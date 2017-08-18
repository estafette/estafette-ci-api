package estafette

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToEventChannels()
	Stop()
	RemoveJobForEstafetteBuild(CiBuilderEvent)
}

type eventWorkerImpl struct {
	waitGroup       *sync.WaitGroup
	quitChannel     chan bool
	ciBuilderClient CiBuilderClient
	eventsChannel   chan CiBuilderEvent
}

// NewEstafetteEventWorker returns a new estafette.EventWorker
func NewEstafetteEventWorker(waitGroup *sync.WaitGroup, ciBuilderClient CiBuilderClient, eventsChannel chan CiBuilderEvent) EventWorker {
	return &eventWorkerImpl{
		waitGroup:       waitGroup,
		quitChannel:     make(chan bool),
		ciBuilderClient: ciBuilderClient,
		eventsChannel:   eventsChannel,
	}
}

func (w *eventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle estafette events via channels
		log.Debug().Msg("Listening to Estafette events channels...")
		for {
			select {
			case ciBuilderEvent := <-w.eventsChannel:
				go func() {
					w.waitGroup.Add(1)
					w.RemoveJobForEstafetteBuild(ciBuilderEvent)
					w.waitGroup.Done()
				}()
			case <-w.quitChannel:
				log.Debug().Msg("Stopping Estafette event worker...")
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
