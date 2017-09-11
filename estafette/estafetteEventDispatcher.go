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
	buildJobLogsWorkerPool chan chan cockroach.BuildJobLogs
	maxWorkers             int
	ciBuilderClient        CiBuilderClient
	cockroachDBClient      cockroach.DBClient
	ciBuilderEventsChannel chan CiBuilderEvent
	buildJobLogsChannel    chan cockroach.BuildJobLogs
}

// NewEstafetteDispatcher returns a new estafette.EventWorker to handle events channeled by estafette.EventDispatcher
func NewEstafetteDispatcher(stopChannel <-chan struct{}, waitGroup *sync.WaitGroup, maxWorkers int, ciBuilderClient CiBuilderClient, cockroachDBClient cockroach.DBClient, ciBuilderEventsChannel chan CiBuilderEvent, buildJobLogsChannel chan cockroach.BuildJobLogs) EventDispatcher {
	return &eventDispatcherImpl{
		waitGroup:              waitGroup,
		stopChannel:            stopChannel,
		ciBuilderWorkerPool:    make(chan chan CiBuilderEvent, maxWorkers),
		buildJobLogsWorkerPool: make(chan chan cockroach.BuildJobLogs, maxWorkers),
		maxWorkers:             maxWorkers,
		ciBuilderClient:        ciBuilderClient,
		cockroachDBClient:      cockroachDBClient,
		ciBuilderEventsChannel: ciBuilderEventsChannel,
		buildJobLogsChannel:    buildJobLogsChannel,
	}
}

// Run starts the dispatcher
func (d *eventDispatcherImpl) Run() {
	// starting n number of workers
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewEstafetteEventWorker(d.stopChannel, d.waitGroup, d.ciBuilderWorkerPool, d.buildJobLogsWorkerPool, d.ciBuilderClient, d.cockroachDBClient)
		worker.ListenToCiBuilderEventChannels()
		worker.ListenToBuildJobLogsEventChannels()
	}

	go d.dispatchCiBuilderEvents()
	go d.dispatchBuildJobLogsEvents()
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

func (d *eventDispatcherImpl) dispatchBuildJobLogsEvents() {
	for {
		select {
		case pushEvent := <-d.buildJobLogsChannel:
			// a job request has been received
			go func(pushEvent cockroach.BuildJobLogs) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				eventsChannel := <-d.buildJobLogsWorkerPool

				// dispatch the job to the worker job channel
				eventsChannel <- pushEvent
			}(pushEvent)
		}
	}
}
