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
	WaitGroup       *sync.WaitGroup
	QuitChannel     chan bool
	CiBuilderClient CiBuilderClient
	EventsChannel   chan CiBuilderEvent
}

// NewEstafetteEventWorker returns a new estafette.EventWorker
func NewEstafetteEventWorker(waitGroup *sync.WaitGroup, ciBuilderClient CiBuilderClient, eventsChannel chan CiBuilderEvent) EventWorker {
	return &eventWorkerImpl{
		WaitGroup:       waitGroup,
		QuitChannel:     make(chan bool),
		CiBuilderClient: ciBuilderClient,
		EventsChannel:   eventsChannel,
	}
}

func (w *eventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle estafette events via channels
		log.Debug().Msg("Listening to Estafette events channels...")
		for {
			select {
			case ciBuilderEvent := <-w.EventsChannel:
				go func() {
					w.WaitGroup.Add(1)
					w.RemoveJobForEstafetteBuild(ciBuilderEvent)
					w.WaitGroup.Done()
				}()
			case <-w.QuitChannel:
				log.Debug().Msg("Stopping Estafette event worker...")
				return
			}
		}
	}()
}

func (w *eventWorkerImpl) Stop() {
	go func() {
		w.QuitChannel <- true
	}()
}

func (w *eventWorkerImpl) RemoveJobForEstafetteBuild(ciBuilderEvent CiBuilderEvent) {

	// create ci builder job
	err := w.CiBuilderClient.RemoveCiBuilderJob(ciBuilderEvent.JobName)
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
