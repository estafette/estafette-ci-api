package main

import (
	"context"
	"fmt"
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
	createJob(string, string) (*batchv1.Job, error)
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

	repoName := strings.Replace(pushEvent.Repository.FullName, "/", "-", -1)
	if len(repoName) > 50 {
		repoName = repoName[:50]
	}

	// max 63 chars
	jobName := fmt.Sprintf("build-%v-%v", repoName, pushEvent.After[:6])

	return k.createJob(jobName, authenticatedGitURL)
}

// CreateJobForBitbucketPushEvent creates a kubernetes job to clone the authenticated git url
func (k *Kubernetes) CreateJobForBitbucketPushEvent(pushEvent BitbucketRepositoryPushEvent, authenticatedGitURL string) (job *batchv1.Job, err error) {

	repoName := strings.Replace(pushEvent.Repository.FullName, "/", "-", -1)
	if len(repoName) > 50 {
		repoName = repoName[:50]
	}

	// max 63 chars
	jobName := fmt.Sprintf("build-%v-%v", repoName, pushEvent.Push.Changes[0].New.Target.Hash[:6])

	return k.createJob(jobName, authenticatedGitURL)
}

func (k *Kubernetes) createJob(jobName, authenticatedGitURL string) (job *batchv1.Job, err error) {

	containerName := "clone"
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
							Args:  []string{"clone", authenticatedGitURL},
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
