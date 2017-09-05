package github

import (
	"strings"
	"sync"

	"github.com/estafette/estafette-ci-api/estafette"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToEventChannels()
	CreateJobForGithubPush(PushEvent)
}

type eventWorkerImpl struct {
	waitGroup       *sync.WaitGroup
	stopChannel     <-chan struct{}
	eventsChannel   chan PushEvent
	apiClient       APIClient
	ciBuilderClient estafette.CiBuilderClient
}

// NewGithubEventWorker returns a new github.EventWorker to handle events channeled by github.EventHandler
func NewGithubEventWorker(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, apiClient APIClient, ciBuilderClient estafette.CiBuilderClient, eventsChannel chan PushEvent) EventWorker {
	return &eventWorkerImpl{
		waitGroup:       waitGroup,
		stopChannel:     stopChannel,
		eventsChannel:   eventsChannel,
		apiClient:       apiClient,
		ciBuilderClient: ciBuilderClient,
	}
}

func (w *eventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle github events via channels
		log.Debug().Msg("Listening to Github events channels...")
		for {
			select {
			case pushEvent := <-w.eventsChannel:
				go func() {
					w.waitGroup.Add(1)
					w.CreateJobForGithubPush(pushEvent)
					w.waitGroup.Done()
				}()
			case <-w.stopChannel:
				log.Debug().Msg("Stopping Github event worker...")
				return
			}
		}
	}()
}

func (w *eventWorkerImpl) CreateJobForGithubPush(pushEvent PushEvent) {

	// check to see that it's a cloneable event
	if !strings.HasPrefix(pushEvent.Ref, "refs/heads/") {
		return
	}

	// get access token
	accessToken, err := w.apiClient.GetInstallationToken(pushEvent.Installation.ID)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving access token failed")
		return
	}

	// get manifest file
	manifestExists, manifestString, err := w.apiClient.GetEstafetteManifest(accessToken, pushEvent)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return
	}

	if !manifestExists {
		log.Info().Interface("pushEvent", pushEvent).Msgf("No Estafette manifest for repo %v and revision %v, not creating a job", pushEvent.Repository.FullName, pushEvent.After)
		return
	}

	mft, err := manifest.ReadManifest(manifestString)
	builderTrack := "stable"
	if err != nil {
		log.Warn().Err(err).Str("manifest", manifestString).Msgf("Deserializing Estafette manifest for repo %v and revision %v failed, continuing though so developer gets useful feedback", pushEvent.Repository.FullName, pushEvent.After)
	} else {
		builderTrack = mft.Builder.Track
	}

	log.Debug().Interface("pushEvent", pushEvent).Interface("manifest", mft).Msgf("Estafette manifest for repo %v and revision %v exists creating a builder job...", pushEvent.Repository.FullName, pushEvent.After)

	// get authenticated url for the repository
	authenticatedRepositoryURL, err := w.apiClient.GetAuthenticatedRepositoryURL(accessToken, pushEvent.Repository.HTMLURL)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving authenticated repository failed")
		return
	}

	// define ci builder params
	ciBuilderParams := estafette.CiBuilderParams{
		RepoFullName:         pushEvent.Repository.FullName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           strings.Replace(pushEvent.Ref, "refs/heads/", "", 1),
		RepoRevision:         pushEvent.After,
		EnvironmentVariables: map[string]string{"ESTAFETTE_GITHUB_API_TOKEN": accessToken.Token},
		Track:                builderTrack,
	}

	// create ci builder job
	_, err = w.ciBuilderClient.CreateCiBuilderJob(ciBuilderParams)
	if err != nil {
		log.Error().Err(err).
			Str("fullname", ciBuilderParams.RepoFullName).
			Str("url", ciBuilderParams.RepoURL).
			Str("branch", ciBuilderParams.RepoBranch).
			Str("revision", ciBuilderParams.RepoRevision).
			Str("track", ciBuilderParams.Track).
			Msgf("Creating estafette-ci-builder job for Github repository %v revision %v failed", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)

		return
	}

	log.Info().
		Str("fullname", ciBuilderParams.RepoFullName).
		Str("url", ciBuilderParams.RepoURL).
		Str("branch", ciBuilderParams.RepoBranch).
		Str("revision", ciBuilderParams.RepoRevision).
		Str("track", ciBuilderParams.Track).
		Msgf("Created estafette-ci-builder job for Github repository %v revision %v", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)
}
