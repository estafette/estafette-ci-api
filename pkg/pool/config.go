package pool

import "context"

// Config for a worker pool,
type Config[J, R any] struct {
	// Size of the pool
	Size int
	// MaxRetry for failed jobs, it is cumulative for all jobs
	MaxRetry int
	// JobQueueLimit for jobs channel, will block SendJobs call until full
	JobQueueLimit int
	// ResultQueueLimit for results channel, will block SendJobs call until full
	ResultQueueLimit int
	// HandlePanic for jobs that fail with panic
	HandlePanic bool
	// Worker of the pool
	Worker func(context.Context, J) (R, error)
}

// DefaultConfig returns a new Config[J, R] with JobQueueLimit and ResultQueueLimit equal to 100 * size
func DefaultConfig[J, R any](Size int, Worker func(ctx context.Context, job J) (R, error)) *Config[J, R] {
	return &Config[J, R]{
		Size:             Size,
		JobQueueLimit:    100 * Size,
		ResultQueueLimit: 100 * Size,
		MaxRetry:         0,
		HandlePanic:      false,
		Worker:           Worker,
	}
}

// NewConfig returns a new Config[J, R]
func NewConfig[J, R any](Size, JobQueueLimit, ResultQueueLimit, MaxRetry int, HandlePanic bool, Worker func(ctx context.Context, job J) (R, error)) *Config[J, R] {
	return &Config[J, R]{
		Size:             Size,
		JobQueueLimit:    JobQueueLimit,
		ResultQueueLimit: ResultQueueLimit,
		MaxRetry:         MaxRetry,
		HandlePanic:      HandlePanic,
		Worker:           Worker,
	}
}
