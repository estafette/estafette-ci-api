package builderapi

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v4"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/estafette/estafette-ci-api/pkg/api"
	"github.com/estafette/estafette-ci-api/pkg/clients/dockerhubapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/rs/zerolog/log"
)

var (
	// ErrJobNotFound is returned if a job can't be found
	ErrJobNotFound = errors.New("The job can't be found")
)

// Client is the interface for running kubernetes commands specific to this application
//
//go:generate mockgen -package=builderapi -destination ./mock.go -source=client.go
type Client interface {
	CreateCiBuilderJob(ctx context.Context, params CiBuilderParams) (job *batchv1.Job, err error)
	RemoveCiBuilderJob(ctx context.Context, jobName string) (err error)
	CancelCiBuilderJob(ctx context.Context, jobName string) (err error)
	RemoveCiBuilderConfigMap(ctx context.Context, configmapName string) (err error)
	RemoveCiBuilderSecret(ctx context.Context, secretName string) (err error)
	RemoveCiBuilderImagePullSecret(ctx context.Context, secretName string) (err error)
	TailCiBuilderJobLogs(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) (err error)
	GetJobName(ctx context.Context, jobType contracts.JobType, repoOwner, repoName, id string) (jobname string)
}

// NewClient returns a new estafette.Client
func NewClient(config *api.APIConfig, encryptedConfig *api.APIConfig, secretHelper crypt.SecretHelper, kubeClientset *kubernetes.Clientset, dockerHubClient dockerhubapi.Client) Client {
	return &client{
		kubeClientset:   kubeClientset,
		dockerHubClient: dockerHubClient,
		config:          config,
		encryptedConfig: encryptedConfig,
		secretHelper:    secretHelper,
	}
}

type client struct {
	kubeClientset   *kubernetes.Clientset
	dockerHubClient dockerhubapi.Client
	config          *api.APIConfig
	encryptedConfig *api.APIConfig
	secretHelper    crypt.SecretHelper
}

// CreateCiBuilderJob creates an estafette-ci-builder job in Kubernetes to run the estafette build
func (c *client) CreateCiBuilderJob(ctx context.Context, ciBuilderParams CiBuilderParams) (job *batchv1.Job, err error) {

	if err := ciBuilderParams.BuilderConfig.Validate(); err != nil {
		return nil, err
	}

	// create job name of max 63 chars
	jobName := c.getCiBuilderJobName(ctx, ciBuilderParams)

	log.Debug().Msgf("Creating job %v...", jobName)

	// check # of found secrets
	manifestBytes, err := json.Marshal(ciBuilderParams.BuilderConfig.Manifest)
	if err == nil {
		c.inspectSecrets(c.secretHelper, string(manifestBytes), ciBuilderParams.GetFullRepoPath(), "manifest")
	}

	// extend builder config to parameterize the builder and replace all other envvars to improve security
	localBuilderConfig, err := c.getBuilderConfig(ctx, ciBuilderParams, jobName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed creating job %v builder config...", jobName)
	}
	builderConfigJSONBytes, err := json.Marshal(localBuilderConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed marshalling job %v builder config...", jobName)
	}
	builderConfigValue := string(builderConfigJSONBytes)

	// check # of found secrets
	c.inspectSecrets(c.secretHelper, builderConfigValue, ciBuilderParams.GetFullRepoPath(), "builderconfig before reencrypting")

	builderConfigValue, newKey, err := c.secretHelper.ReencryptAllEnvelopes(builderConfigValue, ciBuilderParams.GetFullRepoPath(), false)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed re-encrypting job %v builder config secrets...", jobName)
	}

	// check # of found secrets
	c.inspectSecrets(crypt.NewSecretHelper(newKey, false), builderConfigValue, ciBuilderParams.GetFullRepoPath(), "builderconfig after reencrypting")

	// create configmap for builder config
	err = c.createCiBuilderConfigMap(ctx, ciBuilderParams, jobName, builderConfigValue)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed creating job %v configmap...", jobName)
	}

	// create secret for decryption key secret
	err = c.createCiBuilderSecret(ctx, ciBuilderParams, jobName, newKey)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed creating job %v secret...", jobName)
	}

	createImagePullSecret, err := c.createCiBuilderImagePullSecret(ctx, ciBuilderParams, jobName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed creating job %v image pull secret...", jobName)
	}

	// other job config
	repository := "estafette/estafette-ci-builder"
	tag := *ciBuilderParams.BuilderConfig.Track
	image := fmt.Sprintf("%v:%v", repository, tag)
	imagePullPolicy := v1.PullAlways
	if ciBuilderParams.OperatingSystem != manifest.OperatingSystemWindows {
		digest, err := c.dockerHubClient.GetDigestCached(ctx, repository, tag)
		if err == nil && digest.Digest != "" {
			image = fmt.Sprintf("%v@%v", repository, digest.Digest)
			imagePullPolicy = v1.PullIfNotPresent
		}
	}
	privileged := true

	volumes, volumeMounts := c.getCiBuilderJobVolumesAndMounts(ctx, ciBuilderParams, localBuilderConfig, jobName)

	labels := map[string]string{
		"createdBy": "estafette",
		"jobType":   string(ciBuilderParams.BuilderConfig.JobType),
	}

	terminationGracePeriodSeconds := int64(120)

	job = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: c.config.Jobs.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"cluster-autoscaler.kubernetes.io/safe-to-evict": "false",
			},
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"cluster-autoscaler.kubernetes.io/safe-to-evict": "false",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName:            c.config.Jobs.ServiceAccountName,
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []v1.Container{
						{
							Name:            "estafette-ci-builder",
							Image:           image,
							ImagePullPolicy: imagePullPolicy,
							Args: []string{
								"--run-as-job",
							},
							Env: c.getCiBuilderJobEnvironmentVariables(ctx, ciBuilderParams, localBuilderConfig),
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
							},
							Resources:    c.getCiBuilderJobResources(ctx, ciBuilderParams),
							VolumeMounts: volumeMounts,
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
					Volumes:       volumes,
					Affinity:      c.getCiBuilderJobAffinity(ctx, ciBuilderParams, localBuilderConfig),
					Tolerations:   c.getCiBuilderJobTolerations(ctx, ciBuilderParams, localBuilderConfig),
				},
			},
		},
	}

	if createImagePullSecret {
		job.Spec.Template.Spec.ImagePullSecrets = append(job.Spec.Template.Spec.ImagePullSecrets, v1.LocalObjectReference{
			Name: c.getImagePullSecretName(jobName),
		})
	}

	job, err = c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return job, errors.Wrapf(err, "Failed creating job %v job...", jobName)
	}

	log.Debug().Msgf("Job %v is created", jobName)

	return
}

// RemoveCiBuilderJob waits for a job to finish and then removes it
func (c *client) RemoveCiBuilderJob(ctx context.Context, jobName string) (err error) {

	log.Debug().Msgf("Removing job %v after completion...", jobName)

	// check if job exists
	job, err := c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(ErrJobNotFound, "Job %v does not exist: %v", jobName, err)
	}
	if job == nil {
		return errors.Wrapf(ErrJobNotFound, "Job %v does not exist", jobName)
	}

	err = c.awaitCiBuilderJob(ctx, job)
	if err != nil {
		return errors.Wrapf(err, "Failed waiting job %v after completion", jobName)
	}

	err = c.removeCiBuilderJobCore(ctx, job)
	if err != nil {
		return errors.Wrapf(err, "Failed removing job %v after completion", jobName)
	}

	log.Debug().Msgf("Job %v is removed after completion", jobName)

	return
}

// CancelCiBuilderJob removes a job and its pods to cancel a build/release
func (c *client) CancelCiBuilderJob(ctx context.Context, jobName string) (err error) {

	log.Debug().Msgf("Canceling job %v...", jobName)

	// check if job exists
	job, err := c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(ErrJobNotFound, "Job %v does not exist: %v", jobName, err)
	}
	if job == nil {
		return errors.Wrapf(ErrJobNotFound, "Job %v does not exist", jobName)
	}

	err = c.removeCiBuilderJobCore(ctx, job)
	if err != nil {
		return errors.Wrapf(err, "Failed canceling job %v", jobName)
	}

	log.Debug().Msgf("Job %v is canceled", jobName)

	return
}

func (c *client) awaitCiBuilderJob(ctx context.Context, job *batchv1.Job) (err error) {
	// check if job is finished
	if job.Status.Succeeded != 1 {
		log.Debug().Str("jobName", job.Name).Msgf("Job is not done yet, watching job %v to succeed", job.Name)

		// watch for job updates
		timeoutSeconds := int64(300)
		watcher, err := c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector:  fields.OneTermEqualSelector("metadata.name", job.Name).String(),
			TimeoutSeconds: &timeoutSeconds,
		})

		if err != nil {
			log.Warn().Err(err).Str("jobName", job.Name).Msgf("Watcher call for job %v failed, ignoring", job.Name)
		} else {
			// wait for job to succeed
			for {
				event, ok := <-watcher.ResultChan()
				if !ok {
					log.Warn().Msgf("Watcher for job %v is closed, ignoring", job.Name)
					break
				}
				if event.Type == watch.Modified {
					job, ok := event.Object.(*batchv1.Job)
					if !ok {
						log.Warn().Msgf("Watcher for job %v returns event object of incorrect type, ignoring", job.Name)
						break
					}
					if job.Status.Succeeded == 1 {
						break
					}
				}
			}
		}
	}

	return nil
}

func (c *client) createCiBuilderConfigMap(ctx context.Context, ciBuilderParams CiBuilderParams, jobName, builderConfigValue string) (err error) {

	builderConfigConfigmapName := jobName

	configmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builderConfigConfigmapName,
			Namespace: c.config.Jobs.Namespace,
			Labels: map[string]string{
				"createdBy": "estafette",
				"jobType":   string(ciBuilderParams.BuilderConfig.JobType),
			},
		},
		Data: map[string]string{
			"builder-config.json": builderConfigValue,
		},
	}

	_, err = c.kubeClientset.CoreV1().ConfigMaps(c.config.Jobs.Namespace).Create(ctx, configmap, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "Creating configmap %v failed", builderConfigConfigmapName)
	}

	log.Debug().Msgf("Configmap %v is created", builderConfigConfigmapName)

	return nil
}

func (c *client) createCiBuilderSecret(ctx context.Context, ciBuilderParams CiBuilderParams, jobName, newKey string) (err error) {
	decryptionKeySecretName := jobName
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      decryptionKeySecretName,
			Namespace: c.config.Jobs.Namespace,
			Labels: map[string]string{
				"createdBy": "estafette",
				"jobType":   string(ciBuilderParams.BuilderConfig.JobType),
			},
		},
		Data: map[string][]byte{
			"secretDecryptionKey": []byte(newKey),
		},
	}

	_, err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "Creating secret %v failed", decryptionKeySecretName)
	}

	log.Debug().Msgf("Secret %v is created", decryptionKeySecretName)

	return nil
}

func (c *client) createCiBuilderImagePullSecret(ctx context.Context, ciBuilderParams CiBuilderParams, jobName string) (created bool, err error) {

	registryPullCredentials := contracts.GetCredentialsByType(c.config.Credentials, "container-registry-pull")
	if len(registryPullCredentials) == 0 {
		return false, nil
	}

	imagePullSecretName := c.getImagePullSecretName(jobName)

	username := registryPullCredentials[0].AdditionalProperties["username"].(string)
	password := registryPullCredentials[0].AdditionalProperties["password"].(string)

	dockerconfig := map[string]map[string]map[string]string{
		"auths": {
			"https://index.docker.io/v1/": map[string]string{
				"username": username,
				"password": password,
				"auth":     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", username, password))),
			},
		},
	}

	jsonBytes, err := json.Marshal(dockerconfig)
	if err != nil {
		return
	}

	// create image pull secret
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imagePullSecretName,
			Namespace: c.config.Jobs.Namespace,
			Labels: map[string]string{
				"createdBy": "estafette",
				"jobType":   string(ciBuilderParams.BuilderConfig.JobType),
			},
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": jsonBytes,
		},
	}

	_, err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "Creating secret %v failed", imagePullSecretName)
	}

	log.Debug().Msgf("Secret %v is created", imagePullSecretName)

	return true, nil
}

func (c *client) RemoveCiBuilderConfigMap(ctx context.Context, jobName string) (err error) {

	configmapName := jobName

	// check if configmap exists
	_, err = c.kubeClientset.CoreV1().ConfigMaps(c.config.Jobs.Namespace).Get(ctx, configmapName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Get call for configmap %v failed", configmapName)
	}

	// delete configmap
	propagationPolicy := metav1.DeletePropagationForeground
	err = c.kubeClientset.CoreV1().ConfigMaps(c.config.Jobs.Namespace).Delete(ctx, configmapName, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return errors.Wrapf(err, "Deleting configmap %v failed", configmapName)
	}

	log.Debug().Msgf("Configmap %v is deleted", configmapName)

	return
}

func (c *client) RemoveCiBuilderSecret(ctx context.Context, jobName string) (err error) {

	secretName := jobName

	// check if secret exists
	_, err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Get call for secret %v failed", secretName)
	}

	// delete secret
	propagationPolicy := metav1.DeletePropagationForeground
	err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Delete(ctx, secretName, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return errors.Wrapf(err, "Deleting secret %v failed", secretName)
	}

	log.Debug().Msgf("Secret %v is deleted", secretName)

	return
}

func (c *client) RemoveCiBuilderImagePullSecret(ctx context.Context, jobName string) (err error) {

	secretName := c.getImagePullSecretName(jobName)

	// check if secret exists
	_, err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Get call for secret %v failed", secretName)
	}

	// delete secret
	propagationPolicy := metav1.DeletePropagationForeground
	err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Delete(ctx, secretName, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return errors.Wrapf(err, "Deleting secret %v failed", secretName)
	}

	log.Debug().Msgf("Secret %v is deleted", secretName)

	return
}

func (c *client) removeCiBuilderJobCore(ctx context.Context, job *batchv1.Job) (err error) {

	// delete job
	propagationPolicy := metav1.DeletePropagationForeground
	removeJobErr := c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	removeConfigmapErr := c.RemoveCiBuilderConfigMap(ctx, job.Name)
	removeSecretErr := c.RemoveCiBuilderSecret(ctx, job.Name)
	removeImagePullSecretErr := c.RemoveCiBuilderImagePullSecret(ctx, job.Name)

	if removeJobErr != nil {
		return errors.Wrapf(removeJobErr, "Deleting job %v failed", job.Name)
	}

	if removeConfigmapErr != nil {
		return errors.Wrapf(removeConfigmapErr, "Removing configmap for job %v failed", job.Name)
	}

	if removeSecretErr != nil {
		return errors.Wrapf(removeSecretErr, "Removing secret for job %v failed", job.Name)
	}

	if removeImagePullSecretErr != nil {
		return errors.Wrapf(removeImagePullSecretErr, "Removing image pull secret for job %v failed", job.Name)
	}

	return nil
}

// TailCiBuilderJobLogs tails logs of a running job
func (c *client) TailCiBuilderJobLogs(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) (err error) {

	// close channel so api handler can finish it's response
	defer close(logChannel)

	log.Debug().Msgf("TailCiBuilderJobLogs - listing pods with job-name=%v namespace=%v", jobName, c.config.Jobs.Namespace)

	labelSelector := labels.Set{
		"job-name": jobName,
	}
	pods, err := c.kubeClientset.CoreV1().Pods(c.config.Jobs.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})

	log.Debug().Msgf("TailCiBuilderJobLogs - retrieved %v pods", len(pods.Items))
	for _, pod := range pods.Items {
		err = c.waitIfPodIsPending(ctx, labelSelector, &pod, jobName)
		if err != nil {
			return
		}

		if pod.Status.Phase != v1.PodRunning {
			log.Warn().Msgf("TailCiBuilderJobLogs - pod %v for job %v has unsupported phase %v", pod.Name, jobName, pod.Status.Phase)
			continue
		}

		err = c.followPodLogs(ctx, &pod, jobName, logChannel)
		if err != nil {
			return
		}
	}

	log.Debug().Msgf("TailCiBuilderJobLogs - done following logs stream for all %v pods for job %v", len(pods.Items), jobName)

	return
}

func (c *client) waitIfPodIsPending(ctx context.Context, labelSelector labels.Set, pod *v1.Pod, jobName string) (err error) {

	if pod.Status.Phase == v1.PodPending {

		log.Debug().Msg("TailCiBuilderJobLogs - pod is pending, waiting for running state...")

		// watch for pod to go into Running state (or out of Pending state)
		timeoutSeconds := int64(300)
		watcher, err := c.kubeClientset.CoreV1().Pods(c.config.Jobs.Namespace).Watch(ctx, metav1.ListOptions{
			LabelSelector:  labelSelector.String(),
			TimeoutSeconds: &timeoutSeconds,
		})
		if err != nil {
			return err
		}

		for {
			event, ok := <-watcher.ResultChan()
			if !ok {
				log.Warn().Msgf("Watcher for pod with job-name=%v is closed", jobName)
				break
			}
			if event.Type == watch.Modified {
				modifiedPod, ok := event.Object.(*v1.Pod)
				if !ok {
					log.Warn().Msgf("Watcher for pod with job-name=%v returns event object of incorrect type", jobName)
					break
				}
				if modifiedPod.Status.Phase != v1.PodPending {
					*pod = *modifiedPod
					break
				}
			}
		}
	}

	return nil
}

func (c *client) followPodLogs(ctx context.Context, pod *v1.Pod, jobName string, logChannel chan contracts.TailLogLine) (err error) {
	log.Debug().Msg("TailCiBuilderJobLogs - pod has running state...")

	req := c.kubeClientset.CoreV1().Pods(c.config.Jobs.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{
		Follow: true,
	})
	logsStream, err := req.Stream(ctx)
	if err != nil {
		return errors.Wrapf(err, "Failed opening logs stream for pod %v for job %v", pod.Name, jobName)
	}
	defer logsStream.Close()

	reader := bufio.NewReader(logsStream)
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			log.Debug().Msgf("EOF in logs stream for pod %v for job %v, exiting tailing", pod.Name, jobName)
			break
		}
		if err != nil {
			return err
		}

		// only forward if it's a json object with property 'tailLogLine'
		var zeroLogLine ZeroLogLine
		err = json.Unmarshal(line, &zeroLogLine)
		if err == nil {
			if zeroLogLine.TailLogLine != nil {
				logChannel <- *zeroLogLine.TailLogLine
			}
		} else {
			log.Warn().Err(err).Str("line", string(line)).Msgf("Tailed log from pod %v for job %v is not of type json", pod.Name, jobName)
		}
	}
	log.Debug().Msgf("Done following logs stream for pod %v for job %v", pod.Name, jobName)

	return nil
}

// GetJobName returns the job name for a build or release job
func (c *client) GetJobName(ctx context.Context, jobType contracts.JobType, repoOwner, repoName, id string) string {

	// create job name of max 63 chars
	maxJobNameLength := 63

	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	fullRepoName := re.ReplaceAllString(fmt.Sprintf("%v/%v", repoOwner, repoName), "-")

	maxRepoNameLength := maxJobNameLength - len(jobType) - 1 - len(id) - 1
	if len(fullRepoName) > maxRepoNameLength {
		fullRepoName = fullRepoName[:maxRepoNameLength]
	}

	return strings.ToLower(fmt.Sprintf("%v-%v-%v", jobType, fullRepoName, id))
}

func (c *client) getImagePullSecretName(jobName string) string {

	jobName = strings.TrimPrefix(jobName, "release-")
	jobName = strings.TrimPrefix(jobName, "build-")

	return "pull-" + jobName
}

func (c *client) getBuilderConfig(ctx context.Context, ciBuilderParams CiBuilderParams, jobName string) (contracts.BuilderConfig, error) {

	localBuilderConfig := ciBuilderParams.BuilderConfig

	if err := localBuilderConfig.Validate(); err != nil {
		return localBuilderConfig, err
	}

	now := time.Now().UTC()
	expiry := now.Add(time.Duration(6) * time.Hour)

	jwt, err := api.GenerateJWT(c.config, now, expiry, jwtgo.MapClaims{
		"job": jobName,
	})
	if err != nil {
		return contracts.BuilderConfig{}, err
	}

	localBuilderConfig.JobName = &jobName
	localBuilderConfig.CIServer = &contracts.CIServerConfig{
		BaseURL:          c.config.APIServer.BaseURL,
		BuilderEventsURL: strings.TrimRight(c.config.APIServer.ServiceURL, "/") + "/api/commands",
		JWT:              jwt,
		JWTExpiry:        expiry,
	}

	switch localBuilderConfig.JobType {
	case contracts.JobTypeBuild:
		localBuilderConfig.Stages = localBuilderConfig.Manifest.Stages

		localBuilderConfig.CIServer.PostLogsURL = strings.TrimRight(c.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/builds/%v/logs", localBuilderConfig.Git.RepoSource, localBuilderConfig.Git.RepoOwner, localBuilderConfig.Git.RepoName, localBuilderConfig.Build.ID)
		localBuilderConfig.CIServer.CancelJobURL = strings.TrimRight(c.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/builds/%v", localBuilderConfig.Git.RepoSource, localBuilderConfig.Git.RepoOwner, localBuilderConfig.Git.RepoName, localBuilderConfig.Build.ID)

	case contracts.JobTypeRelease:
		releaseExists := false
		for _, r := range localBuilderConfig.Manifest.Releases {
			if r.Name == localBuilderConfig.Release.Name {
				releaseExists = true
				localBuilderConfig.Stages = r.Stages
			}
		}
		if !releaseExists {
			localBuilderConfig.Stages = []*manifest.EstafetteStage{}
		}

		localBuilderConfig.CIServer.PostLogsURL = strings.TrimRight(c.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/releases/%v/logs", localBuilderConfig.Git.RepoSource, localBuilderConfig.Git.RepoOwner, localBuilderConfig.Git.RepoName, localBuilderConfig.Release.ID)
		localBuilderConfig.CIServer.CancelJobURL = strings.TrimRight(c.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/releases/%v", localBuilderConfig.Git.RepoSource, localBuilderConfig.Git.RepoOwner, localBuilderConfig.Git.RepoName, localBuilderConfig.Release.ID)

	case contracts.JobTypeBot:
		botExists := false
		for _, b := range localBuilderConfig.Manifest.Bots {
			if b.Name == localBuilderConfig.Bot.Name {
				botExists = true
				localBuilderConfig.Stages = b.Stages
			}
		}
		if !botExists {
			localBuilderConfig.Stages = []*manifest.EstafetteStage{}
		}

		localBuilderConfig.CIServer.PostLogsURL = strings.TrimRight(c.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/bots/%v/logs", localBuilderConfig.Git.RepoSource, localBuilderConfig.Git.RepoOwner, localBuilderConfig.Git.RepoName, localBuilderConfig.Bot.ID)
		localBuilderConfig.CIServer.CancelJobURL = strings.TrimRight(c.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/bots/%v", localBuilderConfig.Git.RepoSource, localBuilderConfig.Git.RepoOwner, localBuilderConfig.Git.RepoName, localBuilderConfig.Bot.ID)

	}

	// get configured credentials
	credentials := c.encryptedConfig.Credentials

	// add dynamic github api token credential
	if token, ok := ciBuilderParams.EnvironmentVariables["ESTAFETTE_GITHUB_API_TOKEN"]; ok {

		encryptedTokenEnvelope, err := c.secretHelper.EncryptEnvelope(token, crypt.DefaultPipelineAllowList)
		if err != nil {
			return contracts.BuilderConfig{}, err
		}

		credentials = append(credentials, &contracts.CredentialConfig{
			Name: "github-api-token",
			Type: "github-api-token",
			AdditionalProperties: map[string]interface{}{
				"token": encryptedTokenEnvelope,
			},
		})
	}

	// add dynamic bitbucket api token credential
	if token, ok := ciBuilderParams.EnvironmentVariables["ESTAFETTE_BITBUCKET_API_TOKEN"]; ok {

		encryptedTokenEnvelope, err := c.secretHelper.EncryptEnvelope(token, crypt.DefaultPipelineAllowList)
		if err != nil {
			return contracts.BuilderConfig{}, err
		}

		credentials = append(credentials, &contracts.CredentialConfig{
			Name: "bitbucket-api-token",
			Type: "bitbucket-api-token",
			AdditionalProperties: map[string]interface{}{
				"token": encryptedTokenEnvelope,
			},
		})
	}

	// add dynamic cloudsource api token credential
	if token, ok := ciBuilderParams.EnvironmentVariables["ESTAFETTE_CLOUDSOURCE_API_TOKEN"]; ok {

		encryptedTokenEnvelope, err := c.secretHelper.EncryptEnvelope(token, crypt.DefaultPipelineAllowList)
		if err != nil {
			return contracts.BuilderConfig{}, err
		}

		credentials = append(credentials, &contracts.CredentialConfig{
			Name: "cloudsource-api-token",
			Type: "cloudsource-api-token",
			AdditionalProperties: map[string]interface{}{
				"token": encryptedTokenEnvelope,
			},
		})
	}

	// filter to only what's needed by the build/release job
	trustedImages := contracts.FilterTrustedImages(c.encryptedConfig.TrustedImages, localBuilderConfig.Stages, ciBuilderParams.GetFullRepoPath())
	credentials = contracts.FilterCredentials(credentials, trustedImages, ciBuilderParams.GetFullRepoPath(), localBuilderConfig.Git.RepoBranch)

	// add container-registry credentials to allow private registry images to be used in stages
	credentials = contracts.AddCredentialsIfNotPresent(credentials, contracts.FilterCredentialsByPipelinesAllowList(contracts.GetCredentialsByType(c.encryptedConfig.Credentials, "container-registry"), ciBuilderParams.GetFullRepoPath()))
	credentials = contracts.AddCredentialsIfNotPresent(credentials, contracts.FilterCredentialsByPipelinesAllowList(contracts.GetCredentialsByType(c.encryptedConfig.Credentials, "container-registry-pull"), ciBuilderParams.GetFullRepoPath()))

	localBuilderConfig.Credentials = credentials
	localBuilderConfig.TrustedImages = trustedImages

	if localBuilderConfig.Manifest.Version.SemVer != nil {
		versionParams := manifest.EstafetteVersionParams{
			AutoIncrement: localBuilderConfig.Version.CurrentCounter,
			Branch:        localBuilderConfig.Git.RepoBranch,
			Revision:      localBuilderConfig.Git.RepoRevision,
		}
		patchWithLabel := localBuilderConfig.Manifest.Version.SemVer.GetPatchWithLabel(versionParams)
		label := localBuilderConfig.Manifest.Version.SemVer.GetLabel(versionParams)

		localBuilderConfig.Version.Major = &localBuilderConfig.Manifest.Version.SemVer.Major
		localBuilderConfig.Version.Minor = &localBuilderConfig.Manifest.Version.SemVer.Minor
		localBuilderConfig.Version.Patch = &patchWithLabel
		localBuilderConfig.Version.Label = &label
	}

	if c.config != nil && c.config.APIServer != nil && c.config.APIServer.DockerConfigPerOperatingSystem != nil {
		if dc, ok := c.config.APIServer.DockerConfigPerOperatingSystem[ciBuilderParams.OperatingSystem]; ok {

			var copiedDockerConfig contracts.DockerConfig
			err = copier.CopyWithOption(&copiedDockerConfig, dc, copier.Option{IgnoreEmpty: true, DeepCopy: true})
			if err != nil {
				return localBuilderConfig, err
			}

			localBuilderConfig.DockerConfig = &copiedDockerConfig
		}
	}

	return localBuilderConfig, nil
}

func (c *client) getCiBuilderJobEnvironmentVariables(ctx context.Context, ciBuilderParams CiBuilderParams, builderConfig contracts.BuilderConfig) (environmentVariables []v1.EnvVar) {

	// ensure a json log format is used by the ci-builder, so logs can be tailed
	logFormat := os.Getenv("ESTAFETTE_LOG_FORMAT")
	if logFormat != foundation.LogFormatJSON && logFormat != foundation.LogFormatV3 {
		logFormat = foundation.LogFormatJSON
	}

	environmentVariables = []v1.EnvVar{
		{
			Name:  "BUILDER_CONFIG_PATH",
			Value: "/configs/builder-config.json",
		},
		{
			Name:  "ESTAFETTE_LOG_FORMAT",
			Value: logFormat,
		},
		{
			Name: "POD_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "POD_UID",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "POD_NODE_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
	}

	// configure jaeger for ci-builder
	jaegerServiceName := os.Getenv("JAEGER_SERVICE_NAME")
	jaegerDisabled := os.Getenv("JAEGER_DISABLED")

	if jaegerServiceName != "" && jaegerDisabled != "true" {
		environmentVariables = append(environmentVariables, []v1.EnvVar{
			{
				Name:  "JAEGER_SERVICE_NAME",
				Value: "estafette-ci-builder",
			},
			{
				Name: "JAEGER_AGENT_HOST",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
			{
				Name:  "JAEGER_SAMPLER_TYPE",
				Value: "const",
			},
			{
				Name:  "JAEGER_SAMPLER_PARAM",
				Value: "1",
			},
		}...)
	} else {
		environmentVariables = append(environmentVariables,
			v1.EnvVar{
				Name:  "JAEGER_DISABLED",
				Value: "true",
			})
	}

	// forward all envars prefixed with JAEGER_ to builder job except for the ones already set above
	for _, e := range os.Environ() {
		kvPair := strings.SplitN(e, "=", 2)

		if len(kvPair) == 2 {
			envvarName := kvPair[0]
			envvarValue := kvPair[1]

			if strings.HasPrefix(envvarName, "JAEGER_") && envvarName != "JAEGER_SERVICE_NAME" && envvarName != "JAEGER_AGENT_HOST" && envvarName != "JAEGER_SAMPLER_TYPE" && envvarName != "JAEGER_SAMPLER_PARAM" && envvarName != "JAEGER_DISABLED" && envvarValue != "" {
				environmentVariables = append(environmentVariables, v1.EnvVar{
					Name:  envvarName,
					Value: envvarValue,
				})
			}
		}
	}

	if ciBuilderParams.OperatingSystem == manifest.OperatingSystemWindows || ciBuilderParams.BuilderConfig.Manifest.Builder.BuilderType == manifest.BuilderTypeKubernetes {
		workingDirectoryVolumeName := "working-directory"
		tempDirectoryVolumeName := "temp-directory"

		// this is the path on the host mounted into any of the stage containers; with docker-outside-docker the daemon can't see paths inside the ci-builder container
		estafetteWorkdirName := "ESTAFETTE_WORKDIR"
		estafetteWorkdirValue := "/var/lib/kubelet/pods/$(POD_UID)/volumes/kubernetes.io~empty-dir/" + workingDirectoryVolumeName
		if ciBuilderParams.OperatingSystem == manifest.OperatingSystemWindows {
			estafetteWorkdirValue = "c:" + estafetteWorkdirValue
		}
		environmentVariables = append(environmentVariables,
			v1.EnvVar{
				Name:  estafetteWorkdirName,
				Value: estafetteWorkdirValue,
			},
		)

		tempWorkdirName := "ESTAFETTE_TEMPDIR"
		tempWorkdirValue := "/var/lib/kubelet/pods/$(POD_UID)/volumes/kubernetes.io~empty-dir/" + tempDirectoryVolumeName
		if ciBuilderParams.OperatingSystem == manifest.OperatingSystemWindows {
			tempWorkdirValue = "c:" + tempWorkdirValue
		}
		environmentVariables = append(environmentVariables,
			v1.EnvVar{
				Name:  tempWorkdirName,
				Value: tempWorkdirValue,
			},
		)
	}

	return environmentVariables
}

func (c *client) getCiBuilderJobVolumesAndMounts(ctx context.Context, ciBuilderParams CiBuilderParams, builderConfig contracts.BuilderConfig, jobName string) (volumes []v1.Volume, volumeMounts []v1.VolumeMount) {

	storageMedium := v1.StorageMediumDefault
	if ciBuilderParams.BuilderConfig.Manifest.Builder.StorageMedium == manifest.StorageMediumMemory {
		storageMedium = v1.StorageMediumMemory
	}

	workingDirectoryVolumeMountPath := "/estafette-work"
	if ciBuilderParams.OperatingSystem == manifest.OperatingSystemWindows {
		workingDirectoryVolumeMountPath = "c:" + workingDirectoryVolumeMountPath
	}

	tempDirectoryVolumeMountPath := "/tmp"
	if ciBuilderParams.OperatingSystem == manifest.OperatingSystemWindows {
		tempDirectoryVolumeMountPath = "C:/Windows/TEMP"
	}

	volumes = []v1.Volume{
		{
			Name: "app-configs",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: jobName,
					},
				},
			},
		},
		{
			Name: "app-secret",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: jobName,
				},
			},
		},
		{
			Name: "working-directory",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium: storageMedium,
				},
			},
		},
		{
			Name: "temp-directory",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium: storageMedium,
				},
			},
		},
	}

	volumeMounts = []v1.VolumeMount{
		{
			Name:      "app-configs",
			MountPath: "/configs",
		},
		{
			Name:      "app-secret",
			MountPath: "/secrets",
		},
		{
			Name:      "working-directory",
			MountPath: workingDirectoryVolumeMountPath,
		},
		{
			Name:      "temp-directory",
			MountPath: tempDirectoryVolumeMountPath,
		},
	}

	if builderConfig.DockerConfig != nil && builderConfig.DockerConfig.RunType == contracts.DockerRunTypeDinD {
		volumes = append(volumes, v1.Volume{
			Name: "docker-graph-storage",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "docker-graph-storage",
			MountPath: "/var/lib/docker",
		})
	}

	if ciBuilderParams.OperatingSystem == manifest.OperatingSystemWindows && ciBuilderParams.BuilderConfig.Manifest.Builder.BuilderType != manifest.BuilderTypeKubernetes {
		// windows builds uses docker-outside-docker, for which the hosts docker socket needs to be mounted into the ci-builder container
		dockerSocketVolumeName := "docker-socket"
		dockerSocketVolumeHostPath := `\\.\pipe\docker_engine`
		volumes = append(volumes, v1.Volume{
			Name: dockerSocketVolumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: dockerSocketVolumeHostPath,
				},
			},
		})

		dockerSocketVolumeMountPath := `\\.\pipe\docker_engine`
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      dockerSocketVolumeName,
			MountPath: dockerSocketVolumeMountPath,
		})

		// in order not to have to install the docker cli into the ci-builder container it's mounted from the host as well
		dockerCLIVolumeName := "docker-cli"
		dockerCLIVolumeHostPath := `C:/Program Files/Docker`
		volumes = append(volumes, v1.Volume{
			Name: dockerCLIVolumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: dockerCLIVolumeHostPath,
				},
			},
		})

		dockerCLIVolumeMountPath := `C:/Program Files/Docker`
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      dockerCLIVolumeName,
			MountPath: dockerCLIVolumeMountPath,
		})
	}

	return volumes, volumeMounts
}

func (c *client) getCiBuilderJobTolerations(ctx context.Context, ciBuilderParams CiBuilderParams, builderConfig contracts.BuilderConfig) (tolerations []v1.Toleration) {
	tolerations = []v1.Toleration{}

	switch ciBuilderParams.BuilderConfig.JobType {
	case contracts.JobTypeBuild:
		if c.config.Jobs.BuildAffinityAndTolerations != nil && c.config.Jobs.BuildAffinityAndTolerations.Tolerations != nil && len(c.config.Jobs.BuildAffinityAndTolerations.Tolerations) > 0 {
			tolerations = append(tolerations, c.config.Jobs.BuildAffinityAndTolerations.Tolerations...)
		}
	case contracts.JobTypeRelease:
		if c.config.Jobs.ReleaseAffinityAndTolerations != nil && c.config.Jobs.ReleaseAffinityAndTolerations.Tolerations != nil && len(c.config.Jobs.ReleaseAffinityAndTolerations.Tolerations) > 0 {
			tolerations = append(tolerations, c.config.Jobs.ReleaseAffinityAndTolerations.Tolerations...)
		}
	case contracts.JobTypeBot:
		if c.config.Jobs.BotAffinityAndTolerations != nil && c.config.Jobs.BotAffinityAndTolerations.Tolerations != nil && len(c.config.Jobs.BotAffinityAndTolerations.Tolerations) > 0 {
			tolerations = append(tolerations, c.config.Jobs.BotAffinityAndTolerations.Tolerations...)
		}
	}

	// ensure it runs on a windows node
	if ciBuilderParams.OperatingSystem == manifest.OperatingSystemWindows {
		tolerations = append(tolerations, v1.Toleration{
			Effect:   v1.TaintEffectNoSchedule,
			Key:      "node.kubernetes.io/os",
			Operator: v1.TolerationOpEqual,
			Value:    "windows",
		})
	}

	return tolerations
}

func (c *client) getCiBuilderJobAffinity(ctx context.Context, ciBuilderParams CiBuilderParams, builderConfig contracts.BuilderConfig) (affinity *v1.Affinity) {

	switch ciBuilderParams.BuilderConfig.JobType {
	case contracts.JobTypeBuild:
		if c.config.Jobs.BuildAffinityAndTolerations != nil && c.config.Jobs.BuildAffinityAndTolerations.Affinity != nil {

			var deepcopyAffinity v1.Affinity
			err := copier.CopyWithOption(&deepcopyAffinity, *c.config.Jobs.BuildAffinityAndTolerations.Affinity, copier.Option{IgnoreEmpty: true, DeepCopy: true})
			if err != nil {
				log.Warn().Err(err).Msgf("Failed creating deep copy of build job affinity")
			}

			affinity = &deepcopyAffinity
		}
	case contracts.JobTypeRelease:
		if c.config.Jobs.ReleaseAffinityAndTolerations != nil && c.config.Jobs.ReleaseAffinityAndTolerations.Affinity != nil {

			var deepcopyAffinity v1.Affinity
			err := copier.CopyWithOption(&deepcopyAffinity, *c.config.Jobs.ReleaseAffinityAndTolerations.Affinity, copier.Option{IgnoreEmpty: true, DeepCopy: true})
			if err != nil {
				log.Warn().Err(err).Msgf("Failed creating deep copy of release job affinity")
			}

			affinity = &deepcopyAffinity
		}
	case contracts.JobTypeBot:
		if c.config.Jobs.BotAffinityAndTolerations != nil && c.config.Jobs.BotAffinityAndTolerations.Affinity != nil {

			var deepcopyAffinity v1.Affinity
			err := copier.CopyWithOption(&deepcopyAffinity, *c.config.Jobs.BotAffinityAndTolerations.Affinity, copier.Option{IgnoreEmpty: true, DeepCopy: true})
			if err != nil {
				log.Warn().Err(err).Msgf("Failed creating deep copy of bot job affinity")
			}

			affinity = &deepcopyAffinity
		}
	}

	if ciBuilderParams.OperatingSystem == manifest.OperatingSystemWindows {
		if affinity == nil {
			affinity = &v1.Affinity{}
		}
		if affinity.NodeAffinity == nil {
			affinity.NodeAffinity = &v1.NodeAffinity{}
		}
		if affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{}
		}
		if affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms == nil {
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []v1.NodeSelectorTerm{}
		}
		if len(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) == 0 {
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, v1.NodeSelectorTerm{
				MatchExpressions: []v1.NodeSelectorRequirement{},
			})
		}

		// ensure it runs on a windows node
		operatingSystemAffinityKey := "kubernetes.io/os"
		operatingSystemAffinityValue := ciBuilderParams.OperatingSystem

		affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions = append(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions, v1.NodeSelectorRequirement{
			Key:      operatingSystemAffinityKey,
			Operator: v1.NodeSelectorOpIn,
			Values:   []string{string(operatingSystemAffinityValue)},
		})
	}

	return
}

func (c *client) getCiBuilderJobResources(ctx context.Context, ciBuilderParams CiBuilderParams) (resources v1.ResourceRequirements) {

	// define resource request and limit values from job resources struct, so we can autotune later on
	cpuRequest := fmt.Sprintf("%f", ciBuilderParams.JobResources.CPURequest)
	cpuLimit := fmt.Sprintf("%f", ciBuilderParams.JobResources.CPULimit)
	memoryRequest := fmt.Sprintf("%.0f", ciBuilderParams.JobResources.MemoryRequest)
	memoryLimit := fmt.Sprintf("%.0f", ciBuilderParams.JobResources.MemoryLimit)

	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			"cpu":    resource.MustParse(cpuRequest),
			"memory": resource.MustParse(memoryRequest),
		},
		Limits: v1.ResourceList{
			"cpu":    resource.MustParse(cpuLimit),
			"memory": resource.MustParse(memoryLimit),
		},
	}
}

func (c *client) getCiBuilderJobName(ctx context.Context, ciBuilderParams CiBuilderParams) string {

	var id string
	switch ciBuilderParams.BuilderConfig.JobType {
	case contracts.JobTypeBuild:
		id = ciBuilderParams.BuilderConfig.Build.ID
	case contracts.JobTypeRelease:
		id = ciBuilderParams.BuilderConfig.Release.ID
	case contracts.JobTypeBot:
		id = ciBuilderParams.BuilderConfig.Bot.ID
	}

	return c.GetJobName(ctx, ciBuilderParams.BuilderConfig.JobType, ciBuilderParams.BuilderConfig.Git.RepoOwner, ciBuilderParams.BuilderConfig.Git.RepoName, id)
}

func (c *client) inspectSecrets(secretHelper crypt.SecretHelper, input, pipeline, when string) {
	values, err := secretHelper.GetAllSecretValues(input, pipeline)
	if err == nil {
		log.Debug().Msgf("[%v] Collected %v secrets for pipeline %v...", when, len(values), pipeline)
	} else {
		log.Debug().Err(err).Msgf("[%v] Failed collecting %v secrets for pipeline %v...", when, len(values), pipeline)
	}
}
