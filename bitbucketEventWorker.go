package main

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type bitbucketWorker struct {
	WaitGroup   *sync.WaitGroup
	QuitChannel chan bool
}

func newBitbucketWorker(waitGroup *sync.WaitGroup) bitbucketWorker {
	return bitbucketWorker{
		WaitGroup:   waitGroup,
		QuitChannel: make(chan bool)}
}

func (w *bitbucketWorker) listenToBitbucketPushEventChannel() {
	go func() {
		// handle github push events via channels
		log.Debug().Msg("Listening to Bitbucket push events channel...")
		for {
			select {
			case pushEvent := <-bitbucketPushEvents:
				w.WaitGroup.Add(1)
				createJobForBitbucketPush(pushEvent)
				w.WaitGroup.Done()
			case <-w.QuitChannel:
				return
			}
		}
	}()
}

func (w *bitbucketWorker) stop() {
	go func() {
		w.QuitChannel <- true
	}()
}

func createJobForBitbucketPush(pushEvent BitbucketRepositoryPushEvent) {

	// check to see that it's a cloneable event
	if len(pushEvent.Push.Changes) == 0 || pushEvent.Push.Changes[0].New == nil || pushEvent.Push.Changes[0].New.Type != "branch" || len(pushEvent.Push.Changes[0].New.Target.Hash) == 0 {
		return
	}

	// get authenticated url for the repository
	bbClient := CreateBitbucketAPIClient(*bitbucketAPIKey, *bitbucketAppOAuthKey, *bitbucketAppOAuthSecret)
	authenticatedRepositoryURL, err := bbClient.GetAuthenticatedRepositoryURL(pushEvent.Repository.Links.HTML.Href)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving authenticated repository failed")
		return
	}

	// create ci builder client
	ciBuilderClient, err := CreateCiBuilderClient()
	if err != nil {
		log.Error().Err(err).Msg("Initializing ci builder client failed")
		return
	}

	// define ci builder params
	ciBuilderParams := CiBuilderParams{
		RepoFullName: pushEvent.Repository.FullName,
		RepoURL:      authenticatedRepositoryURL,
		RepoBranch:   pushEvent.Push.Changes[0].New.Name,
		RepoRevision: pushEvent.Push.Changes[0].New.Target.Hash,
	}

	// create ci builder job
	_, err = ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
	if err != nil {
		log.Error().Err(err).Msg("Creating ci builder job failed")
		return
	}

	log.Debug().
		Str("url", ciBuilderParams.RepoURL).
		Str("branch", ciBuilderParams.RepoBranch).
		Str("revision", ciBuilderParams.RepoRevision).
		Msgf("Created estafette-ci-builder job for Bitbucket repository %v revision %v", ciBuilderParams.RepoURL, ciBuilderParams.RepoRevision)
}
