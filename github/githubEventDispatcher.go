package github

import (
	"sync"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/estafette"
	ghcontracts "github.com/estafette/estafette-ci-api/github/contracts"
)

// EventDispatcher dispatches events pushed to channels to the workers
type EventDispatcher interface {
	// A pool of workers channels that are registered with the dispatcher
	Run()
}

type eventDispatcherImpl struct {
	waitGroup         *sync.WaitGroup
	stopChannel       <-chan struct{}
	workerPool        chan chan ghcontracts.PushEvent
	maxWorkers        int
	eventsChannel     chan ghcontracts.PushEvent
	apiClient         APIClient
	ciBuilderClient   estafette.CiBuilderClient
	cockroachDBClient cockroach.DBClient
}

// NewGithubDispatcher returns a new github.EventWorker to handle events channeled by github.EventDispatcher
func NewGithubDispatcher(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, maxWorkers int, apiClient APIClient, ciBuilderClient estafette.CiBuilderClient, cockroachDBClient cockroach.DBClient, eventsChannel chan ghcontracts.PushEvent) EventDispatcher {
	return &eventDispatcherImpl{
		waitGroup:         waitGroup,
		stopChannel:       stopChannel,
		workerPool:        make(chan chan ghcontracts.PushEvent, maxWorkers),
		maxWorkers:        maxWorkers,
		eventsChannel:     eventsChannel,
		apiClient:         apiClient,
		ciBuilderClient:   ciBuilderClient,
		cockroachDBClient: cockroachDBClient,
	}
}

// Run starts the dispatcher
func (d *eventDispatcherImpl) Run() {
	// starting n number of workers
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewGithubEventWorker(d.stopChannel, d.waitGroup, d.workerPool, d.apiClient, d.ciBuilderClient, d.cockroachDBClient)
		worker.ListenToEventChannels()
	}

	go d.dispatch()
}

func (d *eventDispatcherImpl) dispatch() {
	for {
		select {
		case pushEvent := <-d.eventsChannel:
			// a job request has been received
			go func(pushEvent ghcontracts.PushEvent) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				eventsChannel := <-d.workerPool

				// dispatch the job to the worker job channel
				eventsChannel <- pushEvent
			}(pushEvent)
		}
	}
}
