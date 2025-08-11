package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenBucket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		store       Store
		rate        int
		interval    time.Duration
		opts        []TokenBucketOption
		expectError error
	}{
		{
			name:        "nil store",
			store:       nil,
			rate:        10,
			interval:    time.Second,
			expectError: ErrStoreRequired,
		},
		{
			name:        "zero rate",
			store:       NewMemoryStore(),
			rate:        0,
			interval:    time.Second,
			expectError: ErrInvalidLimit,
		},
		{
			name:        "negative rate",
			store:       NewMemoryStore(),
			rate:        -1,
			interval:    time.Second,
			expectError: ErrInvalidLimit,
		},
		{
			name:        "zero interval",
			store:       NewMemoryStore(),
			rate:        10,
			interval:    0,
			expectError: ErrInvalidInterval,
		},
		{
			name:        "negative interval",
			store:       NewMemoryStore(),
			rate:        10,
			interval:    -1 * time.Second,
			expectError: ErrInvalidInterval,
		},
		{
			name:     "valid with default burst",
			store:    NewMemoryStore(),
			rate:     10,
			interval: time.Second,
		},
		{
			name:     "valid with custom burst",
			store:    NewMemoryStore(),
			rate:     10,
			interval: time.Second,
			opts:     []TokenBucketOption{WithBurst(20)},
		},
		{
			name:     "burst less than rate gets adjusted",
			store:    NewMemoryStore(),
			rate:     10,
			interval: time.Second,
			opts:     []TokenBucketOption{WithBurst(5)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tb, err := NewTokenBucket(tt.store, tt.rate, tt.interval, tt.opts...)
			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)
				assert.Nil(t, tb)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tb)
				assert.GreaterOrEqual(t, tb.burst, tt.rate)
			}
		})
	}
}

func TestTokenBucket_Allow(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	tb, err := NewTokenBucket(store, 5, 100*time.Millisecond, WithBurst(10))
	require.NoError(t, err)

	ctx := context.Background()
	key := "test-key"

	t.Run("empty key", func(t *testing.T) {
		result, err := tb.Allow(ctx, "")
		assert.ErrorIs(t, err, ErrKeyRequired)
		assert.Nil(t, result)
	})

	t.Run("initial burst capacity", func(t *testing.T) {
		for i := range 10 {
			result, err := tb.Allow(ctx, key+"-burst")
			require.NoError(t, err)
			assert.True(t, result.Allowed, "request %d should be allowed", i+1)
			assert.Equal(t, 10, result.Limit)
			assert.Equal(t, 9-i, result.Remaining)
		}

		result, err := tb.Allow(ctx, key+"-burst")
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, 0, result.Remaining)
	})

	t.Run("token refill", func(t *testing.T) {
		testKey := key + "-refill"

		for range 10 {
			result, err := tb.Allow(ctx, testKey)
			require.NoError(t, err)
			assert.True(t, result.Allowed)
		}

		result, err := tb.Allow(ctx, testKey)
		require.NoError(t, err)
		assert.False(t, result.Allowed)

		time.Sleep(110 * time.Millisecond)

		for i := range 5 {
			result, err := tb.Allow(ctx, testKey)
			require.NoError(t, err)
			assert.True(t, result.Allowed, "request %d after refill should be allowed", i+1)
		}

		result, err = tb.Allow(ctx, testKey)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
	})
}

func TestTokenBucket_AllowN(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	tb, err := NewTokenBucket(store, 10, time.Second, WithBurst(20))
	require.NoError(t, err)

	ctx := context.Background()
	key := "test-key-n"

	t.Run("consume multiple tokens", func(t *testing.T) {
		result, err := tb.AllowN(ctx, key, 5)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 15, result.Remaining)

		result, err = tb.AllowN(ctx, key, 10)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 5, result.Remaining)

		result, err = tb.AllowN(ctx, key, 6)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, 5, result.Remaining)
	})

	t.Run("negative n defaults to 1", func(t *testing.T) {
		result, err := tb.AllowN(ctx, key+"-neg", -5)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 19, result.Remaining)
	})

	t.Run("zero n defaults to 1", func(t *testing.T) {
		result, err := tb.AllowN(ctx, key+"-zero", 0)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 19, result.Remaining)
	})
}

func TestTokenBucket_Concurrent(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	tb, err := NewTokenBucket(store, 100, time.Second, WithBurst(100))
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
				result, err := tb.Allow(ctx, key)
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

	assert.Equal(t, 100, allowed, "should allow exactly burst capacity")
	assert.Equal(t, 400, denied, "should deny remaining requests")
}

func TestTokenBucket_Status(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	tb, err := NewTokenBucket(store, 5, 100*time.Millisecond, WithBurst(10))
	require.NoError(t, err)

	ctx := context.Background()
	key := "status-key"

	t.Run("empty key", func(t *testing.T) {
		result, err := tb.Status(ctx, "")
		assert.ErrorIs(t, err, ErrKeyRequired)
		assert.Nil(t, result)
	})

	t.Run("status without consuming", func(t *testing.T) {
		result1, err := tb.Status(ctx, key)
		require.NoError(t, err)
		assert.True(t, result1.Allowed)
		assert.Equal(t, 10, result1.Limit)
		assert.Equal(t, 10, result1.Remaining)

		result2, err := tb.Status(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, result1.Remaining, result2.Remaining)
	})

	t.Run("status after consuming", func(t *testing.T) {
		_, err := tb.AllowN(ctx, key, 5)
		require.NoError(t, err)

		result, err := tb.Status(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 5, result.Remaining)
	})
}

func TestTokenBucket_Reset(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	tb, err := NewTokenBucket(store, 10, time.Second, WithBurst(10))
	require.NoError(t, err)

	ctx := context.Background()
	key := "reset-key"

	t.Run("empty key", func(t *testing.T) {
		err := tb.Reset(ctx, "")
		assert.ErrorIs(t, err, ErrKeyRequired)
	})

	t.Run("reset restores capacity", func(t *testing.T) {
		for range 10 {
			_, err := tb.Allow(ctx, key)
			require.NoError(t, err)
		}

		result, err := tb.Allow(ctx, key)
		require.NoError(t, err)
		assert.False(t, result.Allowed)

		err = tb.Reset(ctx, key)
		require.NoError(t, err)

		result, err = tb.Allow(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, 9, result.Remaining)
	})

	t.Run("reset clears lastRefill", func(t *testing.T) {
		_, err := tb.Allow(ctx, "refill-test")
		require.NoError(t, err)

		tb.mu.RLock()
		_, exists := tb.lastRefill["refill-test"]
		tb.mu.RUnlock()
		assert.True(t, exists)

		err = tb.Reset(ctx, "refill-test")
		require.NoError(t, err)

		tb.mu.RLock()
		_, exists = tb.lastRefill["refill-test"]
		tb.mu.RUnlock()
		assert.False(t, exists)
	})
}

func TestTokenBucket_MemoryLeak(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	tb, err := NewTokenBucket(store, 100, 50*time.Millisecond)
	require.NoError(t, err)

	ctx := context.Background()

	for i := range 100 {
		key := "leak-test-" + string(rune(i))
		_, err := tb.Allow(ctx, key)
		require.NoError(t, err)
	}

	tb.mu.RLock()
	initialSize := len(tb.lastRefill)
	tb.mu.RUnlock()
	assert.Equal(t, 100, initialSize)

	for i := range 100 {
		key := "leak-test-" + string(rune(i))
		err := tb.Reset(ctx, key)
		require.NoError(t, err)
	}

	tb.mu.RLock()
	finalSize := len(tb.lastRefill)
	tb.mu.RUnlock()
	assert.Equal(t, 0, finalSize)
}

func TestTokenBucket_BurstAdjustment(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()

	t.Run("burst smaller than rate gets adjusted", func(t *testing.T) {
		tb, err := NewTokenBucket(store, 10, time.Second, WithBurst(5))
		require.NoError(t, err)
		assert.Equal(t, 10, tb.burst)
	})

	t.Run("burst equal to rate", func(t *testing.T) {
		tb, err := NewTokenBucket(store, 10, time.Second, WithBurst(10))
		require.NoError(t, err)
		assert.Equal(t, 10, tb.burst)
	})

	t.Run("burst larger than rate", func(t *testing.T) {
		tb, err := NewTokenBucket(store, 10, time.Second, WithBurst(20))
		require.NoError(t, err)
		assert.Equal(t, 20, tb.burst)
	})

	t.Run("zero burst option ignored", func(t *testing.T) {
		tb, err := NewTokenBucket(store, 10, time.Second, WithBurst(0))
		require.NoError(t, err)
		assert.Equal(t, 10, tb.burst)
	})

	t.Run("negative burst option ignored", func(t *testing.T) {
		tb, err := NewTokenBucket(store, 10, time.Second, WithBurst(-5))
		require.NoError(t, err)
		assert.Equal(t, 10, tb.burst)
	})
}
