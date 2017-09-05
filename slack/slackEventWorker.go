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
	eventsChannel chan SlashCommand
	apiClient     APIClient
}

// NewSlackEventWorker returns the slack.EventWorker
func NewSlackEventWorker(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, apiClient APIClient, eventsChannel chan SlashCommand) EventWorker {
	return &eventWorkerImpl{
		waitGroup:     waitGroup,
		stopChannel:   stopChannel,
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
			case <-w.stopChannel:
				log.Debug().Msg("Stopping Slack event worker...")
				return
			}
		}
	}()
}
