package estafette

import (
	"strconv"
	"sync"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToCiBuilderEventChannels()
	RemoveJobForEstafetteBuild(CiBuilderEvent)
	UpdateBuildStatus(CiBuilderEvent)
}

type eventWorkerImpl struct {
	waitGroup              *sync.WaitGroup
	stopChannel            <-chan struct{}
	ciBuilderWorkerPool    chan chan CiBuilderEvent
	ciBuilderClient        CiBuilderClient
	cockroachDBClient      cockroach.DBClient
	ciBuilderEventsChannel chan CiBuilderEvent
}

// NewEstafetteEventWorker returns a new estafette.EventWorker
func NewEstafetteEventWorker(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, ciBuilderWorkerPool chan chan CiBuilderEvent, ciBuilderClient CiBuilderClient, cockroachDBClient cockroach.DBClient) EventWorker {
	return &eventWorkerImpl{
		waitGroup:              waitGroup,
		stopChannel:            stopChannel,
		ciBuilderWorkerPool:    ciBuilderWorkerPool,
		ciBuilderClient:        ciBuilderClient,
		cockroachDBClient:      cockroachDBClient,
		ciBuilderEventsChannel: make(chan CiBuilderEvent),
	}
}

func (w *eventWorkerImpl) ListenToCiBuilderEventChannels() {
	go func() {
		// handle estafette events via channels
		log.Debug().Msg("Listening to Estafette ci builder events channels...")
		for {
			// register the current worker into the worker queue.
			w.ciBuilderWorkerPool <- w.ciBuilderEventsChannel

			select {
			case ciBuilderEvent := <-w.ciBuilderEventsChannel:
				go func() {
					w.waitGroup.Add(1)
					w.UpdateBuildStatus(ciBuilderEvent)
					w.RemoveJobForEstafetteBuild(ciBuilderEvent)
					w.waitGroup.Done()
				}()
			case <-w.stopChannel:
				log.Debug().Msg("Stopping Estafette event worker...")
				return
			}
		}
	}()
}

func (w *eventWorkerImpl) RemoveJobForEstafetteBuild(ciBuilderEvent CiBuilderEvent) {

	// create ci builder job
	err := w.ciBuilderClient.RemoveCiBuilderJob(ciBuilderEvent.JobName)
	if err != nil {
		log.Error().Err(err).
			Str("jobName", ciBuilderEvent.JobName).
			Msgf("Removing ci-builder job %v failed", ciBuilderEvent.JobName)

		return
	}

	log.Info().
		Str("jobName", ciBuilderEvent.JobName).
		Msgf("Removed ci-builder job %v", ciBuilderEvent.JobName)
}

func (w *eventWorkerImpl) UpdateBuildStatus(ciBuilderEvent CiBuilderEvent) {

	// check build status for backwards compatibility of builder
	if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.ReleaseID != "" {

		releaseID, err := strconv.Atoi(ciBuilderEvent.ReleaseID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Converting release id %v to integer for job %v failed", ciBuilderEvent.ReleaseID, ciBuilderEvent.JobName)

			return
		}

		err = w.cockroachDBClient.UpdateReleaseStatus(ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, releaseID, ciBuilderEvent.BuildStatus)
		if err != nil {
			log.Error().Err(err).
				Msgf("Updating release status for job %v failed", ciBuilderEvent.JobName)

			return
		}

		log.Info().
			Str("jobName", ciBuilderEvent.JobName).
			Msgf("Updated release status for ci-builder job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)

	} else if ciBuilderEvent.BuildStatus != "" {

		err := w.cockroachDBClient.UpdateBuildStatus(ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, ciBuilderEvent.RepoRevision, ciBuilderEvent.BuildStatus)
		if err != nil {
			log.Error().Err(err).
				Msgf("Updating build status for job %v failed", ciBuilderEvent.JobName)

			return
		}

		log.Info().
			Str("jobName", ciBuilderEvent.JobName).
			Msgf("Updated build status for ci-builder job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)
	}
}
