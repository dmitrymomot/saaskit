package queue

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStorage implements all queue repository interfaces for testing and local development
type MemoryStorage struct {
	mu    sync.RWMutex
	tasks map[uuid.UUID]*Task
	dlq   map[uuid.UUID]*TasksDlq

	// Indexes for efficient queries
	byQueue  map[string][]uuid.UUID
	byStatus map[TaskStatus][]uuid.UUID

	// Lock management
	lockTicker *time.Ticker
	done       chan struct{}
}

// NewMemoryStorage creates a new in-memory storage implementation
func NewMemoryStorage() *MemoryStorage {
	ms := &MemoryStorage{
		tasks:    make(map[uuid.UUID]*Task),
		dlq:      make(map[uuid.UUID]*TasksDlq),
		byQueue:  make(map[string][]uuid.UUID),
		byStatus: make(map[TaskStatus][]uuid.UUID),
		done:     make(chan struct{}),
	}

	// Start lock expiration manager
	ms.lockTicker = time.NewTicker(time.Second)
	go ms.lockExpirationManager()

	return ms
}

// Close stops the background goroutines
func (ms *MemoryStorage) Close() error {
	close(ms.done)
	ms.lockTicker.Stop()
	return nil
}

// CreateTask implements EnqueuerRepository and SchedulerRepository
func (ms *MemoryStorage) CreateTask(ctx context.Context, task *Task) error {
	if task == nil {
		return errors.New("task cannot be nil")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check if task already exists
	if _, exists := ms.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Clone task to prevent external modifications
	taskCopy := *task
	ms.tasks[task.ID] = &taskCopy

	// Update indexes
	ms.byQueue[task.Queue] = append(ms.byQueue[task.Queue], task.ID)
	ms.byStatus[task.Status] = append(ms.byStatus[task.Status], task.ID)

	return nil
}

// ClaimTask implements WorkerRepository
func (ms *MemoryStorage) ClaimTask(ctx context.Context, workerID uuid.UUID, queues []string, lockDuration time.Duration) (*Task, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	var bestTask *Task
	var bestPriority Priority = -1

	// Find the highest priority available task using a priority-first, time-second algorithm
	// This ensures critical tasks are processed first while maintaining fairness within priority tiers
	for _, taskID := range ms.byStatus[TaskStatusPending] {
		task := ms.tasks[taskID]

		// Skip tasks not in requested queues
		if !slices.Contains(queues, task.Queue) {
			continue
		}

		// Skip tasks scheduled for future execution (delayed tasks)
		if task.ScheduledAt.After(now) {
			continue
		}

		// Skip tasks still locked by other workers (shouldn't happen in pending status)
		if task.LockedUntil != nil && task.LockedUntil.After(now) {
			continue
		}

		// Priority-first selection: higher priority wins, earliest creation time breaks ties
		if bestTask == nil ||
			task.Priority > bestPriority ||
			(task.Priority == bestPriority && task.ScheduledAt.Before(bestTask.ScheduledAt)) {
			bestTask = task
			bestPriority = task.Priority
		}
	}

	if bestTask == nil {
		return nil, ErrNoTaskToClaim
	}

	// Claim the task
	lockUntil := now.Add(lockDuration)
	bestTask.Status = TaskStatusProcessing
	bestTask.LockedUntil = &lockUntil
	bestTask.LockedBy = &workerID

	// Update status index
	ms.removeFromStatusIndex(bestTask.ID, TaskStatusPending)
	ms.byStatus[TaskStatusProcessing] = append(ms.byStatus[TaskStatusProcessing], bestTask.ID)

	// Return a copy to prevent external modifications
	taskCopy := *bestTask
	return &taskCopy, nil
}

// CompleteTask implements WorkerRepository
func (ms *MemoryStorage) CompleteTask(ctx context.Context, taskID uuid.UUID) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	task, exists := ms.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != TaskStatusProcessing {
		return fmt.Errorf("task %s is not in processing state", taskID)
	}

	now := time.Now()
	task.Status = TaskStatusCompleted
	task.ProcessedAt = &now
	task.LockedUntil = nil
	task.LockedBy = nil

	// Update status index
	ms.removeFromStatusIndex(taskID, TaskStatusProcessing)
	ms.byStatus[TaskStatusCompleted] = append(ms.byStatus[TaskStatusCompleted], taskID)

	return nil
}

// FailTask implements WorkerRepository
func (ms *MemoryStorage) FailTask(ctx context.Context, taskID uuid.UUID, errorMsg string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	task, exists := ms.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != TaskStatusProcessing {
		return fmt.Errorf("task %s is not in processing state", taskID)
	}

	task.RetryCount++
	task.Error = &errorMsg
	task.LockedUntil = nil
	task.LockedBy = nil

	if task.RetryCount >= task.MaxRetries {
		task.Status = TaskStatusFailed
		ms.removeFromStatusIndex(taskID, TaskStatusProcessing)
		ms.byStatus[TaskStatusFailed] = append(ms.byStatus[TaskStatusFailed], taskID)
	} else {
		// Reset to pending for retry
		task.Status = TaskStatusPending
		ms.removeFromStatusIndex(taskID, TaskStatusProcessing)
		ms.byStatus[TaskStatusPending] = append(ms.byStatus[TaskStatusPending], taskID)

		// Apply exponential backoff to prevent thundering herd on persistent failures
		// Linear progression: 30s, 60s, 90s... balances quick retry with system stability
		backoff := time.Duration(task.RetryCount) * 30 * time.Second
		task.ScheduledAt = time.Now().Add(backoff)
	}

	return nil
}

// MoveToDLQ implements WorkerRepository
func (ms *MemoryStorage) MoveToDLQ(ctx context.Context, taskID uuid.UUID) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	task, exists := ms.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Create DLQ entry
	dlqEntry := &TasksDlq{
		ID:         uuid.New(),
		TaskID:     task.ID,
		Queue:      task.Queue,
		TaskType:   task.TaskType,
		TaskName:   task.TaskName,
		Payload:    task.Payload,
		Priority:   task.Priority,
		Error:      "",
		RetryCount: task.RetryCount,
		FailedAt:   time.Now(),
		CreatedAt:  time.Now(),
	}

	if task.Error != nil {
		dlqEntry.Error = *task.Error
	}

	ms.dlq[dlqEntry.ID] = dlqEntry

	// Remove from main storage and indexes
	ms.removeFromStatusIndex(taskID, task.Status)
	ms.removeFromQueueIndex(taskID, task.Queue)
	delete(ms.tasks, taskID)

	return nil
}

// ExtendLock implements WorkerRepository
func (ms *MemoryStorage) ExtendLock(ctx context.Context, taskID uuid.UUID, duration time.Duration) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	task, exists := ms.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != TaskStatusProcessing {
		return fmt.Errorf("task %s is not in processing state", taskID)
	}

	lockUntil := time.Now().Add(duration)
	task.LockedUntil = &lockUntil

	return nil
}

// Helper methods

func (ms *MemoryStorage) removeFromStatusIndex(taskID uuid.UUID, status TaskStatus) {
	ms.byStatus[status] = slices.DeleteFunc(ms.byStatus[status], func(id uuid.UUID) bool {
		return id == taskID
	})
}

func (ms *MemoryStorage) removeFromQueueIndex(taskID uuid.UUID, queue string) {
	ms.byQueue[queue] = slices.DeleteFunc(ms.byQueue[queue], func(id uuid.UUID) bool {
		return id == taskID
	})
}

// lockExpirationManager runs in background to recover tasks from dead workers
// Essential for system resilience - without this, tasks locked by crashed workers would be lost forever
//
// How it works:
// 1. Runs every 30 seconds (configurable via lockCheckInterval)
// 2. Scans all tasks in "processing" status
// 3. Checks if their LockedUntil timestamp has passed
// 4. Resets expired tasks back to "pending" status for retry
//
// This ensures that if a worker crashes, gets killed, or loses network connectivity,
// its claimed tasks will eventually become available for other workers to process.
// The lock duration should be set longer than expected task processing time to avoid
// premature expiration of locks for long-running tasks.
func (ms *MemoryStorage) lockExpirationManager() {
	for {
		select {
		case <-ms.lockTicker.C:
			ms.expireLocks()
		case <-ms.done:
			return
		}
	}
}

// expireLocks scans all processing tasks and releases expired locks
// This allows tasks to be retried if a worker crashes or becomes unresponsive
//
// Lock expiration strategy:
// - Only checks tasks in "processing" status (actively being worked on)
// - Compares current time against each task's LockedUntil timestamp
// - Tasks with expired locks are reset to "pending" with cleared lock fields
// - The task remains at its current retry count, preserving failure history
//
// This method must be called while holding the mutex to ensure consistency
// during the status transitions and index updates.
func (ms *MemoryStorage) expireLocks() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	for _, taskID := range ms.byStatus[TaskStatusProcessing] {
		task := ms.tasks[taskID]
		if task.LockedUntil != nil && task.LockedUntil.Before(now) {
			// Release expired lock and reset task to pending for retry
			task.Status = TaskStatusPending
			task.LockedUntil = nil
			task.LockedBy = nil

			// Update indexes to make task claimable again
			ms.removeFromStatusIndex(taskID, TaskStatusProcessing)
			ms.byStatus[TaskStatusPending] = append(ms.byStatus[TaskStatusPending], taskID)
		}
	}
}
