package github

import (
	"sync"

	"github.com/estafette/estafette-ci-api/estafette"
)

// EventDispatcher dispatches events pushed to channels to the workers
type EventDispatcher interface {
	// A pool of workers channels that are registered with the dispatcher
	Run()
}

type eventDispatcherImpl struct {
	waitGroup       *sync.WaitGroup
	stopChannel     <-chan struct{}
	workerPool      chan chan PushEvent
	maxWorkers      int
	eventsChannel   chan PushEvent
	apiClient       APIClient
	ciBuilderClient estafette.CiBuilderClient
}

// NewGithubDispatcher returns a new github.EventWorker to handle events channeled by github.EventDispatcher
func NewGithubDispatcher(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, maxWorkers int, apiClient APIClient, ciBuilderClient estafette.CiBuilderClient, eventsChannel chan PushEvent) EventDispatcher {
	return &eventDispatcherImpl{
		waitGroup:       waitGroup,
		stopChannel:     stopChannel,
		workerPool:      make(chan chan PushEvent, maxWorkers),
		maxWorkers:      maxWorkers,
		eventsChannel:   eventsChannel,
		apiClient:       apiClient,
		ciBuilderClient: ciBuilderClient,
	}
}

// Run starts the dispatcher
func (d *eventDispatcherImpl) Run() {
	// starting n number of workers
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewGithubEventWorker(d.stopChannel, d.waitGroup, d.workerPool, d.apiClient, d.ciBuilderClient)
		worker.ListenToEventChannels()
	}

	go d.dispatch()
}

func (d *eventDispatcherImpl) dispatch() {
	for {
		select {
		case pushEvent := <-d.eventsChannel:
			// a job request has been received
			go func(pushEvent PushEvent) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				eventsChannel := <-d.workerPool

				// dispatch the job to the worker job channel
				eventsChannel <- pushEvent
			}(pushEvent)
		}
	}
}
