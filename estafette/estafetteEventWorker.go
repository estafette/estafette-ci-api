package estafette

import (
	"sync"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToCiBuilderEventChannels()
	ListenToBuildJobLogsEventChannels()
	RemoveJobForEstafetteBuild(CiBuilderEvent)
	InsertLogs(cockroach.BuildJobLogs)
}

type eventWorkerImpl struct {
	waitGroup              *sync.WaitGroup
	stopChannel            <-chan struct{}
	ciBuilderWorkerPool    chan chan CiBuilderEvent
	buildJobLogsWorkerPool chan chan cockroach.BuildJobLogs
	ciBuilderClient        CiBuilderClient
	cockroachDBClient      cockroach.DBClient
	ciBuilderEventsChannel chan CiBuilderEvent
	buildJobLogsChannel    chan cockroach.BuildJobLogs
}

// NewEstafetteEventWorker returns a new estafette.EventWorker
func NewEstafetteEventWorker(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, ciBuilderWorkerPool chan chan CiBuilderEvent, buildJobLogsWorkerPool chan chan cockroach.BuildJobLogs, ciBuilderClient CiBuilderClient, cockroachDBClient cockroach.DBClient) EventWorker {
	return &eventWorkerImpl{
		waitGroup:              waitGroup,
		stopChannel:            stopChannel,
		ciBuilderWorkerPool:    ciBuilderWorkerPool,
		buildJobLogsWorkerPool: buildJobLogsWorkerPool,
		ciBuilderClient:        ciBuilderClient,
		cockroachDBClient:      cockroachDBClient,
		ciBuilderEventsChannel: make(chan CiBuilderEvent),
		buildJobLogsChannel:    make(chan cockroach.BuildJobLogs),
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

func (w *eventWorkerImpl) ListenToBuildJobLogsEventChannels() {
	go func() {
		// handle estafette events via channels
		log.Debug().Msg("Listening to Estafette events channels...")
		for {
			// register the current worker into the worker queue.
			w.buildJobLogsWorkerPool <- w.buildJobLogsChannel

			select {
			case buildJobLogs := <-w.buildJobLogsChannel:
				go func() {
					w.waitGroup.Add(1)
					w.InsertLogs(buildJobLogs)
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

func (w *eventWorkerImpl) InsertLogs(buildJobLogs cockroach.BuildJobLogs) {

	err := w.cockroachDBClient.InsertBuildJobLogs(buildJobLogs)
	if err != nil {
		log.Error().Err(err).
			Interface("buildJobLogs", buildJobLogs).
			Msgf("Inserting logs for %v failed", buildJobLogs.RepoFullName)

		return
	}

	log.Info().
		Interface("buildJobLogs", buildJobLogs).
		Msgf("Inserted logs for %v", buildJobLogs.RepoFullName)
}
