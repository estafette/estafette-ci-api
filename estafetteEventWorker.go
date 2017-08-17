package main

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// EstafetteWorker processes events pushed to channels
type EstafetteWorker interface {
	ListenToEstafetteBuildFinishedEventChannel()
	Stop()
	RemoveJobForEstafetteBuild(EstafetteBuildFinishedEvent)
}

type estafetteWorkerImpl struct {
	WaitGroup   *sync.WaitGroup
	QuitChannel chan bool
}

func newEstafetteWorker(waitGroup *sync.WaitGroup) EstafetteWorker {
	return &estafetteWorkerImpl{
		WaitGroup:   waitGroup,
		QuitChannel: make(chan bool)}
}

func (w *estafetteWorkerImpl) ListenToEstafetteBuildFinishedEventChannel() {
	go func() {
		// handle estafette 'build finished' events via channels
		log.Debug().Msg("Listening to Estafette 'build finished' events channel...")
		for {
			select {
			case buildFinishedEvent := <-estafetteBuildFinishedEvents:
				w.WaitGroup.Add(1)
				w.RemoveJobForEstafetteBuild(buildFinishedEvent)
				w.WaitGroup.Done()
			case <-w.QuitChannel:
				log.Info().Msg("Stopping Estafette worker...")
				return
			}
		}
	}()
}

func (w *estafetteWorkerImpl) Stop() {
	go func() {
		w.QuitChannel <- true
	}()
}

func (w *estafetteWorkerImpl) RemoveJobForEstafetteBuild(buildFinishedEvent EstafetteBuildFinishedEvent) {

	// create ci builder client
	ciBuilderClient, err := newCiBuilderClient()
	if err != nil {
		log.Error().Err(err).Msg("Initializing ci builder client failed")
		return
	}

	// create ci builder job
	err = ciBuilderClient.RemoveCiBuilderJob(buildFinishedEvent.JobName)
	if err != nil {
		log.Error().Err(err).
			Str("jobName", buildFinishedEvent.JobName).
			Msgf("Removing ci-builder job %v failed", buildFinishedEvent.JobName)

		return
	}

	log.Debug().
		Str("jobName", buildFinishedEvent.JobName).
		Msgf("Removed ci-builder job %v", buildFinishedEvent.JobName)
}
