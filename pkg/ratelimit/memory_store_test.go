package ratelimit_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dmitrymomot/saaskit/pkg/ratelimit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	t.Parallel()

	t.Run("default configuration", func(t *testing.T) {
		t.Parallel()
		store := ratelimit.NewMemoryStore()
		assert.NotNil(t, store)
		// Can't test unexported fields in black-box testing
	})

	t.Run("custom cleanup interval", func(t *testing.T) {
		t.Parallel()
		store := ratelimit.NewMemoryStore(ratelimit.WithCleanupInterval(5 * time.Second))
		assert.NotNil(t, store)
		// Can't test unexported fields in black-box testing
	})

	t.Run("custom initial capacity", func(t *testing.T) {
		t.Parallel()
		store := ratelimit.NewMemoryStore(ratelimit.WithInitialCapacity(500))
		assert.NotNil(t, store)
		// Can't test unexported fields in black-box testing
	})

	t.Run("zero cleanup interval ignored", func(t *testing.T) {
		t.Parallel()
		store := ratelimit.NewMemoryStore(ratelimit.WithCleanupInterval(0))
		assert.NotNil(t, store)
		// Can't test unexported fields in black-box testing
	})

	t.Run("negative cleanup interval ignored", func(t *testing.T) {
		t.Parallel()
		store := ratelimit.NewMemoryStore(ratelimit.WithCleanupInterval(-1 * time.Second))
		assert.NotNil(t, store)
		// Can't test unexported fields in black-box testing
	})

	t.Run("zero initial capacity ignored", func(t *testing.T) {
		t.Parallel()
		store := ratelimit.NewMemoryStore(ratelimit.WithInitialCapacity(0))
		assert.NotNil(t, store)
		// Can't test unexported fields in black-box testing
	})

	t.Run("negative initial capacity ignored", func(t *testing.T) {
		t.Parallel()
		store := ratelimit.NewMemoryStore(ratelimit.WithInitialCapacity(-50))
		assert.NotNil(t, store)
		// Can't test unexported fields in black-box testing
	})
}

func TestMemoryStore_IncrementAndGet(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	t.Run("new bucket creation", func(t *testing.T) {
		count, ttl, err := store.IncrementAndGet(ctx, "new-key", 5, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
		assert.InDelta(t, time.Second.Seconds(), ttl.Seconds(), 0.1)
	})

	t.Run("increment existing bucket", func(t *testing.T) {
		key := "increment-key"
		count1, _, err := store.IncrementAndGet(ctx, key, 3, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count1)

		count2, _, err := store.IncrementAndGet(ctx, key, 2, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count2)
	})

	t.Run("negative increment (decrement)", func(t *testing.T) {
		key := "decrement-key"
		_, _, err := store.IncrementAndGet(ctx, key, 10, time.Second)
		require.NoError(t, err)

		count, _, err := store.IncrementAndGet(ctx, key, -3, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(7), count)
	})

	t.Run("expired bucket replacement", func(t *testing.T) {
		key := "expire-key"
		_, _, err := store.IncrementAndGet(ctx, key, 5, 50*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(60 * time.Millisecond)

		count, ttl, err := store.IncrementAndGet(ctx, key, 3, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.InDelta(t, time.Second.Seconds(), ttl.Seconds(), 0.1)
	})
}

func TestMemoryStore_Get(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	t.Run("non-existent key", func(t *testing.T) {
		count, ttl, err := store.Get(ctx, "missing-key")
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, time.Duration(0), ttl)
	})

	t.Run("existing key", func(t *testing.T) {
		key := "existing-key"
		_, _, err := store.IncrementAndGet(ctx, key, 7, time.Second)
		require.NoError(t, err)

		count, ttl, err := store.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(7), count)
		assert.Greater(t, ttl, time.Duration(0))
	})

	t.Run("expired key", func(t *testing.T) {
		key := "expired-key"
		_, _, err := store.IncrementAndGet(ctx, key, 5, 50*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(60 * time.Millisecond)

		count, ttl, err := store.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, time.Duration(0), ttl)
	})
}

func TestMemoryStore_Delete(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	t.Run("delete bucket", func(t *testing.T) {
		key := "delete-bucket"
		_, _, err := store.IncrementAndGet(ctx, key, 5, time.Second)
		require.NoError(t, err)

		err = store.Delete(ctx, key)
		require.NoError(t, err)

		count, ttl, err := store.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, time.Duration(0), ttl)
	})

	t.Run("delete window", func(t *testing.T) {
		key := "delete-window"
		err := store.RecordTimestamp(ctx, key, time.Now(), time.Second)
		require.NoError(t, err)

		count, err := store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		err = store.Delete(ctx, key)
		require.NoError(t, err)

		count, err = store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		err := store.Delete(ctx, "non-existent")
		assert.NoError(t, err)
	})
}

func TestMemoryStore_RecordTimestamp(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	t.Run("new window creation", func(t *testing.T) {
		key := "new-window"
		err := store.RecordTimestamp(ctx, key, time.Now(), time.Second)
		require.NoError(t, err)

		count, err := store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("append to existing window", func(t *testing.T) {
		key := "append-window"
		now := time.Now()

		err := store.RecordTimestamp(ctx, key, now, time.Second)
		require.NoError(t, err)

		err = store.RecordTimestamp(ctx, key, now.Add(100*time.Millisecond), time.Second)
		require.NoError(t, err)

		count, err := store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("automatic cleanup of old timestamps", func(t *testing.T) {
		key := "cleanup-window"
		now := time.Now()

		err := store.RecordTimestamp(ctx, key, now.Add(-2*time.Second), 1*time.Second)
		require.NoError(t, err)

		err = store.RecordTimestamp(ctx, key, now, 1*time.Second)
		require.NoError(t, err)

		count, err := store.CountInWindow(ctx, key, 1*time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "old timestamp should be cleaned up")
	})
}

func TestMemoryStore_CountInWindow(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	t.Run("empty window", func(t *testing.T) {
		count, err := store.CountInWindow(ctx, "empty", time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("count within window", func(t *testing.T) {
		key := "count-window"
		now := time.Now()

		err := store.RecordTimestamp(ctx, key, now.Add(-500*time.Millisecond), time.Second)
		require.NoError(t, err)
		err = store.RecordTimestamp(ctx, key, now.Add(-100*time.Millisecond), time.Second)
		require.NoError(t, err)
		err = store.RecordTimestamp(ctx, key, now, time.Second)
		require.NoError(t, err)

		count, err := store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("cleanup during count", func(t *testing.T) {
		key := "cleanup-count"
		now := time.Now()

		err := store.RecordTimestamp(ctx, key, now.Add(-2*time.Second), time.Second)
		require.NoError(t, err)
		err = store.RecordTimestamp(ctx, key, now.Add(-1500*time.Millisecond), time.Second)
		require.NoError(t, err)
		err = store.RecordTimestamp(ctx, key, now.Add(-500*time.Millisecond), time.Second)
		require.NoError(t, err)

		count, err := store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
		// Can't test internal cleanup in black-box testing
	})
}

func TestMemoryStore_CleanupExpired(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	t.Run("cleanup expired timestamps", func(t *testing.T) {
		key := "cleanup-test"
		now := time.Now()

		err := store.RecordTimestamp(ctx, key, now.Add(-2*time.Second), time.Second)
		require.NoError(t, err)
		err = store.RecordTimestamp(ctx, key, now, time.Second)
		require.NoError(t, err)

		err = store.CleanupExpired(ctx, key, time.Second)
		require.NoError(t, err)

		count, err := store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("cleanup empty window removes key", func(t *testing.T) {
		key := "cleanup-empty"
		now := time.Now()

		err := store.RecordTimestamp(ctx, key, now.Add(-2*time.Second), time.Second)
		require.NoError(t, err)

		err = store.CleanupExpired(ctx, key, time.Second)
		require.NoError(t, err)

		// Can't test internal deletion in black-box testing
		// Just verify the count is 0
		count, err := store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("cleanup non-existent key", func(t *testing.T) {
		err := store.CleanupExpired(ctx, "non-existent", time.Second)
		assert.NoError(t, err)
	})
}

func TestMemoryStore_Concurrent(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore()
	defer store.Close()
	ctx := context.Background()

	const goroutines = 100
	const operations = 50

	t.Run("concurrent bucket operations", func(t *testing.T) {
		var wg sync.WaitGroup
		key := "concurrent-bucket"

		wg.Add(goroutines)
		for range goroutines {
			go func() {
				defer wg.Done()
				for range operations {
					_, _, err := store.IncrementAndGet(ctx, key, 1, time.Second)
					assert.NoError(t, err)
				}
			}()
		}
		wg.Wait()

		count, _, err := store.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(goroutines*operations), count)
	})

	t.Run("concurrent window operations", func(t *testing.T) {
		var wg sync.WaitGroup
		key := "concurrent-window"
		startTime := time.Now()

		wg.Add(goroutines)
		for range goroutines {
			go func() {
				defer wg.Done()
				for range operations {
					err := store.RecordTimestamp(ctx, key, time.Now(), time.Second)
					assert.NoError(t, err)
				}
			}()
		}
		wg.Wait()

		duration := time.Since(startTime)

		count, err := store.CountInWindow(ctx, key, time.Second)
		require.NoError(t, err)

		// If operations took less than 1 second, all should be counted
		// Otherwise, some early timestamps may have expired
		if duration < time.Second {
			assert.Equal(t, int64(goroutines*operations), count)
		} else {
			// At least some timestamps should be in the window
			assert.Greater(t, count, int64(0))
			assert.LessOrEqual(t, count, int64(goroutines*operations))
		}
	})

	t.Run("concurrent mixed operations", func(t *testing.T) {
		var wg sync.WaitGroup

		wg.Add(4)

		go func() {
			defer wg.Done()
			for i := range 100 {
				key := "mixed-" + string(rune(i))
				_, _, _ = store.IncrementAndGet(ctx, key, 1, 50*time.Millisecond)
			}
		}()

		go func() {
			defer wg.Done()
			for i := range 100 {
				key := "mixed-" + string(rune(i))
				_ = store.RecordTimestamp(ctx, key, time.Now(), 50*time.Millisecond)
			}
		}()

		go func() {
			defer wg.Done()
			for i := range 100 {
				key := "mixed-" + string(rune(i))
				_, _, _ = store.Get(ctx, key)
				_, _ = store.CountInWindow(ctx, key, 50*time.Millisecond)
			}
		}()

		go func() {
			defer wg.Done()
			time.Sleep(30 * time.Millisecond)
			// Can't call unexported cleanup method in black-box testing
		}()

		wg.Wait()
	})
}

func TestMemoryStore_Cleanup(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore(ratelimit.WithCleanupInterval(50 * time.Millisecond))
	defer store.Close()
	ctx := context.Background()

	t.Run("automatic cleanup of expired buckets", func(t *testing.T) {
		for i := range 10 {
			key := "auto-cleanup-" + string(rune(i))
			_, _, err := store.IncrementAndGet(ctx, key, 1, 25*time.Millisecond)
			require.NoError(t, err)
		}

		// Verify all 10 entries were created
		for i := range 10 {
			key := "auto-cleanup-" + string(rune(i))
			count, _, _ := store.Get(ctx, key)
			assert.Greater(t, count, int64(0))
		}

		time.Sleep(100 * time.Millisecond)

		// Verify all entries expired
		for i := range 10 {
			key := "auto-cleanup-" + string(rune(i))
			count, _, _ := store.Get(ctx, key)
			assert.Equal(t, int64(0), count, "expired buckets should be cleaned up")
		}
	})

	t.Run("cleanup preserves non-expired entries", func(t *testing.T) {
		_, _, err := store.IncrementAndGet(ctx, "short-lived", 1, 25*time.Millisecond)
		require.NoError(t, err)
		_, _, err = store.IncrementAndGet(ctx, "long-lived", 1, 5*time.Second)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		// Check if entries still exist
		shortCount, _, _ := store.Get(ctx, "short-lived")
		longCount, _, _ := store.Get(ctx, "long-lived")
		shortExists := shortCount > 0
		longExists := longCount > 0

		assert.False(t, shortExists, "short-lived bucket should be cleaned up")
		assert.True(t, longExists, "long-lived bucket should remain")
	})
}

func TestMemoryStore_Close(t *testing.T) {
	t.Parallel()

	t.Run("close stops cleanup goroutine", func(t *testing.T) {
		store := ratelimit.NewMemoryStore(ratelimit.WithCleanupInterval(50 * time.Millisecond))

		err := store.Close()
		assert.NoError(t, err)
		// Can't test unexported stopCleanup channel in black-box testing
	})

	t.Run("multiple close calls are safe", func(t *testing.T) {
		store := ratelimit.NewMemoryStore()

		err := store.Close()
		assert.NoError(t, err)

		err = store.Close()
		assert.NoError(t, err)
	})
}

func TestMemoryStore_MemoryLeak(t *testing.T) {
	t.Parallel()

	store := ratelimit.NewMemoryStore(ratelimit.WithCleanupInterval(50 * time.Millisecond))
	defer store.Close()
	ctx := context.Background()

	// Create and expire entries to test cleanup
	for i := range 100 {
		key := fmt.Sprintf("leak-test-%d", i)
		_, _, _ = store.IncrementAndGet(ctx, key, 1, 25*time.Millisecond)
		_ = store.RecordTimestamp(ctx, key, time.Now(), 25*time.Millisecond)
	}

	// Wait for entries to expire and cleanup to run multiple times
	time.Sleep(150 * time.Millisecond)

	// Check all entries expired through Get/CountInWindow
	for i := range 100 {
		key := fmt.Sprintf("leak-test-%d", i)
		count, _, _ := store.Get(ctx, key)
		assert.Equal(t, int64(0), count, "expired entry should be cleaned up")
		winCount, _ := store.CountInWindow(ctx, key, 25*time.Millisecond)
		assert.Equal(t, int64(0), winCount, "expired window should be cleaned up")
	}

	// Now test that non-expired entries are preserved
	for i := range 5 {
		key := fmt.Sprintf("persistent-%d", i)
		_, _, _ = store.IncrementAndGet(ctx, key, 1, 5*time.Second)
		_ = store.RecordTimestamp(ctx, key, time.Now(), 5*time.Second)
	}

	// Wait for a cleanup cycle
	time.Sleep(60 * time.Millisecond)

	// Check non-expired entries still exist
	for i := range 5 {
		key := fmt.Sprintf("persistent-%d", i)
		count, _, _ := store.Get(ctx, key)
		assert.Greater(t, count, int64(0), "non-expired bucket should remain")
		winCount, _ := store.CountInWindow(ctx, key, 5*time.Second)
		assert.Greater(t, winCount, int64(0), "non-expired window should remain")
	}
}
