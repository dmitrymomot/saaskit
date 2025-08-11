package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	t.Parallel()

	t.Run("default configuration", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore()
		assert.NotNil(t, store)
		assert.NotNil(t, store.buckets)
		assert.NotNil(t, store.windows)
		assert.Equal(t, 1*time.Minute, store.cleanupInterval)
		assert.Equal(t, 100, store.initialCapacity)
	})

	t.Run("custom cleanup interval", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore(WithCleanupInterval(5 * time.Second))
		assert.Equal(t, 5*time.Second, store.cleanupInterval)
	})

	t.Run("custom initial capacity", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore(WithInitialCapacity(500))
		assert.Equal(t, 500, store.initialCapacity)
	})

	t.Run("zero cleanup interval ignored", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore(WithCleanupInterval(0))
		assert.Equal(t, 1*time.Minute, store.cleanupInterval)
	})

	t.Run("negative cleanup interval ignored", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore(WithCleanupInterval(-1 * time.Second))
		assert.Equal(t, 1*time.Minute, store.cleanupInterval)
	})

	t.Run("zero initial capacity ignored", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore(WithInitialCapacity(0))
		assert.Equal(t, 100, store.initialCapacity)
	})

	t.Run("negative initial capacity ignored", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore(WithInitialCapacity(-50))
		assert.Equal(t, 100, store.initialCapacity)
	})
}

func TestMemoryStore_IncrementAndGet(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
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

	store := NewMemoryStore()
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

	store := NewMemoryStore()
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

	store := NewMemoryStore()
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

	store := NewMemoryStore()
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

		store.mu.RLock()
		sw := store.windows[key]
		store.mu.RUnlock()

		sw.mu.Lock()
		timestampCount := len(sw.timestamps)
		sw.mu.Unlock()

		assert.Equal(t, 1, timestampCount, "expired timestamps should be removed")
	})
}

func TestMemoryStore_CleanupExpired(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
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

		store.mu.RLock()
		_, exists := store.windows[key]
		store.mu.RUnlock()

		assert.False(t, exists, "empty window should be removed")
	})

	t.Run("cleanup non-existent key", func(t *testing.T) {
		err := store.CleanupExpired(ctx, "non-existent", time.Second)
		assert.NoError(t, err)
	})
}

func TestMemoryStore_Concurrent(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
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
			store.cleanup()
		}()

		wg.Wait()
	})
}

func TestMemoryStore_Cleanup(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore(WithCleanupInterval(50 * time.Millisecond))
	defer store.Close()
	ctx := context.Background()

	t.Run("automatic cleanup of expired buckets", func(t *testing.T) {
		for i := range 10 {
			key := "auto-cleanup-" + string(rune(i))
			_, _, err := store.IncrementAndGet(ctx, key, 1, 25*time.Millisecond)
			require.NoError(t, err)
		}

		store.mu.RLock()
		initialCount := len(store.buckets)
		store.mu.RUnlock()
		assert.Equal(t, 10, initialCount)

		time.Sleep(100 * time.Millisecond)

		store.mu.RLock()
		finalCount := len(store.buckets)
		store.mu.RUnlock()
		assert.Equal(t, 0, finalCount, "expired buckets should be cleaned up")
	})

	t.Run("cleanup preserves non-expired entries", func(t *testing.T) {
		_, _, err := store.IncrementAndGet(ctx, "short-lived", 1, 25*time.Millisecond)
		require.NoError(t, err)
		_, _, err = store.IncrementAndGet(ctx, "long-lived", 1, 5*time.Second)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		store.mu.RLock()
		_, shortExists := store.buckets["short-lived"]
		_, longExists := store.buckets["long-lived"]
		store.mu.RUnlock()

		assert.False(t, shortExists, "short-lived bucket should be cleaned up")
		assert.True(t, longExists, "long-lived bucket should remain")
	})
}

func TestMemoryStore_Close(t *testing.T) {
	t.Parallel()

	t.Run("close stops cleanup goroutine", func(t *testing.T) {
		store := NewMemoryStore(WithCleanupInterval(50 * time.Millisecond))

		err := store.Close()
		assert.NoError(t, err)

		select {
		case <-store.stopCleanup:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("cleanup channel should be closed")
		}
	})

	t.Run("multiple close calls are safe", func(t *testing.T) {
		store := NewMemoryStore()

		err := store.Close()
		assert.NoError(t, err)

		err = store.Close()
		assert.NoError(t, err)
	})
}

func TestMemoryStore_MemoryLeak(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore(WithCleanupInterval(50 * time.Millisecond))
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

	store.mu.RLock()
	bucketCount := len(store.buckets)
	windowCount := len(store.windows)
	store.mu.RUnlock()

	// All entries should be cleaned up since they expired
	assert.Equal(t, 0, bucketCount, "all expired buckets should be cleaned up")
	assert.Equal(t, 0, windowCount, "all expired windows should be cleaned up")

	// Now test that non-expired entries are preserved
	for i := range 5 {
		key := fmt.Sprintf("persistent-%d", i)
		_, _, _ = store.IncrementAndGet(ctx, key, 1, 5*time.Second)
		_ = store.RecordTimestamp(ctx, key, time.Now(), 5*time.Second)
	}

	// Wait for a cleanup cycle
	time.Sleep(60 * time.Millisecond)

	store.mu.RLock()
	bucketCount = len(store.buckets)
	windowCount = len(store.windows)
	store.mu.RUnlock()

	assert.Equal(t, 5, bucketCount, "non-expired buckets should remain")
	assert.Equal(t, 5, windowCount, "non-expired windows should remain")
}
