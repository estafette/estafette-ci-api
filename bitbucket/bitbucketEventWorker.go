package bitbucket

import (
	"sync"

	"github.com/estafette/estafette-ci-api/estafette"
	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToEventChannels()
	Stop()
	CreateJobForBitbucketPush(RepositoryPushEvent)
}

type eventWorkerImpl struct {
	waitGroup       *sync.WaitGroup
	quitChannel     chan bool
	eventsChannel   chan RepositoryPushEvent
	apiClient       APIClient
	CiBuilderClient estafette.CiBuilderClient
}

// NewBitbucketEventWorker returns the bitbucket.EventWorker
func NewBitbucketEventWorker(waitGroup *sync.WaitGroup, apiClient APIClient, ciBuilderClient estafette.CiBuilderClient, eventsChannel chan RepositoryPushEvent) EventWorker {
	return &eventWorkerImpl{
		waitGroup:       waitGroup,
		quitChannel:     make(chan bool),
		eventsChannel:   eventsChannel,
		apiClient:       apiClient,
		CiBuilderClient: ciBuilderClient,
	}
}

func (w *eventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle github events via channels
		log.Debug().Msg("Listening to Bitbucket events channels...")
		for {
			select {
			case pushEvent := <-w.eventsChannel:
				go func() {
					w.waitGroup.Add(1)
					w.CreateJobForBitbucketPush(pushEvent)
					w.waitGroup.Done()
				}()
			case <-w.quitChannel:
				log.Debug().Msg("Stopping Bitbucket event worker...")
				return
			}
		}
	}()
}

func (w *eventWorkerImpl) Stop() {
	go func() {
		w.quitChannel <- true
	}()
}

func (w *eventWorkerImpl) CreateJobForBitbucketPush(pushEvent RepositoryPushEvent) {

	// check to see that it's a cloneable event
	if len(pushEvent.Push.Changes) == 0 || pushEvent.Push.Changes[0].New == nil || pushEvent.Push.Changes[0].New.Type != "branch" || len(pushEvent.Push.Changes[0].New.Target.Hash) == 0 {
		return
	}

	// get access token
	accessToken, err := w.apiClient.GetAccessToken()
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return
	}

	// get manifest file
	manifestExists, manifest, err := w.apiClient.GetEstafetteManifest(accessToken, pushEvent)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return
	}

	if !manifestExists {
		log.Info().Interface("pushEvent", pushEvent).Msgf("No Estafette manifest for repo %v and revision %v, not creating a job", pushEvent.Repository.FullName, pushEvent.Push.Changes[0].New.Target.Hash)
		return
	}
	log.Debug().Interface("pushEvent", pushEvent).Str("manifest", manifest).Msgf("Estafette manifest for repo %v and revision %v exists creating a builder job...", pushEvent.Repository.FullName, pushEvent.Push.Changes[0].New.Target.Hash)

	// get authenticated url for the repository
	authenticatedRepositoryURL, err := w.apiClient.GetAuthenticatedRepositoryURL(accessToken, pushEvent.Repository.Links.HTML.Href)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving authenticated repository failed")
		return
	}

	// define ci builder params
	ciBuilderParams := estafette.CiBuilderParams{
		RepoFullName:         pushEvent.Repository.FullName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           pushEvent.Push.Changes[0].New.Name,
		RepoRevision:         pushEvent.Push.Changes[0].New.Target.Hash,
		EnvironmentVariables: map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken.AccessToken},
	}

	// create ci builder job
	_, err = w.CiBuilderClient.CreateCiBuilderJob(ciBuilderParams)
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
