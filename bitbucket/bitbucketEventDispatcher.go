package bitbucket

import (
	"sync"

	"github.com/estafette/estafette-ci-api/cockroach"
	"github.com/estafette/estafette-ci-api/estafette"
)

// EventDispatcher dispatches events pushed to channels to the workers
type EventDispatcher interface {
	// A pool of workers channels that are registered with the dispatcher
	Run()
}

type eventDispatcherImpl struct {
	waitGroup         *sync.WaitGroup
	stopChannel       <-chan struct{}
	workerPool        chan chan RepositoryPushEvent
	maxWorkers        int
	eventsChannel     chan RepositoryPushEvent
	apiClient         APIClient
	ciBuilderClient   estafette.CiBuilderClient
	cockroachDBClient cockroach.DBClient
}

// NewBitbucketDispatcher returns a new github.EventWorker to handle events channeled by bitbucket.EventDispatcher
func NewBitbucketDispatcher(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, maxWorkers int, apiClient APIClient, ciBuilderClient estafette.CiBuilderClient, cockroachDBClient cockroach.DBClient, eventsChannel chan RepositoryPushEvent) EventDispatcher {
	return &eventDispatcherImpl{
		waitGroup:         waitGroup,
		stopChannel:       stopChannel,
		workerPool:        make(chan chan RepositoryPushEvent, maxWorkers),
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
		worker := NewBitbucketEventWorker(d.stopChannel, d.waitGroup, d.workerPool, d.apiClient, d.ciBuilderClient, d.cockroachDBClient)
		worker.ListenToEventChannels()
	}

	go d.dispatch()
}

func (d *eventDispatcherImpl) dispatch() {
	for {
		select {
		case pushEvent := <-d.eventsChannel:
			// a job request has been received
			go func(pushEvent RepositoryPushEvent) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				eventsChannel := <-d.workerPool

				// dispatch the job to the worker job channel
				eventsChannel <- pushEvent
			}(pushEvent)
		}
	}
}
