package main

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// GithubWorker processes events pushed to channels
type GithubWorker interface {
	ListenToGithubPushEventChannel()
	Stop()
	CreateJobForGithubPush(GithubPushEvent)
}

type githubWorkerImpl struct {
	WaitGroup   *sync.WaitGroup
	QuitChannel chan bool
}

func newGithubWorker(waitGroup *sync.WaitGroup) GithubWorker {
	return &githubWorkerImpl{
		WaitGroup:   waitGroup,
		QuitChannel: make(chan bool)}
}

func (w *githubWorkerImpl) ListenToGithubPushEventChannel() {
	go func() {
		// handle github push events via channels
		log.Debug().Msg("Listening to Github push events channel...")
		for {
			select {
			case pushEvent := <-githubPushEvents:
				w.WaitGroup.Add(1)
				w.CreateJobForGithubPush(pushEvent)
				w.WaitGroup.Done()
			case <-w.QuitChannel:
				log.Info().Msg("Stopping Github worker...")
				return
			}
		}
	}()
}

func (w *githubWorkerImpl) Stop() {
	go func() {
		w.QuitChannel <- true
	}()
}

func (w *githubWorkerImpl) CreateJobForGithubPush(pushEvent GithubPushEvent) {

	// check to see that it's a cloneable event
	if !strings.HasPrefix(pushEvent.Ref, "refs/heads/") {
		return
	}

	// get authenticated url for the repository
	ghClient := newGithubAPIClient(*githubAppPrivateKeyPath, *githubAppID, *githubAppOAuthClientID, *githubAppOAuthClientSecret)
	authenticatedRepositoryURL, err := ghClient.GetAuthenticatedRepositoryURL(pushEvent.Installation.ID, pushEvent.Repository.HTMLURL)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving authenticated repository failed")
		return
	}

	// create ci builder client
	ciBuilderClient, err := newCiBuilderClient()
	if err != nil {
		log.Error().Err(err).Msg("Initializing ci builder client failed")
		return
	}

	// define ci builder params
	ciBuilderParams := CiBuilderParams{
		RepoFullName: pushEvent.Repository.FullName,
		RepoURL:      authenticatedRepositoryURL,
		RepoBranch:   strings.Replace(pushEvent.Ref, "refs/heads/", "", 1),
		RepoRevision: pushEvent.After,
	}

	// create ci builder job
	_, err = ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
	if err != nil {
		log.Error().Err(err).
			Str("fullname", ciBuilderParams.RepoFullName).
			Str("url", ciBuilderParams.RepoURL).
			Str("branch", ciBuilderParams.RepoBranch).
			Str("revision", ciBuilderParams.RepoRevision).
			Msgf("Created estafette-ci-builder job for Github repository %v revision %v failed", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)

		return
	}

	log.Debug().
		Str("fullname", ciBuilderParams.RepoFullName).
		Str("url", ciBuilderParams.RepoURL).
		Str("branch", ciBuilderParams.RepoBranch).
		Str("revision", ciBuilderParams.RepoRevision).
		Msgf("Created estafette-ci-builder job for Github repository %v revision %v", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)
}
