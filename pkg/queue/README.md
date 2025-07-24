# queue

A repository-agnostic task queue with first-class support for immediate, delayed, and periodic execution.

## Overview

The package provides three cooperating components that together deliver a fully-featured background job system. All persistence concerns live behind small repository interfaces, allowing you to back the queue with any storage engine by implementing the required interfaces.

## Internal Usage

This package is internal to the project and provides background job processing capabilities for other project components.

## Features

- One-time, delayed, and periodic task scheduling
- Storage-agnostic design via repository interfaces
- Priority-based task processing (0-100 scale)
- Named queues for work segregation
- Configurable concurrency with automatic lock extension
- Built-in retry logic with dead-letter queue support

## Usage

### Basic Example

```go
import (
    "context"
    "time"

    "github.com/dmitrymomot/saaskit/pkg/queue"
)

// Basic task enqueueing
func example(repo queue.EnqueuerRepository) error {
    e, err := queue.NewEnqueuer(repo)
    if err != nil {
        return err
    }

    payload := struct {
        UserID int64 `json:"user_id"`
    }{UserID: 42}

    // Execute within the next minute
    return e.Enqueue(context.Background(), payload,
        queue.WithDelay(time.Minute),
    )
}
```

### Worker & Handler

```go
// Set up worker with handler
w, _ := queue.NewWorker(repo,
    queue.WithQueues("default", "priority"),
    queue.WithMaxConcurrentTasks(4),
)

// Register strongly-typed handler
handler := queue.NewTaskHandler(func(ctx context.Context, payload struct{
    UserID int64 `json:"user_id"`
}) error {
    // Process the task
    return nil
})

_ = w.RegisterHandler(handler)

// Start processing
ctx, cancel := context.WithCancel(context.Background())
go func() {
    _ = w.Start(ctx)
}()
defer cancel()
```

### Periodic Jobs

```go
// Set up scheduler
s, _ := queue.NewScheduler(repo,
    queue.WithCheckInterval(30*time.Second),
)

// Add periodic task
_ = s.AddTask("cleanup_sessions",
    queue.DailyAt(2, 0), // runs daily at 02:00
    queue.WithTaskPriority(queue.PriorityLow),
)

// Start scheduler
go s.Start(context.Background())
```

### Error Handling

```go
// Check for specific errors
if errors.Is(err, queue.ErrInvalidPriority) {
    // Handle invalid priority
}
```

## Best Practices

### Integration Guidelines

- Run Enqueuer, Scheduler, and Worker in separate services for horizontal scalability
- Implement repository interfaces once and reuse across all components
- Use named queues to route high-priority work to dedicated workers
- Call ExtendLockForTask for long-running tasks to maintain locks

### Project-Specific Considerations

- Configure workers via environment variables using the Config struct
- Tune WithMaxConcurrentTasks and WithPullInterval based on workload
- Create appropriate database indices for queue queries
- Monitor dead-letter queue for failed tasks requiring manual intervention

## API Reference

### Configuration Variables

```go
type Config struct {
    PollInterval       time.Duration `env:"QUEUE_POLL_INTERVAL" envDefault:"5s"`
    LockTimeout        time.Duration `env:"QUEUE_LOCK_TIMEOUT" envDefault:"5m"`
    ShutdownTimeout    time.Duration `env:"QUEUE_SHUTDOWN_TIMEOUT" envDefault:"30s"`
    MaxConcurrentTasks int           `env:"QUEUE_MAX_CONCURRENT_TASKS" envDefault:"10"`
}
```

### Types

```go
// Core task structure
type Task struct {
    ID          uuid.UUID
    Queue       string
    TaskType    TaskType   // TaskTypeOneTime | TaskTypePeriodic
    TaskName    string
    Payload     []byte
    Status      TaskStatus // pending, processing, completed, failed
    Priority    Priority   // 0-100
    RetryCount  int8
    MaxRetries  int8
    ScheduledAt time.Time
    LockedUntil *time.Time
    LockedBy    *uuid.UUID
    ProcessedAt *time.Time
    Error       *string
    CreatedAt   time.Time
}

// Dead-letter queue entry
type TasksDlq struct {
    ID         uuid.UUID
    TaskID     uuid.UUID
    Queue      string
    TaskType   TaskType
    TaskName   string
    Payload    []byte
    Priority   Priority
    Error      string
    RetryCount int8
    FailedAt   time.Time
    CreatedAt  time.Time
}

// Repository interfaces
type EnqueuerRepository interface {
    CreateTask(ctx context.Context, task *Task) error
}

type SchedulerRepository interface {
    CreateTask(ctx context.Context, task *Task) error
    GetPendingTaskByName(ctx context.Context, taskName string) (*Task, error)
}

type WorkerRepository interface {
    ClaimTask(ctx context.Context, workerID uuid.UUID, queues []string, lockDuration time.Duration) (*Task, error)
    CompleteTask(ctx context.Context, taskID uuid.UUID) error
    FailTask(ctx context.Context, taskID uuid.UUID, errorMsg string) error
    MoveToDLQ(ctx context.Context, taskID uuid.UUID) error
    ExtendLock(ctx context.Context, taskID uuid.UUID, duration time.Duration) error
}
```

### Functions

```go
// Constructors
func NewEnqueuer(repo EnqueuerRepository, opts ...EnqueuerOption) (*Enqueuer, error)
func NewScheduler(repo SchedulerRepository, opts ...SchedulerOption) (*Scheduler, error)
func NewWorker(repo WorkerRepository, opts ...WorkerOption) (*Worker, error)

// Handler factories
func NewTaskHandler[T any](handler TaskHandlerFunc[T]) Handler
func NewPeriodicTaskHandler(name string, handler PeriodicTaskHandlerFunc) Handler

// Schedule builders
func EveryMinute() Schedule
func EveryInterval(d time.Duration) Schedule
func EveryMinutes(n int) Schedule
func EveryHours(n int) Schedule
func Hourly() Schedule
func HourlyAt(minute int) Schedule
func Daily() Schedule
func DailyAt(hour, minute int) Schedule
func Weekly(weekday time.Weekday) Schedule
func WeeklyOn(weekday time.Weekday, hour, minute int) Schedule
func Monthly(day int) Schedule
func MonthlyOn(day, hour, minute int) Schedule
```

### Methods

```go
// Enqueuer
(*Enqueuer).Enqueue(ctx context.Context, payload any, opts ...EnqueueOption) error

// Worker
(*Worker).RegisterHandler(handler Handler) error
(*Worker).RegisterHandlers(handlers ...Handler) error
(*Worker).Start(ctx context.Context) error
(*Worker).Stop() error
(*Worker).Run(ctx context.Context) func() error
(*Worker).ExtendLockForTask(ctx context.Context, taskID uuid.UUID, extension time.Duration) error
(*Worker).WorkerInfo() (id string, hostname string, pid int)

// Scheduler
(*Scheduler).AddTask(name string, schedule Schedule, opts ...SchedulerTaskOption) error
(*Scheduler).Start(ctx context.Context) error
(*Scheduler).RemoveTask(name string)
(*Scheduler).ListTasks() []string
```

### Error Types

```go
var (
    ErrRepositoryNil          error
    ErrPayloadNil             error
    ErrPayloadMarshal         error
    ErrTaskCreate             error
    ErrInvalidPriority        error
    ErrNoItemsToEnqueue       error
    ErrHandlerNotFound        error
    ErrNoHandlers             error
    ErrInvalidSchedule        error
    ErrTaskAlreadyRegistered  error
    ErrSchedulerNotConfigured error
    ErrNoScheduleSpecified    error
    ErrFailedToGetNextTask    error
    ErrFailedToUpdateTaskStatus error
    ErrFailedToMoveToDLQ      error
)
```
