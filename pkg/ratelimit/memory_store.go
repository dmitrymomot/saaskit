package ratelimit

import (
	"context"
	"sync"
	"time"
)

// MemoryStore implements an in-memory store for rate limiting using
// separate maps for token bucket counters and sliding window timestamps.
// Includes automatic cleanup of expired entries to prevent memory leaks.
type MemoryStore struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	windows map[string]*slidingWindow

	cleanupInterval time.Duration
	initialCapacity int
	stopCleanup     chan struct{}
	cleanupOnce     sync.Once
}

type bucket struct {
	count     int64
	expiresAt time.Time
}

type slidingWindow struct {
	timestamps []time.Time
	mu         sync.Mutex
}

// MemoryStoreOption configures a MemoryStore.
type MemoryStoreOption func(*MemoryStore)

// WithCleanupInterval sets the cleanup interval for expired entries.
func WithCleanupInterval(interval time.Duration) MemoryStoreOption {
	return func(s *MemoryStore) {
		if interval > 0 {
			s.cleanupInterval = interval
		}
	}
}

// WithInitialCapacity sets the initial capacity for sliding window timestamps.
func WithInitialCapacity(capacity int) MemoryStoreOption {
	return func(s *MemoryStore) {
		if capacity > 0 {
			s.initialCapacity = capacity
		}
	}
}

// NewMemoryStore creates a new in-memory store with automatic cleanup.
func NewMemoryStore(opts ...MemoryStoreOption) *MemoryStore {
	s := &MemoryStore{
		buckets:         make(map[string]*bucket),
		windows:         make(map[string]*slidingWindow),
		cleanupInterval: 1 * time.Minute,
		initialCapacity: 100,
		stopCleanup:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	go s.cleanupLoop()

	return s
}

// IncrementAndGet atomically increments the counter for token bucket algorithm.
// Creates a new bucket if one doesn't exist or has expired.
func (s *MemoryStore) IncrementAndGet(ctx context.Context, key string, incr int, window time.Duration) (int64, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	b, exists := s.buckets[key]

	if !exists || now.After(b.expiresAt) {
		b = &bucket{
			count:     int64(incr),
			expiresAt: now.Add(window),
		}
		s.buckets[key] = b
		return b.count, window, nil
	}

	b.count += int64(incr)
	return b.count, time.Until(b.expiresAt), nil
}

// Get returns the current counter value and TTL for token bucket algorithm.
// Returns (0, 0) if the bucket doesn't exist or has expired.
func (s *MemoryStore) Get(ctx context.Context, key string) (int64, time.Duration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, exists := s.buckets[key]
	if !exists {
		return 0, 0, nil
	}

	now := time.Now()
	if now.After(b.expiresAt) {
		return 0, 0, nil
	}

	return b.count, time.Until(b.expiresAt), nil
}

// Delete removes the given key from both stores.
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.buckets, key)
	delete(s.windows, key)
	return nil
}

// RecordTimestamp adds a timestamp to the sliding window and removes
// expired timestamps to maintain accuracy and prevent memory growth.
func (s *MemoryStore) RecordTimestamp(ctx context.Context, key string, timestamp time.Time, window time.Duration) error {
	s.mu.Lock()
	sw, exists := s.windows[key]
	if !exists {
		sw = &slidingWindow{
			timestamps: make([]time.Time, 0, s.initialCapacity),
		}
		s.windows[key] = sw
	}
	s.mu.Unlock()

	sw.mu.Lock()
	defer sw.mu.Unlock()

	cutoff := timestamp.Add(-window)
	validTimestamps := make([]time.Time, 0, len(sw.timestamps)+1)

	for _, ts := range sw.timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	validTimestamps = append(validTimestamps, timestamp)
	sw.timestamps = validTimestamps

	return nil
}

// CountInWindow returns the number of timestamps within the sliding window
// and cleans up expired timestamps as a side effect for memory efficiency.
func (s *MemoryStore) CountInWindow(ctx context.Context, key string, window time.Duration) (int64, error) {
	s.mu.RLock()
	sw, exists := s.windows[key]
	s.mu.RUnlock()

	if !exists {
		return 0, nil
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	cutoff := time.Now().Add(-window)
	count := int64(0)

	validTimestamps := make([]time.Time, 0, len(sw.timestamps))
	for _, ts := range sw.timestamps {
		if ts.After(cutoff) {
			count++
			validTimestamps = append(validTimestamps, ts)
		}
	}

	sw.timestamps = validTimestamps
	return count, nil
}

// CleanupExpired removes expired timestamps from the sliding window.
func (s *MemoryStore) CleanupExpired(ctx context.Context, key string, window time.Duration) error {
	s.mu.RLock()
	sw, exists := s.windows[key]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	cutoff := time.Now().Add(-window)
	validTimestamps := make([]time.Time, 0, len(sw.timestamps))

	for _, ts := range sw.timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	sw.timestamps = validTimestamps

	if len(sw.timestamps) == 0 {
		s.mu.Lock()
		delete(s.windows, key)
		s.mu.Unlock()
	}

	return nil
}

// cleanupLoop runs in a separate goroutine to periodically remove
// expired buckets and empty windows, preventing memory leaks.
func (s *MemoryStore) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.stopCleanup:
			return
		}
	}
}

// cleanup removes expired token buckets and empty sliding windows.
// Called periodically by cleanupLoop to prevent unbounded memory growth.
func (s *MemoryStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for key, b := range s.buckets {
		if now.After(b.expiresAt) {
			delete(s.buckets, key)
		}
	}

	for key, sw := range s.windows {
		sw.mu.Lock()
		if len(sw.timestamps) == 0 {
			delete(s.windows, key)
		}
		sw.mu.Unlock()
	}
}

// Close stops the cleanup goroutine.
func (s *MemoryStore) Close() error {
	s.cleanupOnce.Do(func() {
		close(s.stopCleanup)
	})
	return nil
}
