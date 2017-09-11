package slack

import (
	"sync"
)

// EventDispatcher dispatches events pushed to channels to the workers
type EventDispatcher interface {
	// A pool of workers channels that are registered with the dispatcher
	Run()
}

type eventDispatcherImpl struct {
	waitGroup     *sync.WaitGroup
	stopChannel   <-chan struct{}
	workerPool    chan chan SlashCommand
	maxWorkers    int
	eventsChannel chan SlashCommand
	apiClient     APIClient
}

// NewSlackDispatcher returns a new slack.EventWorker to handle events channeled by slack.EventDispatcher
func NewSlackDispatcher(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, maxWorkers int, apiClient APIClient, eventsChannel chan SlashCommand) EventDispatcher {
	return &eventDispatcherImpl{
		waitGroup:     waitGroup,
		stopChannel:   stopChannel,
		workerPool:    make(chan chan SlashCommand, maxWorkers),
		maxWorkers:    maxWorkers,
		eventsChannel: eventsChannel,
		apiClient:     apiClient,
	}
}

// Run starts the dispatcher
func (d *eventDispatcherImpl) Run() {
	// starting n number of workers
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewSlackEventWorker(d.stopChannel, d.waitGroup, d.workerPool, d.apiClient)
		worker.ListenToEventChannels()
	}

	go d.dispatch()
}

func (d *eventDispatcherImpl) dispatch() {
	for {
		select {
		case pushEvent := <-d.eventsChannel:
			// a job request has been received
			go func(pushEvent SlashCommand) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				eventsChannel := <-d.workerPool

				// dispatch the job to the worker job channel
				eventsChannel <- pushEvent
			}(pushEvent)
		}
	}
}
