package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EnqueuerRepository defines the interface for task creation
type EnqueuerRepository interface {
	CreateTask(ctx context.Context, task *Task) error
}

// Enqueuer handles task enqueueing
type Enqueuer struct {
	repo            EnqueuerRepository
	defaultQueue    string
	defaultPriority Priority
}

// NewEnqueuer creates a new Enqueuer
func NewEnqueuer(repo EnqueuerRepository, opts ...EnqueuerOption) (*Enqueuer, error) {
	if repo == nil {
		return nil, ErrRepositoryNil
	}

	options := &enqueuerOptions{
		defaultQueue:    DefaultQueueName,
		defaultPriority: PriorityDefault,
	}

	for _, opt := range opts {
		opt(options)
	}

	return &Enqueuer{
		repo:            repo,
		defaultQueue:    options.defaultQueue,
		defaultPriority: options.defaultPriority,
	}, nil
}

// Enqueue adds a new task to the queue
func (e *Enqueuer) Enqueue(ctx context.Context, payload any, opts ...EnqueueOption) error {
	if payload == nil {
		return ErrPayloadNil
	}

	// Apply default options
	options := &enqueueOptions{
		queue:      e.defaultQueue,
		priority:   e.defaultPriority,
		maxRetries: 3,
	}

	// Apply user options
	for _, opt := range opts {
		opt(options)
	}

	// Validate priority
	if !options.priority.Valid() {
		return ErrInvalidPriority
	}

	// Build and store task
	task, err := e.buildTask(payload, options)
	if err != nil {
		return err
	}

	// Store task
	if err := e.repo.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create task %q in queue %q: %w", task.TaskName, task.Queue, err)
	}

	return nil
}

// buildTask constructs a Task from payload and options
func (e *Enqueuer) buildTask(payload any, options *enqueueOptions) (*Task, error) {
	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload of type %T: %w", payload, err)
	}

	// Determine task name
	taskName := options.taskName
	if taskName == "" {
		taskName = qualifiedStructName(payload)
	}

	// Calculate scheduled time
	scheduledAt := time.Now()
	if options.scheduledAt != nil {
		scheduledAt = *options.scheduledAt
	} else if options.delay > 0 {
		scheduledAt = scheduledAt.Add(options.delay)
	}

	return &Task{
		ID:          uuid.New(),
		Queue:       options.queue,
		TaskType:    TaskTypeOneTime,
		TaskName:    taskName,
		Payload:     payloadBytes,
		Status:      TaskStatusPending,
		Priority:    options.priority,
		RetryCount:  0,
		MaxRetries:  options.maxRetries,
		ScheduledAt: scheduledAt,
		CreatedAt:   time.Now(),
	}, nil
}
