package estafette

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToCiBuilderEventChannels()
	RemoveJobForEstafetteBuild(CiBuilderEvent) error
	UpdateBuildStatus(CiBuilderEvent) error
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
		for {
			// register the current worker into the worker queue.
			w.ciBuilderWorkerPool <- w.ciBuilderEventsChannel

			select {
			case ciBuilderEvent := <-w.ciBuilderEventsChannel:
				go func(ciBuilderEvent CiBuilderEvent) {
					w.waitGroup.Add(1)
					err := w.UpdateBuildStatus(ciBuilderEvent)
					if err != nil {
						log.Error().Err(err).Msgf("Failed updating build status for job %v, not removing the job", ciBuilderEvent.JobName)
					} else {
						err = w.RemoveJobForEstafetteBuild(ciBuilderEvent)
						if err != nil {
							log.Error().Err(err).Msgf("Failed removing job %v", ciBuilderEvent.JobName)
						}
					}
					w.waitGroup.Done()
				}(ciBuilderEvent)
			case <-w.stopChannel:
				log.Debug().Msg("Stopping Estafette event worker...")
				return
			}
		}
	}()
}

func (w *eventWorkerImpl) RemoveJobForEstafetteBuild(ciBuilderEvent CiBuilderEvent) (err error) {

	// create ci builder job
	err = w.ciBuilderClient.RemoveCiBuilderJob(ciBuilderEvent.JobName)
	if err != nil {
		log.Error().Err(err).
			Str("jobName", ciBuilderEvent.JobName).
			Msgf("Removing ci-builder job %v failed", ciBuilderEvent.JobName)
		return
	}

	return
}

func (w *eventWorkerImpl) UpdateBuildStatus(ciBuilderEvent CiBuilderEvent) (err error) {

	// check build status for backwards compatibility of builder
	if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.ReleaseID != "" {

		releaseID, err := strconv.Atoi(ciBuilderEvent.ReleaseID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Converting release id %v to integer for job %v failed", ciBuilderEvent.ReleaseID, ciBuilderEvent.JobName)

			return err
		}

		err = w.cockroachDBClient.UpdateReleaseStatus(ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, releaseID, ciBuilderEvent.BuildStatus)
		if err != nil {
			log.Error().Err(err).
				Msgf("Updating release status for job %v failed", ciBuilderEvent.JobName)

			return err
		}

	} else if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.BuildID != "" {

		buildID, err := strconv.Atoi(ciBuilderEvent.BuildID)
		if err != nil {
			log.Error().Err(err).
				Msgf("Converting build id %v to integer for job %v failed", ciBuilderEvent.BuildID, ciBuilderEvent.JobName)

			return err
		}

		err = w.cockroachDBClient.UpdateBuildStatusByID(ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, buildID, ciBuilderEvent.BuildStatus)
		if err != nil {
			log.Error().Err(err).
				Msgf("Updating build status for job %v failed", ciBuilderEvent.JobName)

			return err
		}

	} else if ciBuilderEvent.BuildStatus != "" {

		err := w.cockroachDBClient.UpdateBuildStatus(ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, ciBuilderEvent.RepoBranch, ciBuilderEvent.RepoRevision, ciBuilderEvent.BuildStatus)
		if err != nil {
			log.Error().Err(err).
				Msgf("Updating build status for job %v failed", ciBuilderEvent.JobName)

			return err
		}
	}

	return fmt.Errorf("CiBuilderEvent has invalid state, not updating build status")
}
