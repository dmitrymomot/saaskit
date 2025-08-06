package broadcast

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryBroadcaster_Subscribe(t *testing.T) {
	t.Run("subscribe creates active subscriber", func(t *testing.T) {
		b := NewMemoryBroadcaster[string](10)
		defer b.Close()

		ctx := context.Background()
		sub := b.Subscribe(ctx)
		require.NotNil(t, sub)

		ch := sub.Receive(ctx)
		require.NotNil(t, ch)
	})

	t.Run("subscribe after close returns closed subscriber", func(t *testing.T) {
		b := NewMemoryBroadcaster[string](10)
		err := b.Close()
		require.NoError(t, err)

		ctx := context.Background()
		sub := b.Subscribe(ctx)
		require.NotNil(t, sub)

		ch := sub.Receive(ctx)
		_, ok := <-ch
		assert.False(t, ok)
	})

	t.Run("context cancellation unsubscribes", func(t *testing.T) {
		b := NewMemoryBroadcaster[string](10)
		defer b.Close()

		ctx, cancel := context.WithCancel(context.Background())
		sub := b.Subscribe(ctx)

		cancel()
		time.Sleep(50 * time.Millisecond)

		err := b.Broadcast(context.Background(), Message[string]{Data: "test"})
		require.NoError(t, err)

		select {
		case msg, ok := <-sub.Receive(context.Background()):
			if ok {
				t.Fatalf("should not receive after context cancel, got: %v", msg)
			}
		case <-time.After(50 * time.Millisecond):
		}
	})
}

func TestMemoryBroadcaster_Broadcast(t *testing.T) {
	t.Run("broadcast to single subscriber", func(t *testing.T) {
		b := NewMemoryBroadcaster[string](10)
		defer b.Close()

		ctx := context.Background()
		sub := b.Subscribe(ctx)

		msg := Message[string]{Data: "hello"}
		err := b.Broadcast(ctx, msg)
		require.NoError(t, err)

		received := <-sub.Receive(ctx)
		assert.Equal(t, "hello", received.Data)
	})

	t.Run("broadcast to multiple subscribers", func(t *testing.T) {
		b := NewMemoryBroadcaster[int](10)
		defer b.Close()

		ctx := context.Background()
		const numSubs = 5
		subs := make([]Subscriber[int], numSubs)

		for i := range numSubs {
			subs[i] = b.Subscribe(ctx)
		}

		msg := Message[int]{Data: 42}
		err := b.Broadcast(ctx, msg)
		require.NoError(t, err)

		for i, sub := range subs {
			select {
			case received := <-sub.Receive(ctx):
				assert.Equal(t, 42, received.Data, "subscriber %d", i)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("subscriber %d timeout", i)
			}
		}
	})

	t.Run("broadcast after close is safe", func(t *testing.T) {
		b := NewMemoryBroadcaster[string](10)
		err := b.Close()
		require.NoError(t, err)

		err = b.Broadcast(context.Background(), Message[string]{Data: "test"})
		assert.NoError(t, err)
	})

	t.Run("slow subscriber is dropped", func(t *testing.T) {
		b := NewMemoryBroadcaster[int](1)
		defer b.Close()

		ctx := context.Background()
		sub := b.Subscribe(ctx)

		for i := range 10 {
			err := b.Broadcast(ctx, Message[int]{Data: i})
			require.NoError(t, err)
		}

		time.Sleep(50 * time.Millisecond)

		count := 0
		timeout := time.After(100 * time.Millisecond)
		for {
			select {
			case _, ok := <-sub.Receive(ctx):
				if !ok {
					return
				}
				count++
			case <-timeout:
				assert.LessOrEqual(t, count, 2)
				return
			}
		}
	})
}

func TestMemoryBroadcaster_Close(t *testing.T) {
	t.Run("close closes all subscribers", func(t *testing.T) {
		b := NewMemoryBroadcaster[string](10)

		ctx := context.Background()
		subs := make([]Subscriber[string], 3)
		for i := range subs {
			subs[i] = b.Subscribe(ctx)
		}

		err := b.Close()
		require.NoError(t, err)

		for i, sub := range subs {
			_, ok := <-sub.Receive(ctx)
			assert.False(t, ok, "subscriber %d channel should be closed", i)
		}
	})

	t.Run("double close is safe", func(t *testing.T) {
		b := NewMemoryBroadcaster[string](10)

		err := b.Close()
		require.NoError(t, err)

		err = b.Close()
		require.NoError(t, err)
	})
}

func TestMemoryBroadcaster_Generic(t *testing.T) {
	type CustomMessage struct {
		ID   int
		Text string
	}

	t.Run("struct type", func(t *testing.T) {
		b := NewMemoryBroadcaster[CustomMessage](10)
		defer b.Close()

		ctx := context.Background()
		sub := b.Subscribe(ctx)

		msg := Message[CustomMessage]{
			Data: CustomMessage{ID: 1, Text: "hello"},
		}
		err := b.Broadcast(ctx, msg)
		require.NoError(t, err)

		received := <-sub.Receive(ctx)
		assert.Equal(t, 1, received.Data.ID)
		assert.Equal(t, "hello", received.Data.Text)
	})

	t.Run("slice type", func(t *testing.T) {
		b := NewMemoryBroadcaster[[]byte](10)
		defer b.Close()

		ctx := context.Background()
		sub := b.Subscribe(ctx)

		data := []byte("binary data")
		msg := Message[[]byte]{Data: data}
		err := b.Broadcast(ctx, msg)
		require.NoError(t, err)

		received := <-sub.Receive(ctx)
		assert.Equal(t, data, received.Data)
	})
}

func TestMemoryBroadcaster_Concurrent(t *testing.T) {
	t.Run("concurrent broadcasts", func(t *testing.T) {
		b := NewMemoryBroadcaster[int](1000)
		defer b.Close()

		ctx := context.Background()
		sub := b.Subscribe(ctx)

		const numGoroutines = 10
		const msgsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(base int) {
				defer wg.Done()
				for j := range msgsPerGoroutine {
					err := b.Broadcast(ctx, Message[int]{Data: base*1000 + j})
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		received := make(map[int]bool)
		timeout := time.After(1 * time.Second)

		count := 0
	loop:
		for count < numGoroutines*msgsPerGoroutine {
			select {
			case msg, ok := <-sub.Receive(ctx):
				if !ok {
					break loop
				}
				received[msg.Data] = true
				count++
			case <-timeout:
				break loop
			}
		}

		assert.GreaterOrEqual(t, len(received), 900)
	})

	t.Run("concurrent subscribe/unsubscribe", func(t *testing.T) {
		b := NewMemoryBroadcaster[string](10)
		defer b.Close()

		var wg sync.WaitGroup
		const numGoroutines = 50

		wg.Add(numGoroutines)
		for range numGoroutines {
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()

				sub := b.Subscribe(ctx)
				<-sub.Receive(ctx)
			}()
		}

		go func() {
			for range 100 {
				b.Broadcast(context.Background(), Message[string]{Data: "test"})
				time.Sleep(time.Millisecond)
			}
		}()

		wg.Wait()
	})
}

func BenchmarkMemoryBroadcaster_Broadcast(b *testing.B) {
	broadcaster := NewMemoryBroadcaster[string](100)
	defer broadcaster.Close()

	ctx := context.Background()
	const numSubs = 10

	for range numSubs {
		sub := broadcaster.Subscribe(ctx)
		go func(s Subscriber[string]) {
			for range s.Receive(ctx) {
			}
		}(sub)
	}

	msg := Message[string]{Data: "benchmark"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			broadcaster.Broadcast(ctx, msg)
		}
	})
}

func BenchmarkMemoryBroadcaster_Subscribe(b *testing.B) {
	broadcaster := NewMemoryBroadcaster[string](10)
	defer broadcaster.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sub := broadcaster.Subscribe(ctx)
			sub.Close()
		}
	})
}
