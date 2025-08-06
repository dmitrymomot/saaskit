package queue_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

// MockWorkerRepository is a mock implementation of WorkerRepository
type MockWorkerRepository struct {
	mock.Mock
}

func (m *MockWorkerRepository) ClaimTask(ctx context.Context, workerID uuid.UUID, queues []string, lockDuration time.Duration) (*queue.Task, error) {
	args := m.Called(ctx, workerID, queues, lockDuration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*queue.Task), args.Error(1)
}

func (m *MockWorkerRepository) CompleteTask(ctx context.Context, taskID uuid.UUID) error {
	args := m.Called(ctx, taskID)
	return args.Error(0)
}

func (m *MockWorkerRepository) FailTask(ctx context.Context, taskID uuid.UUID, errorMsg string) error {
	args := m.Called(ctx, taskID, errorMsg)
	return args.Error(0)
}

func (m *MockWorkerRepository) MoveToDLQ(ctx context.Context, taskID uuid.UUID) error {
	args := m.Called(ctx, taskID)
	return args.Error(0)
}

func (m *MockWorkerRepository) ExtendLock(ctx context.Context, taskID uuid.UUID, duration time.Duration) error {
	args := m.Called(ctx, taskID, duration)
	return args.Error(0)
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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		worker, err := queue.NewWorker(mockRepo)
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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		worker, err := queue.NewWorker(mockRepo,
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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		worker, err := queue.NewWorker(mockRepo)
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return nil
		})

		err = worker.RegisterHandler(handler)
		assert.NoError(t, err)
	})

	t.Run("register multiple handlers", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		worker, err := queue.NewWorker(mockRepo)
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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		worker, err := queue.NewWorker(mockRepo)
		require.NoError(t, err)

		err = worker.RegisterHandler(nil)
		assert.NoError(t, err) // Should not error on nil
	})
}

func TestWorker_StartStop(t *testing.T) {
	t.Parallel()

	t.Run("start and stop successfully", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Expect ClaimTask to be called multiple times and return no tasks
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()

		worker, err := queue.NewWorker(mockRepo, queue.WithPullInterval(50*time.Millisecond))
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
		time.Sleep(20 * time.Millisecond)

		err = worker.Stop()
		assert.NoError(t, err)
	})

	t.Run("start without handlers", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		worker, err := queue.NewWorker(mockRepo)
		require.NoError(t, err)

		err = worker.Start(context.Background())
		assert.ErrorIs(t, err, queue.ErrNoHandlers)
	})

	t.Run("double start error", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Expect ClaimTask to be called multiple times
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()

		worker, err := queue.NewWorker(mockRepo)
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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		worker, err := queue.NewWorker(mockRepo)
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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Create task
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

		// Set up expectations
		// First call returns the task, subsequent calls return no task
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(task, nil).Once()
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()
		mockRepo.On("CompleteTask", mock.Anything, task.ID).Return(nil).Once()

		worker, err := queue.NewWorker(mockRepo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		processed := make(chan testPayload, 1)
		handler := queue.NewTaskHandler(func(ctx context.Context, p testPayload) error {
			processed <- p
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

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

		_ = worker.Stop()
	})

	t.Run("task failure with retry", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Create task
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

		// Set up expectations
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(task, nil).Once()
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()
		mockRepo.On("FailTask", mock.Anything, task.ID, "processing failed").Return(nil).Once()

		worker, err := queue.NewWorker(mockRepo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		attempts := atomic.Int32{}
		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			attempts.Add(1)
			return errors.New("processing failed")
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for first attempt
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, int32(1), attempts.Load())

		_ = worker.Stop()
	})

	t.Run("task failure to DLQ after max retries", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Create task already at max retries
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

		// Set up expectations
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(task, nil).Once()
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()
		mockRepo.On("FailTask", mock.Anything, task.ID, "permanent failure").Return(nil).Once()
		mockRepo.On("MoveToDLQ", mock.Anything, task.ID).Return(nil).Once()

		worker, err := queue.NewWorker(mockRepo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return errors.New("permanent failure")
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(150 * time.Millisecond)

		_ = worker.Stop()
	})

	t.Run("missing handler moves to DLQ", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Create task with unregistered handler
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

		// Set up expectations
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(task, nil).Once()
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()
		mockRepo.On("FailTask", mock.Anything, task.ID, "no handler registered for task type: unregistered.Handler").Return(nil).Once()
		mockRepo.On("MoveToDLQ", mock.Anything, task.ID).Return(nil).Once()

		worker, err := queue.NewWorker(mockRepo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		// Register handler for different task type
		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(150 * time.Millisecond)

		_ = worker.Stop()
	})

	t.Run("handler panic recovery", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Create task
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

		// Set up expectations
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(task, nil).Once()
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()
		mockRepo.On("FailTask", mock.Anything, task.ID, mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "panic")
		})).Return(nil).Once()

		worker, err := queue.NewWorker(mockRepo, queue.WithPullInterval(50*time.Millisecond))
		require.NoError(t, err)

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			panic("handler panic!")
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing - needs more time than pull interval
		// to ensure task is claimed, processed (panic), and FailTask is called
		time.Sleep(150 * time.Millisecond)

		// Worker should still be running
		err = worker.Stop()
		assert.NoError(t, err)
	})
}

func TestWorker_ConcurrentProcessing(t *testing.T) {
	t.Parallel()

	t.Run("processes multiple tasks concurrently", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Create multiple tasks
		tasks := make([]*queue.Task, 6)
		for i := range 6 {
			payload := testPayload{Message: "concurrent", Value: i}
			payloadBytes, _ := json.Marshal(payload)
			tasks[i] = &queue.Task{
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
		}

		// Set up expectations - tasks will be claimed and completed
		// Set up claim expectations sequentially
		for _, task := range tasks {
			mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
				Return(task, nil).Once()
		}
		// After all tasks are claimed, return no task
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()

		// Expect CompleteTask for each task
		for _, task := range tasks {
			mockRepo.On("CompleteTask", mock.Anything, task.ID).Return(nil).Once()
		}

		worker, err := queue.NewWorker(mockRepo,
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
			time.Sleep(20 * time.Millisecond)
			processed.Add(1)
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Create task
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

		// Set up expectations
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(task, nil).Once()
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()
		mockRepo.On("CompleteTask", mock.Anything, task.ID).Return(nil).Once()

		worker, err := queue.NewWorker(mockRepo, queue.WithPullInterval(10*time.Millisecond))
		require.NoError(t, err)

		taskStarted := make(chan struct{})
		taskCompleted := atomic.Bool{}

		handler := queue.NewTaskHandler(func(ctx context.Context, payload testPayload) error {
			close(taskStarted)
			time.Sleep(50 * time.Millisecond)
			taskCompleted.Store(true)
			return nil
		})
		err = worker.RegisterHandler(handler)
		require.NoError(t, err)

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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Expect ClaimTask to be called and return no tasks
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{queue.DefaultQueueName}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()

		worker, err := queue.NewWorker(mockRepo, queue.WithPullInterval(50*time.Millisecond))
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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		taskID := uuid.New()

		// Set up expectation
		mockRepo.On("ExtendLock", mock.Anything, taskID, 5*time.Minute).Return(nil).Once()

		worker, err := queue.NewWorker(mockRepo)
		require.NoError(t, err)

		err = worker.ExtendLockForTask(context.Background(), taskID, 5*time.Minute)
		assert.NoError(t, err)
	})
}

func TestWorker_WorkerInfo(t *testing.T) {
	t.Parallel()

	t.Run("returns worker information", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		worker, err := queue.NewWorker(mockRepo)
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

		mockRepo := new(MockWorkerRepository)
		defer mockRepo.AssertExpectations(t)

		// Create tasks for different queues
		tasks := make(map[string]*queue.Task)
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
			tasks[queueName] = task
		}

		// Set up expectations - only tasks from priority and batch queues should be claimed
		// First claim returns priority task
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{"priority", "batch"}, mock.Anything).
			Return(tasks["priority"], nil).Once()
		// Second claim returns batch task
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{"priority", "batch"}, mock.Anything).
			Return(tasks["batch"], nil).Once()
		// All subsequent claims return no task
		mockRepo.On("ClaimTask", mock.Anything, mock.Anything, []string{"priority", "batch"}, mock.Anything).
			Return(nil, queue.ErrNoTaskToClaim).Maybe()

		// Expect CompleteTask for the two tasks that should be processed
		mockRepo.On("CompleteTask", mock.Anything, tasks["priority"].ID).Return(nil).Once()
		mockRepo.On("CompleteTask", mock.Anything, tasks["batch"].ID).Return(nil).Once()

		worker, err := queue.NewWorker(mockRepo,
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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = worker.Start(ctx)
		require.NoError(t, err)

		// Wait for processing with timeout
		deadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(deadline) {
			mu.Lock()
			if processed["should-process-1"] > 0 && processed["should-process-2"] > 0 {
				mu.Unlock()
				break
			}
			mu.Unlock()
			time.Sleep(5 * time.Millisecond)
		}

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
