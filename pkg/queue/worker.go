package queue

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkerRepository defines the interface for worker operations
type WorkerRepository interface {
	// ClaimTask atomically claims the next available task
	ClaimTask(ctx context.Context, workerID uuid.UUID, queues []string, lockDuration time.Duration) (*Task, error)

	// CompleteTask marks task as completed
	CompleteTask(ctx context.Context, taskID uuid.UUID) error

	// FailTask marks task as failed and increments retry count
	FailTask(ctx context.Context, taskID uuid.UUID, errorMsg string) error

	// MoveToDLQ moves task to dead letter queue
	MoveToDLQ(ctx context.Context, taskID uuid.UUID) error

	// ExtendLock extends the lock timeout for long-running tasks (optional)
	ExtendLock(ctx context.Context, taskID uuid.UUID, duration time.Duration) error
}

// Worker processes tasks from the queue
type Worker struct {
	repo     WorkerRepository
	handlers map[string]Handler
	queues   []string
	workerID uuid.UUID
	sem      chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex

	// Configuration
	pullInterval time.Duration
	lockTimeout  time.Duration
	logger       *slog.Logger

	// State management
	ctx    context.Context
	cancel context.CancelFunc
}

// NewWorker creates a new task worker
func NewWorker(repo WorkerRepository, opts ...WorkerOption) (*Worker, error) {
	if repo == nil {
		return nil, ErrRepositoryNil
	}

	// Default options
	options := &workerOptions{
		queues:             []string{DefaultQueueName},
		pullInterval:       5 * time.Second,
		lockTimeout:        5 * time.Minute,
		maxConcurrentTasks: 1,
		logger:             slog.Default(),
	}

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	return &Worker{
		repo:         repo,
		handlers:     make(map[string]Handler),
		queues:       options.queues,
		workerID:     uuid.New(),
		sem:          make(chan struct{}, options.maxConcurrentTasks),
		pullInterval: options.pullInterval,
		lockTimeout:  options.lockTimeout,
		logger:       options.logger,
	}, nil
}

// RegisterHandler registers a single task handler
func (w *Worker) RegisterHandler(handler Handler) error {
	if handler == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.handlers[handler.Name()] = handler
	return nil
}

// RegisterHandlers registers multiple task handlers
func (w *Worker) RegisterHandlers(handlers ...Handler) error {
	for _, h := range handlers {
		if err := w.RegisterHandler(h); err != nil {
			return err
		}
	}
	return nil
}

// Start begins processing tasks in the background
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.cancel != nil {
		w.mu.Unlock()
		return fmt.Errorf("worker already started")
	}

	if len(w.handlers) == 0 {
		w.mu.Unlock()
		return ErrNoHandlers
	}

	w.ctx, w.cancel = context.WithCancel(ctx)
	w.mu.Unlock()

	// Start the main processing loop
	go w.run()

	w.logger.Info("worker started",
		slog.String("worker_id", w.workerID.String()),
		slog.Any("queues", w.queues),
		slog.Int("max_concurrent", cap(w.sem)))

	return nil
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop() error {
	w.mu.Lock()
	if w.cancel == nil {
		w.mu.Unlock()
		return fmt.Errorf("worker not started")
	}
	cancel := w.cancel
	w.cancel = nil
	w.mu.Unlock()

	// Cancel context to stop processing
	cancel()

	// Wait for all active tasks to complete
	w.logger.Info("worker stopping, waiting for active tasks to complete",
		slog.String("worker_id", w.workerID.String()))

	w.wg.Wait()

	w.logger.Info("worker stopped",
		slog.String("worker_id", w.workerID.String()))

	return nil
}

// Run starts the worker and returns a function suitable for errgroup
func (w *Worker) Run(ctx context.Context) func() error {
	return func() error {
		if err := w.Start(ctx); err != nil {
			return err
		}

		<-ctx.Done()

		return w.Stop()
	}
}

// run is the main processing loop
func (w *Worker) run() {
	ticker := time.NewTicker(w.pullInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// Try to acquire a slot
			select {
			case w.sem <- struct{}{}:
				// Got a slot, process task in background
				w.wg.Add(1)
				go func() {
					defer w.wg.Done()
					defer func() { <-w.sem }() // Release slot

					if err := w.pullAndProcess(); err != nil {
						if err != ErrHandlerNotFound {
							w.logger.Error("failed to process task",
								slog.String("worker_id", w.workerID.String()),
								slog.String("error", err.Error()))
						}
					}
				}()
			default:
				// All slots busy, skip this tick
				w.logger.Debug("all worker slots busy, skipping tick",
					slog.String("worker_id", w.workerID.String()))
			}
		}
	}
}

// pullAndProcess pulls a task and processes it
func (w *Worker) pullAndProcess() error {
	// Claim next available task
	task, err := w.repo.ClaimTask(w.ctx, w.workerID, w.queues, w.lockTimeout)
	if err != nil {
		// No task available is expected
		return nil
	}

	w.logger.Debug("claimed task",
		slog.String("worker_id", w.workerID.String()),
		slog.String("task_id", task.ID.String()),
		slog.String("task_name", task.TaskName),
		slog.String("queue", task.Queue))

	// Process the task
	return w.processTask(task)
}

// processTask executes a task with its handler
func (w *Worker) processTask(task *Task) error {
	// Find handler
	w.mu.RLock()
	handler, ok := w.handlers[task.TaskName]
	w.mu.RUnlock()

	if !ok {
		return w.handleMissingHandler(task)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(w.ctx, w.lockTimeout)
	defer cancel()

	// Execute handler
	start := time.Now()
	err := handler.Handle(ctx, task.Payload)
	duration := time.Since(start)

	if err != nil {
		return w.handleTaskFailure(task, err, duration)
	}

	return w.handleTaskSuccess(task, duration)
}

// handleMissingHandler processes tasks that have no registered handler
func (w *Worker) handleMissingHandler(task *Task) error {
	w.logger.Error("no handler registered for task type",
		slog.String("worker_id", w.workerID.String()),
		slog.String("task_id", task.ID.String()),
		slog.String("task_name", task.TaskName))

	// Move to DLQ since no handler means no retry will help
	if err := w.repo.MoveToDLQ(w.ctx, task.ID); err != nil {
		return fmt.Errorf("failed to move task %s to DLQ: %w", task.ID, err)
	}

	return ErrHandlerNotFound
}

// handleTaskFailure processes failed task execution
func (w *Worker) handleTaskFailure(task *Task, execErr error, duration time.Duration) error {
	w.logger.Error("task failed",
		slog.String("worker_id", w.workerID.String()),
		slog.String("task_id", task.ID.String()),
		slog.String("task_name", task.TaskName),
		slog.Int("retry_count", int(task.RetryCount)),
		slog.Int("max_retries", int(task.MaxRetries)),
		slog.Duration("duration", duration),
		slog.String("error", execErr.Error()))

	// Check if this was the last retry
	if task.RetryCount >= task.MaxRetries {
		// Move to DLQ
		if err := w.repo.MoveToDLQ(w.ctx, task.ID); err != nil {
			return fmt.Errorf("failed to move task %s to DLQ after max retries: %w", task.ID, err)
		}

		w.logger.Warn("task moved to dead letter queue",
			slog.String("worker_id", w.workerID.String()),
			slog.String("task_id", task.ID.String()),
			slog.String("task_name", task.TaskName))

		return nil
	}

	// Mark as failed (will be retried)
	if err := w.repo.FailTask(w.ctx, task.ID, execErr.Error()); err != nil {
		return fmt.Errorf("failed to update task %s status to failed: %w", task.ID, err)
	}

	return nil
}

// handleTaskSuccess processes successful task completion
func (w *Worker) handleTaskSuccess(task *Task, duration time.Duration) error {
	if err := w.repo.CompleteTask(w.ctx, task.ID); err != nil {
		return fmt.Errorf("failed to mark task %s as completed: %w", task.ID, err)
	}

	w.logger.Info("task completed successfully",
		slog.String("worker_id", w.workerID.String()),
		slog.String("task_id", task.ID.String()),
		slog.String("task_name", task.TaskName),
		slog.String("queue", task.Queue),
		slog.Duration("duration", duration))

	return nil
}

// ExtendLockForTask extends the lock timeout for a long-running task
// This should be called periodically for tasks that take longer than lockTimeout
func (w *Worker) ExtendLockForTask(ctx context.Context, taskID uuid.UUID, extension time.Duration) error {
	return w.repo.ExtendLock(ctx, taskID, extension)
}

// WorkerInfo returns information about the worker
func (w *Worker) WorkerInfo() (id string, hostname string, pid int) {
	hostname, _ = os.Hostname()
	return w.workerID.String(), hostname, os.Getpid()
}
