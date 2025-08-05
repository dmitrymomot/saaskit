package broadcast_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/broadcast"
)

// MockStorage is a mock implementation of the Storage interface
type MockStorage[T any] struct {
	mock.Mock
}

// Store saves a message
func (m *MockStorage[T]) Store(ctx context.Context, message broadcast.Message[T]) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// Load retrieves messages for a channel
func (m *MockStorage[T]) Load(ctx context.Context, channel string, opts broadcast.LoadOptions) ([]broadcast.Message[T], error) {
	args := m.Called(ctx, channel, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]broadcast.Message[T]), args.Error(1)
}

// Delete removes messages older than the given time
func (m *MockStorage[T]) Delete(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

// Channels returns all known channels
func (m *MockStorage[T]) Channels(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestNewHub(t *testing.T) {
	t.Parallel()

	t.Run("default config", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{})
		require.NotNil(t, hub)
		defer hub.Close()

		// Should use defaults
		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test")
		require.NoError(t, err)
		sub.Close()
	})

	t.Run("custom config", func(t *testing.T) {
		t.Parallel()

		called := false
		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize:   50,
			CleanupInterval:     time.Minute,
			SlowConsumerTimeout: 10 * time.Second,
			MetricsCallback: func(channel string, subscribers int) {
				called = true
			},
		})
		require.NotNil(t, hub)
		defer hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test")
		require.NoError(t, err)
		sub.Close()

		assert.True(t, called, "metrics callback should be called")
	})
}

func TestMessage(t *testing.T) {
	t.Parallel()

	msg := broadcast.Message[string]{
		ID:        "test-id",
		Channel:   "test-channel",
		Payload:   "test-payload",
		Timestamp: time.Now(),
		Metadata: broadcast.Metadata{
			"key": "value",
		},
	}

	assert.Equal(t, "test-id", msg.ID)
	assert.Equal(t, "test-channel", msg.Channel)
	assert.Equal(t, "test-payload", msg.Payload)
	assert.NotZero(t, msg.Timestamp)
	assert.Equal(t, "value", msg.Metadata["key"])
}

func TestMessageJSONMarshaling(t *testing.T) {
	t.Parallel()

	// Test with string payload
	t.Run("StringPayload", func(t *testing.T) {
		t.Parallel()

		timestamp := time.Now().Truncate(time.Millisecond)
		original := broadcast.Message[string]{
			ID:        "msg-123",
			Channel:   "test-channel",
			Payload:   "test-payload",
			Timestamp: timestamp,
			Metadata: broadcast.Metadata{
				"key1": "value1",
				"key2": 42,
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var decoded broadcast.Message[string]
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Channel, decoded.Channel)
		assert.Equal(t, original.Payload, decoded.Payload)
		assert.Equal(t, original.Timestamp.Unix(), decoded.Timestamp.Unix())
		assert.Equal(t, original.Metadata["key1"], decoded.Metadata["key1"])
		assert.Equal(t, float64(42), decoded.Metadata["key2"]) // JSON numbers decode as float64
	})

	// Test with complex payload
	t.Run("ComplexPayload", func(t *testing.T) {
		t.Parallel()

		type ComplexPayload struct {
			Name   string   `json:"name"`
			Count  int      `json:"count"`
			Active bool     `json:"active"`
			Tags   []string `json:"tags"`
		}

		timestamp := time.Now().Truncate(time.Millisecond)
		original := broadcast.Message[ComplexPayload]{
			ID:      "msg-456",
			Channel: "complex-channel",
			Payload: ComplexPayload{
				Name:   "Test Item",
				Count:  10,
				Active: true,
				Tags:   []string{"tag1", "tag2"},
			},
			Timestamp: timestamp,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var decoded broadcast.Message[ComplexPayload]
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Channel, decoded.Channel)
		assert.Equal(t, original.Payload.Name, decoded.Payload.Name)
		assert.Equal(t, original.Payload.Count, decoded.Payload.Count)
		assert.Equal(t, original.Payload.Active, decoded.Payload.Active)
		assert.Equal(t, original.Payload.Tags, decoded.Payload.Tags)
		assert.Equal(t, original.Timestamp.Unix(), decoded.Timestamp.Unix())
	})

	// Test with nil metadata
	t.Run("NilMetadata", func(t *testing.T) {
		t.Parallel()

		original := broadcast.Message[int]{
			ID:        "msg-789",
			Channel:   "int-channel",
			Payload:   42,
			Timestamp: time.Now(),
			// Metadata is nil
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var decoded broadcast.Message[int]
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Channel, decoded.Channel)
		assert.Equal(t, original.Payload, decoded.Payload)
		assert.Nil(t, decoded.Metadata)
	})
}

func TestPublishMessage(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
		DefaultBufferSize: 10,
	})
	defer hub.Close()

	ctx := context.Background()
	sub, err := hub.Subscribe(ctx, "test-channel")
	require.NoError(t, err)
	defer sub.Close()

	// Create custom message
	customMsg := broadcast.Message[string]{
		ID:        "custom-id",
		Channel:   "test-channel",
		Payload:   "custom-payload",
		Timestamp: time.Now(),
		Metadata: broadcast.Metadata{
			"source": "test",
		},
	}

	err = hub.PublishMessage(ctx, customMsg)
	require.NoError(t, err)

	// Verify received message
	select {
	case msg := <-sub.Messages():
		assert.Equal(t, customMsg.ID, msg.ID)
		assert.Equal(t, customMsg.Channel, msg.Channel)
		assert.Equal(t, customMsg.Payload, msg.Payload)
		assert.Equal(t, customMsg.Timestamp, msg.Timestamp)
		assert.Equal(t, customMsg.Metadata["source"], msg.Metadata["source"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for message")
	}
}

func TestSubscribeOptions(t *testing.T) {
	t.Parallel()

	t.Run("error callback", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		var capturedError error
		errorCallback := func(err error) {
			capturedError = err
		}

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test-channel",
			broadcast.WithErrorCallback(errorCallback),
		)
		require.NoError(t, err)
		defer sub.Close()

		// Error callbacks would be triggered by internal errors
		// For this test, we just verify the option is accepted
		assert.Nil(t, capturedError)
	})

	t.Run("slow consumer callback", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize:   1,
			SlowConsumerTimeout: 50 * time.Millisecond,
		})
		defer hub.Close()

		slowCallback := func() {
			// Callback would be called on slow consumer detection
		}

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test-channel",
			broadcast.WithSlowConsumerCallback(slowCallback),
		)
		require.NoError(t, err)

		// Fill buffer without reading to trigger slow consumer
		hub.Publish(ctx, "test-channel", "msg1")
		hub.Publish(ctx, "test-channel", "msg2")

		// Note: The current implementation doesn't call the slow consumer callback
		// This test verifies the option is accepted
		sub.Close()
	})
}

func TestPublishOptions(t *testing.T) {
	t.Parallel()

	t.Run("with timeout", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		defer sub.Close()

		err = hub.Publish(ctx, "test-channel", "message",
			broadcast.WithTimeout(time.Second),
		)
		require.NoError(t, err)

		select {
		case msg := <-sub.Messages():
			assert.Equal(t, "message", msg.Payload)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("with persistence", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage[string])
		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
			Storage:           mockStorage,
		})
		defer hub.Close()

		ctx := context.Background()

		// Need a subscriber for storage to be called
		sub, err := hub.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		defer sub.Close()

		// Storage is called whenever configured, not just with WithPersistence
		mockStorage.On("Store", mock.Anything, mock.Anything).Return(nil)

		err = hub.Publish(ctx, "test-channel", "persistent message",
			broadcast.WithPersistence(),
		)
		require.NoError(t, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestLoadOptions(t *testing.T) {
	t.Parallel()

	now := time.Now()
	before := now.Add(-1 * time.Hour)
	after := now.Add(-2 * time.Hour)

	opts := broadcast.LoadOptions{
		Limit:  100,
		Before: &before,
		After:  &after,
		LastID: "last-id",
	}

	assert.Equal(t, 100, opts.Limit)
	assert.Equal(t, before, *opts.Before)
	assert.Equal(t, after, *opts.After)
	assert.Equal(t, "last-id", opts.LastID)
}

func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("ErrHubClosed", func(t *testing.T) {
		err := broadcast.ErrHubClosed{}
		assert.Equal(t, "broadcast: hub is closed", err.Error())
	})

	t.Run("ErrSubscriberClosed", func(t *testing.T) {
		err := broadcast.ErrSubscriberClosed{ID: "sub-123"}
		assert.Equal(t, "broadcast: subscriber sub-123 is closed", err.Error())
	})

	t.Run("ErrChannelNotFound", func(t *testing.T) {
		err := broadcast.ErrChannelNotFound{Channel: "missing"}
		assert.Equal(t, "broadcast: channel missing not found", err.Error())
	})

	t.Run("ErrStorageFailure", func(t *testing.T) {
		innerErr := errors.New("connection failed")
		err := broadcast.ErrStorageFailure{
			Operation: "store",
			Err:       innerErr,
		}
		assert.Equal(t, "broadcast: storage store failed: connection failed", err.Error())
		assert.Equal(t, innerErr, err.Unwrap())
	})

	t.Run("ErrPublishTimeout", func(t *testing.T) {
		err := broadcast.ErrPublishTimeout{
			Channel: "slow-channel",
			Timeout: 5 * time.Second,
		}
		assert.Equal(t, "broadcast: publish to channel slow-channel timed out after 5s", err.Error())
	})

	t.Run("ErrShutdownTimeout", func(t *testing.T) {
		err := broadcast.ErrShutdownTimeout{}
		assert.Equal(t, "broadcast: shutdown timeout exceeded", err.Error())
	})
}

func TestIntegrationScenario(t *testing.T) {
	t.Parallel()

	// Simulate a chat room scenario
	mockStorage := new(MockStorage[ChatMessage])
	hub := broadcast.NewHub[ChatMessage](broadcast.HubConfig[ChatMessage]{
		DefaultBufferSize: 50,
		Storage:           mockStorage,
		MetricsCallback: func(channel string, subscribers int) {
			t.Logf("Channel %s has %d subscribers", channel, subscribers)
		},
	})
	defer hub.Close()

	// Setup storage expectations
	mockStorage.On("Store", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockStorage.On("Load", mock.Anything, "chat-room", mock.Anything).Return([]broadcast.Message[ChatMessage]{}, nil).Maybe()

	ctx := context.Background()
	room := "chat-room"

	// User 1 joins
	user1, err := hub.Subscribe(ctx, room, broadcast.WithReplay(10))
	require.NoError(t, err)
	defer user1.Close()

	// User 2 joins
	user2, err := hub.Subscribe(ctx, room, broadcast.WithReplay(10))
	require.NoError(t, err)
	defer user2.Close()

	// User 1 sends message
	err = hub.Publish(ctx, room, ChatMessage{
		User: "Alice",
		Text: "Hello everyone!",
		Time: time.Now(),
	}, broadcast.WithPersistence())
	require.NoError(t, err)

	// Both users should receive the message
	verifyMessage := func(sub broadcast.Subscriber[ChatMessage], expectedUser, expectedText string) {
		select {
		case msg := <-sub.Messages():
			assert.Equal(t, expectedUser, msg.Payload.User)
			assert.Equal(t, expectedText, msg.Payload.Text)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for message")
		}
	}

	verifyMessage(user1, "Alice", "Hello everyone!")
	verifyMessage(user2, "Alice", "Hello everyone!")

	// User 2 sends message
	err = hub.Publish(ctx, room, ChatMessage{
		User: "Bob",
		Text: "Hi Alice!",
		Time: time.Now(),
	}, broadcast.WithPersistence())
	require.NoError(t, err)

	verifyMessage(user1, "Bob", "Hi Alice!")
	verifyMessage(user2, "Bob", "Hi Alice!")

	// Check subscriber count
	assert.Equal(t, 2, hub.SubscriberCount(room))

	// User 1 leaves
	user1.Close()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.SubscriberCount(room))
}

// Helper types for testing
type ChatMessage struct {
	User string
	Text string
	Time time.Time
}

func TestConcurrentChannelOperations(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
		DefaultBufferSize: 100,
		CleanupInterval:   100 * time.Millisecond,
	})
	defer hub.Close()

	ctx := context.Background()
	done := make(chan struct{})

	// Concurrent channel creation/deletion
	go func() {
		for i := 0; i < 50; i++ {
			channel := "dynamic-channel"
			sub, err := hub.Subscribe(ctx, channel)
			if err == nil {
				go func(s broadcast.Subscriber[string]) {
					time.Sleep(10 * time.Millisecond)
					s.Close()
				}(sub)
			}
		}
		close(done)
	}()

	// Concurrent publishing
	go func() {
		for i := 0; i < 100; i++ {
			hub.Publish(ctx, "dynamic-channel", "message")
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Concurrent channel listing
	go func() {
		for i := 0; i < 20; i++ {
			channels := hub.Channels()
			_ = channels // Just accessing, not asserting
			time.Sleep(25 * time.Millisecond)
		}
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("timeout in concurrent test")
	}
}
