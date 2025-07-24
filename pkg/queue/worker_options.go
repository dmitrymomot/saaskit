package queue

import (
	"log/slog"
	"time"
)

// WorkerOption is a functional option for configuring a worker
type WorkerOption func(*workerOptions)

type workerOptions struct {
	queues             []string
	pullInterval       time.Duration
	lockTimeout        time.Duration
	maxConcurrentTasks int
	logger             *slog.Logger
}

// WithQueues sets which queues the worker should pull from
func WithQueues(queues ...string) WorkerOption {
	return func(o *workerOptions) {
		o.queues = queues
	}
}

// WithPullInterval sets how often the worker checks for new tasks
func WithPullInterval(d time.Duration) WorkerOption {
	return func(o *workerOptions) {
		if d > 0 {
			o.pullInterval = d
		}
	}
}

// WithLockTimeout sets the lock duration for tasks
func WithLockTimeout(d time.Duration) WorkerOption {
	return func(o *workerOptions) {
		if d > 0 {
			o.lockTimeout = d
		}
	}
}

// WithMaxConcurrentTasks sets the maximum number of concurrent tasks
func WithMaxConcurrentTasks(n int) WorkerOption {
	return func(o *workerOptions) {
		if n > 0 {
			o.maxConcurrentTasks = n
		}
	}
}

// WithWorkerLogger sets the logger for the worker
func WithWorkerLogger(logger *slog.Logger) WorkerOption {
	return func(o *workerOptions) {
		if logger != nil {
			o.logger = logger
		}
	}
}
