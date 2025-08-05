//go:build load

package broadcast_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/broadcast"
)

func TestHub_SlowConsumer_Load(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[string](broadcast.HubConfig[string]{
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

func TestHub_Concurrent_Load(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[int](broadcast.HubConfig[int]{
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
				if count >= numPublishers*messagesPerPublisher/numSubscribers {
					break
				}
			}

			mu.Lock()
			received[subID] = count
			mu.Unlock()
		}()
	}

	// Start publishers
	for i := 0; i < numPublishers; i++ {
		publisherID := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < messagesPerPublisher; j++ {
				message := publisherID*messagesPerPublisher + j
				err := hub.Publish(ctx, "concurrent-channel", message)
				require.NoError(t, err)
			}
		}()
	}

	// Wait for completion with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("test timed out")
	}

	// Verify each subscriber received some messages
	mu.Lock()
	totalReceived := 0
	for subID, count := range received {
		assert.Greater(t, count, 0, "subscriber %d should have received messages", subID)
		totalReceived += count
	}
	mu.Unlock()

	// Total messages sent should equal total received (accounting for fan-out)
	expectedTotal := numPublishers * messagesPerPublisher * numSubscribers
	assert.Equal(t, expectedTotal, totalReceived, "message count mismatch")
}

func TestHub_HighThroughput_Load(t *testing.T) {
	t.Parallel()

	hub := broadcast.NewHub[int](broadcast.HubConfig[int]{
		DefaultBufferSize: 1000,
	})
	defer hub.Close()

	ctx := context.Background()
	const numMessages = 10000
	const numChannels = 5

	var wg sync.WaitGroup
	totalReceived := int64(0)
	var receivedMu sync.Mutex

	// Create subscribers for each channel
	for ch := 0; ch < numChannels; ch++ {
		channelName := "channel-" + string(rune('A'+ch))
		wg.Add(1)
		go func(chName string) {
			defer wg.Done()

			sub, err := hub.Subscribe(ctx, chName)
			require.NoError(t, err)
			defer sub.Close()

			count := 0
			for range sub.Messages() {
				count++
				if count >= numMessages {
					break
				}
			}

			receivedMu.Lock()
			totalReceived += int64(count)
			receivedMu.Unlock()
		}(channelName)
	}

	// Publisher goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < numMessages; i++ {
			for ch := 0; ch < numChannels; ch++ {
				channelName := "channel-" + string(rune('A'+ch))
				err := hub.Publish(ctx, channelName, i)
				require.NoError(t, err)
			}
		}
	}()

	// Wait for completion with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(60 * time.Second):
		t.Fatal("high throughput test timed out")
	}

	receivedMu.Lock()
	expectedTotal := int64(numMessages * numChannels)
	assert.Equal(t, expectedTotal, totalReceived, "high throughput message count mismatch")
	receivedMu.Unlock()
}
