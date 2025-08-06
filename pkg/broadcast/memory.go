package broadcast

import (
	"context"
	"sync"
)

type MemoryBroadcaster[T any] struct {
	subscribers map[*subscriber[T]]struct{}
	bufferSize  int
	closed      bool
	mu          sync.RWMutex
}

func NewMemoryBroadcaster[T any](bufferSize int) *MemoryBroadcaster[T] {
	return &MemoryBroadcaster[T]{
		subscribers: make(map[*subscriber[T]]struct{}),
		bufferSize:  max(bufferSize, 1),
	}
}

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
	
	go func() {
		<-ctx.Done()
		b.unsubscribe(sub)
	}()
	
	return sub
}

func (b *MemoryBroadcaster[T]) Broadcast(ctx context.Context, msg Message[T]) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if b.closed {
		return nil
	}
	
	for sub := range b.subscribers {
		if !sub.send(msg) {
			go b.unsubscribe(sub)
		}
	}
	
	return nil
}

func (b *MemoryBroadcaster[T]) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.closed {
		return nil
	}
	
	b.closed = true
	
	for sub := range b.subscribers {
		_ = sub.Close()
	}
	
	clear(b.subscribers)
	return nil
}

func (b *MemoryBroadcaster[T]) unsubscribe(sub *subscriber[T]) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	delete(b.subscribers, sub)
	_ = sub.Close()
}