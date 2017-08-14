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
	"github.com/rs/zerolog/log"
)

// Kubernetes wraps the Kubernetes Client
type Kubernetes struct {
	Client *k8s.Client
}

// KubernetesClient is the interface for running kubernetes commands specific to this application
type KubernetesClient interface {
	CreateJobForGithubPushEvent(GithubPushEvent, string) (*batchv1.Job, error)
	CreateJobForBitbucketPushEvent(BitbucketRepositoryPushEvent, string) (*batchv1.Job, error)
}

// NewKubernetesClient return a Kubernetes client
func NewKubernetesClient() (kubernetes KubernetesClient, err error) {

	client, err := k8s.NewInClusterClient()
	if err != nil {
		log.Error().Err(err).Msg("Creating Kubernetes api client failed")
		return
	}

	kubernetes = &Kubernetes{
		Client: client,
	}

	return
}

// CreateJobForGithubPushEvent creates a kubernetes job to clone the authenticated git url
func (k *Kubernetes) CreateJobForGithubPushEvent(pushEvent GithubPushEvent, authenticatedGitURL string) (job *batchv1.Job, err error) {

	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	repoName := re.ReplaceAllString(pushEvent.Repository.FullName, "-")
	if len(repoName) > 50 {
		repoName = repoName[:50]
	}

	// max 63 chars
	jobName := strings.ToLower(fmt.Sprintf("build-%v-%v", repoName, pushEvent.After[:6]))

	args := []string{"clone", "--depth=10", authenticatedGitURL}

	containerName := "git-clone"
	image := "alpine/git"
	restartPolicy := "Never"

	job = &batchv1.Job{
		Metadata: &metav1.ObjectMeta{
			Name:      &jobName,
			Namespace: &k.Client.Namespace,
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
							Args:  args,
						},
					},
					RestartPolicy: &restartPolicy,
				},
			},
		},
	}

	job, err = k.Client.BatchV1().CreateJob(context.Background(), job)

	return
}

// CreateJobForBitbucketPushEvent creates a kubernetes job to clone the authenticated git url
func (k *Kubernetes) CreateJobForBitbucketPushEvent(pushEvent BitbucketRepositoryPushEvent, authenticatedGitURL string) (job *batchv1.Job, err error) {

	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	repoName := re.ReplaceAllString(pushEvent.Repository.FullName, "-")
	if len(repoName) > 50 {
		repoName = repoName[:50]
	}

	// max 63 chars
	jobName := strings.ToLower(fmt.Sprintf("build-%v-%v", repoName, pushEvent.Push.Changes[0].New.Target.Hash[:6]))

	estafetteGitURLName := "ESTAFETTE_GIT_URL"
	estafetteGitURLValue := authenticatedGitURL
	estafetteGitBranchName := "ESTAFETTE_GIT_BRANCH"
	estafetteGitBranchValue := pushEvent.Push.Changes[0].New.Name
	estafetteGitRevisionName := "ESTAFETTE_GIT_REVISION"
	estafetteGitRevisionValue := pushEvent.Push.Changes[0].New.Target.Hash

	containerName := "estafette-ci-builder"
	image := "estafette/estafette-ci-builder:0.0.151"
	restartPolicy := "Never"

	job = &batchv1.Job{
		Metadata: &metav1.ObjectMeta{
			Name:      &jobName,
			Namespace: &k.Client.Namespace,
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
							Env: []*apiv1.EnvVar{
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
							},
						},
					},
					RestartPolicy: &restartPolicy,
				},
			},
		},
	}

	job, err = k.Client.BatchV1().CreateJob(context.Background(), job)

	return
}
