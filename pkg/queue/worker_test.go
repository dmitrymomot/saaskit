package queue_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

// Mock repository for worker tests
type mockWorkerRepo struct {
	mu         sync.Mutex
	tasks      map[uuid.UUID]*queue.Task
	dlq        map[uuid.UUID]*queue.Task
	claimFunc  func(ctx context.Context, workerID uuid.UUID, queues []string, lockDuration time.Duration) (*queue.Task, error)
	claimCount atomic.Int32
}

func newMockWorkerRepo() *mockWorkerRepo {
	return &mockWorkerRepo{
		tasks: make(map[uuid.UUID]*queue.Task),
		dlq:   make(map[uuid.UUID]*queue.Task),
	}
}

func (m *mockWorkerRepo) ClaimTask(ctx context.Context, workerID uuid.UUID, queues []string, lockDuration time.Duration) (*queue.Task, error) {
	m.claimCount.Add(1)

	if m.claimFunc != nil {
		return m.claimFunc(ctx, workerID, queues, lockDuration)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.tasks {
		// Check queue
		queueMatch := false
		for _, q := range queues {
			if task.Queue == q {
				queueMatch = true
				break
			}
		}
		if !queueMatch {
			continue
		}

		// Check if available
		if task.Status == queue.TaskStatusPending && task.ScheduledAt.Before(time.Now()) {
			// Claim it
			task.Status = queue.TaskStatusProcessing
			lockedUntil := time.Now().Add(lockDuration)
			task.LockedUntil = &lockedUntil
			task.LockedBy = &workerID
			return &queue.Task{
				ID:          task.ID,
				Queue:       task.Queue,
				TaskType:    task.TaskType,
				TaskName:    task.TaskName,
				Payload:     task.Payload,
				Status:      task.Status,
				Priority:    task.Priority,
				RetryCount:  task.RetryCount,
				MaxRetries:  task.MaxRetries,
				ScheduledAt: task.ScheduledAt,
				LockedUntil: task.LockedUntil,
				LockedBy:    task.LockedBy,
				CreatedAt:   task.CreatedAt,
			}, nil
		}
	}

	return nil, queue.ErrNoTaskToClaim
}

func (m *mockWorkerRepo) CompleteTask(ctx context.Context, taskID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != queue.TaskStatusProcessing {
		return fmt.Errorf("task %s not in processing state", taskID)
	}

	task.Status = queue.TaskStatusCompleted
	now := time.Now()
	task.ProcessedAt = &now
	return nil
}

func (m *mockWorkerRepo) FailTask(ctx context.Context, taskID uuid.UUID, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Status = queue.TaskStatusFailed
	task.Error = &errorMsg
	task.RetryCount++

	// Reset to pending if retries remain
	if task.RetryCount < task.MaxRetries {
		task.Status = queue.TaskStatusPending
		// Add backoff
		task.ScheduledAt = time.Now().Add(time.Duration(task.RetryCount) * time.Second)
		task.LockedUntil = nil
		task.LockedBy = nil
	}

	return nil
}

func (m *mockWorkerRepo) MoveToDLQ(ctx context.Context, taskID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Move to DLQ
	m.dlq[taskID] = task
	delete(m.tasks, taskID)

	return nil
}

func (m *mockWorkerRepo) ExtendLock(ctx context.Context, taskID uuid.UUID, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != queue.TaskStatusProcessing {
		return fmt.Errorf("task %s not in processing state", taskID)
	}

	lockedUntil := time.Now().Add(duration)
	task.LockedUntil = &lockedUntil
	return nil
}

func (m *mockWorkerRepo) addTask(task *queue.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID] = task
}

// Test payload types
type testPayload struct {
	Message string `json:"message"`
	Value   int    `json:"value"`
}

func TestWorker_NewWorker(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)
		require.NotNil(t, worker)
	})

	t.Run("nil repository error", func(t *testing.T) {
		t.Parallel()

		worker, err := queue.NewWorker(nil)
		assert.ErrorIs(t, err, queue.ErrRepositoryNil)
		assert.Nil(t, worker)
	})

	t.Run("with options", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo,
			queue.WithQueues("queue1", "queue2"),
			queue.WithPullInterval(1*time.Second),
			queue.WithLockTimeout(10*time.Minute),
			queue.WithMaxConcurrentTasks(5),
		)
		require.NoError(t, err)
		require.NotNil(t, worker)
	})
}

func TestWorker_RegisterHandler(t *testing.T) {
	t.Parallel()

	t.Run("register single handler", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return nil
		})

		err = worker.RegisterHandler(handler)
		assert.NoError(t, err)
	})

	t.Run("register multiple handlers", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)

		handler1 := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return nil
		})
		handler2 := queue.NewPeriodicTaskHandler("periodic-task", func(ctx context.Context) error {
			return nil
		})

		err = worker.RegisterHandlers(handler1, handler2)
		assert.NoError(t, err)
	})

	t.Run("register nil handler", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)

		err = worker.RegisterHandler(nil)
		assert.NoError(t, err) // Should not error on nil
	})
}

func TestWorker_StartStop(t *testing.T) {
	t.Parallel()

	t.Run("start and stop successfully", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Let it run for a bit
		time.Sleep(100 * time.Millisecond)

		err = worker.Stop()
		assert.NoError(t, err)
	})

	t.Run("start without handlers", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)

		err = worker.Start(context.Background())
		assert.ErrorIs(t, err, queue.ErrNoHandlers)
	})

	t.Run("double start error", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		err = worker.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")

		_ = worker.Stop()
	})

	t.Run("stop without start", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)

		err = worker.Stop()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

func TestWorker_ProcessTask(t *testing.T) {
	t.Parallel()

	t.Run("successful task processing", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		processed := make(chan testPayload, 1)
		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			processed <- payload
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		// Add a task
		payload := testPayload{Message: "test", Value: 42}
		payloadBytes, _ := json.Marshal(payload)
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "queue_test.testPayload",
			Payload:     payloadBytes,
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			MaxRetries:  3,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		repo.addTask(task)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing
		select {
		case p := <-processed:
			assert.Equal(t, payload.Message, p.Message)
			assert.Equal(t, payload.Value, p.Value)
		case <-time.After(2 * time.Second):
			t.Fatal("task not processed in time")
		}

		// Verify task completed
		repo.mu.Lock()
		assert.Equal(t, queue.TaskStatusCompleted, repo.tasks[task.ID].Status)
		repo.mu.Unlock()

		_ = worker.Stop()
	})

	t.Run("task failure with retry", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		attempts := atomic.Int32{}
		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			attempts.Add(1)
			return errors.New("processing failed")
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		// Add a task
		payload := testPayload{Message: "fail", Value: 0}
		payloadBytes, _ := json.Marshal(payload)
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "queue_test.testPayload",
			Payload:     payloadBytes,
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			RetryCount:  0,
			MaxRetries:  2,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		repo.addTask(task)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for first attempt
		time.Sleep(200 * time.Millisecond)

		// Verify task failed but can retry
		repo.mu.Lock()
		assert.Equal(t, queue.TaskStatusPending, repo.tasks[task.ID].Status)
		assert.Equal(t, int8(1), repo.tasks[task.ID].RetryCount)
		assert.NotNil(t, repo.tasks[task.ID].Error)
		repo.mu.Unlock()

		_ = worker.Stop()
	})

	t.Run("task failure to DLQ after max retries", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return errors.New("permanent failure")
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		// Add a task already at max retries minus 1
		// When it fails, FailTask will increment to MaxRetries
		// Worker checks pre-increment value, so we need RetryCount = MaxRetries - 1
		payload := testPayload{Message: "dlq", Value: 0}
		payloadBytes, _ := json.Marshal(payload)
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "queue_test.testPayload",
			Payload:     payloadBytes,
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			RetryCount:  3, // Already at max, so worker will move to DLQ
			MaxRetries:  3,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		repo.addTask(task)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(200 * time.Millisecond)

		// Verify task moved to DLQ
		repo.mu.Lock()
		_, inTasks := repo.tasks[task.ID]
		_, inDLQ := repo.dlq[task.ID]
		repo.mu.Unlock()

		assert.False(t, inTasks, "task should not be in regular tasks")
		assert.True(t, inDLQ, "task should be in DLQ")

		_ = worker.Stop()
	})

	t.Run("missing handler moves to DLQ", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		// Register handler for different task type
		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		// Add a task with unregistered handler
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "unregistered.Handler",
			Payload:     []byte("{}"),
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			MaxRetries:  3,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		repo.addTask(task)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(200 * time.Millisecond)

		// Verify task moved to DLQ
		repo.mu.Lock()
		_, inTasks := repo.tasks[task.ID]
		_, inDLQ := repo.dlq[task.ID]
		repo.mu.Unlock()

		assert.False(t, inTasks, "task should not be in regular tasks")
		assert.True(t, inDLQ, "task should be in DLQ")

		_ = worker.Stop()
	})

	t.Run("handler panic recovery", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			panic("handler panic!")
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		// Add a task
		payload := testPayload{Message: "panic", Value: 0}
		payloadBytes, _ := json.Marshal(payload)
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "queue_test.testPayload",
			Payload:     payloadBytes,
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			RetryCount:  0,
			MaxRetries:  3,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		repo.addTask(task)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(200 * time.Millisecond)

		// Verify task failed but didn't crash worker
		repo.mu.Lock()
		assert.Equal(t, queue.TaskStatusPending, repo.tasks[task.ID].Status) // Should retry
		assert.Equal(t, int8(1), repo.tasks[task.ID].RetryCount)
		assert.NotNil(t, repo.tasks[task.ID].Error)
		assert.Contains(t, *repo.tasks[task.ID].Error, "panic")
		repo.mu.Unlock()

		// Worker should still be running
		err = worker.Stop()
		assert.NoError(t, err)
	})
}

func TestWorker_ConcurrentProcessing(t *testing.T) {
	t.Parallel()

	t.Run("processes multiple tasks concurrently", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo,
			queue.WithPullInterval(10*time.Millisecond),
			queue.WithMaxConcurrentTasks(3),
		)
		require.NoError(t, err)

		// Track concurrent executions
		concurrent := atomic.Int32{}
		maxConcurrent := atomic.Int32{}
		processed := atomic.Int32{}

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			current := concurrent.Add(1)
			defer concurrent.Add(-1)

			// Update max concurrent
			for {
				max := maxConcurrent.Load()
				if current <= max || maxConcurrent.CompareAndSwap(max, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(100 * time.Millisecond)
			processed.Add(1)
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		// Add multiple tasks
		for i := range 6 {
			payload := testPayload{Message: "concurrent", Value: i}
			payloadBytes, _ := json.Marshal(payload)
			task := &queue.Task{
				ID:          uuid.New(),
				Queue:       queue.DefaultQueueName,
				TaskType:    queue.TaskTypeOneTime,
				TaskName:    "queue_test.testPayload",
				Payload:     payloadBytes,
				Status:      queue.TaskStatusPending,
				Priority:    queue.PriorityMedium,
				MaxRetries:  3,
				ScheduledAt: time.Now().Add(-time.Minute),
				CreatedAt:   time.Now(),
			}
			repo.addTask(task)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for all tasks to process
		deadline := time.Now().Add(2 * time.Second)
		for processed.Load() < 6 && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}

		assert.Equal(t, int32(6), processed.Load(), "all tasks should be processed")
		assert.Equal(t, int32(3), maxConcurrent.Load(), "max concurrent should be 3")

		_ = worker.Stop()
	})
}

func TestWorker_GracefulShutdown(t *testing.T) {
	t.Parallel()

	t.Run("waits for active tasks to complete", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo, queue.WithPullInterval(10*time.Millisecond))
		require.NoError(t, err)

		taskStarted := make(chan struct{})
		taskCompleted := atomic.Bool{}

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			close(taskStarted)
			time.Sleep(200 * time.Millisecond)
			taskCompleted.Store(true)
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		// Add a task
		payload := testPayload{Message: "shutdown", Value: 1}
		payloadBytes, _ := json.Marshal(payload)
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "queue_test.testPayload",
			Payload:     payloadBytes,
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			MaxRetries:  3,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		repo.addTask(task)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for task to start
		<-taskStarted

		// Stop worker while task is running
		stopDone := make(chan error, 1)
		go func() {
			stopDone <- worker.Stop()
		}()

		// Stop should wait for task to complete
		select {
		case err := <-stopDone:
			assert.NoError(t, err)
			assert.True(t, taskCompleted.Load(), "task should have completed before stop returned")
		case <-time.After(1 * time.Second):
			t.Fatal("stop did not complete in time")
		}
	})
}

func TestWorker_RunFunction(t *testing.T) {
	t.Parallel()

	t.Run("run function for errgroup", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		runFunc := worker.Run(ctx)
		err = runFunc()
		assert.NoError(t, err) // Should exit cleanly when context is cancelled
	})
}

func TestWorker_ExtendLockForTask(t *testing.T) {
	t.Parallel()

	t.Run("extends lock successfully", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)

		taskID := uuid.New()
		task := &queue.Task{
			ID:          taskID,
			Status:      queue.TaskStatusProcessing,
			LockedUntil: ptrTime(time.Now().Add(time.Minute)),
		}
		repo.addTask(task)

		err = worker.ExtendLockForTask(context.Background(), taskID, 5*time.Minute)
		assert.NoError(t, err)

		// Verify lock was extended
		repo.mu.Lock()
		assert.NotNil(t, repo.tasks[taskID].LockedUntil)
		assert.True(t, repo.tasks[taskID].LockedUntil.After(time.Now().Add(4*time.Minute)))
		repo.mu.Unlock()
	})
}

func TestWorker_WorkerInfo(t *testing.T) {
	t.Parallel()

	t.Run("returns worker information", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo)
		require.NoError(t, err)

		id, hostname, pid := worker.WorkerInfo()
		assert.NotEmpty(t, id)
		assert.NotEmpty(t, hostname)
		assert.Greater(t, pid, 0)
	})
}

func TestWorker_QueueFiltering(t *testing.T) {
	t.Parallel()

	t.Run("processes only specified queues", func(t *testing.T) {
		t.Parallel()

		repo := newMockWorkerRepo()
		worker, err := queue.NewWorker(repo,
			queue.WithQueues("priority", "batch"),
			queue.WithPullInterval(50*time.Millisecond),
		)
		require.NoError(t, err)

		processed := make(map[string]int)
		mu := sync.Mutex{}

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			mu.Lock()
			processed[payload.Message]++
			mu.Unlock()
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		// Add tasks to different queues
		queues := map[string]string{
			"priority": "should-process-1",
			"batch":    "should-process-2",
			"ignored":  "should-not-process",
		}

		for queueName, message := range queues {
			payload := testPayload{Message: message, Value: 1}
			payloadBytes, _ := json.Marshal(payload)
			task := &queue.Task{
				ID:          uuid.New(),
				Queue:       queueName,
				TaskType:    queue.TaskTypeOneTime,
				TaskName:    "queue_test.testPayload",
				Payload:     payloadBytes,
				Status:      queue.TaskStatusPending,
				Priority:    queue.PriorityMedium,
				MaxRetries:  3,
				ScheduledAt: time.Now().Add(-time.Minute),
				CreatedAt:   time.Now(),
			}
			repo.addTask(task)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(300 * time.Millisecond)

		// Verify correct tasks were processed
		mu.Lock()
		assert.Equal(t, 1, processed["should-process-1"])
		assert.Equal(t, 1, processed["should-process-2"])
		assert.Equal(t, 0, processed["should-not-process"])
		mu.Unlock()

		_ = worker.Stop()
	})
}

// Helper function
func ptrTime(t time.Time) *time.Time {
	return &t
}

func TestWorkerWithLogger(t *testing.T) {
	t.Parallel()

	// Create a custom logger
	customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create worker with custom logger
	storage := queue.NewMemoryStorage()
	worker, err := queue.NewWorker(storage, queue.WithWorkerLogger(customLogger))
	require.NoError(t, err)

	// The worker should be created successfully with the custom logger
	assert.NotNil(t, worker)

	// The main purpose of this test is to ensure the logger option is accepted
	// and doesn't cause any issues during initialization
}
