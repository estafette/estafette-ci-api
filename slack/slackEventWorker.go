package slack

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToEventChannels()
}

type eventWorkerImpl struct {
	waitGroup     *sync.WaitGroup
	stopChannel   <-chan struct{}
	workerPool    chan chan SlashCommand
	eventsChannel chan SlashCommand
	apiClient     APIClient
}

// NewSlackEventWorker returns the slack.EventWorker
func NewSlackEventWorker(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, workerPool chan chan SlashCommand, apiClient APIClient) EventWorker {
	return &eventWorkerImpl{
		waitGroup:     waitGroup,
		stopChannel:   stopChannel,
		workerPool:    workerPool,
		eventsChannel: make(chan SlashCommand),
		apiClient:     apiClient,
	}
}

func (w *eventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle slack events via channels
		log.Debug().Msg("Listening to Slack events channels...")
		for {
			// register the current worker into the worker queue.
			w.workerPool <- w.eventsChannel

			select {
			case slashCommand := <-w.eventsChannel:
				go func() {
					w.waitGroup.Add(1)
					log.Debug().Interface("slashCommand", slashCommand).Msg("Received slash command via channel")
					w.waitGroup.Done()
				}()
			case <-w.stopChannel:
				log.Debug().Msg("Stopping Slack event worker...")
				return
			}
		}
	}()
}
