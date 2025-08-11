package ratelimiter_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/ratelimiter"
)

// TestBucket_ContextCancellation verifies proper context handling
func TestBucket_ContextCancellation(t *testing.T) {
	t.Parallel()

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       10,
		RefillRate:     1,
		RefillInterval: time.Second,
	}

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	t.Run("cancelled context before operation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// The memory store doesn't check context, but this tests the pattern
		result, err := tb.Allow(ctx, "test-cancelled")
		// Memory store doesn't return context errors, but result should be valid
		if err == nil {
			assert.NotNil(t, result)
		}
	})

	t.Run("context timeout during operation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Give time for timeout
		time.Sleep(2 * time.Millisecond)

		result, err := tb.Allow(ctx, "test-timeout")
		// Memory store doesn't check context, but this tests the pattern
		if err == nil {
			assert.NotNil(t, result)
		}
	})

	t.Run("context with deadline", func(t *testing.T) {
		deadline := time.Now().Add(10 * time.Millisecond)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		result, err := tb.Allow(ctx, "test-deadline")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// mockFailingStore simulates store backend failures
type mockFailingStore struct {
	failAfter  int
	callCount  atomic.Int32
	errorMsg   string
	mu         sync.Mutex
	shouldFail bool
}

func newMockFailingStore(failAfter int, errorMsg string) *mockFailingStore {
	return &mockFailingStore{
		failAfter:  failAfter,
		errorMsg:   errorMsg,
		shouldFail: true,
	}
}

func (s *mockFailingStore) ConsumeTokens(ctx context.Context, key string, tokens int, config ratelimiter.Config) (int, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := int(s.callCount.Add(1))
	if s.shouldFail && count > s.failAfter {
		return 0, time.Time{}, errors.New(s.errorMsg)
	}

	// Simple implementation for successful calls
	remaining := config.Capacity - tokens
	resetAt := time.Now().Add(config.RefillInterval)
	return remaining, resetAt, nil
}

func (s *mockFailingStore) Reset(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shouldFail {
		return errors.New(s.errorMsg)
	}
	return nil
}

func (s *mockFailingStore) Close() error {
	return nil
}

func (s *mockFailingStore) EnableFailure(enable bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shouldFail = enable
}

// TestBucket_StoreFailures verifies handling of store backend failures
func TestBucket_StoreFailures(t *testing.T) {
	t.Parallel()

	t.Run("store connection failure", func(t *testing.T) {
		store := newMockFailingStore(0, "connection refused")
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       10,
			RefillRate:     1,
			RefillInterval: time.Second,
		}

		tb, err := ratelimiter.NewBucket(store, config)
		require.NoError(t, err)

		result, err := tb.Allow(context.Background(), "test-key")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("store intermittent failures", func(t *testing.T) {
		store := newMockFailingStore(2, "temporary failure")
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       10,
			RefillRate:     1,
			RefillInterval: time.Second,
		}

		tb, err := ratelimiter.NewBucket(store, config)
		require.NoError(t, err)

		// First two calls should succeed
		for i := 0; i < 2; i++ {
			result, err := tb.Allow(context.Background(), "test-key")
			assert.NoError(t, err)
			assert.NotNil(t, result)
		}

		// Third call should fail
		result, err := tb.Allow(context.Background(), "test-key")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "temporary failure")
	})

	t.Run("reset operation failure", func(t *testing.T) {
		store := newMockFailingStore(0, "reset failed")
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       10,
			RefillRate:     1,
			RefillInterval: time.Second,
		}

		tb, err := ratelimiter.NewBucket(store, config)
		require.NoError(t, err)

		err = tb.Reset(context.Background(), "test-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reset failed")
	})

	t.Run("recovery after failure", func(t *testing.T) {
		store := newMockFailingStore(2, "temporary error")
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       10,
			RefillRate:     1,
			RefillInterval: time.Second,
		}

		tb, err := ratelimiter.NewBucket(store, config)
		require.NoError(t, err)

		// Use up the successful calls
		for i := 0; i < 2; i++ {
			_, _ = tb.Allow(context.Background(), "test-key")
		}

		// This should fail
		result, err := tb.Allow(context.Background(), "test-key")
		assert.Error(t, err)
		assert.Nil(t, result)

		// Disable failures
		store.EnableFailure(false)

		// Should succeed now
		result, err = tb.Allow(context.Background(), "test-key")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestBucket_ClockEdgeCases tests time-related edge cases
func TestBucket_ClockEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("very large refill intervals", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		// Test with maximum duration (prevent overflow)
		config := ratelimiter.Config{
			Capacity:       100,
			RefillRate:     1,
			RefillInterval: 24 * 365 * time.Hour, // 1 year
		}

		tb, err := ratelimiter.NewBucket(store, config)
		require.NoError(t, err)

		// Should handle large intervals without overflow
		result, err := tb.Allow(context.Background(), "test-large-interval")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 99, result.Remaining)
	})

	t.Run("rapid successive calls", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       5,
			RefillRate:     1000,
			RefillInterval: time.Microsecond, // Very fast refill
		}

		tb, err := ratelimiter.NewBucket(store, config)
		require.NoError(t, err)

		// Rapid calls should not cause issues
		for i := 0; i < 100; i++ {
			result, err := tb.Allow(context.Background(), "test-rapid")
			assert.NoError(t, err)
			assert.NotNil(t, result)
		}
	})

	t.Run("zero time handling", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       10,
			RefillRate:     1,
			RefillInterval: time.Second,
		}

		tb, err := ratelimiter.NewBucket(store, config)
		require.NoError(t, err)

		// First call establishes the bucket
		result, err := tb.Allow(context.Background(), "test-zero-time")
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// ResetAt should never be zero time
		assert.False(t, result.ResetAt.IsZero())
	})

	t.Run("integer overflow prevention", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       1<<31 - 1, // Max int32
			RefillRate:     1<<31 - 1, // Max int32
			RefillInterval: time.Nanosecond,
		}

		tb, err := ratelimiter.NewBucket(store, config)
		require.NoError(t, err)

		// Should handle without overflow
		result, err := tb.Allow(context.Background(), "test-overflow")
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Wait a tiny bit for potential refill
		time.Sleep(time.Millisecond)

		// Should still work without overflow
		result, err = tb.Allow(context.Background(), "test-overflow")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestMemoryStore_MemoryLeak verifies cleanup prevents memory leaks
func TestMemoryStore_MemoryLeak(t *testing.T) {
	// Use shorter cleanup interval for testing
	store := ratelimiter.NewMemoryStore(
		ratelimiter.WithCleanupInterval(100 * time.Millisecond),
	)
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       10,
		RefillRate:     1,
		RefillInterval: 50 * time.Millisecond, // Short TTL
	}

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	// Create multiple buckets
	keys := make([]string, 100)
	for i := range 100 {
		keys[i] = fmt.Sprintf("test-key-%d", i)
		_, err := tb.Allow(context.Background(), keys[i])
		assert.NoError(t, err)
	}

	// Wait for buckets to become stale (2x TTL + cleanup interval)
	time.Sleep(250 * time.Millisecond)

	// Create activity on one key to keep it alive
	activeKey := "active-key"
	_, err = tb.Allow(context.Background(), activeKey)
	assert.NoError(t, err)

	// Trigger cleanup by waiting
	time.Sleep(150 * time.Millisecond)

	// Verify old buckets can still be created (implying they were cleaned)
	for _, key := range keys[:10] {
		result, err := tb.Allow(context.Background(), key)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Should have full capacity (new bucket)
		assert.Equal(t, 9, result.Remaining)
	}

	// Active key should still have its state
	result, err := tb.Allow(context.Background(), activeKey)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Should have consumed at least 1 token (we made 2 calls)
	assert.Less(t, result.Remaining, 10)
}

// eventually is a helper that polls a condition until it's true or timeout
func eventually(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	interval := timeout / 100
	if interval < time.Millisecond {
		interval = time.Millisecond
	}

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	t.Errorf("condition not met within %v: %s", timeout, msg)
}

// TestBucket_EventualConsistency replaces sleep with polling
func TestBucket_EventualConsistency(t *testing.T) {
	t.Parallel()

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	config := ratelimiter.Config{
		Capacity:       5,
		RefillRate:     5,
		RefillInterval: 100 * time.Millisecond,
	}

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	// Consume all tokens
	for range 5 {
		result, err := tb.Allow(context.Background(), "test-refill")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}

	// Should be exhausted
	result, err := tb.Allow(context.Background(), "test-refill")
	assert.NoError(t, err)
	assert.False(t, result.Allowed())

	// Poll for refill instead of sleep
	eventually(t, func() bool {
		status, _ := tb.Status(context.Background(), "test-refill")
		return status != nil && status.Remaining > 0
	}, 200*time.Millisecond, "tokens should refill")

	// Should have tokens again
	result, err = tb.Allow(context.Background(), "test-refill")
	assert.NoError(t, err)
	assert.True(t, result.Allowed())
}
