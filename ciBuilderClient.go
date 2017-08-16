package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ericchiang/k8s"
	apiv1 "github.com/ericchiang/k8s/api/v1"
	batchv1 "github.com/ericchiang/k8s/apis/batch/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// CiBuilderClient is the interface for running kubernetes commands specific to this application
type CiBuilderClient interface {
	CreateCiBuilderJob(CiBuilderParams) (*batchv1.Job, error)
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
	// temporarily pass build version equal to revision from the outside until estafette supports versioning
	estafetteBuildVersionName := "ESTAFETTE_BUILD_VERSION"
	estafetteBuildVersionValue := ciBuilderParams.RepoRevision

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
	}

	for key, value := range ciBuilderParams.EnvironmentVariables {
		environmentVariables = append(environmentVariables, &apiv1.EnvVar{
			Name:  &key,
			Value: &value,
		})
	}

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
