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

type Kubernetes struct {
	Client *k8s.Client
}

type KubernetesClient interface {
	CreateJob(GithubPushEvent, string) (*batchv1.Job, error)
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

// CreateJob creates a kubernetes job to clone the authenticated git url
func (k *Kubernetes) CreateJob(pushEvent GithubPushEvent, authenticatedGitURL string) (job *batchv1.Job, err error) {

	name := fmt.Sprintf("%v-%v", strings.Replace(pushEvent.Repository.FullName, "/", "-", -1), pushEvent.After[0:6])
	containerName := "clone"
	image := "alpine/git"

	job = &batchv1.Job{
		Metadata: &metav1.ObjectMeta{
			Name:      &name,
			Namespace: &k.Client.Namespace,
		},
		Spec: &batchv1.JobSpec{
			Template: &apiv1.PodTemplateSpec{
				Spec: &apiv1.PodSpec{
					Containers: []*apiv1.Container{
						&apiv1.Container{
							Name:  &containerName,
							Image: &image,
							Args:  []string{"clone", authenticatedGitURL},
						},
					},
				},
			},
		},
	}

	job, err = k.Client.BatchV1().CreateJob(context.Background(), job)

	return
}
