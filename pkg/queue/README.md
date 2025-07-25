# queue

Repository-agnostic task queue with first-class support for immediate, delayed, and periodic execution.

## Features

- **One-time tasks** - Execute immediately or with delay
- **Periodic tasks** - Schedule jobs with flexible intervals (hourly, daily, weekly, monthly)
- **Priority queue** - Tasks processed by priority (0-100 scale)
- **Retry mechanism** - Automatic retries with configurable limits and dead letter queue

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/queue"
```

## Usage

```go
// Define your task payload
type SendEmailPayload struct {
    UserID int64
    Email  string
}

// Create enqueuer with repository implementation
enqueuer, err := queue.NewEnqueuer(repo)
if err != nil {
    return err
}

// Enqueue a task with 5 minute delay
err = enqueuer.Enqueue(ctx,
    SendEmailPayload{UserID: 42, Email: "user@example.com"},
    queue.WithDelay(5*time.Minute),
    queue.WithPriority(queue.PriorityHigh),
)
```

## Common Operations

### Process Tasks with Worker

```go
// Create worker and register handlers
worker, _ := queue.NewWorker(repo,
    queue.WithWorkerPullInterval(time.Second),
    queue.WithWorkerConcurrency(10),
)

// Register typed handler
worker.RegisterHandler(queue.NewTaskHandler(func(ctx context.Context, payload SendEmailPayload) error {
    // Send email logic here
    return nil
}))

// Start processing
worker.Start(ctx)
```

### Schedule Periodic Tasks

```go
// Create scheduler
scheduler, _ := queue.NewScheduler(repo)

// Add daily cleanup task
scheduler.AddTask("cleanup_sessions",
    queue.DailyAt(2, 0), // 2:00 AM
    queue.WithTaskPriority(queue.PriorityLow),
)

// Add periodic handler
worker.RegisterHandler(queue.NewPeriodicTaskHandler("cleanup_sessions", func(ctx context.Context) error {
    // Cleanup logic here
    return nil
}))

// Start scheduler
go scheduler.Start(ctx)
```

## Error Handling

```go
// Package errors:
var (
    ErrRepositoryNil         = errors.New("repository cannot be nil")
    ErrPayloadNil            = errors.New("payload cannot be nil")
    ErrInvalidPriority       = errors.New("priority must be between 0 and 100")
    ErrHandlerNotFound       = errors.New("no handler registered for task type")
    ErrNoHandlers            = errors.New("no task handlers registered")
    ErrTaskAlreadyRegistered = errors.New("task already registered")
)

// Usage:
if errors.Is(err, queue.ErrHandlerNotFound) {
    // handle missing handler
}
```

## Configuration

```go
// Enqueuer options
enqueuer, _ := queue.NewEnqueuer(repo,
    queue.WithDefaultQueue("emails"),
    queue.WithDefaultPriority(queue.PriorityMedium),
)

// Worker options
worker, _ := queue.NewWorker(repo,
    queue.WithWorkerQueues("default", "emails", "notifications"),
    queue.WithWorkerConcurrency(20),
    queue.WithWorkerLockTimeout(5*time.Minute),
)

// Scheduler options
scheduler, _ := queue.NewScheduler(repo,
    queue.WithCheckInterval(30*time.Second),
)
```

## API Documentation

```bash
# Full API documentation
go doc github.com/dmitrymomot/saaskit/pkg/queue

# Specific function or type
go doc github.com/dmitrymomot/saaskit/pkg/queue.Schedule
```

## Notes

- Tasks are immutable once persisted; retries tracked via RetryCount field
- Failed tasks automatically move to dead letter queue after max retries
- Repository interfaces allow any storage backend (PostgreSQL, Redis, MongoDB)
