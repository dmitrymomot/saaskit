package broadcast

import (
	"context"
	"sync"
)

// Message wraps data of type T for type-safe broadcasting.
type Message[T any] struct {
	Data T
}

// Subscriber receives messages from a Broadcaster.
// Implementations must be safe for concurrent use.
type Subscriber[T any] interface {
	// Receive returns a channel for receiving broadcast messages.
	// The context parameter allows implementations to respect cancellation
	// during blocking operations (e.g., in Redis or NATS adapters).
	// For the in-memory implementation, the context is not used but kept
	// for interface consistency across all adapter implementations.
	Receive(ctx context.Context) <-chan Message[T]
	
	// Close closes the subscriber and releases resources.
	// After Close, the receive channel is closed and no more messages will be received.
	// Close is idempotent and safe to call multiple times.
	Close() error
}

// Broadcaster sends messages to multiple subscribers.
// Implementations should handle slow consumers gracefully,
// typically by dropping messages rather than blocking.
type Broadcaster[T any] interface {
	// Subscribe creates a new subscriber that will receive all broadcast messages.
	// The context controls the lifetime of the subscription - when the context
	// is cancelled, the subscription is automatically cleaned up.
	Subscribe(ctx context.Context) Subscriber[T]
	
	// Broadcast sends a message to all active subscribers.
	// Messages may be dropped for slow consumers to prevent blocking.
	// The context parameter is kept for consistency but may not be used
	// by all implementations.
	Broadcast(ctx context.Context, msg Message[T]) error
	
	// Close shuts down the broadcaster and closes all subscribers.
	// After Close, Subscribe and Broadcast will return closed subscribers
	// and have no effect, respectively.
	Close() error
}

type subscriber[T any] struct {
	ch     chan Message[T]
	closed bool
	mu     sync.RWMutex
}

func newSubscriber[T any](bufferSize int) *subscriber[T] {
	return &subscriber[T]{
		ch: make(chan Message[T], bufferSize),
	}
}

func (s *subscriber[T]) Receive(ctx context.Context) <-chan Message[T] {
	return s.ch
}

func (s *subscriber[T]) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.closed {
		close(s.ch)
		s.closed = true
	}
	return nil
}

func (s *subscriber[T]) send(msg Message[T]) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.closed {
		return false
	}
	
	select {
	case s.ch <- msg:
		return true
	default:
		return false
	}
}