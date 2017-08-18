package main

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// EstafetteEventWorker processes events pushed to channels
type EstafetteEventWorker interface {
	ListenToEventChannels()
	Stop()
	RemoveJobForEstafetteBuild(EstafetteCiBuilderEvent)
}

type estafetteEventWorkerImpl struct {
	WaitGroup   *sync.WaitGroup
	QuitChannel chan bool
}

func newEstafetteEventWorker(waitGroup *sync.WaitGroup) EstafetteEventWorker {
	return &estafetteEventWorkerImpl{
		WaitGroup:   waitGroup,
		QuitChannel: make(chan bool)}
}

func (w *estafetteEventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle estafette events via channels
		log.Debug().Msg("Listening to Estafette events channels...")
		for {
			select {
			case ciBuilderEvent := <-estafetteCiBuilderEvents:
				log.Debug().Interface("event", ciBuilderEvent).Msgf("Received event for job %v from channel", ciBuilderEvent.JobName)
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

func (w *estafetteEventWorkerImpl) Stop() {
	go func() {
		w.QuitChannel <- true
	}()
}

func (w *estafetteEventWorkerImpl) RemoveJobForEstafetteBuild(ciBuilderEvent EstafetteCiBuilderEvent) {

	// create ci builder client
	ciBuilderClient, err := newCiBuilderClient()
	if err != nil {
		log.Error().Err(err).Msg("Initializing ci builder client failed")
		return
	}

	// create ci builder job
	err = ciBuilderClient.RemoveCiBuilderJob(ciBuilderEvent.JobName)
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
