package estafette

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ericchiang/k8s"
	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/ericchiang/k8s/apis/resource"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/docker"
	contracts "github.com/estafette/estafette-ci-contracts"
	crypt "github.com/estafette/estafette-ci-crypt"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

// CiBuilderClient is the interface for running kubernetes commands specific to this application
type CiBuilderClient interface {
	CreateCiBuilderJob(CiBuilderParams) (*batchv1.Job, error)
	RemoveCiBuilderJob(string) error
	CancelCiBuilderJob(string) error
	TailCiBuilderJobLogs(string, chan contracts.TailLogLine) error
	GetJobName(string, string, string, string) string
	GetBuilderConfig(CiBuilderParams, string) contracts.BuilderConfig
}

type ciBuilderClientImpl struct {
	kubeClient                      *k8s.Client
	dockerHubClient                 docker.DockerHubAPIClient
	config                          config.APIConfig
	encryptedConfig                 config.APIConfig
	secretHelper                    crypt.SecretHelper
	PrometheusOutboundAPICallTotals *prometheus.CounterVec
}

// NewCiBuilderClient returns a new estafette.CiBuilderClient
func NewCiBuilderClient(config config.APIConfig, encryptedConfig config.APIConfig, secretHelper crypt.SecretHelper, prometheusOutboundAPICallTotals *prometheus.CounterVec) (ciBuilderClient CiBuilderClient, err error) {

	var kubeClient *k8s.Client

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {

		kubeClient, err = k8s.NewInClusterClient()
		if err != nil {
			log.Error().Err(err).Msg("Creating k8s client failed")
			return
		}

	} else {

		homeDir := os.Getenv("HOME")

		data, err := ioutil.ReadFile(fmt.Sprintf("%v/.kube/config", homeDir))
		if err != nil {
			log.Error().Err(err).Msg("Reading kube config failed")
			return nil, err
		}

		var config k8s.Config
		if err := yaml.Unmarshal(data, &config); err != nil {
			log.Error().Err(err).Msg("Deserializing kube config failed")
			return nil, err
		}

		kubeClient, err = k8s.NewClient(&config)
	}

	dockerHubClient, err := docker.NewDockerHubAPIClient()
	if err != nil {
		return
	}

	ciBuilderClient = &ciBuilderClientImpl{
		kubeClient:                      kubeClient,
		dockerHubClient:                 dockerHubClient,
		config:                          config,
		encryptedConfig:                 encryptedConfig,
		secretHelper:                    secretHelper,
		PrometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}

	return
}

// CreateCiBuilderJob creates an estafette-ci-builder job in Kubernetes to run the estafette build
func (cbc *ciBuilderClientImpl) CreateCiBuilderJob(ciBuilderParams CiBuilderParams) (job *batchv1.Job, err error) {

	// create job name of max 63 chars
	id := strconv.Itoa(ciBuilderParams.BuildID)
	if ciBuilderParams.JobType == "release" {
		id = strconv.Itoa(ciBuilderParams.ReleaseID)
	}

	jobName := cbc.GetJobName(ciBuilderParams.JobType, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, id)

	log.Info().Msgf("Creating job %v...", jobName)

	// extend builder config to parameterize the builder and replace all other envvars to improve security
	localBuilderConfig := cbc.GetBuilderConfig(ciBuilderParams, jobName)

	builderConfigName := "BUILDER_CONFIG"
	builderConfigJSONBytes, err := json.Marshal(localBuilderConfig)
	if err != nil {
		return
	}
	builderConfigValue := string(builderConfigJSONBytes)
	builderConfigValue, newKey, err := cbc.secretHelper.ReencryptAllEnvelopes(builderConfigValue, true)
	if err != nil {
		return
	}

	environmentVariables := []*corev1.EnvVar{
		&corev1.EnvVar{
			Name:  &builderConfigName,
			Value: &builderConfigValue,
		},
	}

	// define resource request and limit values to fit reasonably well inside a n1-highmem-4 machine
	cpuRequest := "1.0"
	cpuLimit := "3.0"
	memoryRequest := "2.0Gi"
	memoryLimit := "20.0Gi"

	// other job config
	containerName := "estafette-ci-builder"
	repository := "estafette/estafette-ci-builder"
	tag := ciBuilderParams.Track
	image := fmt.Sprintf("%v:%v", repository, tag)
	imagePullPolicy := "Always"
	digest, err := cbc.dockerHubClient.GetDigestCached(repository, tag)
	if err == nil && digest.Digest != "" {
		image = fmt.Sprintf("%v@%v", repository, digest.Digest)
		imagePullPolicy = "IfNotPresent"
	}
	restartPolicy := "Never"
	privileged := true

	preemptibleAffinityWeight := int32(10)
	preemptibleAffinityKey := "cloud.google.com/gke-preemptible"
	preemptibleAffinityOperator := "In"

	volumeMounts := []*corev1.VolumeMount{}
	volumes := []*corev1.Volume{}

	job = &batchv1.Job{
		Metadata: &metav1.ObjectMeta{
			Name:      &jobName,
			Namespace: &cbc.kubeClient.Namespace,
			Labels: map[string]string{
				"createdBy": "estafette",
				"jobType":   ciBuilderParams.JobType,
			},
		},
		Spec: &batchv1.JobSpec{
			Template: &corev1.PodTemplateSpec{
				Metadata: &metav1.ObjectMeta{
					Labels: map[string]string{
						"createdBy": "estafette",
						"jobType":   ciBuilderParams.JobType,
					},
				},
				Spec: &corev1.PodSpec{
					Containers: []*corev1.Container{
						&corev1.Container{
							Name:            &containerName,
							Image:           &image,
							ImagePullPolicy: &imagePullPolicy,
							Args: []string{
								fmt.Sprintf("--secret-decryption-key-base64=%v", newKey),
								"--run-as-job",
							},
							Env: environmentVariables,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
							},
							Resources: &corev1.ResourceRequirements{
								Requests: map[string]*resource.Quantity{
									"cpu":    &resource.Quantity{String_: &cpuRequest},
									"memory": &resource.Quantity{String_: &memoryRequest},
								},
								Limits: map[string]*resource.Quantity{
									"cpu":    &resource.Quantity{String_: &cpuLimit},
									"memory": &resource.Quantity{String_: &memoryLimit},
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					RestartPolicy: &restartPolicy,

					Volumes: volumes,

					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []*corev1.PreferredSchedulingTerm{
								&corev1.PreferredSchedulingTerm{
									Weight: &preemptibleAffinityWeight,
									// A node selector term, associated with the corresponding weight.
									Preference: &corev1.NodeSelectorTerm{
										MatchExpressions: []*corev1.NodeSelectorRequirement{
											&corev1.NodeSelectorRequirement{
												Key:      &preemptibleAffinityKey,
												Operator: &preemptibleAffinityOperator,
												Values:   []string{"true"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// "error":"unregistered type *v1.Job",
	err = cbc.kubeClient.Create(context.Background(), job)
	cbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "kubernetes"}).Inc()

	if err != nil {
		return
	}

	log.Info().Msgf("Job %v is created", jobName)

	return
}

// RemoveCiBuilderJob waits for a job to finish and then removes it
func (cbc *ciBuilderClientImpl) RemoveCiBuilderJob(jobName string) (err error) {

	log.Info().Msgf("Deleting job %v...", jobName)

	// check if job is finished
	var job batchv1.Job
	err = cbc.kubeClient.Get(context.Background(), cbc.kubeClient.Namespace, jobName, &job)
	cbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "kubernetes"}).Inc()
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("Get call for job %v failed", jobName)
	}

	if err != nil || *job.Status.Succeeded != 1 {
		log.Debug().Str("jobName", jobName).Msgf("Job is not done yet, watching for job %v to succeed", jobName)

		// watch for job updates
		var job batchv1.Job
		watcher, err := cbc.kubeClient.Watch(context.Background(), cbc.kubeClient.Namespace, &job, k8s.Timeout(time.Duration(300)*time.Second))
		defer watcher.Close()

		cbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "kubernetes"}).Inc()
		if err != nil {
			log.Error().Err(err).
				Str("jobName", jobName).
				Msgf("Watcher call for job %v failed", jobName)
		} else {
			// wait for job to succeed
			for {
				job := new(batchv1.Job)
				event, err := watcher.Next(job)
				if err != nil {
					log.Error().Err(err)
					break
				}

				if event == k8s.EventModified && *job.Metadata.Name == jobName && *job.Status.Succeeded == 1 {
					break
				}
			}
		}
	}

	// delete job
	err = cbc.kubeClient.Delete(context.Background(), &job)
	cbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "kubernetes"}).Inc()
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("Deleting job %v failed", jobName)
		return
	}

	log.Info().Msgf("Job %v is deleted", jobName)

	return
}

// CancelCiBuilderJob removes a job and its pods to cancel a build/release
func (cbc *ciBuilderClientImpl) CancelCiBuilderJob(jobName string) (err error) {

	log.Info().Msgf("Canceling job %v...", jobName)

	// check if job is finished
	var job batchv1.Job
	err = cbc.kubeClient.Get(context.Background(), cbc.kubeClient.Namespace, jobName, &job)
	cbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "kubernetes"}).Inc()
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("Get call for job %v failed", jobName)
		return
	}

	// delete job
	err = cbc.kubeClient.Delete(context.Background(), &job)
	cbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "kubernetes"}).Inc()
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("Canceling job %v failed", jobName)
		return
	}

	log.Info().Msgf("Job %v is canceled", jobName)

	return
}

// TailCiBuilderJobLogs tails logs of a running job
func (cbc *ciBuilderClientImpl) TailCiBuilderJobLogs(jobName string, logChannel chan contracts.TailLogLine) (err error) {

	// close channel so api handler can finish it's response
	defer close(logChannel)

	labels := new(k8s.LabelSelector)
	labels.Eq("job-name", jobName)

	var pods corev1.PodList
	if err := cbc.kubeClient.List(context.Background(), cbc.kubeClient.Namespace, &pods, labels.Selector()); err != nil {
		return err
	}

	for _, pod := range pods.Items {

		if *pod.Status.Phase == "Pending" {
			// watch for pod to go into Running state (or out of Pending state)
			var pendingPod corev1.Pod
			watcher, err := cbc.kubeClient.Watch(context.Background(), cbc.kubeClient.Namespace, &pendingPod, k8s.Timeout(time.Duration(300)*time.Second))

			if err != nil {
				return err
			}

			// wait for pod to change Phase to succeed
			defer watcher.Close()
			for {
				watchedPod := new(corev1.Pod)
				event, err := watcher.Next(watchedPod)
				if err != nil {
					return err
				}

				if event == k8s.EventModified && *watchedPod.Metadata.Name == *pod.Metadata.Name && *watchedPod.Status.Phase != "Pending" {
					pod = watchedPod
					break
				}
			}
		}

		if *pod.Status.Phase != "Running" {
			log.Warn().Msgf("Post %v for job %v has unsupported phase %v", *pod.Metadata.Name, jobName, *pod.Status.Phase)
		}

		// follow logs from pod
		url := fmt.Sprintf("%v/api/v1/namespaces/%v/pods/%v/log?follow=true", cbc.kubeClient.Endpoint, cbc.kubeClient.Namespace, *pod.Metadata.Name)

		ct := "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Error().Err(err).Msgf("Failed generating request for retrieving logs from pod %v for job %v", *pod.Metadata.Name, jobName)
			return err
		}
		if cbc.kubeClient.SetHeaders != nil {
			if err := cbc.kubeClient.SetHeaders(req.Header); err != nil {
				log.Error().Err(err).Msgf("Failed setting request headers for retrieving logs from pod %v for job %v", *pod.Metadata.Name, jobName)
				return err
			}
		}
		req = req.WithContext(context.Background())

		req.Header.Set("Accept", ct)

		resp, err := cbc.kubeClient.Client.Do(req)
		if err != nil {
			log.Error().Err(err).Msgf("Failed performing request for retrieving logs from pod %v for job %v", *pod.Metadata.Name, jobName)
			return err
		}

		if resp.StatusCode/100 != 2 {
			errorMessage := fmt.Sprintf("Request for retrieving logs from pod %v for job %v has status code %v", *pod.Metadata.Name, jobName, resp.StatusCode)
			log.Error().Msg(errorMessage)
			return fmt.Errorf(errorMessage)
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Warn().Err(err).Msgf("Error while reading lines from logs from pod %v for job %v", *pod.Metadata.Name, jobName)
			}

			// only forward if it's a json object with property 'tailLogLine'
			var zeroLogLine zeroLogLine
			err = json.Unmarshal(line, &zeroLogLine)
			if err == nil {
				if zeroLogLine.TailLogLine != nil {
					logChannel <- *zeroLogLine.TailLogLine
				}
			} else {
				log.Error().Err(err).Str("line", string(line)).Msgf("Tailed log from pod %v for job %v is not of type json", *pod.Metadata.Name, jobName)
			}
		}
	}

	return
}

// GetJobName returns the job name for a build or release job
func (cbc *ciBuilderClientImpl) GetJobName(jobType, repoOwner, repoSource, id string) string {

	// create job name of max 63 chars
	maxJobNameLength := 63

	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	repoName := re.ReplaceAllString(fmt.Sprintf("%v/%v", repoOwner, repoSource), "-")

	maxRepoNameLength := maxJobNameLength - len(jobType) - 1 - len(id) - 1
	if len(repoName) > maxRepoNameLength {
		repoName = repoName[:maxRepoNameLength]
	}

	return strings.ToLower(fmt.Sprintf("%v-%v-%v", jobType, repoName, id))
}

// GetJobName returns the job name for a build or release job
func (cbc *ciBuilderClientImpl) GetBuilderConfig(ciBuilderParams CiBuilderParams, jobName string) contracts.BuilderConfig {

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
	credentials := cbc.encryptedConfig.Credentials

	// add dynamic github api token credential
	if token, ok := ciBuilderParams.EnvironmentVariables["ESTAFETTE_GITHUB_API_TOKEN"]; ok {
		credentials = append(credentials, &contracts.CredentialConfig{
			Name: "github-api-token",
			Type: "github-api-token",
			AdditionalProperties: map[string]interface{}{
				"token": token,
			},
		})
	}

	// add dynamic bitbucket api token credential
	if token, ok := ciBuilderParams.EnvironmentVariables["ESTAFETTE_BITBUCKET_API_TOKEN"]; ok {
		credentials = append(credentials, &contracts.CredentialConfig{
			Name: "bitbucket-api-token",
			Type: "bitbucket-api-token",
			AdditionalProperties: map[string]interface{}{
				"token": token,
			},
		})
	}

	// filter to only what's needed by the build/release job
	trustedImages := contracts.FilterTrustedImages(cbc.encryptedConfig.TrustedImages, stages)
	credentials = contracts.FilterCredentials(credentials, trustedImages)

	// add container-registry credentials to allow private registry images to be used in stages
	credentials = contracts.AddCredentialsIfNotPresent(credentials, contracts.GetCredentialsByType(cbc.encryptedConfig.Credentials, "container-registry"))

	localBuilderConfig := contracts.BuilderConfig{
		Credentials:     credentials,
		TrustedImages:   trustedImages,
		RegistryMirror:  cbc.config.RegistryMirror,
		DockerDaemonMTU: cbc.config.DockerDaemonMTU,
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
			AutoIncrement: ciBuilderParams.AutoIncrement,
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
			AutoIncrement: &ciBuilderParams.AutoIncrement,
		}
	} else {
		localBuilderConfig.BuildVersion = &contracts.BuildVersionConfig{
			Version:       ciBuilderParams.VersionNumber,
			AutoIncrement: &ciBuilderParams.AutoIncrement,
		}
	}

	localBuilderConfig.Manifest = &ciBuilderParams.Manifest

	localBuilderConfig.JobName = &jobName
	localBuilderConfig.CIServer = &contracts.CIServerConfig{
		BaseURL:          cbc.config.APIServer.BaseURL,
		BuilderEventsURL: strings.TrimRight(cbc.config.APIServer.ServiceURL, "/") + "/api/commands",
		PostLogsURL:      strings.TrimRight(cbc.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/builds/%v/logs", ciBuilderParams.RepoSource, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, ciBuilderParams.BuildID),
		APIKey:           cbc.config.Auth.APIKey,
	}

	if ciBuilderParams.ReleaseID > 0 {
		localBuilderConfig.CIServer.PostLogsURL = strings.TrimRight(cbc.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/%v/releases/%v/logs", ciBuilderParams.RepoSource, ciBuilderParams.RepoOwner, ciBuilderParams.RepoName, ciBuilderParams.ReleaseID)
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

	return localBuilderConfig
}
