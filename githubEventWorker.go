package main

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// GithubEventWorker processes events pushed to channels
type GithubEventWorker interface {
	ListenToEventChannels()
	Stop()
	CreateJobForGithubPush(GithubPushEvent)
}

type githubEventWorkerImpl struct {
	WaitGroup   *sync.WaitGroup
	QuitChannel chan bool
}

func newGithubEventWorker(waitGroup *sync.WaitGroup) GithubEventWorker {
	return &githubEventWorkerImpl{
		WaitGroup:   waitGroup,
		QuitChannel: make(chan bool)}
}

func (w *githubEventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle github events via channels
		log.Debug().Msg("Listening to Github events channels...")
		for {
			select {
			case pushEvent := <-githubPushEvents:
				go func() {
					w.WaitGroup.Add(1)
					w.CreateJobForGithubPush(pushEvent)
					w.WaitGroup.Done()
				}()
			case <-w.QuitChannel:
				log.Debug().Msg("Stopping Github event worker...")
				return
			}
		}
	}()
}

func (w *githubEventWorkerImpl) Stop() {
	go func() {
		w.QuitChannel <- true
	}()
}

func (w *githubEventWorkerImpl) CreateJobForGithubPush(pushEvent GithubPushEvent) {

	// check to see that it's a cloneable event
	if !strings.HasPrefix(pushEvent.Ref, "refs/heads/") {
		return
	}

	// create github api client
	ghClient := newGithubAPIClient(*githubAppPrivateKeyPath, *githubAppID, *githubAppOAuthClientID, *githubAppOAuthClientSecret)

	// get access token
	accessToken, err := ghClient.GetInstallationToken(pushEvent.Installation.ID)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving access token failed")
		return
	}

	// get manifest file
	manifestExists, manifest, err := ghClient.GetEstafetteManifest(accessToken, pushEvent)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return
	}

	if !manifestExists {
		log.Info().Interface("pushEvent", pushEvent).Msgf("No Estaffette manifest for repo %v and revision %v, not creating a job", pushEvent.Repository.FullName, pushEvent.After)
		return
	}
	log.Debug().Interface("pushEvent", pushEvent).Str("manifest", manifest).Msgf("Estaffette manifest for repo %v and revision %v exists creating a builder job...", pushEvent.Repository.FullName, pushEvent.After)

	// get authenticated url for the repository
	authenticatedRepositoryURL, err := ghClient.GetAuthenticatedRepositoryURL(accessToken, pushEvent.Repository.HTMLURL)
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
		RepoFullName:         pushEvent.Repository.FullName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           strings.Replace(pushEvent.Ref, "refs/heads/", "", 1),
		RepoRevision:         pushEvent.After,
		EnvironmentVariables: map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken.Token},
	}

	// create ci builder job
	_, err = ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
	if err != nil {
		log.Error().Err(err).
			Str("fullname", ciBuilderParams.RepoFullName).
			Str("url", ciBuilderParams.RepoURL).
			Str("branch", ciBuilderParams.RepoBranch).
			Str("revision", ciBuilderParams.RepoRevision).
			Msgf("Creating estafette-ci-builder job for Github repository %v revision %v failed", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)

		return
	}

	log.Info().
		Str("fullname", ciBuilderParams.RepoFullName).
		Str("url", ciBuilderParams.RepoURL).
		Str("branch", ciBuilderParams.RepoBranch).
		Str("revision", ciBuilderParams.RepoRevision).
		Msgf("Created estafette-ci-builder job for Github repository %v revision %v", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)
}
