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
	lastAccess time.Time // Used by cleanup to identify stale buckets
}

// MemoryStore implements Store interface using in-memory storage.
type MemoryStore struct {
	mu      sync.RWMutex
	buckets map[string]*bucket

	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// MemoryStoreOption configures a MemoryStore.
type MemoryStoreOption func(*MemoryStore)

// WithCleanupInterval sets the cleanup interval for removing stale buckets.
// Set to 0 to disable automatic cleanup.
func WithCleanupInterval(interval time.Duration) MemoryStoreOption {
	return func(ms *MemoryStore) {
		ms.cleanupInterval = interval
	}
}

// NewMemoryStore creates a new in-memory store with optional cleanup.
func NewMemoryStore(opts ...MemoryStoreOption) *MemoryStore {
	ms := &MemoryStore{
		buckets:         make(map[string]*bucket),
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(ms)
	}

	// Start background cleanup only if interval is set
	if ms.cleanupInterval > 0 {
		go ms.cleanup()
	}

	return ms
}

// ConsumeTokens attempts to consume tokens from the bucket.
func (ms *MemoryStore) ConsumeTokens(ctx context.Context, key string, tokens int, config Config) (remaining int, resetAt time.Time, err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	b, exists := ms.buckets[key]

	if !exists {
		b = &bucket{
			tokens:     config.Capacity,
			lastRefill: now,
			lastAccess: now,
		}
		ms.buckets[key] = b
	}

	// Token bucket algorithm: calculate how many refill intervals have passed
	// and add the corresponding tokens, then consume the requested amount
	elapsed := now.Sub(b.lastRefill)
	// Cap intervals to prevent integer overflow in high-capacity/low-rate scenarios
	maxIntervals := int64(config.Capacity/config.RefillRate + 1)
	intervalsElapsed := int(min(int64(elapsed/config.RefillInterval), maxIntervals))

	if intervalsElapsed > 0 {
		tokensToAdd := intervalsElapsed * config.RefillRate
		b.tokens = min(b.tokens+tokensToAdd, config.Capacity)
		b.lastRefill = now // Prevent time drift accumulation
	}

	b.tokens -= tokens
	remaining = b.tokens
	b.lastAccess = now

	resetAt = b.lastRefill.Add(config.RefillInterval)

	return remaining, resetAt, nil
}

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

// removeStale removes buckets that haven't been accessed recently to prevent memory leaks.
func (ms *MemoryStore) removeStale() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	staleThreshold := 1 * time.Hour

	for key, b := range ms.buckets {
		if now.Sub(b.lastAccess) > staleThreshold {
			delete(ms.buckets, key)
		}
	}
}

// Close stops the cleanup goroutine. Safe to call multiple times.
func (ms *MemoryStore) Close() {
	select {
	case <-ms.stopCleanup:
		// Already closed
	default:
		close(ms.stopCleanup)
	}
}
