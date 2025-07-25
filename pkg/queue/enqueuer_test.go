package queue_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

// Mock repository for enqueuer tests
type mockEnqueuerRepo struct {
	createFunc func(ctx context.Context, task *queue.Task) error
	tasks      []*queue.Task
}

func (m *mockEnqueuerRepo) CreateTask(ctx context.Context, task *queue.Task) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, task)
	}
	m.tasks = append(m.tasks, task)
	return nil
}

// Test payload types
type enqueueTestPayload struct {
	Message string `json:"message"`
	Value   int    `json:"value"`
}

// Type that cannot be marshaled to JSON
type unmarshalablePayload struct {
	Ch chan int
}

func TestEnqueuer_NewEnqueuer(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		t.Parallel()

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)
		require.NotNil(t, enqueuer)
	})

	t.Run("nil repository error", func(t *testing.T) {
		t.Parallel()

		enqueuer, err := queue.NewEnqueuer(nil)
		assert.ErrorIs(t, err, queue.ErrRepositoryNil)
		assert.Nil(t, enqueuer)
	})

	t.Run("with options", func(t *testing.T) {
		t.Parallel()

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo,
			queue.WithDefaultQueue("custom-queue"),
			queue.WithDefaultPriority(queue.PriorityHigh),
		)
		require.NoError(t, err)
		require.NotNil(t, enqueuer)
	})
}

func TestEnqueuer_Enqueue(t *testing.T) {
	t.Parallel()

	t.Run("successful enqueue with defaults", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "test", Value: 42}
		err = enqueuer.Enqueue(context.Background(), payload)
		require.NoError(t, err)

		// Verify task was created
		require.Len(t, repo.tasks, 1)
		task := repo.tasks[0]
		assert.NotEqual(t, uuid.Nil, task.ID)
		assert.Equal(t, queue.DefaultQueueName, task.Queue)
		assert.Equal(t, queue.TaskTypeOneTime, task.TaskType)
		assert.Equal(t, "queue_test.enqueueTestPayload", task.TaskName)
		assert.NotEmpty(t, task.Payload)
		assert.Equal(t, queue.TaskStatusPending, task.Status)
		assert.Equal(t, queue.PriorityDefault, task.Priority)
		assert.Equal(t, int8(0), task.RetryCount)
		assert.Equal(t, int8(3), task.MaxRetries)
		assert.False(t, task.ScheduledAt.After(time.Now()))
		assert.False(t, task.CreatedAt.IsZero())
	})

	t.Run("enqueue with custom options", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "custom", Value: 100}
		scheduledTime := time.Now().Add(time.Hour)

		err = enqueuer.Enqueue(context.Background(), payload,
			queue.WithQueue("priority-queue"),
			queue.WithPriority(queue.PriorityMax),
			queue.WithMaxRetries(5),
			queue.WithTaskName("custom.task.Name"),
			queue.WithScheduledAt(scheduledTime),
		)
		require.NoError(t, err)

		// Verify custom options
		require.Len(t, repo.tasks, 1)
		task := repo.tasks[0]
		assert.Equal(t, "priority-queue", task.Queue)
		assert.Equal(t, queue.PriorityMax, task.Priority)
		assert.Equal(t, int8(5), task.MaxRetries)
		assert.Equal(t, "custom.task.Name", task.TaskName)
		assert.True(t, task.ScheduledAt.Equal(scheduledTime))
	})

	t.Run("enqueue with delay", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "delayed", Value: 1}
		beforeEnqueue := time.Now()

		err = enqueuer.Enqueue(context.Background(), payload,
			queue.WithDelay(30*time.Second),
		)
		require.NoError(t, err)

		// Verify delay
		require.Len(t, repo.tasks, 1)
		task := repo.tasks[0]
		assert.True(t, task.ScheduledAt.After(beforeEnqueue.Add(29*time.Second)))
		assert.True(t, task.ScheduledAt.Before(beforeEnqueue.Add(31*time.Second)))
	})

	t.Run("scheduledAt overrides delay", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "scheduled", Value: 1}
		scheduledTime := time.Now().Add(2 * time.Hour)

		err = enqueuer.Enqueue(context.Background(), payload,
			queue.WithDelay(30*time.Second),      // This should be ignored
			queue.WithScheduledAt(scheduledTime), // This takes precedence
		)
		require.NoError(t, err)

		// Verify scheduled time wins
		require.Len(t, repo.tasks, 1)
		task := repo.tasks[0]
		assert.True(t, task.ScheduledAt.Equal(scheduledTime))
	})

	t.Run("nil payload error", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		err = enqueuer.Enqueue(context.Background(), nil)
		assert.ErrorIs(t, err, queue.ErrPayloadNil)
		assert.Empty(t, repo.tasks)
	})

	t.Run("invalid priority error", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "invalid", Value: 1}
		err = enqueuer.Enqueue(context.Background(), payload,
			queue.WithPriority(queue.Priority(101)), // Invalid priority
		)
		assert.ErrorIs(t, err, queue.ErrInvalidPriority)
		assert.Empty(t, repo.tasks)
	})

	t.Run("marshal payload error", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		// Channel cannot be marshaled to JSON
		payload := unmarshalablePayload{Ch: make(chan int)}
		err = enqueuer.Enqueue(context.Background(), payload)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal payload")
		assert.Empty(t, repo.tasks)
	})

	t.Run("repository error", func(t *testing.T) {

		repoErr := errors.New("database connection lost")
		repo := &mockEnqueuerRepo{
			createFunc: func(ctx context.Context, task *queue.Task) error {
				return repoErr
			},
		}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "fail", Value: 1}
		err = enqueuer.Enqueue(context.Background(), payload)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create task")
		assert.Contains(t, err.Error(), "database connection lost")
	})
}

func TestEnqueuer_TaskNameGeneration(t *testing.T) {
	t.Parallel()

	t.Run("struct payload", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "test", Value: 1}
		err = enqueuer.Enqueue(context.Background(), payload)
		require.NoError(t, err)

		require.Len(t, repo.tasks, 1)
		assert.Equal(t, "queue_test.enqueueTestPayload", repo.tasks[0].TaskName)
	})

	t.Run("pointer to struct payload", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := &enqueueTestPayload{Message: "test", Value: 1}
		err = enqueuer.Enqueue(context.Background(), payload)
		require.NoError(t, err)

		require.Len(t, repo.tasks, 1)
		assert.Equal(t, "queue_test.enqueueTestPayload", repo.tasks[0].TaskName)
	})

	t.Run("map payload", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := map[string]any{"message": "test", "value": 1}
		err = enqueuer.Enqueue(context.Background(), payload)
		require.NoError(t, err)

		require.Len(t, repo.tasks, 1)
		assert.Equal(t, "map[string]interface {}", repo.tasks[0].TaskName)
	})

	t.Run("custom task name overrides", func(t *testing.T) {

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "test", Value: 1}
		err = enqueuer.Enqueue(context.Background(), payload,
			queue.WithTaskName("my.custom.TaskName"),
		)
		require.NoError(t, err)

		require.Len(t, repo.tasks, 1)
		assert.Equal(t, "my.custom.TaskName", repo.tasks[0].TaskName)
	})
}

func TestEnqueuer_DefaultConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("uses configured defaults", func(t *testing.T) {
		t.Parallel()

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo,
			queue.WithDefaultQueue("high-priority"),
			queue.WithDefaultPriority(queue.PriorityHigh),
		)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "test", Value: 1}
		err = enqueuer.Enqueue(context.Background(), payload)
		require.NoError(t, err)

		// Verify defaults were used
		require.Len(t, repo.tasks, 1)
		task := repo.tasks[0]
		assert.Equal(t, "high-priority", task.Queue)
		assert.Equal(t, queue.PriorityHigh, task.Priority)
	})

	t.Run("options override defaults", func(t *testing.T) {
		t.Parallel()

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo,
			queue.WithDefaultQueue("default-queue"),
			queue.WithDefaultPriority(queue.PriorityLow),
		)
		require.NoError(t, err)

		payload := enqueueTestPayload{Message: "test", Value: 1}
		err = enqueuer.Enqueue(context.Background(), payload,
			queue.WithQueue("override-queue"),
			queue.WithPriority(queue.PriorityMax),
		)
		require.NoError(t, err)

		// Verify options override defaults
		require.Len(t, repo.tasks, 1)
		task := repo.tasks[0]
		assert.Equal(t, "override-queue", task.Queue)
		assert.Equal(t, queue.PriorityMax, task.Priority)
	})
}

func TestEnqueuer_PayloadMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("preserves payload data", func(t *testing.T) {
		t.Parallel()

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := enqueueTestPayload{
			Message: "test message with special chars: 👍",
			Value:   -12345,
		}
		err = enqueuer.Enqueue(context.Background(), payload)
		require.NoError(t, err)

		// Verify payload was correctly marshaled
		require.Len(t, repo.tasks, 1)
		task := repo.tasks[0]

		var decoded enqueueTestPayload
		err = json.Unmarshal(task.Payload, &decoded)
		require.NoError(t, err)
		assert.Equal(t, payload.Message, decoded.Message)
		assert.Equal(t, payload.Value, decoded.Value)
	})

	t.Run("handles complex nested structures", func(t *testing.T) {
		t.Parallel()

		type complexPayload struct {
			ID     string              `json:"id"`
			Data   map[string]any      `json:"data"`
			Tags   []string            `json:"tags"`
			Nested *enqueueTestPayload `json:"nested"`
		}

		repo := &mockEnqueuerRepo{}
		enqueuer, err := queue.NewEnqueuer(repo)
		require.NoError(t, err)

		payload := complexPayload{
			ID: "test-123",
			Data: map[string]any{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			Tags: []string{"tag1", "tag2", "tag3"},
			Nested: &enqueueTestPayload{
				Message: "nested",
				Value:   99,
			},
		}

		err = enqueuer.Enqueue(context.Background(), payload)
		require.NoError(t, err)

		// Verify complex payload was marshaled correctly
		require.Len(t, repo.tasks, 1)
		task := repo.tasks[0]

		var decoded complexPayload
		err = json.Unmarshal(task.Payload, &decoded)
		require.NoError(t, err)
		assert.Equal(t, payload.ID, decoded.ID)
		assert.Equal(t, len(payload.Data), len(decoded.Data))
		assert.Equal(t, payload.Data["key1"], decoded.Data["key1"])
		// JSON unmarshals numbers as float64 by default
		assert.Equal(t, float64(payload.Data["key2"].(int)), decoded.Data["key2"])
		assert.Equal(t, payload.Data["key3"], decoded.Data["key3"])
		assert.Equal(t, payload.Tags, decoded.Tags)
		assert.Equal(t, payload.Nested.Message, decoded.Nested.Message)
		assert.Equal(t, payload.Nested.Value, decoded.Nested.Value)
	})
}
