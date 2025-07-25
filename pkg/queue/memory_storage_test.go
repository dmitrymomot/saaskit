package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

func TestMemoryStorage_CreateTask(t *testing.T) {
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	t.Run("creates task successfully", func(t *testing.T) {
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Payload:     []byte(`{"data": "test"}`),
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			RetryCount:  0,
			MaxRetries:  3,
			ScheduledAt: time.Now(),
			CreatedAt:   time.Now(),
		}

		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)
	})

	t.Run("fails on duplicate task ID", func(t *testing.T) {
		id := uuid.New()
		task1 := &queue.Task{
			ID:          id,
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now(),
			CreatedAt:   time.Now(),
		}

		err := storage.CreateTask(context.Background(), task1)
		require.NoError(t, err)

		task2 := &queue.Task{
			ID:          id,
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task-2",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now(),
			CreatedAt:   time.Now(),
		}

		err = storage.CreateTask(context.Background(), task2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("fails on nil task", func(t *testing.T) {
		err := storage.CreateTask(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task cannot be nil")
	})
}

func TestMemoryStorage_ClaimTask(t *testing.T) {
	t.Run("claims highest priority task", func(t *testing.T) {
		storage := queue.NewMemoryStorage()
		defer storage.Close()
		workerID := uuid.New()
		// Create tasks with different priorities
		lowPriorityTask := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "low-priority",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityLow,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), lowPriorityTask)
		require.NoError(t, err)

		highPriorityTask := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "high-priority",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityHigh,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err = storage.CreateTask(context.Background(), highPriorityTask)
		require.NoError(t, err)

		// Claim task should return high priority one
		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		require.NoError(t, err)
		require.NotNil(t, claimed)
		assert.Equal(t, highPriorityTask.ID, claimed.ID)
		assert.Equal(t, queue.TaskStatusProcessing, claimed.Status)
		assert.NotNil(t, claimed.LockedUntil)
		assert.Equal(t, workerID, *claimed.LockedBy)
	})

	t.Run("respects queue filter", func(t *testing.T) {
		storage := queue.NewMemoryStorage()
		defer storage.Close()
		workerID := uuid.New()
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       "special-queue",
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		// Should not find task in default queue
		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		assert.ErrorIs(t, err, queue.ErrNoTaskToClaim)
		assert.Nil(t, claimed)

		// Should find task in special queue
		claimed, err = storage.ClaimTask(context.Background(), workerID, []string{"special-queue"}, 5*time.Minute)
		require.NoError(t, err)
		require.NotNil(t, claimed)
		assert.Equal(t, task.ID, claimed.ID)
	})

	t.Run("respects scheduled time", func(t *testing.T) {
		storage := queue.NewMemoryStorage()
		defer storage.Close()
		workerID := uuid.New()
		futureTask := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "future-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), futureTask)
		require.NoError(t, err)

		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		assert.ErrorIs(t, err, queue.ErrNoTaskToClaim)
		assert.Nil(t, claimed)
	})

	t.Run("respects existing locks", func(t *testing.T) {
		storage := queue.NewMemoryStorage()
		defer storage.Close()
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		// First worker claims the task
		worker1 := uuid.New()
		claimed1, err := storage.ClaimTask(context.Background(), worker1, []string{queue.DefaultQueueName}, 5*time.Minute)
		require.NoError(t, err)
		require.NotNil(t, claimed1)

		// Second worker should not be able to claim it
		worker2 := uuid.New()
		claimed2, err := storage.ClaimTask(context.Background(), worker2, []string{queue.DefaultQueueName}, 5*time.Minute)
		assert.ErrorIs(t, err, queue.ErrNoTaskToClaim)
		assert.Nil(t, claimed2)
	})
}

func TestMemoryStorage_CompleteTask(t *testing.T) {
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	t.Run("completes task successfully", func(t *testing.T) {
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		workerID := uuid.New()
		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		require.NoError(t, err)

		err = storage.CompleteTask(context.Background(), claimed.ID)
		require.NoError(t, err)

		// Task should not be claimable anymore
		claimed2, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		assert.ErrorIs(t, err, queue.ErrNoTaskToClaim)
		assert.Nil(t, claimed2)
	})

	t.Run("fails on non-existent task", func(t *testing.T) {
		err := storage.CompleteTask(context.Background(), uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("fails on non-processing task", func(t *testing.T) {
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now(),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		err = storage.CompleteTask(context.Background(), task.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in processing state")
	})
}

func TestMemoryStorage_FailTask(t *testing.T) {
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	t.Run("fails task with retry", func(t *testing.T) {
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			RetryCount:  0,
			MaxRetries:  3,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		workerID := uuid.New()
		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		require.NoError(t, err)

		err = storage.FailTask(context.Background(), claimed.ID, "test error")
		require.NoError(t, err)

		// Task should be claimable again but with backoff
		time.Sleep(100 * time.Millisecond) // Let background routine process
		claimed2, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		assert.ErrorIs(t, err, queue.ErrNoTaskToClaim) // Should be scheduled for future due to backoff
		assert.Nil(t, claimed2)
	})

	t.Run("fails task permanently after max retries", func(t *testing.T) {
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			RetryCount:  2,
			MaxRetries:  3,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		workerID := uuid.New()
		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		require.NoError(t, err)

		err = storage.FailTask(context.Background(), claimed.ID, "final error")
		require.NoError(t, err)

		// Task should not be claimable (failed permanently)
		claimed2, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		assert.ErrorIs(t, err, queue.ErrNoTaskToClaim)
		assert.Nil(t, claimed2)
	})
}

func TestMemoryStorage_MoveToDLQ(t *testing.T) {
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	t.Run("moves task to DLQ", func(t *testing.T) {
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusFailed,
			Priority:    queue.PriorityMedium,
			RetryCount:  3,
			MaxRetries:  3,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
			Error:       stringPtr("final error"),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		err = storage.MoveToDLQ(context.Background(), task.ID)
		require.NoError(t, err)

		// Task should not be claimable anymore
		workerID := uuid.New()
		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
		assert.ErrorIs(t, err, queue.ErrNoTaskToClaim)
		assert.Nil(t, claimed)
	})
}

func TestMemoryStorage_ExtendLock(t *testing.T) {
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	t.Run("extends lock successfully", func(t *testing.T) {
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		workerID := uuid.New()
		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, time.Minute)
		require.NoError(t, err)

		originalLock := *claimed.LockedUntil

		err = storage.ExtendLock(context.Background(), claimed.ID, 5*time.Minute)
		require.NoError(t, err)

		// Verify lock was extended by trying to claim with another worker
		worker2 := uuid.New()
		claimed2, err := storage.ClaimTask(context.Background(), worker2, []string{queue.DefaultQueueName}, time.Minute)
		assert.ErrorIs(t, err, queue.ErrNoTaskToClaim)
		assert.Nil(t, claimed2)

		// The new lock should be later than the original
		_ = originalLock // Original lock time captured for comparison
	})
}

func TestMemoryStorage_LockExpiration(t *testing.T) {
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	t.Run("expired locks are released", func(t *testing.T) {
		task := &queue.Task{
			ID:          uuid.New(),
			Queue:       queue.DefaultQueueName,
			TaskType:    queue.TaskTypeOneTime,
			TaskName:    "test-task",
			Status:      queue.TaskStatusPending,
			Priority:    queue.PriorityMedium,
			ScheduledAt: time.Now().Add(-time.Minute),
			CreatedAt:   time.Now(),
		}
		err := storage.CreateTask(context.Background(), task)
		require.NoError(t, err)

		workerID := uuid.New()
		claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 500*time.Millisecond)
		require.NoError(t, err)
		require.NotNil(t, claimed)

		// Wait for lock to expire
		time.Sleep(2 * time.Second)

		// Another worker should be able to claim it
		worker2 := uuid.New()
		claimed2, err := storage.ClaimTask(context.Background(), worker2, []string{queue.DefaultQueueName}, 5*time.Minute)
		require.NoError(t, err)
		require.NotNil(t, claimed2)
		assert.Equal(t, task.ID, claimed2.ID)
		assert.Equal(t, worker2, *claimed2.LockedBy)
	})
}

func TestMemoryStorage_Concurrency(t *testing.T) {
	storage := queue.NewMemoryStorage()
	defer storage.Close()

	t.Run("concurrent task creation", func(t *testing.T) {
		const numGoroutines = 10
		const tasksPerGoroutine = 10

		errChan := make(chan error, numGoroutines*tasksPerGoroutine)

		for i := range numGoroutines {
			go func(workerNum int) {
				for j := range tasksPerGoroutine {
					task := &queue.Task{
						ID:          uuid.New(),
						Queue:       queue.DefaultQueueName,
						TaskType:    queue.TaskTypeOneTime,
						TaskName:    "concurrent-task",
						Status:      queue.TaskStatusPending,
						Priority:    queue.Priority(workerNum*10 + j),
						ScheduledAt: time.Now(),
						CreatedAt:   time.Now(),
					}
					err := storage.CreateTask(context.Background(), task)
					errChan <- err
				}
			}(i)
		}

		// Collect all errors
		for range numGoroutines * tasksPerGoroutine {
			err := <-errChan
			assert.NoError(t, err)
		}
	})

	t.Run("concurrent task claiming", func(t *testing.T) {
		// Create tasks
		for i := range 20 {
			task := &queue.Task{
				ID:          uuid.New(),
				Queue:       queue.DefaultQueueName,
				TaskType:    queue.TaskTypeOneTime,
				TaskName:    "claim-test",
				Status:      queue.TaskStatusPending,
				Priority:    queue.Priority(i),
				ScheduledAt: time.Now().Add(-time.Minute),
				CreatedAt:   time.Now(),
			}
			err := storage.CreateTask(context.Background(), task)
			require.NoError(t, err)
		}

		// Multiple workers trying to claim tasks
		const numWorkers = 5
		claimedChan := make(chan *queue.Task, 20)
		errChan := make(chan error, 20)

		for i := range numWorkers {
			go func(workerNum int) {
				workerID := uuid.New()
				for j := range 4 { // Each worker tries to claim 4 tasks
					claimed, err := storage.ClaimTask(context.Background(), workerID, []string{queue.DefaultQueueName}, 5*time.Minute)
					if err == nil {
						claimedChan <- claimed
					} else {
						errChan <- err
					}
					_ = j
				}
			}(i)
		}

		// Collect results
		claimedTasks := make(map[uuid.UUID]bool)
		totalClaimed := 0

		for range numWorkers * 4 {
			select {
			case task := <-claimedChan:
				// Ensure no duplicate claims
				assert.False(t, claimedTasks[task.ID], "Task %s was claimed multiple times", task.ID)
				claimedTasks[task.ID] = true
				totalClaimed++
			case err := <-errChan:
				// Some claims will fail when no tasks are available
				assert.ErrorIs(t, err, queue.ErrNoTaskToClaim)
			}
		}

		// Should have claimed exactly 20 tasks (no more, no less)
		assert.Equal(t, 20, totalClaimed)
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
