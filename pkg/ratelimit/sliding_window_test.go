package ratelimit_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/ratelimit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlidingWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		store       ratelimit.SlidingWindowStore
		limit       int
		window      time.Duration
		expectError error
	}{
		{
			name:        "nil store",
			store:       nil,
			limit:       10,
			window:      time.Second,
			expectError: ratelimit.ErrStoreRequired,
		},
		{
			name:        "zero limit",
			store:       ratelimit.NewMemoryStore(),
			limit:       0,
			window:      time.Second,
			expectError: ratelimit.ErrInvalidLimit,
		},
		{
			name:        "negative limit",
			store:       ratelimit.NewMemoryStore(),
			limit:       -1,
			window:      time.Second,
			expectError: ratelimit.ErrInvalidLimit,
		},
		{
			name:        "zero window",
			store:       ratelimit.NewMemoryStore(),
			limit:       10,
			window:      0,
			expectError: ratelimit.ErrInvalidInterval,
		},
		{
			name:        "negative window",
			store:       ratelimit.NewMemoryStore(),
			limit:       10,
			window:      -1 * time.Second,
			expectError: ratelimit.ErrInvalidInterval,
		},
		{
			name:   "valid configuration",
			store:  ratelimit.NewMemoryStore(),
			limit:  10,
			window: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sw, err := ratelimit.NewSlidingWindow(tt.store, tt.limit, tt.window)
			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)
				assert.Nil(t, sw)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sw)
				// Can't test unexported fields in black-box testing
			}
		})
	}
}

func TestSlidingWindow_Allow(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	sw, err := ratelimit.NewSlidingWindow(store, 5, 100*time.Millisecond)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test-key"

	t.Run("empty key", func(t *testing.T) {
		result, err := sw.Allow(ctx, "")
		assert.ErrorIs(t, err, ratelimit.ErrKeyRequired)
		assert.Nil(t, result)
	})

	t.Run("enforce limit within window", func(t *testing.T) {
		for i := range 5 {
			result, err := sw.Allow(ctx, key)
			require.NoError(t, err)
			assert.True(t, result.Allowed, "request %d should be allowed", i+1)
			assert.Equal(t, 5, result.Limit)
			assert.Equal(t, 4-i, result.Remaining)
		}

		result, err := sw.Allow(ctx, key)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, 0, result.Remaining)
	})

	t.Run("sliding window expiration", func(t *testing.T) {
		testKey := key + "-sliding"

		for range 5 {
			result, err := sw.Allow(ctx, testKey)
			require.NoError(t, err)
			assert.True(t, result.Allowed)
		}

		result, err := sw.Allow(ctx, testKey)
		require.NoError(t, err)
		assert.False(t, result.Allowed)

		time.Sleep(110 * time.Millisecond)

		result, err = sw.Allow(ctx, testKey)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 4, result.Remaining)
	})
}

func TestSlidingWindow_AllowN(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	sw, err := ratelimit.NewSlidingWindow(store, 10, time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	key := "test-key-n"

	t.Run("consume multiple requests", func(t *testing.T) {
		result, err := sw.AllowN(ctx, key, 3)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 7, result.Remaining)

		result, err = sw.AllowN(ctx, key, 5)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 2, result.Remaining)

		result, err = sw.AllowN(ctx, key, 3)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, 2, result.Remaining)
	})

	t.Run("negative n returns error", func(t *testing.T) {
		_, err := sw.AllowN(ctx, key+"-neg", -5)
		require.Error(t, err)
		assert.Equal(t, ratelimit.ErrInvalidLimit, err)
	})

	t.Run("zero n returns error", func(t *testing.T) {
		_, err := sw.AllowN(ctx, key+"-zero", 0)
		require.Error(t, err)
		assert.Equal(t, ratelimit.ErrInvalidLimit, err)
	})
}

func TestSlidingWindow_WindowBoundaries(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	windowDuration := 200 * time.Millisecond // Increased window for more predictable test
	sw, err := ratelimit.NewSlidingWindow(store, 3, windowDuration)
	require.NoError(t, err)

	ctx := context.Background()
	key := "boundary-key"

	// Request 1 at T+0
	result, err := sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Request 2 at T+50ms
	time.Sleep(50 * time.Millisecond)
	result, err = sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Request 3 at T+100ms
	time.Sleep(50 * time.Millisecond)
	result, err = sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Request 4 at T+100ms - should be denied (3 requests in window)
	result, err = sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed, "should be denied, still 3 requests in window")

	// Wait until T+210ms - first request should expire
	time.Sleep(110 * time.Millisecond)

	// Request 5 at T+210ms - should be allowed (first request expired)
	result, err = sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed, "first request should have expired")
}

func TestSlidingWindow_Concurrent(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	sw, err := ratelimit.NewSlidingWindow(store, 100, time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	key := "concurrent-key"

	const goroutines = 50
	const requestsPerGoroutine = 10

	var allowed, denied int
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range requestsPerGoroutine {
				result, err := sw.Allow(ctx, key)
				if err == nil {
					mu.Lock()
					if result.Allowed {
						allowed++
					} else {
						denied++
					}
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, 100, allowed, "should allow exactly limit")
	assert.Equal(t, 400, denied, "should deny remaining requests")
}

func TestSlidingWindow_Status(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	sw, err := ratelimit.NewSlidingWindow(store, 5, time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	key := "status-key"

	t.Run("empty key", func(t *testing.T) {
		result, err := sw.Status(ctx, "")
		assert.ErrorIs(t, err, ratelimit.ErrKeyRequired)
		assert.Nil(t, result)
	})

	t.Run("status without consuming", func(t *testing.T) {
		result1, err := sw.Status(ctx, key)
		require.NoError(t, err)
		assert.True(t, result1.Allowed)
		assert.Equal(t, 5, result1.Limit)
		assert.Equal(t, 5, result1.Remaining)

		result2, err := sw.Status(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, result1.Remaining, result2.Remaining)
	})

	t.Run("status after consuming", func(t *testing.T) {
		_, err := sw.AllowN(ctx, key, 3)
		require.NoError(t, err)

		result, err := sw.Status(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 2, result.Remaining)
	})
}

func TestSlidingWindow_Reset(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	sw, err := ratelimit.NewSlidingWindow(store, 5, time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	key := "reset-key"

	t.Run("empty key", func(t *testing.T) {
		err := sw.Reset(ctx, "")
		assert.ErrorIs(t, err, ratelimit.ErrKeyRequired)
	})

	t.Run("reset clears timestamps", func(t *testing.T) {
		for range 5 {
			_, err := sw.Allow(ctx, key)
			require.NoError(t, err)
		}

		result, err := sw.Allow(ctx, key)
		require.NoError(t, err)
		assert.False(t, result.Allowed)

		err = sw.Reset(ctx, key)
		require.NoError(t, err)

		result, err = sw.Allow(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 4, result.Remaining)
	})
}

func TestSlidingWindow_AccurateTimestamps(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	sw, err := ratelimit.NewSlidingWindow(store, 2, 50*time.Millisecond)
	require.NoError(t, err)

	ctx := context.Background()
	key := "timestamp-accuracy"

	start := time.Now()
	result, err := sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	time.Sleep(25 * time.Millisecond)

	result, err = sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	result, err = sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed, "third request should be denied")

	elapsed := time.Since(start)
	waitTime := 50*time.Millisecond - elapsed + 5*time.Millisecond
	time.Sleep(waitTime)

	result, err = sw.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed, "should allow after first timestamp expires")
}
