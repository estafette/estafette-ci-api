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
						log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msgf("Failed updating build status for job %v to %v, not removing the job", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)
					} else if ciBuilderEvent.BuildStatus != "canceled" {
						err = w.RemoveJobForEstafetteBuild(ciBuilderEvent)
						if err != nil {
							log.Error().Err(err).Interface("ciBuilderEvent", ciBuilderEvent).Msgf("Failed removing job %v", ciBuilderEvent.JobName)
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
	return w.ciBuilderClient.RemoveCiBuilderJob(ciBuilderEvent.JobName)
}

func (w *eventWorkerImpl) UpdateBuildStatus(ciBuilderEvent CiBuilderEvent) (err error) {

	log.Debug().Interface("ciBuilderEvent", ciBuilderEvent).Msgf("UpdateBuildStatus executing...")

	if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.ReleaseID != "" {

		releaseID, err := strconv.Atoi(ciBuilderEvent.ReleaseID)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Converted release id %v", releaseID)

		err = w.cockroachDBClient.UpdateReleaseStatus(ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, releaseID, ciBuilderEvent.BuildStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated release status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)

		return err

	} else if ciBuilderEvent.BuildStatus != "" && ciBuilderEvent.BuildID != "" {

		buildID, err := strconv.Atoi(ciBuilderEvent.BuildID)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Converted build id %v", buildID)

		err = w.cockroachDBClient.UpdateBuildStatusByID(ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, buildID, ciBuilderEvent.BuildStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated build status for job %v to %v", ciBuilderEvent.JobName, ciBuilderEvent.BuildStatus)

		return err

		// check build status for backwards compatibility of builder
	} else if ciBuilderEvent.BuildStatus != "" {

		err := w.cockroachDBClient.UpdateBuildStatus(ciBuilderEvent.RepoSource, ciBuilderEvent.RepoOwner, ciBuilderEvent.RepoName, ciBuilderEvent.RepoBranch, ciBuilderEvent.RepoRevision, ciBuilderEvent.BuildStatus)
		if err != nil {
			return err
		}

		log.Debug().Msgf("Updated build status for job %v", ciBuilderEvent.JobName)

		return err
	}

	return fmt.Errorf("CiBuilderEvent has invalid state, not updating build status")
}
