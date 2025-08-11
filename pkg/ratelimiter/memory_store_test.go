package ratelimiter_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/ratelimiter"
)

func TestMemoryStore_ConsumeTokens(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       10,
		RefillRate:     2,
		RefillInterval: 100 * time.Millisecond,
	}

	t.Run("creates new bucket with full capacity", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		remaining, resetAt, err := store.ConsumeTokens(ctx, "new-key", 3, config)
		assert.NoError(t, err)
		assert.Equal(t, 7, remaining)
		assert.NotZero(t, resetAt)
	})

	t.Run("consumes tokens correctly", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		key := "test-consume"

		remaining, _, err := store.ConsumeTokens(ctx, key, 4, config)
		assert.NoError(t, err)
		assert.Equal(t, 6, remaining)

		remaining, _, err = store.ConsumeTokens(ctx, key, 3, config)
		assert.NoError(t, err)
		assert.Equal(t, 3, remaining)

		remaining, _, err = store.ConsumeTokens(ctx, key, 5, config)
		assert.NoError(t, err)
		assert.Equal(t, -2, remaining)
	})

	t.Run("refills tokens over time", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		key := "test-refill"

		remaining, _, err := store.ConsumeTokens(ctx, key, config.Capacity, config)
		assert.NoError(t, err)
		assert.Equal(t, 0, remaining)

		time.Sleep(config.RefillInterval + 10*time.Millisecond)

		remaining, _, err = store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		assert.Equal(t, config.RefillRate, remaining)

		time.Sleep(config.RefillInterval)

		remaining, _, err = store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		assert.Equal(t, config.RefillRate*2, remaining)
	})

	t.Run("caps tokens at capacity", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		key := "test-cap"

		_, _, err := store.ConsumeTokens(ctx, key, 5, config)
		require.NoError(t, err)

		time.Sleep(config.RefillInterval * 10)

		remaining, _, err := store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		assert.Equal(t, config.Capacity, remaining)
	})

	t.Run("handles zero token consumption", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		key := "test-zero"

		remaining1, _, err := store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		assert.Equal(t, config.Capacity, remaining1)

		remaining2, _, err := store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		assert.Equal(t, remaining1, remaining2)
	})

	t.Run("handles negative remaining correctly", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		key := "test-negative"

		remaining, _, err := store.ConsumeTokens(ctx, key, config.Capacity+5, config)
		assert.NoError(t, err)
		assert.Equal(t, -5, remaining)

		time.Sleep(config.RefillInterval + 10*time.Millisecond)

		remaining, _, err = store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		assert.Equal(t, -5+config.RefillRate, remaining)
	})
}

func TestMemoryStore_Reset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       10,
		RefillRate:     1,
		RefillInterval: 100 * time.Millisecond,
	}

	t.Run("resets existing bucket", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		key := "test-reset"

		_, _, err := store.ConsumeTokens(ctx, key, 8, config)
		require.NoError(t, err)

		err = store.Reset(ctx, key)
		assert.NoError(t, err)

		remaining, _, err := store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		assert.Equal(t, config.Capacity, remaining)
	})

	t.Run("reset non-existent key succeeds", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		err := store.Reset(ctx, "non-existent")
		assert.NoError(t, err)
	})
}

func TestMemoryStore_WithCleanupInterval(t *testing.T) {
	t.Parallel()

	t.Run("custom cleanup interval", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore(
			ratelimiter.WithCleanupInterval(50 * time.Millisecond),
		)
		defer store.Close()

		ctx := context.Background()
		config := ratelimiter.Config{
			Capacity:       10,
			RefillRate:     1,
			RefillInterval: 10 * time.Millisecond,
		}

		_, _, err := store.ConsumeTokens(ctx, "temp-key", 1, config)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("disabled cleanup with zero interval", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore(
			ratelimiter.WithCleanupInterval(0),
		)
		defer store.Close()

		ctx := context.Background()
		config := ratelimiter.Config{
			Capacity:       10,
			RefillRate:     1,
			RefillInterval: 10 * time.Millisecond,
		}

		_, _, err := store.ConsumeTokens(ctx, "no-cleanup", 1, config)
		assert.NoError(t, err)
	})
}

func TestMemoryStore_Close(t *testing.T) {
	t.Parallel()

	t.Run("close stops cleanup", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore(
			ratelimiter.WithCleanupInterval(50 * time.Millisecond),
		)

		store.Close()
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("multiple close calls are safe", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()

		store.Close()
		store.Close()
		store.Close()
	})

	t.Run("operations work after close", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		store.Close()

		ctx := context.Background()
		config := ratelimiter.Config{
			Capacity:       10,
			RefillRate:     1,
			RefillInterval: 100 * time.Millisecond,
		}

		remaining, _, err := store.ConsumeTokens(ctx, "after-close", 1, config)
		assert.NoError(t, err)
		assert.Equal(t, 9, remaining)
	})
}

func TestMemoryStore_IntegerOverflowPrevention(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("prevents overflow with large refill calculations", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       1000,
			RefillRate:     100,
			RefillInterval: time.Millisecond,
		}

		key := "overflow-test"

		_, _, err := store.ConsumeTokens(ctx, key, config.Capacity, config)
		require.NoError(t, err)

		// Sleep for 100ms to simulate many refill intervals passing
		time.Sleep(100 * time.Millisecond)

		remaining, _, err := store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		// Should be capped at capacity, not overflowed
		assert.Equal(t, config.Capacity, remaining)
	})

	t.Run("handles max int values", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		config := ratelimiter.Config{
			Capacity:       1<<31 - 1,
			RefillRate:     1000,
			RefillInterval: time.Millisecond,
		}

		key := "max-int"

		remaining, _, err := store.ConsumeTokens(ctx, key, 1, config)
		assert.NoError(t, err)
		assert.Equal(t, config.Capacity-1, remaining)
	})
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       100,
		RefillRate:     10,
		RefillInterval: 100 * time.Millisecond,
	}

	t.Run("concurrent consumption same key", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		key := "concurrent-same"
		goroutines := 10
		tokensPerGoroutine := 5

		var wg sync.WaitGroup
		wg.Add(goroutines)

		results := make([]int, goroutines)

		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				remaining, _, err := store.ConsumeTokens(ctx, key, tokensPerGoroutine, config)
				if err == nil {
					results[idx] = remaining
				}
			}(i)
		}

		wg.Wait()

		finalRemaining, _, err := store.ConsumeTokens(ctx, key, 0, config)
		assert.NoError(t, err)
		assert.Equal(t, config.Capacity-(goroutines*tokensPerGoroutine), finalRemaining)
	})

	t.Run("concurrent different keys", func(t *testing.T) {
		store := ratelimiter.NewMemoryStore()
		defer store.Close()

		goroutines := 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := range goroutines {
			go func(idx int) {
				defer wg.Done()
				key := "key-" + string(rune('a'+idx))

				for j := range 5 {
					_, _, err := store.ConsumeTokens(ctx, key, j+1, config)
					assert.NoError(t, err)
				}

				if idx%2 == 0 {
					err := store.Reset(ctx, key)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()
	})
}
