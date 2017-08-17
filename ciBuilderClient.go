package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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
	KubeClient *k8s.Client
}

// newCiBuilderClient return a estafette ci builder client
func newCiBuilderClient() (ciBuilderClient CiBuilderClient, err error) {

	kubeClient, err := k8s.NewInClusterClient()
	if err != nil {
		log.Error().Err(err).Msg("Creating k8s client failed")
		return
	}

	ciBuilderClient = &ciBuilderClientImpl{
		KubeClient: kubeClient,
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
	estafetteCiServerBaseURLValue := *estafetteCiServerBaseURL

	// temporarily pass build version equal to revision from the outside until estafette supports versioning
	estafetteBuildVersionName := "ESTAFETTE_BUILD_VERSION"
	estafetteBuildVersionValue := ciBuilderParams.RepoRevision
	estafetteBuildVersionPatchName := "ESTAFETTE_BUILD_VERSION_PATCH"
	estafetteBuildVersionPatchValue := "1"

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
	image := fmt.Sprintf("estafette/estafette-ci-builder:%v", *estafetteCiBuilderVersion)
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

	// track call via prometheus
	outgoingAPIRequestTotal.With(prometheus.Labels{"target": "kubernetes"}).Inc()

	job, err = cbc.KubeClient.BatchV1().CreateJob(context.Background(), job)

	return
}

// RemoveCiBuilderJob waits for a job to finish and then removes it
func (cbc *ciBuilderClientImpl) RemoveCiBuilderJob(jobName string) (err error) {

	// watch for job updates
	watcher, err := cbc.KubeClient.BatchV1().WatchJobs(context.Background(), cbc.KubeClient.Namespace)
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("WatchJobs call for job %v failed", jobName)
		return
	}

	// wait for job to succeed
	for {
		event, job, err := watcher.Next()
		if err != nil {
			log.Error().Err(err)
			return err
		}

		if *event.Type == k8s.EventModified && *job.Metadata.Name == jobName && *job.Status.Succeeded == 1 {
			break
		}
	}

	// delete job
	err = cbc.KubeClient.BatchV1().DeleteJob(context.Background(), jobName, cbc.KubeClient.Namespace)
	if err != nil {
		log.Error().Err(err).
			Str("jobName", jobName).
			Msgf("Deleting job %v failed", jobName)
		return
	}

	return
}
