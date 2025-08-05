package broadcast_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/broadcast"
)

func TestHub_Subscribe(t *testing.T) {
	t.Parallel()

	t.Run("successful subscription", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		require.NotNil(t, sub)
		defer sub.Close()

		assert.Equal(t, "test-channel", sub.Channel())
		assert.NotEmpty(t, sub.ID())
		assert.NotNil(t, sub.Messages())
	})

	t.Run("subscription with options", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test-channel",
			broadcast.WithBufferSize(50),
		)
		require.NoError(t, err)
		require.NotNil(t, sub)
		defer sub.Close()

		// Buffer size is internal, so we test it indirectly
		// by publishing many messages without reading
		for i := 0; i < 50; i++ {
			err := hub.Publish(ctx, "test-channel", "msg")
			require.NoError(t, err)
		}
	})

	t.Run("subscription with context cancellation", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx, cancel := context.WithCancel(context.Background())
		sub, err := hub.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		require.NotNil(t, sub)

		// Cancel context should close subscriber
		cancel()
		time.Sleep(50 * time.Millisecond) // Allow cleanup

		// Messages channel should be closed
		select {
		case _, ok := <-sub.Messages():
			assert.False(t, ok, "messages channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for channel close")
		}
	})

	t.Run("subscribe to closed hub", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{})
		hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test-channel")
		assert.Error(t, err)
		assert.Nil(t, sub)
		assert.IsType(t, broadcast.ErrHubClosed{}, err)
	})

	t.Run("multiple subscribers same channel", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		sub1, err := hub.Subscribe(ctx, "shared-channel")
		require.NoError(t, err)
		defer sub1.Close()

		sub2, err := hub.Subscribe(ctx, "shared-channel")
		require.NoError(t, err)
		defer sub2.Close()

		assert.Equal(t, 2, hub.SubscriberCount("shared-channel"))
		assert.NotEqual(t, sub1.ID(), sub2.ID())
	})

	t.Run("metrics callback", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex
		metrics := make(map[string]int)

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
			MetricsCallback: func(channel string, subscribers int) {
				mu.Lock()
				metrics[channel] = subscribers
				mu.Unlock()
			},
		})
		defer hub.Close()

		ctx := context.Background()
		sub1, err := hub.Subscribe(ctx, "metrics-channel")
		require.NoError(t, err)

		mu.Lock()
		assert.Equal(t, 1, metrics["metrics-channel"])
		mu.Unlock()

		sub2, err := hub.Subscribe(ctx, "metrics-channel")
		require.NoError(t, err)

		mu.Lock()
		assert.Equal(t, 2, metrics["metrics-channel"])
		mu.Unlock()

		sub1.Close()
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, 1, metrics["metrics-channel"])
		mu.Unlock()

		sub2.Close()
	})
}

func TestHub_Publish(t *testing.T) {
	t.Parallel()

	t.Run("publish to single subscriber", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		defer sub.Close()

		err = hub.Publish(ctx, "test-channel", "hello world")
		require.NoError(t, err)

		select {
		case msg := <-sub.Messages():
			assert.Equal(t, "hello world", msg.Payload)
			assert.Equal(t, "test-channel", msg.Channel)
			assert.NotEmpty(t, msg.ID)
			assert.False(t, msg.Timestamp.IsZero())
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("publish to multiple subscribers", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		var subs []broadcast.Subscriber[string]

		// Create 3 subscribers
		for i := 0; i < 3; i++ {
			sub, err := hub.Subscribe(ctx, "multi-channel")
			require.NoError(t, err)
			subs = append(subs, sub)
		}
		defer func() {
			for _, sub := range subs {
				sub.Close()
			}
		}()

		err := hub.Publish(ctx, "multi-channel", "broadcast message")
		require.NoError(t, err)

		// All subscribers should receive the message
		for i, sub := range subs {
			select {
			case msg := <-sub.Messages():
				assert.Equal(t, "broadcast message", msg.Payload, "subscriber %d", i)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("timeout waiting for message on subscriber %d", i)
			}
		}
	})

	t.Run("publish with metadata", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "metadata-channel")
		require.NoError(t, err)
		defer sub.Close()

		metadata := broadcast.Metadata{
			"user_id": "123",
			"action":  "test",
		}

		err = hub.Publish(ctx, "metadata-channel", "with metadata",
			broadcast.WithMetadata(metadata),
		)
		require.NoError(t, err)

		select {
		case msg := <-sub.Messages():
			assert.Equal(t, "with metadata", msg.Payload)
			assert.Equal(t, "123", msg.Metadata["user_id"])
			assert.Equal(t, "test", msg.Metadata["action"])
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("publish to non-existent channel", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})
		defer hub.Close()

		ctx := context.Background()
		err := hub.Publish(ctx, "no-subscribers", "message")
		assert.NoError(t, err) // Should not error, just no-op
	})

	t.Run("publish to closed hub", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{})
		hub.Close()

		ctx := context.Background()
		err := hub.Publish(ctx, "test-channel", "message")
		assert.Error(t, err)
		assert.IsType(t, broadcast.ErrHubClosed{}, err)
	})

	t.Run("publish with context cancellation", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 1,
		})
		defer hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		defer sub.Close()

		// Cancel context immediately
		pubCtx, cancel := context.WithCancel(ctx)
		cancel()

		err = hub.Publish(pubCtx, "test-channel", "message")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestHub_SlowConsumer(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[string](broadcast.HubConfig{
		DefaultBufferSize:   1,
		SlowConsumerTimeout: 50 * time.Millisecond,
	})
	defer hub.Close()

	ctx := context.Background()
	sub, err := hub.Subscribe(ctx, "slow-channel", broadcast.WithBufferSize(1))
	require.NoError(t, err)
	defer sub.Close()

	// Fill the buffer
	err = hub.Publish(ctx, "slow-channel", "message 1")
	require.NoError(t, err)

	// This should timeout but not error (slow consumer is handled gracefully)
	err = hub.Publish(ctx, "slow-channel", "message 2")
	require.NoError(t, err)

	// Third message also times out
	err = hub.Publish(ctx, "slow-channel", "message 3")
	require.NoError(t, err)

	// We can still read the first message
	select {
	case msg := <-sub.Messages():
		assert.Equal(t, "message 1", msg.Payload)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("should have received first message")
	}

	// Subsequent messages may have been dropped due to slow consumer
	// This tests that slow consumer timeout doesn't crash the system
}

func TestHub_Channels(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[string](broadcast.HubConfig{
		DefaultBufferSize: 10,
	})
	defer hub.Close()

	ctx := context.Background()

	// Initially no channels
	assert.Empty(t, hub.Channels())

	// Subscribe to create channels
	sub1, err := hub.Subscribe(ctx, "channel-1")
	require.NoError(t, err)
	defer sub1.Close()

	sub2, err := hub.Subscribe(ctx, "channel-2")
	require.NoError(t, err)
	defer sub2.Close()

	channels := hub.Channels()
	assert.Len(t, channels, 2)
	assert.Contains(t, channels, "channel-1")
	assert.Contains(t, channels, "channel-2")
}

func TestHub_SubscriberCount(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[string](broadcast.HubConfig{
		DefaultBufferSize: 10,
	})
	defer hub.Close()

	ctx := context.Background()

	// No subscribers initially
	assert.Equal(t, 0, hub.SubscriberCount("test-channel"))

	// Add subscribers
	var subs []broadcast.Subscriber[string]
	for i := 0; i < 3; i++ {
		sub, err := hub.Subscribe(ctx, "test-channel")
		require.NoError(t, err)
		subs = append(subs, sub)
	}

	assert.Equal(t, 3, hub.SubscriberCount("test-channel"))

	// Close one subscriber
	subs[0].Close()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, hub.SubscriberCount("test-channel"))

	// Close remaining
	for i := 1; i < len(subs); i++ {
		subs[i].Close()
	}
}

func TestHub_Storage(t *testing.T) {
	t.Parallel()

	t.Run("store messages", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage)
		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
			Storage:           mockStorage,
		})
		defer hub.Close()

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "storage-channel")
		require.NoError(t, err)
		defer sub.Close()

		// Expect storage call
		mockStorage.On("Store", mock.Anything, mock.MatchedBy(func(msg broadcast.Message[any]) bool {
			return msg.Channel == "storage-channel" && msg.Payload == "stored message"
		})).Return(nil)

		err = hub.Publish(ctx, "storage-channel", "stored message",
			broadcast.WithPersistence(),
		)
		require.NoError(t, err)

		mockStorage.AssertExpectations(t)
	})

	t.Run("storage error", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage)
		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
			Storage:           mockStorage,
		})
		defer hub.Close()

		ctx := context.Background()

		// Need a subscriber for the channel to exist
		sub, err := hub.Subscribe(ctx, "error-channel")
		require.NoError(t, err)
		defer sub.Close()

		storageErr := errors.New("storage failed")
		mockStorage.On("Store", mock.Anything, mock.Anything).Return(storageErr)

		err = hub.Publish(ctx, "error-channel", "message")
		require.Error(t, err)
		assert.IsType(t, &broadcast.ErrStorageFailure{}, err)
		assert.Contains(t, err.Error(), "storage failed")
	})

	t.Run("replay messages", func(t *testing.T) {
		t.Parallel()

		mockStorage := new(MockStorage)
		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
			Storage:           mockStorage,
		})
		defer hub.Close()

		// Setup replay messages
		replayMessages := []broadcast.Message[any]{
			{
				ID:        "msg1",
				Channel:   "replay-channel",
				Payload:   "replay 1",
				Timestamp: time.Now().Add(-1 * time.Minute),
			},
			{
				ID:        "msg2",
				Channel:   "replay-channel",
				Payload:   "replay 2",
				Timestamp: time.Now().Add(-30 * time.Second),
			},
		}

		mockStorage.On("Load", mock.Anything, "replay-channel", mock.MatchedBy(func(opts broadcast.LoadOptions) bool {
			return opts.Limit == 5
		})).Return(replayMessages, nil)

		ctx := context.Background()
		sub, err := hub.Subscribe(ctx, "replay-channel",
			broadcast.WithReplay(5),
		)
		require.NoError(t, err)
		defer sub.Close()

		// Should receive replay messages
		for i := 0; i < 2; i++ {
			select {
			case msg := <-sub.Messages():
				assert.Equal(t, replayMessages[i].ID, msg.ID)
				assert.Equal(t, replayMessages[i].Payload.(string), msg.Payload)
			case <-time.After(200 * time.Millisecond):
				t.Fatal("timeout waiting for replay message")
			}
		}

		mockStorage.AssertExpectations(t)
	})
}

func TestHub_Close(t *testing.T) {
	t.Parallel()

	t.Run("graceful shutdown", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
		})

		ctx := context.Background()
		sub1, err := hub.Subscribe(ctx, "channel-1")
		require.NoError(t, err)

		sub2, err := hub.Subscribe(ctx, "channel-2")
		require.NoError(t, err)

		// Close hub
		err = hub.Close()
		require.NoError(t, err)

		// Subscribers should be closed
		select {
		case _, ok := <-sub1.Messages():
			assert.False(t, ok)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for sub1 close")
		}

		select {
		case _, ok := <-sub2.Messages():
			assert.False(t, ok)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for sub2 close")
		}

		// Double close should not error
		err = hub.Close()
		assert.NoError(t, err)
	})

	t.Run("close with cleanup interval", func(t *testing.T) {
		t.Parallel()

		hub := broadcast.NewHub[string](broadcast.HubConfig{
			DefaultBufferSize: 10,
			CleanupInterval:   50 * time.Millisecond,
		})

		err := hub.Close()
		require.NoError(t, err)
	})
}

func TestHub_Concurrent(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[int](broadcast.HubConfig{
		DefaultBufferSize: 100,
	})
	defer hub.Close()

	ctx := context.Background()
	const numPublishers = 10
	const numSubscribers = 5
	const messagesPerPublisher = 100

	var wg sync.WaitGroup
	received := make(map[int]int) // subscriber -> count
	var mu sync.Mutex

	// Start subscribers
	for i := 0; i < numSubscribers; i++ {
		subID := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			sub, err := hub.Subscribe(ctx, "concurrent-channel")
			require.NoError(t, err)
			defer sub.Close()

			count := 0
			for range sub.Messages() {
				count++
			}

			mu.Lock()
			received[subID] = count
			mu.Unlock()
		}()
	}

	// Give subscribers time to start
	time.Sleep(50 * time.Millisecond)

	// Start publishers
	for i := 0; i < numPublishers; i++ {
		pubID := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < messagesPerPublisher; j++ {
				err := hub.Publish(ctx, "concurrent-channel", pubID*1000+j)
				require.NoError(t, err)
			}
		}()
	}

	// Wait for all publishers to finish
	time.Sleep(200 * time.Millisecond)

	// Close hub to signal subscribers
	hub.Close()

	// Wait for all goroutines
	wg.Wait()

	// Each subscriber should receive all messages
	expectedTotal := numPublishers * messagesPerPublisher
	mu.Lock()
	for subID, count := range received {
		assert.Equal(t, expectedTotal, count, "subscriber %d received %d messages, expected %d", subID, count, expectedTotal)
	}
	mu.Unlock()
}

func TestSubscriber_Close(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[string](broadcast.HubConfig{
		DefaultBufferSize: 10,
	})
	defer hub.Close()

	ctx := context.Background()
	sub, err := hub.Subscribe(ctx, "test-channel")
	require.NoError(t, err)

	// Close should be idempotent
	err = sub.Close()
	assert.NoError(t, err)

	err = sub.Close()
	assert.NoError(t, err)

	// Messages channel should be closed
	select {
	case _, ok := <-sub.Messages():
		assert.False(t, ok)
	default:
		t.Fatal("messages channel should be closed")
	}
}
