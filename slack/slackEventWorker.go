package slack

import (
	"sync"

	"github.com/rs/zerolog/log"
)

// EventWorker processes events pushed to channels
type EventWorker interface {
	ListenToEventChannels()
	Stop()
}

type eventWorkerImpl struct {
	waitGroup     *sync.WaitGroup
	quitChannel   chan bool
	eventsChannel chan SlashCommand
	apiClient     APIClient
}

// NewSlackEventWorker returns the slack.EventWorker
func NewSlackEventWorker(waitGroup *sync.WaitGroup, apiClient APIClient, eventsChannel chan SlashCommand) EventWorker {
	return &eventWorkerImpl{
		waitGroup:     waitGroup,
		quitChannel:   make(chan bool),
		eventsChannel: eventsChannel,
		apiClient:     apiClient,
	}
}

func (w *eventWorkerImpl) ListenToEventChannels() {
	go func() {
		// handle slack events via channels
		log.Debug().Msg("Listening to Slack events channels...")
		for {
			select {
			case slashCommand := <-w.eventsChannel:
				go func() {
					w.waitGroup.Add(1)
					log.Debug().Interface("slashCommand", slashCommand).Msg("Received slash command via channel")
					w.waitGroup.Done()
				}()
			case <-w.quitChannel:
				log.Debug().Msg("Stopping Slack event worker...")
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
