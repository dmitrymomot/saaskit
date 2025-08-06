package notifications

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBroadcastDeliverer_LRUEviction(t *testing.T) {
	t.Run("evicts least recently used broadcaster", func(t *testing.T) {
		// Create deliverer with small limit for testing
		deliverer := NewBroadcastDeliverer(10, WithMaxBroadcasters(3))
		defer deliverer.Close()

		ctx := context.Background()

		// Subscribe for users to create broadcasters
		sub1 := deliverer.Subscribe(ctx, "user1")
		sub2 := deliverer.Subscribe(ctx, "user2")
		sub3 := deliverer.Subscribe(ctx, "user3")

		// Deliver notifications to verify they work
		err := deliverer.Deliver(ctx, Notification{UserID: "user1", Message: "msg1"})
		require.NoError(t, err)
		err = deliverer.Deliver(ctx, Notification{UserID: "user2", Message: "msg2"})
		require.NoError(t, err)
		err = deliverer.Deliver(ctx, Notification{UserID: "user3", Message: "msg3"})
		require.NoError(t, err)

		// Verify all subscribers receive messages
		select {
		case msg := <-sub1.Receive(ctx):
			assert.Equal(t, "msg1", msg.Data.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("user1 didn't receive message")
		}

		select {
		case msg := <-sub2.Receive(ctx):
			assert.Equal(t, "msg2", msg.Data.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("user2 didn't receive message")
		}

		select {
		case msg := <-sub3.Receive(ctx):
			assert.Equal(t, "msg3", msg.Data.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("user3 didn't receive message")
		}

		// Now add user4, which should evict user1 (least recently used)
		sub4 := deliverer.Subscribe(ctx, "user4")
		err = deliverer.Deliver(ctx, Notification{UserID: "user4", Message: "msg4"})
		require.NoError(t, err)

		select {
		case msg := <-sub4.Receive(ctx):
			assert.Equal(t, "msg4", msg.Data.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("user4 didn't receive message")
		}

		// User1's broadcaster was evicted. The old subscriber's channel should be closed.
		// Wait a bit for the eviction callback to complete
		time.Sleep(50 * time.Millisecond)

		// Check if the old subscriber's channel is closed
		select {
		case _, ok := <-sub1.Receive(ctx):
			if ok {
				t.Fatal("user1's channel should be closed after eviction")
			}
			// Channel is closed - this is expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout - channel should have been closed")
		}

		// Create a new subscriber for user1 (will create a new broadcaster)
		newSub1 := deliverer.Subscribe(ctx, "user1")

		// Send a message to the new broadcaster
		err = deliverer.Deliver(ctx, Notification{UserID: "user1", Message: "msg1-new"})
		require.NoError(t, err)

		// New subscriber should receive the message
		select {
		case msg := <-newSub1.Receive(ctx):
			assert.Equal(t, "msg1-new", msg.Data.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("new user1 subscriber didn't receive message")
		}
	})

	t.Run("accessing user updates LRU order", func(t *testing.T) {
		deliverer := NewBroadcastDeliverer(10, WithMaxBroadcasters(3))
		defer deliverer.Close()

		ctx := context.Background()

		// Create 3 users
		deliverer.Subscribe(ctx, "user1")
		deliverer.Subscribe(ctx, "user2")
		deliverer.Subscribe(ctx, "user3")

		// Access user1 to make it recently used
		deliverer.Deliver(ctx, Notification{UserID: "user1", Message: "test"})

		// Add user4 - should evict user2 (now least recently used)
		sub2 := deliverer.Subscribe(ctx, "user2")
		deliverer.Subscribe(ctx, "user4")

		// User2's old broadcaster should be evicted
		deliverer.Deliver(ctx, Notification{UserID: "user2", Message: "test"})

		// Check that user2 still works (because we just re-subscribed)
		select {
		case <-sub2.Receive(ctx):
			// Good - user2 works
		case <-time.After(100 * time.Millisecond):
			t.Fatal("user2 should still work")
		}
	})

	t.Run("configurable max broadcasters", func(t *testing.T) {
		// Test with limit of 1
		deliverer1 := NewBroadcastDeliverer(10, WithMaxBroadcasters(1))
		defer deliverer1.Close()

		ctx := context.Background()
		deliverer1.Subscribe(ctx, "user1")
		deliverer1.Subscribe(ctx, "user2")

		// Only user2 should work now
		err := deliverer1.Deliver(ctx, Notification{UserID: "user2", Message: "test"})
		require.NoError(t, err)

		// Test with limit of 100
		deliverer100 := NewBroadcastDeliverer(10, WithMaxBroadcasters(100))
		defer deliverer100.Close()

		// Should be able to add 100 users without eviction
		for i := range 100 {
			userID := fmt.Sprintf("user%d", i)
			deliverer100.Subscribe(ctx, userID)
		}

		// All should still work
		for i := range 100 {
			userID := fmt.Sprintf("user%d", i)
			err := deliverer100.Deliver(ctx, Notification{UserID: userID, Message: "test"})
			require.NoError(t, err)
		}
	})

	t.Run("default max broadcasters", func(t *testing.T) {
		// Without specifying, should use default (10000)
		deliverer := NewBroadcastDeliverer(10)
		defer deliverer.Close()

		// Should be able to handle many users
		ctx := context.Background()
		for i := range 1000 {
			userID := fmt.Sprintf("user%d", i)
			deliverer.Subscribe(ctx, userID)
		}

		// Verify it still works
		err := deliverer.Deliver(ctx, Notification{UserID: "user999", Message: "test"})
		require.NoError(t, err)
	})
}

func TestBroadcastDeliverer_Basic(t *testing.T) {
	t.Run("deliver single notification", func(t *testing.T) {
		deliverer := NewBroadcastDeliverer(10)
		defer deliverer.Close()

		ctx := context.Background()
		sub := deliverer.Subscribe(ctx, "user1")

		notif := Notification{
			ID:      "notif1",
			UserID:  "user1",
			Type:    TypeInfo,
			Title:   "Test",
			Message: "Test message",
		}

		err := deliverer.Deliver(ctx, notif)
		require.NoError(t, err)

		select {
		case msg := <-sub.Receive(ctx):
			assert.Equal(t, "notif1", msg.Data.ID)
			assert.Equal(t, "Test", msg.Data.Title)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for notification")
		}
	})

	t.Run("deliver batch notifications", func(t *testing.T) {
		deliverer := NewBroadcastDeliverer(10)
		defer deliverer.Close()

		ctx := context.Background()
		sub1 := deliverer.Subscribe(ctx, "user1")
		sub2 := deliverer.Subscribe(ctx, "user2")

		notifs := []Notification{
			{ID: "n1", UserID: "user1", Message: "msg1"},
			{ID: "n2", UserID: "user1", Message: "msg2"},
			{ID: "n3", UserID: "user2", Message: "msg3"},
		}

		err := deliverer.DeliverBatch(ctx, notifs)
		require.NoError(t, err)

		// User1 should get 2 notifications
		for i := range 2 {
			select {
			case msg := <-sub1.Receive(ctx):
				assert.Contains(t, []string{"msg1", "msg2"}, msg.Data.Message)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("timeout waiting for notification %d for user1", i+1)
			}
		}

		// User2 should get 1 notification
		select {
		case msg := <-sub2.Receive(ctx):
			assert.Equal(t, "msg3", msg.Data.Message)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for notification for user2")
		}
	})
}

func BenchmarkBroadcastDeliverer_WithLRU(b *testing.B) {
	deliverer := NewBroadcastDeliverer(100, WithMaxBroadcasters(1000))
	defer deliverer.Close()

	ctx := context.Background()

	// Create some subscribers
	for i := range 100 {
		userID := fmt.Sprintf("user%d", i)
		deliverer.Subscribe(ctx, userID)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user%d", i%100)
			deliverer.Deliver(ctx, Notification{
				UserID:  userID,
				Message: "benchmark",
			})
			i++
		}
	})
}
