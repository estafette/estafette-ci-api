package estafette

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/docker"

	yaml "gopkg.in/yaml.v2"

	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/resource"
	apiv1 "github.com/ericchiang/k8s/api/v1"
	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// CiBuilderClient is the interface for running kubernetes commands specific to this application
type CiBuilderClient interface {
	CreateCiBuilderJob(CiBuilderParams) (*batchv1.Job, error)
	RemoveCiBuilderJob(string) error
}

type ciBuilderClientImpl struct {
	kubeClient                      *k8s.Client
	dockerHubClient                 docker.DockerHubAPIClient
	config                          config.APIServerConfig
	secretDecryptionKey             string
	PrometheusOutboundAPICallTotals *prometheus.CounterVec
}

// NewCiBuilderClient returns a new estafette.CiBuilderClient
func NewCiBuilderClient(config config.APIServerConfig, secretDecryptionKey string, prometheusOutboundAPICallTotals *prometheus.CounterVec) (ciBuilderClient CiBuilderClient, err error) {

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
		secretDecryptionKey:             secretDecryptionKey,
		PrometheusOutboundAPICallTotals: prometheusOutboundAPICallTotals,
	}

	return
}

// CreateCiBuilderJob creates an estafette-ci-builder job in Kubernetes to run the estafette build
func (cbc *ciBuilderClientImpl) CreateCiBuilderJob(ciBuilderParams CiBuilderParams) (job *batchv1.Job, err error) {

	// create job name of max 63 chars
	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	repoName := re.ReplaceAllString(ciBuilderParams.RepoFullName, "-")
	if len(repoName) > 50 {
		repoName = repoName[:50]
	}
	jobName := strings.ToLower(fmt.Sprintf("build-%v-%v", repoName, ciBuilderParams.RepoRevision[:6]))

	// create envvars for job
	estafetteGitSourceName := "ESTAFETTE_GIT_SOURCE"
	estafetteGitSourceValue := ciBuilderParams.RepoSource
	estafetteGitNameName := "ESTAFETTE_GIT_NAME"
	estafetteGitNameValue := ciBuilderParams.RepoFullName
	estafetteGitURLName := "ESTAFETTE_GIT_URL"
	estafetteGitURLValue := ciBuilderParams.RepoURL
	estafetteGitBranchName := "ESTAFETTE_GIT_BRANCH"
	estafetteGitBranchValue := ciBuilderParams.RepoBranch
	estafetteGitRevisionName := "ESTAFETTE_GIT_REVISION"
	estafetteGitRevisionValue := ciBuilderParams.RepoRevision
	estafetteBuildJobNameName := "ESTAFETTE_BUILD_JOB_NAME"
	estafetteBuildJobNameValue := jobName
	estafetteCiServerBaseURLName := "ESTAFETTE_CI_SERVER_BASE_URL"
	estafetteCiServerBaseURLValue := cbc.config.BaseURL
	estafetteCiServerBuilderEventsURLName := "ESTAFETTE_CI_SERVER_BUILDER_EVENTS_URL"
	estafetteCiServerBuilderEventsURLValue := strings.TrimRight(cbc.config.ServiceURL, "/") + "/api/commands"
	estafetteCiServerBuilderPostLogsURLName := "ESTAFETTE_CI_SERVER_POST_LOGS_URL"
	estafetteCiServerBuilderPostLogsURLValue := strings.TrimRight(cbc.config.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/builds/%v/logs", ciBuilderParams.RepoSource, ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)
	estafetteCiAPIKeyName := "ESTAFETTE_CI_API_KEY"
	estafetteCiAPIKeyValue := cbc.config.APIKey
	estafetteCiBuilderTrackName := "ESTAFETTE_CI_BUILDER_TRACK"
	estafetteCiBuilderTrackValue := ciBuilderParams.Track
	estafetteManifestJSONKeyName := "ESTAFETTE_CI_MANIFEST_JSON"
	manifestJSONBytes, err := json.Marshal(ciBuilderParams.Manifest)
	estafetteManifestJSONKeyValue := string(manifestJSONBytes)

	// temporarily pass build version equal to revision from the outside until estafette supports versioning
	estafetteBuildVersionName := "ESTAFETTE_BUILD_VERSION"
	estafetteBuildVersionValue := ciBuilderParams.VersionNumber
	estafetteBuildVersionPatchName := "ESTAFETTE_BUILD_VERSION_PATCH"
	estafetteBuildVersionPatchValue := fmt.Sprint(ciBuilderParams.AutoIncrement)

	environmentVariables := []*apiv1.EnvVar{
		&apiv1.EnvVar{
			Name:  &estafetteGitSourceName,
			Value: &estafetteGitSourceValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteGitNameName,
			Value: &estafetteGitNameValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteGitURLName,
			Value: &estafetteGitURLValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteGitBranchName,
			Value: &estafetteGitBranchValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteGitRevisionName,
			Value: &estafetteGitRevisionValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteBuildVersionName,
			Value: &estafetteBuildVersionValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteBuildVersionPatchName,
			Value: &estafetteBuildVersionPatchValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteBuildJobNameName,
			Value: &estafetteBuildJobNameValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteCiServerBaseURLName,
			Value: &estafetteCiServerBaseURLValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteCiServerBuilderEventsURLName,
			Value: &estafetteCiServerBuilderEventsURLValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteCiServerBuilderPostLogsURLName,
			Value: &estafetteCiServerBuilderPostLogsURLValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteCiAPIKeyName,
			Value: &estafetteCiAPIKeyValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteCiBuilderTrackName,
			Value: &estafetteCiBuilderTrackValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteManifestJSONKeyName,
			Value: &estafetteManifestJSONKeyValue,
		},
	}

	for key, value := range ciBuilderParams.EnvironmentVariables {
		environmentVariables = append(environmentVariables, &apiv1.EnvVar{
			Name:  &key,
			Value: &value,
		})
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
	if err == nil {
		image = fmt.Sprintf("%v@%v", repository, digest.Digest)
		imagePullPolicy = "IfNotPresent"
	}
	restartPolicy := "Never"
	privileged := true

	preemptibleAffinityWeight := int32(10)
	preemptibleAffinityKey := "cloud.google.com/gke-preemptible"
	preemptibleAffinityOperator := "In"

	job = &batchv1.Job{
		Metadata: &metav1.ObjectMeta{
			Name:      &jobName,
			Namespace: &cbc.kubeClient.Namespace,
			Labels: map[string]string{
				"createdBy": "estafette",
			},
		},
		Spec: &batchv1.JobSpec{
			Template: &apiv1.PodTemplateSpec{
				Metadata: &metav1.ObjectMeta{
					Labels: map[string]string{
						"createdBy": "estafette",
					},
				},
				Spec: &apiv1.PodSpec{
					Containers: []*apiv1.Container{
						&apiv1.Container{
							Name:            &containerName,
							Image:           &image,
							ImagePullPolicy: &imagePullPolicy,
							Args:            []string{fmt.Sprintf("--secret-decryption-key=%v", cbc.secretDecryptionKey)},
							Env:             environmentVariables,
							SecurityContext: &apiv1.SecurityContext{
								Privileged: &privileged,
							},
							Resources: &apiv1.ResourceRequirements{
								Requests: map[string]*resource.Quantity{
									"cpu":    &resource.Quantity{String_: &cpuRequest},
									"memory": &resource.Quantity{String_: &memoryRequest},
								},
								Limits: map[string]*resource.Quantity{
									"cpu":    &resource.Quantity{String_: &cpuLimit},
									"memory": &resource.Quantity{String_: &memoryLimit},
								},
							},
						},
					},
					RestartPolicy: &restartPolicy,

					Affinity: &apiv1.Affinity{
						NodeAffinity: &apiv1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []*apiv1.PreferredSchedulingTerm{
								&apiv1.PreferredSchedulingTerm{
									Weight: &preemptibleAffinityWeight,
									// A node selector term, associated with the corresponding weight.
									Preference: &apiv1.NodeSelectorTerm{
										MatchExpressions: []*apiv1.NodeSelectorRequirement{
											&apiv1.NodeSelectorRequirement{
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

	err = cbc.kubeClient.Create(context.Background(), job)
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("Create call for job %v failed", jobName)
	}
	cbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "kubernetes"}).Inc()

	return
}

// RemoveCiBuilderJob waits for a job to finish and then removes it
func (cbc *ciBuilderClientImpl) RemoveCiBuilderJob(jobName string) (err error) {

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
		log.Debug().
			Str("jobName", jobName).
			Msgf("Job is not done yet, watching for job %v to succeed", jobName)

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
	log.Debug().
		Str("jobName", jobName).
		Msgf("Job %v is done, deleting it...", jobName)

	// delete job
	err = cbc.kubeClient.Delete(context.Background(), &job)
	cbc.PrometheusOutboundAPICallTotals.With(prometheus.Labels{"target": "kubernetes"}).Inc()
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("Deleting job %v failed", jobName)
		return
	}

	return
}
