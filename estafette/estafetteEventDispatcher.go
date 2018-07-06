package estafette

import (
	"sync"

	"github.com/estafette/estafette-ci-api/cockroach"
)

// EventDispatcher dispatches events pushed to channels to the workers
type EventDispatcher interface {
	// A pool of workers channels that are registered with the dispatcher
	Run()
}

type eventDispatcherImpl struct {
	waitGroup              *sync.WaitGroup
	stopChannel            <-chan struct{}
	ciBuilderWorkerPool    chan chan CiBuilderEvent
	maxWorkers             int
	ciBuilderClient        CiBuilderClient
	cockroachDBClient      cockroach.DBClient
	ciBuilderEventsChannel chan CiBuilderEvent
}

// NewEstafetteDispatcher returns a new estafette.EventWorker to handle events channeled by estafette.EventDispatcher
func NewEstafetteDispatcher(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, maxWorkers int, ciBuilderClient CiBuilderClient, cockroachDBClient cockroach.DBClient, ciBuilderEventsChannel chan CiBuilderEvent) EventDispatcher {
	return &eventDispatcherImpl{
		waitGroup:              waitGroup,
		stopChannel:            stopChannel,
		ciBuilderWorkerPool:    make(chan chan CiBuilderEvent, maxWorkers),
		maxWorkers:             maxWorkers,
		ciBuilderClient:        ciBuilderClient,
		cockroachDBClient:      cockroachDBClient,
		ciBuilderEventsChannel: ciBuilderEventsChannel,
	}
}

// Run starts the dispatcher
func (d *eventDispatcherImpl) Run() {
	// starting n number of workers
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewEstafetteEventWorker(d.stopChannel, d.waitGroup, d.ciBuilderWorkerPool, d.ciBuilderClient, d.cockroachDBClient)
		worker.ListenToCiBuilderEventChannels()
	}

	go d.dispatchCiBuilderEvents()
}

func (d *eventDispatcherImpl) dispatchCiBuilderEvents() {
	for {
		select {
		case pushEvent := <-d.ciBuilderEventsChannel:
			// a job request has been received
			go func(pushEvent CiBuilderEvent) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				eventsChannel := <-d.ciBuilderWorkerPool

				// dispatch the job to the worker job channel
				eventsChannel <- pushEvent
			}(pushEvent)
		}
	}
}
