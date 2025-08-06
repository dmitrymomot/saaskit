package broadcast

import (
	"context"
	"sync"
)

// MemoryBroadcaster drops messages for slow consumers rather than blocking the broadcast operation.
// All methods are safe for concurrent use.
type MemoryBroadcaster[T any] struct {
	subscribers map[*subscriber[T]]struct{}
	bufferSize  int
	closed      bool
	mu          sync.RWMutex
	cleanupWg   sync.WaitGroup // tracks cleanup goroutines
}

// NewMemoryBroadcaster creates a new in-memory broadcaster.
// The bufferSize parameter determines the channel buffer size for each subscriber.
// A minimum buffer size of 1 is enforced. When a subscriber's buffer is full,
// new messages will be dropped for that subscriber rather than blocking the broadcast.
func NewMemoryBroadcaster[T any](bufferSize int) *MemoryBroadcaster[T] {
	return &MemoryBroadcaster[T]{
		subscribers: make(map[*subscriber[T]]struct{}),
		// Minimum buffer size of 1 prevents zero-buffer channels which would
		// make all sends blocking and defeat the non-blocking design
		bufferSize: max(bufferSize, 1),
	}
}

// Subscribe creates a new subscriber that will receive all broadcast messages.
// The subscription is automatically cleaned up when the provided context is cancelled.
// If the broadcaster is already closed, returns a closed subscriber.
func (b *MemoryBroadcaster[T]) Subscribe(ctx context.Context) Subscriber[T] {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		sub := newSubscriber[T](b.bufferSize)
		_ = sub.Close()
		return sub
	}

	sub := newSubscriber[T](b.bufferSize)
	b.subscribers[sub] = struct{}{}

	// Auto-cleanup on context cancellation
	if ctx.Done() != nil {
		b.cleanupWg.Add(1)
		go func() {
			defer b.cleanupWg.Done()
			<-ctx.Done()
			b.unsubscribe(sub)
		}()
	}

	return sub
}

// Broadcast sends a message to all active subscribers.
// Messages are sent non-blocking - if a subscriber's channel is full,
// the message is dropped for that subscriber and they are marked for removal.
// Returns nil even if some subscribers didn't receive the message.
func (b *MemoryBroadcaster[T]) Broadcast(ctx context.Context, msg Message[T]) error {
	// Use RLock for read-heavy operations: broadcasts are frequent,
	// subscriber map changes (add/remove) are infrequent
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil
	}

	for sub := range b.subscribers {
		if !sub.send(msg) {
			// Remove slow/closed subscribers asynchronously to avoid blocking
			// this broadcast. Using goroutine prevents write-lock contention
			// during read-heavy broadcast operations
			go b.unsubscribe(sub)
		}
	}

	return nil
}

// Close shuts down the broadcaster and closes all subscribers.
// It is safe to call Close multiple times.
// After Close, new subscriptions will receive already-closed subscribers,
// and Broadcast will have no effect.
func (b *MemoryBroadcaster[T]) Close() error {
	b.mu.Lock()

	if b.closed {
		b.mu.Unlock()
		return nil
	}

	b.closed = true

	// Close all subscribers
	for sub := range b.subscribers {
		_ = sub.Close()
	}

	clear(b.subscribers)
	b.mu.Unlock()

	// Wait for all cleanup goroutines to prevent race conditions between
	// Close() and async unsubscribe operations from Broadcast()
	b.cleanupWg.Wait()

	return nil
}

func (b *MemoryBroadcaster[T]) unsubscribe(sub *subscriber[T]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, sub)
	_ = sub.Close()
}
