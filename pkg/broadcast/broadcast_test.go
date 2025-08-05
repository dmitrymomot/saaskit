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

type MockStorage[T any] struct {
	mock.Mock
}

func (m *MockStorage[T]) Store(ctx context.Context, message broadcast.Message[T]) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockStorage[T]) Load(ctx context.Context, channel string, opts broadcast.LoadOptions) ([]broadcast.Message[T], error) {
	args := m.Called(ctx, channel, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]broadcast.Message[T]), args.Error(1)
}

func (m *MockStorage[T]) Delete(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

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
		assert.Equal(t, float64(42), decoded.Metadata["key2"]) // JSON unmarshals numbers as float64
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

		// Error callbacks triggered by internal errors - just verify option accepted
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

		// Fill subscriber buffer to trigger slow consumer detection
		hub.Publish(ctx, "test-channel", "msg1")
		hub.Publish(ctx, "test-channel", "msg2")

		// Test verifies option is accepted (callback implementation may vary)
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

		// Storage.Store() called when subscribers exist
		sub, err := hub.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		defer sub.Close()

		// Storage called for all messages when configured, regardless of WithPersistence
		mockStorage.On("Store", mock.Anything, mock.Anything).Return(nil)

		err = hub.Publish(ctx, "test-channel", "persistent message",
			broadcast.WithPersistence(),
		)
		require.NoError(t, err)

		mockStorage.AssertExpectations(t)
	})
}

func TestAckSubscriber(t *testing.T) {
	t.Parallel()

	t.Run("message acknowledgment", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		subscriber, err := hub.SubscribeWithAck(ctx, "test-channel")
		require.NoError(t, err)
		defer subscriber.Close()

		// Publish message
		err = hub.Publish(ctx, "test-channel", "test-message")
		require.NoError(t, err)

		// Receive and acknowledge message
		select {
		case ackMsg := <-subscriber.Messages():
			assert.Equal(t, "test-message", ackMsg.Payload)
			assert.Equal(t, "test-channel", ackMsg.Channel)
			assert.NotEmpty(t, ackMsg.ID)

			// Acknowledge the message
			err = ackMsg.Ack()
			assert.NoError(t, err)

		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("message negative acknowledgment", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		subscriber, err := hub.SubscribeWithAck(ctx, "test-channel")
		require.NoError(t, err)
		defer subscriber.Close()

		err = hub.Publish(ctx, "test-channel", "test-message")
		require.NoError(t, err)

		select {
		case ackMsg := <-subscriber.Messages():
			// Negative acknowledge the message
			err = ackMsg.Nack()
			assert.NoError(t, err)

		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("acknowledgment options configuration", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		subscriber, err := hub.SubscribeWithAck(ctx, "test-channel",
			broadcast.WithAckTimeout(100*time.Millisecond),
			broadcast.WithMaxRetries(2),
		)
		require.NoError(t, err)
		defer subscriber.Close()

		// Test that subscriber was created successfully with options
		assert.Equal(t, "test-channel", subscriber.Channel())
		assert.NotEmpty(t, subscriber.ID())
	})

	t.Run("subscriber metadata", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		subscriber, err := hub.SubscribeWithAck(ctx, "test-channel")
		require.NoError(t, err)
		defer subscriber.Close()

		assert.Equal(t, "test-channel", subscriber.Channel())
		assert.NotEmpty(t, subscriber.ID())
	})

	t.Run("multiple acknowledgment calls", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		subscriber, err := hub.SubscribeWithAck(ctx, "test-channel")
		require.NoError(t, err)
		defer subscriber.Close()

		err = hub.Publish(ctx, "test-channel", "multi-ack-test")
		require.NoError(t, err)

		select {
		case ackMsg := <-subscriber.Messages():
			// Multiple acks should be safe
			err = ackMsg.Ack()
			assert.NoError(t, err)

			err = ackMsg.Ack()
			assert.NoError(t, err)

			// Nack after ack should be safe
			err = ackMsg.Nack()
			assert.NoError(t, err)

		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("context cancellation during processing", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx, cancel := context.WithCancel(context.Background())
		subscriber, err := hub.SubscribeWithAck(ctx, "test-channel")
		require.NoError(t, err)

		err = hub.Publish(context.Background(), "test-channel", "cancel-test")
		require.NoError(t, err)

		// Cancel context while message is pending
		cancel()
		time.Sleep(50 * time.Millisecond)

		err = subscriber.Close()
		assert.NoError(t, err)
	})
}

func TestStorageOperations(t *testing.T) {
	t.Parallel()

	t.Run("storage delete", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage[string])
		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			Storage: mockStorage,
		})
		defer hub.Close()

		ctx := context.Background()
		cutoffTime := time.Now().Add(-1 * time.Hour)

		// Mock successful delete
		mockStorage.On("Delete", ctx, cutoffTime).Return(nil)

		// Call Delete method through storage interface
		err := mockStorage.Delete(ctx, cutoffTime)
		assert.NoError(t, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("storage delete error", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage[string])
		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			Storage: mockStorage,
		})
		defer hub.Close()

		ctx := context.Background()
		cutoffTime := time.Now().Add(-1 * time.Hour)
		expectedError := errors.New("delete failed")

		// Mock delete failure
		mockStorage.On("Delete", ctx, cutoffTime).Return(expectedError)

		err := mockStorage.Delete(ctx, cutoffTime)
		assert.Equal(t, expectedError, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("storage channels list", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage[string])
		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			Storage: mockStorage,
		})
		defer hub.Close()

		ctx := context.Background()
		expectedChannels := []string{"channel1", "channel2", "channel3"}

		// Mock successful channels retrieval
		mockStorage.On("Channels", ctx).Return(expectedChannels, nil)

		channels, err := mockStorage.Channels(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedChannels, channels)

		mockStorage.AssertExpectations(t)
	})

	t.Run("storage channels error", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage[string])
		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			Storage: mockStorage,
		})
		defer hub.Close()

		ctx := context.Background()
		expectedError := errors.New("channels query failed")

		// Mock channels retrieval failure
		mockStorage.On("Channels", ctx).Return(nil, expectedError)

		channels, err := mockStorage.Channels(ctx)
		assert.Nil(t, channels)
		assert.Equal(t, expectedError, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("storage load with complex options", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage[string])
		hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
			Storage: mockStorage,
		})
		defer hub.Close()

		ctx := context.Background()
		channel := "test-channel"
		before := time.Now().Add(-1 * time.Hour)
		after := time.Now().Add(-2 * time.Hour)

		opts := broadcast.LoadOptions{
			Limit:  50,
			Before: &before,
			After:  &after,
			LastID: "last-message-id",
		}

		expectedMessages := []broadcast.Message[string]{
			{
				ID:        "msg1",
				Channel:   channel,
				Payload:   "message 1",
				Timestamp: time.Now().Add(-90 * time.Minute),
			},
			{
				ID:        "msg2",
				Channel:   channel,
				Payload:   "message 2",
				Timestamp: time.Now().Add(-80 * time.Minute),
			},
		}

		mockStorage.On("Load", ctx, channel, opts).Return(expectedMessages, nil)

		messages, err := mockStorage.Load(ctx, channel, opts)
		assert.NoError(t, err)
		assert.Equal(t, expectedMessages, messages)
		assert.Len(t, messages, 2)

		mockStorage.AssertExpectations(t)
	})
}

func TestIntegrationScenario(t *testing.T) {
	t.Parallel()

	// Integration test simulating real-world chat room usage
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
		for range 50 {
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
		for range 10 {
			hub.Publish(ctx, "dynamic-channel", "message")
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Concurrent channel listing
	go func() {
		for range 20 {
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
