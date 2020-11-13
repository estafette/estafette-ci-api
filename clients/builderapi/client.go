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
	"strconv"
	"strings"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/estafette/estafette-ci-api/api"
	"github.com/estafette/estafette-ci-api/clients/dockerhubapi"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

var (
	// ErrJobNotFound is returned if a job can't be found
	ErrJobNotFound = errors.New("The job can't be found")
)

// Client is the interface for running kubernetes commands specific to this application
type Client interface {
	CreateCiBuilderJob(ctx context.Context, params CiBuilderParams) (job *batchv1.Job, err error)
	RemoveCiBuilderJob(ctx context.Context, jobName string) (err error)
	CancelCiBuilderJob(ctx context.Context, jobName string) (err error)
	RemoveCiBuilderConfigMap(ctx context.Context, configmapName string) (err error)
	RemoveCiBuilderSecret(ctx context.Context, secretName string) (err error)
	RemoveCiBuilderImagePullSecret(ctx context.Context, secretName string) (err error)
	TailCiBuilderJobLogs(ctx context.Context, jobName string, logChannel chan contracts.TailLogLine) (err error)
	GetJobName(ctx context.Context, jobType, repoOwner, repoName, id string) (jobname string)
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

	// create job name of max 63 chars
	jobName := c.getCiBuilderJobName(ctx, ciBuilderParams)

	log.Info().Msgf("Creating job %v...", jobName)

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
	builderConfigValue, newKey, err := c.secretHelper.ReencryptAllEnvelopes(builderConfigValue, ciBuilderParams.GetFullRepoPath(), false)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed re-encrypting job %v builder config secrets...", jobName)
	}

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
	tag := ciBuilderParams.Track
	image := fmt.Sprintf("%v:%v", repository, tag)
	imagePullPolicy := v1.PullAlways
	digest, err := c.dockerHubClient.GetDigestCached(ctx, repository, tag)
	if err == nil && digest.Digest != "" {
		image = fmt.Sprintf("%v@%v", repository, digest.Digest)
		imagePullPolicy = v1.PullIfNotPresent
	}
	privileged := true

	volumes, volumeMounts := c.getCiBuilderJobVolumesAndMounts(ctx, ciBuilderParams, jobName)

	labels := map[string]string{
		"createdBy": "estafette",
		"jobType":   ciBuilderParams.JobType,
	}

	job = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: c.config.Jobs.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "estafette-ci-builder",
							Image:           image,
							ImagePullPolicy: imagePullPolicy,
							Args: []string{
								"--run-as-job",
							},
							Env: c.getCiBuilderJobEnvironmentVariables(ctx, ciBuilderParams),
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
							},
							Resources:    c.getCiBuilderJobResources(ctx, ciBuilderParams),
							VolumeMounts: volumeMounts,
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
					Volumes:       volumes,
					Affinity:      c.getCiBuilderJobAffinity(ctx, ciBuilderParams),
					Tolerations:   c.getCiBuilderJobTolerations(ctx, ciBuilderParams),
				},
			},
		},
	}

	if createImagePullSecret {
		job.Spec.Template.Spec.ImagePullSecrets = append(job.Spec.Template.Spec.ImagePullSecrets, v1.LocalObjectReference{
			Name: c.getImagePullSecretName(jobName),
		})
	}

	job, err = c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Create(job)
	if err != nil {
		return job, errors.Wrapf(err, "Failed creating job %v job...", jobName)
	}

	log.Info().Msgf("Job %v is created", jobName)

	return
}

// RemoveCiBuilderJob waits for a job to finish and then removes it
func (c *client) RemoveCiBuilderJob(ctx context.Context, jobName string) (err error) {

	log.Info().Msgf("Removing job %v after completion...", jobName)

	// check if job exists
	job, err := c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Get(jobName, metav1.GetOptions{})
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

	log.Info().Msgf("Job %v is removed after completion", jobName)

	return
}

// CancelCiBuilderJob removes a job and its pods to cancel a build/release
func (c *client) CancelCiBuilderJob(ctx context.Context, jobName string) (err error) {

	log.Info().Msgf("Canceling job %v...", jobName)

	// check if job exists
	job, err := c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Get(jobName, metav1.GetOptions{})
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

	log.Info().Msgf("Job %v is canceled", jobName)

	return
}

func (c *client) awaitCiBuilderJob(ctx context.Context, job *batchv1.Job) (err error) {
	// check if job is finished
	if job.Status.Succeeded != 1 {
		log.Debug().Str("jobName", job.Name).Msgf("Job is not done yet, watching job %v to succeed", job.Name)

		// watch for job updates
		timeoutSeconds := int64(300)
		watcher, err := c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Watch(metav1.ListOptions{
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
				"jobType":   ciBuilderParams.JobType,
			},
		},
		Data: map[string]string{
			"builder-config.json": builderConfigValue,
		},
	}

	_, err = c.kubeClientset.CoreV1().ConfigMaps(c.config.Jobs.Namespace).Create(configmap)
	if err != nil {
		return errors.Wrapf(err, "Creating configmap %v failed", builderConfigConfigmapName)
	}

	log.Info().Msgf("Configmap %v is created", builderConfigConfigmapName)

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
				"jobType":   ciBuilderParams.JobType,
			},
		},
		Data: map[string][]byte{
			"secretDecryptionKey": []byte(newKey),
		},
	}

	_, err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Create(secret)
	if err != nil {
		return errors.Wrapf(err, "Creating secret %v failed", decryptionKeySecretName)
	}

	log.Info().Msgf("Secret %v is created", decryptionKeySecretName)

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
				"jobType":   ciBuilderParams.JobType,
			},
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": jsonBytes,
		},
	}

	_, err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Create(secret)
	if err != nil {
		return false, errors.Wrapf(err, "Creating secret %v failed", imagePullSecretName)
	}

	log.Info().Msgf("Secret %v is created", imagePullSecretName)

	return true, nil
}

func (c *client) RemoveCiBuilderConfigMap(ctx context.Context, jobName string) (err error) {

	configmapName := jobName

	// check if configmap exists
	_, err = c.kubeClientset.CoreV1().ConfigMaps(c.config.Jobs.Namespace).Get(configmapName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Get call for configmap %v failed", configmapName)
	}

	// delete configmap
	err = c.kubeClientset.CoreV1().ConfigMaps(c.config.Jobs.Namespace).Delete(configmapName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "Deleting configmap %v failed", configmapName)
	}

	log.Info().Msgf("Configmap %v is deleted", configmapName)

	return
}

func (c *client) RemoveCiBuilderSecret(ctx context.Context, jobName string) (err error) {

	secretName := jobName

	// check if secret exists
	_, err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Get call for secret %v failed", secretName)
	}

	// delete secret
	err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Delete(secretName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "Deleting secret %v failed", secretName)
	}

	log.Info().Msgf("Secret %v is deleted", secretName)

	return
}

func (c *client) RemoveCiBuilderImagePullSecret(ctx context.Context, jobName string) (err error) {

	secretName := c.getImagePullSecretName(jobName)

	// check if secret exists
	_, err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Get call for secret %v failed", secretName)
	}

	// delete secret
	err = c.kubeClientset.CoreV1().Secrets(c.config.Jobs.Namespace).Delete(secretName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "Deleting secret %v failed", secretName)
	}

	log.Info().Msgf("Secret %v is deleted", secretName)

	return
}

func (c *client) removeCiBuilderJobCore(ctx context.Context, job *batchv1.Job) (err error) {

	// delete job
	removeJobErr := c.kubeClientset.BatchV1().Jobs(c.config.Jobs.Namespace).Delete(job.Name, &metav1.DeleteOptions{})
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
	pods, err := c.kubeClientset.CoreV1().Pods(c.config.Jobs.Namespace).List(metav1.ListOptions{
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
		watcher, err := c.kubeClientset.CoreV1().Pods(c.config.Jobs.Namespace).Watch(metav1.ListOptions{
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
	logsStream, err := req.Stream()
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
			log.Warn().Err(err).Msgf("Error while reading lines from logs from pod %v for job %v", pod.Name, jobName)
			continue
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
func (c *client) GetJobName(ctx context.Context, jobType, repoOwner, repoName, id string) string {

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

	// retrieve stages to filter trusted images and credentials
	stages := ciBuilderParams.Manifest.Stages
	if ciBuilderParams.JobType == "release" {

		releaseExists := false
		for _, r := range ciBuilderParams.Manifest.Releases {
			if r.Name == ciBuilderParams.ReleaseName {
				releaseExists = true
				stages = r.Stages
			}
		}
		if !releaseExists {
			stages = []*manifest.EstafetteStage{}
		}
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
	trustedImages := contracts.FilterTrustedImages(c.encryptedConfig.TrustedImages, stages, ciBuilderParams.GetFullRepoPath())
	credentials = contracts.FilterCredentials(credentials, trustedImages, ciBuilderParams.GetFullRepoPath())

	// add container-registry credentials to allow private registry images to be used in stages
	credentials = contracts.AddCredentialsIfNotPresent(credentials, contracts.FilterCredentialsByPipelinesAllowList(contracts.GetCredentialsByType(c.encryptedConfig.Credentials, "container-registry"), ciBuilderParams.GetFullRepoPath()))
	credentials = contracts.AddCredentialsIfNotPresent(credentials, contracts.FilterCredentialsByPipelinesAllowList(contracts.GetCredentialsByType(c.encryptedConfig.Credentials, "container-registry-pull"), ciBuilderParams.GetFullRepoPath()))

	localBuilderConfig := contracts.BuilderConfig{
		Credentials:     credentials,
		TrustedImages:   trustedImages,
		RegistryMirror:  c.config.RegistryMirror,
		DockerDaemonMTU: c.config.DockerDaemonMTU,
		DockerDaemonBIP: c.config.DockerDaemonBIP,
		DockerNetwork:   c.config.DockerNetwork,
	}

	localBuilderConfig.Action = &ciBuilderParams.JobType
	localBuilderConfig.Track = &ciBuilderParams.Track
	localBuilderConfig.Git = &contracts.GitConfig{
		RepoSource:   ciBuilderParams.RepoSource,
		RepoOwner:    ciBuilderParams.RepoOwner,
		RepoName:     ciBuilderParams.RepoName,
		RepoBranch:   ciBuilderParams.RepoBranch,
		RepoRevision: ciBuilderParams.RepoRevision,
	}
	if ciBuilderParams.Manifest.Version.SemVer != nil {
		versionParams := manifest.EstafetteVersionParams{
			AutoIncrement: ciBuilderParams.CurrentCounter,
			Branch:        ciBuilderParams.RepoBranch,
			Revision:      ciBuilderParams.RepoRevision,
		}
		patchWithLabel := ciBuilderParams.Manifest.Version.SemVer.GetPatchWithLabel(versionParams)
		label := ciBuilderParams.Manifest.Version.SemVer.GetLabel(versionParams)
		localBuilderConfig.BuildVersion = &contracts.BuildVersionConfig{
			Version:       ciBuilderParams.VersionNumber,
			Major:         &ciBuilderParams.Manifest.Version.SemVer.Major,
			Minor:         &ciBuilderParams.Manifest.Version.SemVer.Minor,
			Patch:         &patchWithLabel,
			Label:         &label,
			AutoIncrement: &ciBuilderParams.CurrentCounter,

			// set counters to enable release locking for older revisions
			CurrentCounter:          ciBuilderParams.CurrentCounter,
			MaxCounter:              ciBuilderParams.MaxCounter,
			MaxCounterCurrentBranch: ciBuilderParams.MaxCounterCurrentBranch,
		}
	} else {
		localBuilderConfig.BuildVersion = &contracts.BuildVersionConfig{
			Version:       ciBuilderParams.VersionNumber,
			AutoIncrement: &ciBuilderParams.CurrentCounter,

			// set counters to enable release locking for older revisions
			CurrentCounter:          ciBuilderParams.CurrentCounter,
			MaxCounter:              ciBuilderParams.MaxCounter,
			MaxCounterCurrentBranch: ciBuilderParams.MaxCounterCurrentBranch,
		}
	}

	localBuilderConfig.Manifest = &ciBuilderParams.Manifest

	jwt, err := api.GenerateJWT(c.config, time.Duration(6)*time.Hour, jwtgo.MapClaims{
		"job": jobName,
	})
	if err != nil {
		return contracts.BuilderConfig{}, err
	}

	localBuilderConfig.JobName = &jobName
	localBuilderConfig.CIServer = &contracts.CIServerConfig{
		BaseURL:          c.config.APIServer.BaseURL,
		BuilderEventsURL: strings.TrimRight(c.config.APIServer.ServiceURL, "/") + "/api/commands",
		PostLogsURL:      strings.TrimRight(c.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/builds/%v/logs", ciBuilderParams.RepoSource, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, ciBuilderParams.BuildID),
		JWT:              jwt,
	}

	if ciBuilderParams.ReleaseID > 0 {
		localBuilderConfig.CIServer.PostLogsURL = strings.TrimRight(c.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/releases/%v/logs", ciBuilderParams.RepoSource, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, ciBuilderParams.ReleaseID)
	}

	if *localBuilderConfig.Action == "build" {
		localBuilderConfig.BuildParams = &contracts.BuildParamsConfig{
			BuildID: ciBuilderParams.BuildID,
		}
	}
	if *localBuilderConfig.Action == "release" {
		localBuilderConfig.ReleaseParams = &contracts.ReleaseParamsConfig{
			ReleaseName:   ciBuilderParams.ReleaseName,
			ReleaseID:     ciBuilderParams.ReleaseID,
			ReleaseAction: ciBuilderParams.ReleaseAction,
			TriggeredBy:   ciBuilderParams.ReleaseTriggeredBy,
		}
	}

	localBuilderConfig.Events = make([]*manifest.EstafetteEvent, 0)
	for _, e := range ciBuilderParams.TriggeredByEvents {
		localBuilderConfig.Events = append(localBuilderConfig.Events, &e)
	}

	return localBuilderConfig, nil
}

func (c *client) getCiBuilderJobEnvironmentVariables(ctx context.Context, ciBuilderParams CiBuilderParams) (environmentVariables []v1.EnvVar) {

	environmentVariables = []v1.EnvVar{
		{
			Name:  "BUILDER_CONFIG_PATH",
			Value: "/configs/builder-config.json",
		},
		{
			Name:  "ESTAFETTE_LOG_FORMAT",
			Value: os.Getenv("ESTAFETTE_LOG_FORMAT"),
		},
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
		{
			Name: "POD_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}

	// forward all envars prefixed with JAEGER_ to builder job
	for _, e := range os.Environ() {
		kvPair := strings.SplitN(e, "=", 2)

		if len(kvPair) == 2 {
			envvarName := kvPair[0]
			envvarValue := kvPair[1]

			if strings.HasPrefix(envvarName, "JAEGER_") && envvarName != "JAEGER_SERVICE_NAME" && envvarName != "JAEGER_AGENT_HOST" && envvarName != "JAEGER_SAMPLER_TYPE" && envvarName != "JAEGER_SAMPLER_PARAM" && envvarValue != "" {
				environmentVariables = append(environmentVariables, v1.EnvVar{
					Name:  envvarName,
					Value: envvarValue,
				})
			}
		}
	}

	if ciBuilderParams.OperatingSystem == "windows" {
		workingDirectoryVolumeName := "working-directory"

		// docker in kubernetes on windows is still at 18.09.7, which has api version 1.39
		// todo - use auto detect for the docker api version
		dockerAPIVersionName := "DOCKER_API_VERSION"
		dockerAPIVersionValue := "1.39"
		environmentVariables = append(environmentVariables,
			v1.EnvVar{
				Name:  dockerAPIVersionName,
				Value: dockerAPIVersionValue,
			},
		)

		podUIDName := "POD_UID"
		podUIDFieldPath := "metadata.uid"
		environmentVariables = append(environmentVariables,
			v1.EnvVar{
				Name: podUIDName,
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: podUIDFieldPath,
					},
				},
			},
		)

		// this is the path on the host mounted into any of the stage containers; with docker-outside-docker the daemon can't see paths inside the ci-builder container
		estafetteWorkdirName := "ESTAFETTE_WORKDIR"
		estafetteWorkdirValue := "c:/var/lib/kubelet/pods/$(POD_UID)/volumes/kubernetes.io~empty-dir/" + workingDirectoryVolumeName
		environmentVariables = append(environmentVariables,
			v1.EnvVar{
				Name:  estafetteWorkdirName,
				Value: estafetteWorkdirValue,
			},
		)
	}

	return environmentVariables
}

func (c *client) getCiBuilderJobVolumesAndMounts(ctx context.Context, ciBuilderParams CiBuilderParams, jobName string) (volumes []v1.Volume, volumeMounts []v1.VolumeMount) {

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
	}

	if ciBuilderParams.OperatingSystem == "windows" {
		// use emptydir volume in order to be able to have docker daemon on host mount path into internal container
		workingDirectoryVolumeName := "working-directory"
		volumes = append(volumes, v1.Volume{
			Name: workingDirectoryVolumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})

		workingDirectoryVolumeMountPath := "C:/estafette-work"
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      workingDirectoryVolumeName,
			MountPath: workingDirectoryVolumeMountPath,
		})

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

		volumes = append(volumes, v1.Volume{
			Name: dockerSocketVolumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: dockerSocketVolumeHostPath,
				},
			},
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

func (c *client) getCiBuilderJobTolerations(ctx context.Context, ciBuilderParams CiBuilderParams) (tolerations []v1.Toleration) {
	tolerations = []v1.Toleration{}

	if ciBuilderParams.OperatingSystem == "windows" {
		tolerations = append(tolerations, v1.Toleration{
			Effect:   v1.TaintEffectNoSchedule,
			Key:      "node.kubernetes.io/os",
			Operator: v1.TolerationOpEqual,
			Value:    "windows",
		})
	} else if ciBuilderParams.JobType != "release" {
		// to run build jobs on preemptibles when node autoprovisioning is used, add a toleration for it
		tolerations = append(tolerations, v1.Toleration{
			Effect:   v1.TaintEffectNoSchedule,
			Key:      "cloud.google.com/gke-preemptible",
			Operator: v1.TolerationOpEqual,
			Value:    "true",
		})
	}

	return tolerations
}

func (c *client) getCiBuilderJobAffinity(ctx context.Context, ciBuilderParams CiBuilderParams) (affinity *v1.Affinity) {

	preemptibleAffinityWeight := int32(10)
	preemptibleAffinityKey := "cloud.google.com/gke-preemptible"

	operatingSystemAffinityKey := "beta.kubernetes.io/os"
	operatingSystemAffinityValue := ciBuilderParams.OperatingSystem

	if ciBuilderParams.JobType == "release" {
		// keep off of preemptibles
		return &v1.Affinity{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      preemptibleAffinityKey,
									Operator: v1.NodeSelectorOpDoesNotExist,
								},
							},
						},
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      operatingSystemAffinityKey,
									Operator: v1.NodeSelectorOpIn,
									Values:   []string{operatingSystemAffinityValue},
								},
							},
						},
					},
				},
			},
		}
	}

	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
				{
					Weight: preemptibleAffinityWeight,
					Preference: v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      preemptibleAffinityKey,
								Operator: v1.NodeSelectorOpIn,
								Values:   []string{"true"},
							},
						},
					},
				},
			},
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      operatingSystemAffinityKey,
								Operator: v1.NodeSelectorOpIn,
								Values:   []string{operatingSystemAffinityValue},
							},
						},
					},
				},
			},
		},
	}
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

	id := strconv.Itoa(ciBuilderParams.BuildID)
	if ciBuilderParams.JobType == "release" {
		id = strconv.Itoa(ciBuilderParams.ReleaseID)
	}

	return c.GetJobName(ctx, ciBuilderParams.JobType, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, id)
}
