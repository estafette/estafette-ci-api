package estafette

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

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

// CiBuilderParams contains the parameters required to create a ci builder job
type CiBuilderParams struct {
	RepoFullName         string
	RepoURL              string
	RepoBranch           string
	RepoRevision         string
	EnvironmentVariables map[string]string
}

type ciBuilderClientImpl struct {
	KubeClient                *k8s.Client
	EstafetteCiServerBaseURL  string
	EstafetteCiAPIKey         string
	EstafetteCiBuilderVersion string
}

// NewCiBuilderClient returns a new estafette.CiBuilderClient
func NewCiBuilderClient(estafetteCiServerBaseURL, estafetteCiAPIKey, estafetteCiBuilderVersion string) (ciBuilderClient CiBuilderClient, err error) {

	kubeClient, err := k8s.NewInClusterClient()
	if err != nil {
		log.Error().Err(err).Msg("Creating k8s client failed")
		return
	}

	ciBuilderClient = &ciBuilderClientImpl{
		KubeClient:                kubeClient,
		EstafetteCiServerBaseURL:  estafetteCiServerBaseURL,
		EstafetteCiAPIKey:         estafetteCiAPIKey,
		EstafetteCiBuilderVersion: estafetteCiBuilderVersion,
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
	estafetteCiServerBaseURLValue := cbc.EstafetteCiServerBaseURL
	estafetteCiServerBuilderEventsURLName := "ESTAFETTE_CI_SERVER_BUILDER_EVENTS_URL"
	estafetteCiServerBuilderEventsURLValue := strings.TrimRight(cbc.EstafetteCiServerBaseURL, "/") + "/events/estafette/ci-builder"
	estafetteCiAPIKeyName := "ESTAFETTE_CI_API_KEY"
	estafetteCiAPIKeyValue := cbc.EstafetteCiAPIKey

	// temporarily pass build version equal to revision from the outside until estafette supports versioning
	estafetteBuildVersionName := "ESTAFETTE_BUILD_VERSION"
	estafetteBuildVersionValue := ciBuilderParams.RepoRevision
	estafetteBuildVersionPatchName := "ESTAFETTE_BUILD_VERSION_PATCH"
	estafetteBuildVersionPatchValue := "1"
	estafetteGcrProjectName := "ESTAFETTE_GCR_PROJECT"
	estafetteGcrProjectValue := "travix-com"

	environmentVariables := []*apiv1.EnvVar{
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
			Name:  &estafetteGcrProjectName,
			Value: &estafetteGcrProjectValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteCiServerBuilderEventsURLName,
			Value: &estafetteCiServerBuilderEventsURLValue,
		},
		&apiv1.EnvVar{
			Name:  &estafetteCiAPIKeyName,
			Value: &estafetteCiAPIKeyValue,
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
	image := fmt.Sprintf("estafette/estafette-ci-builder:%v", cbc.EstafetteCiBuilderVersion)
	restartPolicy := "Never"
	privileged := true

	job = &batchv1.Job{
		Metadata: &metav1.ObjectMeta{
			Name:      &jobName,
			Namespace: &cbc.KubeClient.Namespace,
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
							Name:  &containerName,
							Image: &image,
							Env:   environmentVariables,
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
				},
			},
		},
	}

	job, err = cbc.KubeClient.BatchV1().CreateJob(context.Background(), job)
	OutgoingAPIRequestTotal.With(prometheus.Labels{"target": "kubernetes"}).Inc()

	return
}

// RemoveCiBuilderJob waits for a job to finish and then removes it
func (cbc *ciBuilderClientImpl) RemoveCiBuilderJob(jobName string) (err error) {

	// check if job is finished
	job, err := cbc.KubeClient.BatchV1().GetJob(context.Background(), jobName, cbc.KubeClient.Namespace)
	OutgoingAPIRequestTotal.With(prometheus.Labels{"target": "kubernetes"}).Inc()
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("GetJob call for job %v failed", jobName)
	}

	if err != nil || *job.Status.Succeeded != 1 {
		log.Debug().
			Str("jobName", jobName).
			Msgf("Job is not done yet, watching for job %v to succeed", jobName)

		// watch for job updates
		watcher, err := cbc.KubeClient.BatchV1().WatchJobs(context.Background(), cbc.KubeClient.Namespace, k8s.Timeout(time.Duration(60)*time.Second))
		OutgoingAPIRequestTotal.With(prometheus.Labels{"target": "kubernetes"}).Inc()
		if err != nil {
			log.Error().Err(err).
				Str("jobName", jobName).
				Msgf("WatchJobs call for job %v failed", jobName)
		} else {
			// wait for job to succeed
			for {
				event, job, err := watcher.Next()
				if err != nil {
					log.Error().Err(err)
					break
				}

				if *event.Type == k8s.EventModified && *job.Metadata.Name == jobName && *job.Status.Succeeded == 1 {
					break
				}
			}
		}
	}
	log.Debug().
		Str("jobName", jobName).
		Msgf("Job %v is done, deleting it...", jobName)

	// delete job
	err = cbc.KubeClient.BatchV1().DeleteJob(context.Background(), jobName, cbc.KubeClient.Namespace)
	OutgoingAPIRequestTotal.With(prometheus.Labels{"target": "kubernetes"}).Inc()
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("Deleting job %v failed", jobName)
		return
	}

	return
}
