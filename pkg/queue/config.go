package queue

import "time"

// Config holds the configuration for the task queue
type Config struct {
	PollInterval       time.Duration `env:"QUEUE_POLL_INTERVAL" envDefault:"5s"`
	LockTimeout        time.Duration `env:"QUEUE_LOCK_TIMEOUT" envDefault:"5m"`
	ShutdownTimeout    time.Duration `env:"QUEUE_SHUTDOWN_TIMEOUT" envDefault:"30s"`
	MaxConcurrentTasks int           `env:"QUEUE_MAX_CONCURRENT_TASKS" envDefault:"10"`
}
