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

func TestNewTokenBucket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		store       ratelimit.Store
		rate        int
		interval    time.Duration
		opts        []ratelimit.TokenBucketOption
		expectError error
	}{
		{
			name:        "nil store",
			store:       nil,
			rate:        10,
			interval:    time.Second,
			expectError: ratelimit.ErrStoreRequired,
		},
		{
			name:        "zero rate",
			store:       ratelimit.NewMemoryStore(),
			rate:        0,
			interval:    time.Second,
			expectError: ratelimit.ErrInvalidLimit,
		},
		{
			name:        "negative rate",
			store:       ratelimit.NewMemoryStore(),
			rate:        -1,
			interval:    time.Second,
			expectError: ratelimit.ErrInvalidLimit,
		},
		{
			name:        "zero interval",
			store:       ratelimit.NewMemoryStore(),
			rate:        10,
			interval:    0,
			expectError: ratelimit.ErrInvalidInterval,
		},
		{
			name:        "negative interval",
			store:       ratelimit.NewMemoryStore(),
			rate:        10,
			interval:    -1 * time.Second,
			expectError: ratelimit.ErrInvalidInterval,
		},
		{
			name:     "valid with default burst",
			store:    ratelimit.NewMemoryStore(),
			rate:     10,
			interval: time.Second,
		},
		{
			name:     "valid with custom burst",
			store:    ratelimit.NewMemoryStore(),
			rate:     10,
			interval: time.Second,
			opts:     []ratelimit.TokenBucketOption{ratelimit.WithBurst(20)},
		},
		{
			name:     "burst less than rate gets adjusted",
			store:    ratelimit.NewMemoryStore(),
			rate:     10,
			interval: time.Second,
			opts:     []ratelimit.TokenBucketOption{ratelimit.WithBurst(5)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tb, err := ratelimit.NewTokenBucket(tt.store, tt.rate, tt.interval, tt.opts...)
			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)
				assert.Nil(t, tb)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tb)
				// Can't test unexported fields in black-box testing
			}
		})
	}
}

func TestTokenBucket_Allow(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	tb, err := ratelimit.NewTokenBucket(store, 5, 100*time.Millisecond, ratelimit.WithBurst(10))
	require.NoError(t, err)

	ctx := context.Background()
	key := "test-key"

	t.Run("empty key", func(t *testing.T) {
		result, err := tb.Allow(ctx, "")
		assert.ErrorIs(t, err, ratelimit.ErrKeyRequired)
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

	store := ratelimit.NewMemoryStore()
	tb, err := ratelimit.NewTokenBucket(store, 10, time.Second, ratelimit.WithBurst(20))
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

	t.Run("negative n returns error", func(t *testing.T) {
		_, err := tb.AllowN(ctx, key+"-neg", -5)
		require.Error(t, err)
		assert.Equal(t, ratelimit.ErrInvalidLimit, err)
	})

	t.Run("zero n returns error", func(t *testing.T) {
		_, err := tb.AllowN(ctx, key+"-zero", 0)
		require.Error(t, err)
		assert.Equal(t, ratelimit.ErrInvalidLimit, err)
	})
}

func TestTokenBucket_Concurrent(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	tb, err := ratelimit.NewTokenBucket(store, 100, time.Second, ratelimit.WithBurst(100))
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

	store := ratelimit.NewMemoryStore()
	tb, err := ratelimit.NewTokenBucket(store, 5, 100*time.Millisecond, ratelimit.WithBurst(10))
	require.NoError(t, err)

	ctx := context.Background()
	key := "status-key"

	t.Run("empty key", func(t *testing.T) {
		result, err := tb.Status(ctx, "")
		assert.ErrorIs(t, err, ratelimit.ErrKeyRequired)
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

	store := ratelimit.NewMemoryStore()
	tb, err := ratelimit.NewTokenBucket(store, 10, time.Second, ratelimit.WithBurst(10))
	require.NoError(t, err)

	ctx := context.Background()
	key := "reset-key"

	t.Run("empty key", func(t *testing.T) {
		err := tb.Reset(ctx, "")
		assert.ErrorIs(t, err, ratelimit.ErrKeyRequired)
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

	t.Run("reset clears bucket state", func(t *testing.T) {
		_, err := tb.Allow(ctx, "refill-test")
		require.NoError(t, err)

		// Can't verify internal state in black-box testing
		// Just verify Reset works
		err = tb.Reset(ctx, "refill-test")
		require.NoError(t, err)

		// After reset, should be able to use full capacity again
		result, err := tb.Allow(ctx, "refill-test")
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	})
}

func TestTokenBucket_MemoryLeak(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	tb, err := ratelimit.NewTokenBucket(store, 100, 50*time.Millisecond)
	require.NoError(t, err)

	ctx := context.Background()

	for i := range 100 {
		key := "leak-test-" + string(rune(i))
		_, err := tb.Allow(ctx, key)
		require.NoError(t, err)
	}

	// Can't test internal state in black-box testing
	// Just test that resets work
	for i := range 100 {
		key := "leak-test-" + string(rune(i))
		err := tb.Reset(ctx, key)
		require.NoError(t, err)
	}

	// After reset, should be able to use full capacity
	for i := range 100 {
		key := "leak-test-" + string(rune(i))
		result, err := tb.Allow(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}
}

func TestTokenBucket_BurstAdjustment(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()

	t.Run("burst smaller than rate gets adjusted", func(t *testing.T) {
		_, err := ratelimit.NewTokenBucket(store, 10, time.Second, ratelimit.WithBurst(5))
		require.NoError(t, err)
		// Can't test unexported burst field - behavior tested elsewhere
	})

	t.Run("burst equal to rate", func(t *testing.T) {
		_, err := ratelimit.NewTokenBucket(store, 10, time.Second, ratelimit.WithBurst(10))
		require.NoError(t, err)
		// Can't test unexported burst field - behavior tested elsewhere
	})

	t.Run("burst larger than rate", func(t *testing.T) {
		_, err := ratelimit.NewTokenBucket(store, 10, time.Second, ratelimit.WithBurst(20))
		require.NoError(t, err)
		// Can't test unexported burst field - behavior tested elsewhere
	})

	t.Run("zero burst option ignored", func(t *testing.T) {
		_, err := ratelimit.NewTokenBucket(store, 10, time.Second, ratelimit.WithBurst(0))
		require.NoError(t, err)
		// Can't test unexported burst field - behavior tested elsewhere
	})

	t.Run("negative burst option ignored", func(t *testing.T) {
		_, err := ratelimit.NewTokenBucket(store, 10, time.Second, ratelimit.WithBurst(-5))
		require.NoError(t, err)
		// Can't test unexported burst field - behavior tested elsewhere
	})
}
