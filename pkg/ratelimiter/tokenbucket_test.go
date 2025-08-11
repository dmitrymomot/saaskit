package ratelimiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/ratelimiter"
)

func TestNewBucket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      ratelimiter.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: ratelimiter.Config{
				Capacity:       100,
				RefillRate:     10,
				RefillInterval: time.Second,
			},
			expectError: false,
		},
		{
			name: "zero capacity",
			config: ratelimiter.Config{
				Capacity:       0,
				RefillRate:     10,
				RefillInterval: time.Second,
			},
			expectError: true,
			errorMsg:    "capacity must be positive",
		},
		{
			name: "negative capacity",
			config: ratelimiter.Config{
				Capacity:       -1,
				RefillRate:     10,
				RefillInterval: time.Second,
			},
			expectError: true,
			errorMsg:    "capacity must be positive",
		},
		{
			name: "zero refill rate",
			config: ratelimiter.Config{
				Capacity:       100,
				RefillRate:     0,
				RefillInterval: time.Second,
			},
			expectError: true,
			errorMsg:    "refill rate must be positive",
		},
		{
			name: "negative refill rate",
			config: ratelimiter.Config{
				Capacity:       100,
				RefillRate:     -5,
				RefillInterval: time.Second,
			},
			expectError: true,
			errorMsg:    "refill rate must be positive",
		},
		{
			name: "zero refill interval",
			config: ratelimiter.Config{
				Capacity:       100,
				RefillRate:     10,
				RefillInterval: 0,
			},
			expectError: true,
			errorMsg:    "refill interval must be positive",
		},
		{
			name: "negative refill interval",
			config: ratelimiter.Config{
				Capacity:       100,
				RefillRate:     10,
				RefillInterval: -time.Second,
			},
			expectError: true,
			errorMsg:    "refill interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := ratelimiter.NewMemoryStore()
			defer store.Close()

			tb, err := ratelimiter.NewBucket(store, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, tb)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tb)
			}
		})
	}
}

func TestBucket_Allow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       5,
		RefillRate:     1,
		RefillInterval: 100 * time.Millisecond,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	t.Run("allows requests within capacity", func(t *testing.T) {
		key := "test-allow"

		for i := 0; i < config.Capacity; i++ {
			result, err := tb.Allow(ctx, key)
			assert.NoError(t, err)
			assert.True(t, result.Allowed())
			assert.Equal(t, config.Capacity-i-1, result.Remaining)
		}

		result, err := tb.Allow(ctx, key)
		assert.NoError(t, err)
		assert.False(t, result.Allowed())
		assert.Equal(t, -1, result.Remaining)
	})

	t.Run("refills tokens after interval", func(t *testing.T) {
		key := "test-refill"

		// Consume all tokens
		for range config.Capacity {
			result, err := tb.Allow(ctx, key)
			require.NoError(t, err)
			require.True(t, result.Allowed())
		}

		// Should be denied (no tokens left)
		result, err := tb.Allow(ctx, key)
		assert.NoError(t, err)
		assert.False(t, result.Allowed())
		assert.Equal(t, -1, result.Remaining)

		// Wait for two refill intervals to ensure at least one refill happens
		time.Sleep(config.RefillInterval * 2)

		// Check status first to see current state
		status, err := tb.Status(ctx, key)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, status.Remaining, config.RefillRate)

		// Now should be allowed after refill
		result, err = tb.Allow(ctx, key)
		assert.NoError(t, err)
		assert.True(t, result.Allowed())
	})
}

func TestBucket_AllowN(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       10,
		RefillRate:     2,
		RefillInterval: 100 * time.Millisecond,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	t.Run("allows N requests within capacity", func(t *testing.T) {
		key := "test-allowN"

		result, err := tb.AllowN(ctx, key, 5)
		assert.NoError(t, err)
		assert.True(t, result.Allowed())
		assert.Equal(t, 5, result.Remaining)

		result, err = tb.AllowN(ctx, key, 3)
		assert.NoError(t, err)
		assert.True(t, result.Allowed())
		assert.Equal(t, 2, result.Remaining)

		result, err = tb.AllowN(ctx, key, 5)
		assert.NoError(t, err)
		assert.False(t, result.Allowed())
		assert.Equal(t, -3, result.Remaining)
	})

	t.Run("rejects zero tokens", func(t *testing.T) {
		key := "test-zero"

		result, err := tb.AllowN(ctx, key, 0)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ratelimiter.ErrInvalidTokenCount)
	})

	t.Run("rejects negative tokens", func(t *testing.T) {
		key := "test-negative"

		result, err := tb.AllowN(ctx, key, -5)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ratelimiter.ErrInvalidTokenCount)
	})

	t.Run("handles burst requests", func(t *testing.T) {
		key := "test-burst"

		result, err := tb.AllowN(ctx, key, config.Capacity)
		assert.NoError(t, err)
		assert.True(t, result.Allowed())
		assert.Equal(t, 0, result.Remaining)

		result, err = tb.AllowN(ctx, key, 1)
		assert.NoError(t, err)
		assert.False(t, result.Allowed())
	})
}

func TestBucket_Status(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       10,
		RefillRate:     1,
		RefillInterval: 100 * time.Millisecond,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	t.Run("returns status without consuming tokens", func(t *testing.T) {
		key := "test-status"

		result, err := tb.Status(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, config.Capacity, result.Limit)
		assert.Equal(t, config.Capacity, result.Remaining)

		result, err = tb.Status(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, config.Capacity, result.Remaining)

		allowResult, err := tb.AllowN(ctx, key, 3)
		require.NoError(t, err)
		require.True(t, allowResult.Allowed())

		result, err = tb.Status(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, 7, result.Remaining)
	})

	t.Run("shows negative remaining when exceeded", func(t *testing.T) {
		key := "test-status-exceeded"

		_, err := tb.AllowN(ctx, key, config.Capacity+2)
		require.NoError(t, err)

		result, err := tb.Status(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, -2, result.Remaining)
		assert.False(t, result.Allowed())
	})
}

func TestBucket_Reset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       5,
		RefillRate:     1,
		RefillInterval: time.Second,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	t.Run("resets consumed tokens", func(t *testing.T) {
		key := "test-reset"

		for range config.Capacity {
			result, err := tb.Allow(ctx, key)
			require.NoError(t, err)
			require.True(t, result.Allowed())
		}

		result, err := tb.Allow(ctx, key)
		assert.NoError(t, err)
		assert.False(t, result.Allowed())

		err = tb.Reset(ctx, key)
		assert.NoError(t, err)

		result, err = tb.Status(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, config.Capacity, result.Remaining)
	})

	t.Run("reset non-existent key succeeds", func(t *testing.T) {
		err := tb.Reset(ctx, "non-existent")
		assert.NoError(t, err)
	})
}

func TestBucket_MultipleKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       3,
		RefillRate:     1,
		RefillInterval: 100 * time.Millisecond,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	t.Run("independent rate limits per key", func(t *testing.T) {
		key1 := "user1"
		key2 := "user2"

		for range config.Capacity {
			result, err := tb.Allow(ctx, key1)
			require.NoError(t, err)
			require.True(t, result.Allowed())
		}

		result1, err := tb.Allow(ctx, key1)
		assert.NoError(t, err)
		assert.False(t, result1.Allowed())

		result2, err := tb.Allow(ctx, key2)
		assert.NoError(t, err)
		assert.True(t, result2.Allowed())
		assert.Equal(t, config.Capacity-1, result2.Remaining)
	})
}
