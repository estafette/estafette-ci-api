package bitbucket

import (
	"strings"
	"sync"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/estafette"
	"github.com/estafette/estafette-ci-contracts"
	manifest "github.com/estafette/estafette-ci-manifest"
	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToEventChannels()
	CreateJobForBitbucketPush(RepositoryPushEvent)
}

type eventWorkerImpl struct {
	waitGroup         *sync.WaitGroup
	stopChannel       <-chan struct{}
	workerPool        chan chan RepositoryPushEvent
	eventsChannel     chan RepositoryPushEvent
	apiClient         APIClient
	CiBuilderClient   estafette.CiBuilderClient
	cockroachDBClient cockroach.DBClient
}

// NewBitbucketEventWorker returns the bitbucket.EventWorker
func NewBitbucketEventWorker(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, workerPool chan chan RepositoryPushEvent, apiClient APIClient, ciBuilderClient estafette.CiBuilderClient, cockroachDBClient cockroach.DBClient) EventWorker {
	return &eventWorkerImpl{
		waitGroup:         waitGroup,
		stopChannel:       stopChannel,
		workerPool:        workerPool,
		eventsChannel:     make(chan RepositoryPushEvent),
		apiClient:         apiClient,
		CiBuilderClient:   ciBuilderClient,
		cockroachDBClient: cockroachDBClient,
	}
}

func (w *eventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle github events via channels
		log.Debug().Msg("Listening to Bitbucket events channels...")
		for {
			// register the current worker into the worker queue.
			w.workerPool <- w.eventsChannel

			select {
			case pushEvent := <-w.eventsChannel:
				go func() {
					w.waitGroup.Add(1)
					w.CreateJobForBitbucketPush(pushEvent)
					w.waitGroup.Done()
				}()
			case <-w.stopChannel:
				log.Debug().Msg("Stopping Bitbucket event worker...")
				return
			}
		}
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
	manifestExists, manifestString, err := w.apiClient.GetEstafetteManifest(accessToken, pushEvent)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return
	}

	if !manifestExists {
		log.Info().Interface("pushEvent", pushEvent).Msgf("No Estafette manifest for repo %v and revision %v, not creating a job", pushEvent.Repository.FullName, pushEvent.Push.Changes[0].New.Target.Hash)
		return
	}

	mft, err := manifest.ReadManifest(manifestString)
	builderTrack := "stable"
	hasValidManifest := false
	if err != nil {
		log.Warn().Err(err).Str("manifest", manifestString).Msgf("Deserializing Estafette manifest for repo %v and revision %v failed, continuing though so developer gets useful feedback", pushEvent.Repository.FullName, pushEvent.Push.Changes[0].New.Target.Hash)
	} else {
		builderTrack = mft.Builder.Track
		hasValidManifest = true
	}

	log.Debug().Interface("pushEvent", pushEvent).Interface("manifest", mft).Msgf("Estafette manifest for repo %v and revision %v exists creating a builder job...", pushEvent.Repository.FullName, pushEvent.Push.Changes[0].New.Target.Hash)

	// inject steps
	mft, err = estafette.InjectSteps(mft, builderTrack, "bitbucket")
	if err != nil {
		log.Error().Err(err).
			Msg("Failed injecting steps")
		return
	}

	// get authenticated url for the repository
	authenticatedRepositoryURL, err := w.apiClient.GetAuthenticatedRepositoryURL(accessToken, pushEvent.Repository.Links.HTML.Href)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving authenticated repository failed")
		return
	}

	// get autoincrement number
	autoincrement, err := w.cockroachDBClient.GetAutoIncrement("bitbucket", pushEvent.Repository.FullName)
	if err != nil {
		log.Warn().Err(err).
			Msgf("Failed generating autoincrement for Bitbucket repository %v", pushEvent.Repository.FullName)
	}

	// set build version number
	buildVersion := ""
	buildStatus := "failed"
	if hasValidManifest {
		buildVersion = mft.Version.Version(manifest.EstafetteVersionParams{
			AutoIncrement: autoincrement,
			Branch:        pushEvent.Push.Changes[0].New.Name,
			Revision:      pushEvent.Push.Changes[0].New.Target.Hash,
		})
		buildStatus = "running"
	}

	// store build in db
	err = w.cockroachDBClient.InsertBuild(contracts.Build{
		RepoSource:   "bitbucket.org",
		RepoOwner:    strings.Split(pushEvent.Repository.FullName, "/")[0],
		RepoName:     strings.Split(pushEvent.Repository.FullName, "/")[1],
		RepoBranch:   pushEvent.Push.Changes[0].New.Name,
		RepoRevision: pushEvent.Push.Changes[0].New.Target.Hash,
		BuildVersion: buildVersion,
		BuildStatus:  buildStatus,
		Labels:       "",
		Manifest:     manifestString,
	})
	if err != nil {
		log.Warn().Err(err).
			Msgf("Failed inserting build into db for Bitbucket repository %v", pushEvent.Repository.FullName)
	}

	// define ci builder params
	ciBuilderParams := estafette.CiBuilderParams{
		RepoSource:           "bitbucket.org",
		RepoFullName:         pushEvent.Repository.FullName,
		RepoURL:              authenticatedRepositoryURL,
		RepoBranch:           pushEvent.Push.Changes[0].New.Name,
		RepoRevision:         pushEvent.Push.Changes[0].New.Target.Hash,
		EnvironmentVariables: map[string]string{"ESTAFETTE_BITBUCKET_API_TOKEN": accessToken.AccessToken},
		Track:                builderTrack,
		AutoIncrement:        autoincrement,
		VersionNumber:        buildVersion,
		HasValidManifest:     hasValidManifest,
		Manifest:             mft,
	}

	// create ci builder job
	_, err = w.CiBuilderClient.CreateCiBuilderJob(ciBuilderParams)
	if err != nil {
		log.Error().Err(err).
			Interface("params", ciBuilderParams).
			Msgf("Creating estafette-ci-builder job for Bitbucket repository %v revision %v failed", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)

		return
	}

	log.Info().
		Interface("params", ciBuilderParams).
		Msgf("Created estafette-ci-builder job for Bitbucket repository %v revision %v", ciBuilderParams.RepoFullName, ciBuilderParams.RepoRevision)
}
