package queue_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

// Test payload types
type handlerTestPayload struct {
	Message string `json:"message"`
	Value   int    `json:"value"`
}

func TestNewTaskHandler(t *testing.T) {
	t.Parallel()

	t.Run("creates handler with correct name", func(t *testing.T) {
		t.Parallel()

		handler := queue.NewTaskHandler(func(ctx context.Context, payload handlerTestPayload) error {
			return nil
		})

		assert.Equal(t, "queue_test.handlerTestPayload", handler.Name())
	})

	t.Run("handler processes payload correctly", func(t *testing.T) {
		t.Parallel()

		received := make(chan handlerTestPayload, 1)
		handler := queue.NewTaskHandler(func(ctx context.Context, payload handlerTestPayload) error {
			received <- payload
			return nil
		})

		// Create JSON payload
		expected := handlerTestPayload{Message: "test", Value: 42}
		jsonPayload, err := json.Marshal(expected)
		require.NoError(t, err)

		// Handle the payload
		err = handler.Handle(context.Background(), jsonPayload)
		require.NoError(t, err)

		// Verify payload was received correctly
		select {
		case actual := <-received:
			assert.Equal(t, expected.Message, actual.Message)
			assert.Equal(t, expected.Value, actual.Value)
		default:
			t.Fatal("handler did not receive payload")
		}
	})

	t.Run("handler returns error from function", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("processing failed")
		handler := queue.NewTaskHandler(func(ctx context.Context, payload handlerTestPayload) error {
			return expectedErr
		})

		jsonPayload, _ := json.Marshal(handlerTestPayload{})
		err := handler.Handle(context.Background(), jsonPayload)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("handler returns unmarshal error", func(t *testing.T) {
		t.Parallel()

		handler := queue.NewTaskHandler(func(ctx context.Context, payload handlerTestPayload) error {
			return nil
		})

		// Invalid JSON
		err := handler.Handle(context.Background(), []byte("invalid json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
	})

	t.Run("handler with pointer type", func(t *testing.T) {
		t.Parallel()

		handler := queue.NewTaskHandler(func(ctx context.Context, payload *handlerTestPayload) error {
			// Verify we got a non-nil pointer
			if payload == nil {
				return errors.New("payload is nil")
			}
			if payload.Message != "pointer test" {
				return errors.New("wrong message")
			}
			return nil
		})

		assert.Equal(t, "queue_test.handlerTestPayload", handler.Name())

		payload := handlerTestPayload{Message: "pointer test", Value: 1}
		jsonPayload, _ := json.Marshal(payload)
		err := handler.Handle(context.Background(), jsonPayload)
		assert.NoError(t, err)
	})
}

func TestNewPeriodicTaskHandler(t *testing.T) {
	t.Parallel()

	t.Run("creates handler with specified name", func(t *testing.T) {
		handler := queue.NewPeriodicTaskHandler("daily-cleanup", func(ctx context.Context) error {
			return nil
		})

		assert.Equal(t, "daily-cleanup", handler.Name())
	})

	t.Run("handler executes function", func(t *testing.T) {
		executed := false
		handler := queue.NewPeriodicTaskHandler("test-periodic", func(ctx context.Context) error {
			executed = true
			return nil
		})

		err := handler.Handle(context.Background(), nil)
		require.NoError(t, err)
		assert.True(t, executed, "handler should have executed")
	})

	t.Run("handler ignores payload", func(t *testing.T) {
		handler := queue.NewPeriodicTaskHandler("ignore-payload", func(ctx context.Context) error {
			return nil
		})

		// Should work with any payload (or nil)
		err := handler.Handle(context.Background(), []byte(`{"ignored": true}`))
		assert.NoError(t, err)

		err = handler.Handle(context.Background(), nil)
		assert.NoError(t, err)
	})

	t.Run("handler returns error from function", func(t *testing.T) {
		expectedErr := errors.New("periodic task failed")
		handler := queue.NewPeriodicTaskHandler("error-task", func(ctx context.Context) error {
			return expectedErr
		})

		err := handler.Handle(context.Background(), nil)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("handler respects context", func(t *testing.T) {
		handler := queue.NewPeriodicTaskHandler("context-aware", func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		})

		// Test with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := handler.Handle(ctx, nil)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestHandler_ComplexPayloads(t *testing.T) {
	t.Parallel()

	t.Run("nested struct payload", func(t *testing.T) {
		t.Parallel()

		type Address struct {
			Street  string `json:"street"`
			City    string `json:"city"`
			ZipCode string `json:"zip_code"`
		}

		type User struct {
			ID       int            `json:"id"`
			Name     string         `json:"name"`
			Email    string         `json:"email"`
			Address  Address        `json:"address"`
			Tags     []string       `json:"tags"`
			Metadata map[string]any `json:"metadata"`
		}

		handler := queue.NewTaskHandler(func(ctx context.Context, user User) error {
			// Validate received data
			if user.ID != 123 {
				return errors.New("wrong ID")
			}
			if user.Address.City != "San Francisco" {
				return errors.New("wrong city")
			}
			if len(user.Tags) != 3 {
				return errors.New("wrong tags count")
			}
			return nil
		})

		user := User{
			ID:    123,
			Name:  "John Doe",
			Email: "john@example.com",
			Address: Address{
				Street:  "123 Main St",
				City:    "San Francisco",
				ZipCode: "94105",
			},
			Tags: []string{"premium", "verified", "active"},
			Metadata: map[string]any{
				"signup_date": "2024-01-01",
				"referral_id": 456,
			},
		}

		jsonPayload, err := json.Marshal(user)
		require.NoError(t, err)

		err = handler.Handle(context.Background(), jsonPayload)
		assert.NoError(t, err)
	})

	t.Run("slice payload", func(t *testing.T) {
		t.Parallel()

		type BatchRequest struct {
			Items []string `json:"items"`
		}

		handler := queue.NewTaskHandler(func(ctx context.Context, batch BatchRequest) error {
			if len(batch.Items) != 3 {
				return errors.New("wrong item count")
			}
			return nil
		})

		batch := BatchRequest{
			Items: []string{"item1", "item2", "item3"},
		}

		jsonPayload, _ := json.Marshal(batch)
		err := handler.Handle(context.Background(), jsonPayload)
		assert.NoError(t, err)
	})
}

func TestHandler_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty payload struct", func(t *testing.T) {
		t.Parallel()

		type EmptyPayload struct{}

		handler := queue.NewTaskHandler(func(ctx context.Context, payload EmptyPayload) error {
			return nil
		})

		err := handler.Handle(context.Background(), []byte("{}"))
		assert.NoError(t, err)
	})

	t.Run("payload with json tags", func(t *testing.T) {
		t.Parallel()

		type TaggedPayload struct {
			CamelCase   string `json:"camelCase"`
			SnakeCase   string `json:"snake_case"`
			OmitEmpty   string `json:"omit_empty,omitempty"`
			IgnoreField string `json:"-"`
		}

		handler := queue.NewTaskHandler(func(ctx context.Context, payload TaggedPayload) error {
			if payload.CamelCase != "test1" {
				return errors.New("camelCase wrong")
			}
			if payload.SnakeCase != "test2" {
				return errors.New("snake_case wrong")
			}
			if payload.IgnoreField != "" {
				return errors.New("ignored field should be empty")
			}
			return nil
		})

		jsonStr := `{"camelCase":"test1","snake_case":"test2","omit_empty":"","ignored":"should not appear"}`
		err := handler.Handle(context.Background(), []byte(jsonStr))
		assert.NoError(t, err)
	})

	t.Run("payload with time fields", func(t *testing.T) {
		t.Parallel()

		type TimePayload struct {
			CreatedAt time.Time  `json:"created_at"`
			UpdatedAt *time.Time `json:"updated_at,omitempty"`
		}

		handler := queue.NewTaskHandler(func(ctx context.Context, payload TimePayload) error {
			if payload.CreatedAt.IsZero() {
				return errors.New("created_at is zero")
			}
			if payload.UpdatedAt != nil && payload.UpdatedAt.IsZero() {
				return errors.New("updated_at is zero")
			}
			return nil
		})

		now := time.Now()
		payload := TimePayload{
			CreatedAt: now,
			UpdatedAt: &now,
		}

		jsonPayload, err := json.Marshal(payload)
		require.NoError(t, err)

		err = handler.Handle(context.Background(), jsonPayload)
		assert.NoError(t, err)
	})
}
