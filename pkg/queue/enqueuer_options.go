package queue

import "time"

// EnqueuerOption is a functional option for configuring an Enqueuer
type EnqueuerOption func(*enqueuerOptions)

type enqueuerOptions struct {
	defaultQueue    string
	defaultPriority Priority
}

// WithDefaultQueue sets the default queue name
func WithDefaultQueue(queue string) EnqueuerOption {
	return func(o *enqueuerOptions) {
		if queue != "" {
			o.defaultQueue = queue
		}
	}
}

// WithDefaultPriority sets the default priority
func WithDefaultPriority(priority Priority) EnqueuerOption {
	return func(o *enqueuerOptions) {
		if priority.Valid() {
			o.defaultPriority = priority
		}
	}
}

// EnqueueOption is a functional option for the Enqueue method
type EnqueueOption func(*enqueueOptions)

type enqueueOptions struct {
	queue       string
	priority    Priority
	maxRetries  int8
	delay       time.Duration
	scheduledAt *time.Time
	taskName    string
}

// WithQueue sets the queue for the task
func WithQueue(queue string) EnqueueOption {
	return func(o *enqueueOptions) {
		if queue != "" {
			o.queue = queue
		}
	}
}

// WithPriority sets the priority for the task
func WithPriority(priority Priority) EnqueueOption {
	return func(o *enqueueOptions) {
		o.priority = priority
	}
}

// WithMaxRetries sets the maximum number of retries (0-10)
// Capped at 10 to prevent infinite retry loops on persistent failures
func WithMaxRetries(maxRetries int8) EnqueueOption {
	return func(o *enqueueOptions) {
		if maxRetries >= 0 && maxRetries <= 10 {
			o.maxRetries = maxRetries
		}
	}
}

// WithDelay sets a delay before the task can be processed
func WithDelay(delay time.Duration) EnqueueOption {
	return func(o *enqueueOptions) {
		if delay > 0 {
			o.delay = delay
		}
	}
}

// WithScheduledAt sets a specific time for the task to be processed
func WithScheduledAt(scheduledAt time.Time) EnqueueOption {
	return func(o *enqueueOptions) {
		o.scheduledAt = &scheduledAt
	}
}

// WithTaskName sets a custom task name
func WithTaskName(name string) EnqueueOption {
	return func(o *enqueueOptions) {
		if name != "" {
			o.taskName = name
		}
	}
}
