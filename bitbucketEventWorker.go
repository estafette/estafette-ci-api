package main

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// BitbucketEventWorker processes events pushed to channels
type BitbucketEventWorker interface {
	ListenToEventChannels()
	Stop()
	CreateJobForBitbucketPush(BitbucketRepositoryPushEvent)
}

type bitbucketEventWorkerImpl struct {
	WaitGroup   *sync.WaitGroup
	QuitChannel chan bool
}

func newBitbucketEventWorker(waitGroup *sync.WaitGroup) BitbucketEventWorker {
	return &bitbucketEventWorkerImpl{
		WaitGroup:   waitGroup,
		QuitChannel: make(chan bool)}
}

func (w *bitbucketEventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle github events via channels
		log.Debug().Msg("Listening to Bitbucket events channels...")
		for {
			select {
			case pushEvent := <-bitbucketPushEvents:
				go func() {
					w.WaitGroup.Add(1)
					w.CreateJobForBitbucketPush(pushEvent)
					w.WaitGroup.Done()
				}()
			case <-w.QuitChannel:
				log.Debug().Msg("Stopping Bitbucket event worker...")
				return
			}
		}
	}()
}

func (w *bitbucketEventWorkerImpl) Stop() {
	go func() {
		w.QuitChannel <- true
	}()
}

func (w *bitbucketEventWorkerImpl) CreateJobForBitbucketPush(pushEvent BitbucketRepositoryPushEvent) {

	// check to see that it's a cloneable event
	if len(pushEvent.Push.Changes) == 0 || pushEvent.Push.Changes[0].New == nil || pushEvent.Push.Changes[0].New.Type != "branch" || len(pushEvent.Push.Changes[0].New.Target.Hash) == 0 {
		return
	}

	// get authenticated url for the repository
	bbClient := newBitbucketAPIClient(*bitbucketAPIKey, *bitbucketAppOAuthKey, *bitbucketAppOAuthSecret)
	authenticatedRepositoryURL, accessToken, err := bbClient.GetAuthenticatedRepositoryURL(pushEvent.Repository.Links.HTML.Href)
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
		RepoBranch:           pushEvent.Push.Changes[0].New.Name,
		RepoRevision:         pushEvent.Push.Changes[0].New.Target.Hash,
		EnvironmentVariables: map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken.AccessToken},
	}

	// create ci builder job
	_, err = ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
	if err != nil {
		log.Error().Err(err).
			Str("fullname", ciBuilderParams.RepoFullName).
			Str("url", ciBuilderParams.RepoURL).
			Str("branch", ciBuilderParams.RepoBranch).
			Str("revision", ciBuilderParams.RepoRevision).
			Msgf("Creating estafette-ci-builder job for Bitbucket repository %v revision %v failed", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)

		return
	}

	log.Info().
		Str("fullname", ciBuilderParams.RepoFullName).
		Str("url", ciBuilderParams.RepoURL).
		Str("branch", ciBuilderParams.RepoBranch).
		Str("revision", ciBuilderParams.RepoRevision).
		Msgf("Created estafette-ci-builder job for Bitbucket repository %v revision %v", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)
}
