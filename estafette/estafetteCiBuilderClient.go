package estafette

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/estafette/estafette-ci-api/config"
	"github.com/estafette/estafette-ci-api/docker"

	yaml "gopkg.in/yaml.v2"

	"github.com/ericchiang/k8s"
	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/ericchiang/k8s/apis/resource"
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
	config                          config.APIConfig
	secretDecryptionKey             string
	PrometheusOutboundAPICallTotals *prometheus.CounterVec
}

// NewCiBuilderClient returns a new estafette.CiBuilderClient
func NewCiBuilderClient(config config.APIConfig, secretDecryptionKey string, prometheusOutboundAPICallTotals *prometheus.CounterVec) (ciBuilderClient CiBuilderClient, err error) {

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
	maxRepoNameLength := 50
	revisionOrID := ciBuilderParams.RepoRevision[:6]
	if ciBuilderParams.ReleaseID > 0 {
		revisionOrID = strconv.Itoa(ciBuilderParams.ReleaseID)
		maxRepoNameLength = 54 - len(revisionOrID)
	}
	if len(repoName) > maxRepoNameLength {
		repoName = repoName[:maxRepoNameLength]
	}
	jobType := "build"
	if ciBuilderParams.ReleaseID > 0 {
		jobType = "release"
	}
	jobName := strings.ToLower(fmt.Sprintf("%v-%v-%v", jobType, repoName, revisionOrID))

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
	estafetteCiServerBaseURLValue := cbc.config.APIServer.BaseURL
	estafetteCiServerBuilderEventsURLName := "ESTAFETTE_CI_SERVER_BUILDER_EVENTS_URL"
	estafetteCiServerBuilderEventsURLValue := strings.TrimRight(cbc.config.APIServer.ServiceURL, "/") + "/api/commands"
	estafetteCiServerBuilderPostLogsURLName := "ESTAFETTE_CI_SERVER_POST_LOGS_URL"
	estafetteCiServerBuilderPostLogsURLValue := strings.TrimRight(cbc.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/builds/%v/logs", ciBuilderParams.RepoSource, ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)
	if ciBuilderParams.ReleaseID > 0 {
		estafetteCiServerBuilderPostLogsURLValue = strings.TrimRight(cbc.config.APIServer.ServiceURL, "/") + fmt.Sprintf("/api/pipelines/%v/%v/releases/%v/logs", ciBuilderParams.RepoSource, ciBuilderParams.RepoFullName, ciBuilderParams.ReleaseID)
	}
	estafetteCiAPIKeyName := "ESTAFETTE_CI_API_KEY"
	estafetteCiAPIKeyValue := cbc.config.APIServer.APIKey
	estafetteCiBuilderTrackName := "ESTAFETTE_CI_BUILDER_TRACK"
	estafetteCiBuilderTrackValue := ciBuilderParams.Track
	estafetteManifestJSONKeyName := "ESTAFETTE_CI_MANIFEST_JSON"
	manifestJSONBytes, err := json.Marshal(ciBuilderParams.Manifest)
	estafetteManifestJSONKeyValue := string(manifestJSONBytes)
	estafetteRegistriesJSONKeyName := "ESTAFETTE_CI_REGISTRIES_JSON"
	registriesJSONBytes, err := json.Marshal(cbc.config.PrivateContainerRegistries)
	estafetteRegistriesJSONKeyValue := string(registriesJSONBytes)

	// temporarily pass build version equal to revision from the outside until estafette supports versioning
	estafetteBuildVersionName := "ESTAFETTE_BUILD_VERSION"
	estafetteBuildVersionValue := ciBuilderParams.VersionNumber
	estafetteBuildVersionPatchName := "ESTAFETTE_BUILD_VERSION_PATCH"
	estafetteBuildVersionPatchValue := fmt.Sprint(ciBuilderParams.AutoIncrement)
	estafetteReleaseNameName := "ESTAFETTE_RELEASE_NAME"
	estafetteReleaseNameValue := ciBuilderParams.ReleaseName
	estafetteReleaseIDName := "ESTAFETTE_RELEASE_ID"
	estafetteReleaseIDValue := strconv.Itoa(ciBuilderParams.ReleaseID)

	environmentVariables := []*corev1.EnvVar{
		&corev1.EnvVar{
			Name:  &estafetteGitSourceName,
			Value: &estafetteGitSourceValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteGitNameName,
			Value: &estafetteGitNameValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteGitURLName,
			Value: &estafetteGitURLValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteGitBranchName,
			Value: &estafetteGitBranchValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteGitRevisionName,
			Value: &estafetteGitRevisionValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteBuildVersionName,
			Value: &estafetteBuildVersionValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteBuildVersionPatchName,
			Value: &estafetteBuildVersionPatchValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteBuildJobNameName,
			Value: &estafetteBuildJobNameValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteCiServerBaseURLName,
			Value: &estafetteCiServerBaseURLValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteCiServerBuilderEventsURLName,
			Value: &estafetteCiServerBuilderEventsURLValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteCiServerBuilderPostLogsURLName,
			Value: &estafetteCiServerBuilderPostLogsURLValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteCiAPIKeyName,
			Value: &estafetteCiAPIKeyValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteCiBuilderTrackName,
			Value: &estafetteCiBuilderTrackValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteManifestJSONKeyName,
			Value: &estafetteManifestJSONKeyValue,
		},
		&corev1.EnvVar{
			Name:  &estafetteRegistriesJSONKeyName,
			Value: &estafetteRegistriesJSONKeyValue,
		},
	}

	if ciBuilderParams.ReleaseID > 0 {
		environmentVariables = append(environmentVariables, &corev1.EnvVar{
			Name:  &estafetteReleaseNameName,
			Value: &estafetteReleaseNameValue,
		})
		environmentVariables = append(environmentVariables, &corev1.EnvVar{
			Name:  &estafetteReleaseIDName,
			Value: &estafetteReleaseIDValue,
		})
	}

	for key, value := range ciBuilderParams.EnvironmentVariables {
		environmentVariables = append(environmentVariables, &corev1.EnvVar{
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
			Template: &corev1.PodTemplateSpec{
				Metadata: &metav1.ObjectMeta{
					Labels: map[string]string{
						"createdBy": "estafette",
					},
				},
				Spec: &corev1.PodSpec{
					Containers: []*corev1.Container{
						&corev1.Container{
							Name:            &containerName,
							Image:           &image,
							ImagePullPolicy: &imagePullPolicy,
							Args:            []string{fmt.Sprintf("--secret-decryption-key=%v", cbc.secretDecryptionKey)},
							Env:             environmentVariables,
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
						},
					},
					RestartPolicy: &restartPolicy,

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
