package broadcast

import (
	"context"
	"sync"
)

type Message[T any] struct {
	Data T
}

type Subscriber[T any] interface {
	Receive(ctx context.Context) <-chan Message[T]
	Close() error
}

type Broadcaster[T any] interface {
	Subscribe(ctx context.Context) Subscriber[T]
	Broadcast(ctx context.Context, msg Message[T]) error
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