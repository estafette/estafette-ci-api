package pool

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
)

// Pool to manage, interact with worker pool
type Pool[J, R any] interface {
	// SendJobs to job que
	SendJobs(jobs ...J)
	// Close closes job que and returns results channel
	Close() <-chan R
	// Errors returns slice of JobError, in case of successful retires intermittent errors are not returned.
	// It will wait for results channel to be closed
	Errors() []JobError
}

type JobError struct {
	Job any
	Err error
}

type singleStagePool[J, R any] struct {
	*Config[J, R]
	running int
	mutex   sync.Mutex
	jobs    chan J
	results chan R
	errors  []JobError
}

// NewPool creates new instance of worker pool and starts workers
func NewPool[J, R any](ctx context.Context, config *Config[J, R]) (Pool[J, R], error) {
	p := &singleStagePool[J, R]{
		Config:  config,
		running: config.Size,
		mutex:   sync.Mutex{},
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	p.startPool(ctx, make(chan J, p.JobQueueLimit), make(chan R, p.ResultQueueLimit))
	return p, nil
}

func (p *singleStagePool[J, R]) validate() error {
	if p.Size <= 0 {
		return errors.New("expected pool size to be more than 0")
	}
	if p.JobQueueLimit <= 0 {
		return errors.New("expected JobQueueLimit to be than 0")
	}
	if p.ResultQueueLimit <= 0 {
		return errors.New("expected ResultQueueLimit to be than 0")
	}
	if p.Worker == nil {
		return fmt.Errorf("expected worker func to be not nil")
	}
	return nil
}

func (p *singleStagePool[J, R]) startPool(ctx context.Context, jobs chan J, results chan R) {
	p.jobs = jobs
	p.results = results
	for index := 0; index < p.Size; index++ {
		go p.startWorker(ctx)
	}
}

func (p *singleStagePool[J, R]) startWorker(ctx context.Context) {
	if p.HandlePanic {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in worker: %+v", r)
				p.removeWorker()
			}
		}()
	}
	for job := range p.jobs {
	ABC:
		if result, err := p.Worker(ctx, job); err == nil {
			p.results <- result
		} else {
			if p.MaxRetry > 0 {
				p.MaxRetry--
				goto ABC
			} else {
				p.errors = append(p.errors, JobError{
					Job: job,
					Err: err,
				})
			}
		}
	}
	p.removeWorker()
}

func (p *singleStagePool[J, R]) removeWorker() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.running--
	if p.running == 0 {
		close(p.results)
	}
}

func (p *singleStagePool[J, R]) SendJobs(jobs ...J) {
	for _, job := range jobs {
		p.jobs <- job
	}
}

func (p *singleStagePool[J, R]) Close() <-chan R {
	close(p.jobs)
	return p.results
}

func (p *singleStagePool[J, R]) Errors() []JobError {
	<-p.results
	return p.errors
}
