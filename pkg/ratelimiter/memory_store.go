package ratelimiter

import (
	"context"
	"sync"
	"time"
)

// bucket represents a token bucket state.
type bucket struct {
	tokens     int
	lastRefill time.Time
}

// MemoryStore implements Store interface using in-memory storage.
type MemoryStore struct {
	mu      sync.RWMutex
	buckets map[string]*bucket

	// cleanup management
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupOnce     sync.Once
}

// NewMemoryStore creates a new in-memory store with automatic cleanup.
func NewMemoryStore() *MemoryStore {
	ms := &MemoryStore{
		buckets:         make(map[string]*bucket),
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go ms.cleanup()

	return ms
}

// ConsumeTokens attempts to consume tokens from the bucket.
func (ms *MemoryStore) ConsumeTokens(ctx context.Context, key string, tokens int, config Config) (remaining int, resetAt time.Time, err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	b, exists := ms.buckets[key]

	if !exists {
		// Create new bucket with full capacity
		b = &bucket{
			tokens:     config.Capacity,
			lastRefill: now,
		}
		ms.buckets[key] = b
	}

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(b.lastRefill)
	intervalsElapsed := int(elapsed / config.RefillInterval)

	if intervalsElapsed > 0 {
		// Refill tokens
		tokensToAdd := intervalsElapsed * config.RefillRate
		b.tokens = min(b.tokens+tokensToAdd, config.Capacity)
		b.lastRefill = b.lastRefill.Add(time.Duration(intervalsElapsed) * config.RefillInterval)
	}

	// Consume tokens
	b.tokens -= tokens
	remaining = b.tokens

	// Calculate next refill time
	resetAt = b.lastRefill.Add(config.RefillInterval)

	return remaining, resetAt, nil
}

// Reset clears the bucket for the given key.
func (ms *MemoryStore) Reset(ctx context.Context, key string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.buckets, key)
	return nil
}

// cleanup runs periodically to remove stale buckets.
func (ms *MemoryStore) cleanup() {
	ticker := time.NewTicker(ms.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ms.removeStale()
		case <-ms.stopCleanup:
			return
		}
	}
}

// removeStale removes buckets that haven't been accessed in a while.
func (ms *MemoryStore) removeStale() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	staleThreshold := 1 * time.Hour

	for key, b := range ms.buckets {
		if now.Sub(b.lastRefill) > staleThreshold {
			delete(ms.buckets, key)
		}
	}
}

// Close stops the cleanup goroutine.
func (ms *MemoryStore) Close() {
	ms.cleanupOnce.Do(func() {
		close(ms.stopCleanup)
	})
}
