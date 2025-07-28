package tenant_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/tenant"
)

func TestInMemoryCache(t *testing.T) {
	t.Parallel()

	t.Run("stores and retrieves tenant", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)
		testTenant := createTestTenant("acme", true)

		cache.Set(context.Background(), "key1", testTenant, 1*time.Hour)

		retrieved, ok := cache.Get(context.Background(), "key1")
		require.True(t, ok)
		assert.Equal(t, testTenant, retrieved)
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)

		retrieved, ok := cache.Get(context.Background(), "missing")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("respects TTL expiration", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)
		testTenant := createTestTenant("acme", true)

		// Set with very short TTL
		cache.Set(context.Background(), "expire", testTenant, 10*time.Millisecond)

		// Should exist immediately
		retrieved, ok := cache.Get(context.Background(), "expire")
		require.True(t, ok)
		assert.Equal(t, testTenant, retrieved)

		// Wait for expiration
		time.Sleep(20 * time.Millisecond)

		// Should be expired
		retrieved, ok = cache.Get(context.Background(), "expire")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("overwrites existing entries", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)
		tenant1 := createTestTenant("acme", true)
		tenant2 := createTestTenant("globex", true)

		cache.Set(context.Background(), "key", tenant1, 1*time.Hour)
		cache.Set(context.Background(), "key", tenant2, 1*time.Hour)

		retrieved, ok := cache.Get(context.Background(), "key")
		require.True(t, ok)
		assert.Equal(t, tenant2, retrieved)
	})

	t.Run("deletes entries", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)
		testTenant := createTestTenant("acme", true)

		cache.Set(context.Background(), "delete", testTenant, 1*time.Hour)

		// Verify it exists
		_, ok := cache.Get(context.Background(), "delete")
		require.True(t, ok)

		// Delete it
		cache.Delete(context.Background(), "delete")

		// Verify it's gone
		_, ok = cache.Get(context.Background(), "delete")
		assert.False(t, ok)
	})

	t.Run("handles concurrent access", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)
		var wg sync.WaitGroup
		iterations := 100

		// Concurrent writes
		for i := range iterations {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				testTenant := createTestTenant(fmt.Sprintf("tenant%d", i), true)
				cache.Set(context.Background(), "concurrent", testTenant, 1*time.Hour)
			}(i)
		}

		// Concurrent reads
		for range iterations {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cache.Get(context.Background(), "concurrent")
			}()
		}

		// Concurrent deletes
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cache.Delete(context.Background(), "concurrent")
			}()
		}

		wg.Wait()
		// Test should complete without race conditions
	})

	t.Run("cleanup removes expired entries", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)

		// Add entries with different TTLs
		cache.Set(context.Background(), "short", createTestTenant("short", true), 50*time.Millisecond)
		cache.Set(context.Background(), "long", createTestTenant("long", true), 1*time.Hour)

		// Wait for short TTL to expire
		time.Sleep(100 * time.Millisecond)

		// Short should be gone after expiration
		_, ok := cache.Get(context.Background(), "short")
		assert.False(t, ok)

		// Long should still exist
		_, ok = cache.Get(context.Background(), "long")
		assert.True(t, ok)
	})
}

func TestNoOpCache(t *testing.T) {
	t.Parallel()

	t.Run("always returns cache miss", func(t *testing.T) {
		t.Parallel()

		cache := tenant.NewNoOpCache()
		testTenant := createTestTenant("acme", true)

		// Set should be no-op
		cache.Set(context.Background(), "key", testTenant, 1*time.Hour)

		// Get should return false
		retrieved, ok := cache.Get(context.Background(), "key")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("delete is no-op", func(t *testing.T) {
		t.Parallel()

		cache := tenant.NewNoOpCache()

		// Should not panic
		cache.Delete(context.Background(), "any")
	})
}

func TestCache_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles zero TTL", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)
		testTenant := createTestTenant("acme", true)

		// Set with zero TTL (should expire immediately)
		cache.Set(context.Background(), "zero", testTenant, 0)

		// Should be expired
		retrieved, ok := cache.Get(context.Background(), "zero")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("handles negative TTL", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)
		testTenant := createTestTenant("acme", true)

		// Set with negative TTL
		cache.Set(context.Background(), "negative", testTenant, -1*time.Hour)

		// Should be expired
		retrieved, ok := cache.Get(context.Background(), "negative")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("handles empty keys", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)
		testTenant := createTestTenant("acme", true)

		// Empty key should work
		cache.Set(context.Background(), "", testTenant, 1*time.Hour)

		retrieved, ok := cache.Get(context.Background(), "")
		require.True(t, ok)
		assert.Equal(t, testTenant, retrieved)

		cache.Delete(context.Background(), "")
		_, ok = cache.Get(context.Background(), "")
		assert.False(t, ok)
	})
}

// getGoroutineCount returns the current number of goroutines
func getGoroutineCount() int {
	return runtime.NumGoroutine()
}

func TestInMemoryCache_SizeLimits(t *testing.T) {
	t.Parallel()

	t.Run("enforces maximum size", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCacheWithSize(ctx, 3)

		// Add 3 items (at capacity)
		cache.Set(context.Background(), "tenant1", createTestTenant("tenant1", true), 1*time.Hour)
		cache.Set(context.Background(), "tenant2", createTestTenant("tenant2", true), 1*time.Hour)
		cache.Set(context.Background(), "tenant3", createTestTenant("tenant3", true), 1*time.Hour)

		// Verify all exist
		_, ok1 := cache.Get(context.Background(), "tenant1")
		_, ok2 := cache.Get(context.Background(), "tenant2")
		_, ok3 := cache.Get(context.Background(), "tenant3")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.True(t, ok3)

		// Add 4th item, should evict tenant1 (LRU)
		cache.Set(context.Background(), "tenant4", createTestTenant("tenant4", true), 1*time.Hour)

		// tenant1 should be evicted, others should exist
		_, ok1 = cache.Get(context.Background(), "tenant1")
		_, ok2 = cache.Get(context.Background(), "tenant2")
		_, ok3 = cache.Get(context.Background(), "tenant3")
		_, ok4 := cache.Get(context.Background(), "tenant4")
		assert.False(t, ok1, "tenant1 should have been evicted")
		assert.True(t, ok2)
		assert.True(t, ok3)
		assert.True(t, ok4)
	})

	t.Run("LRU eviction works correctly", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCacheWithSize(ctx, 3)

		// Add 3 items
		cache.Set(context.Background(), "tenant1", createTestTenant("tenant1", true), 1*time.Hour)
		cache.Set(context.Background(), "tenant2", createTestTenant("tenant2", true), 1*time.Hour)
		cache.Set(context.Background(), "tenant3", createTestTenant("tenant3", true), 1*time.Hour)

		// Access tenant1 and tenant2 to make them more recently used
		cache.Get(context.Background(), "tenant1")
		cache.Get(context.Background(), "tenant2")

		// Add tenant4, should evict tenant3 (least recently used)
		cache.Set(context.Background(), "tenant4", createTestTenant("tenant4", true), 1*time.Hour)

		// tenant3 should be evicted
		_, ok1 := cache.Get(context.Background(), "tenant1")
		_, ok2 := cache.Get(context.Background(), "tenant2")
		_, ok3 := cache.Get(context.Background(), "tenant3")
		_, ok4 := cache.Get(context.Background(), "tenant4")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.False(t, ok3, "tenant3 should have been evicted as LRU")
		assert.True(t, ok4)
	})

	t.Run("updating existing item doesn't trigger eviction", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCacheWithSize(ctx, 2)

		// Add 2 items (at capacity)
		cache.Set(context.Background(), "tenant1", createTestTenant("tenant1", true), 1*time.Hour)
		cache.Set(context.Background(), "tenant2", createTestTenant("tenant2", true), 1*time.Hour)

		// Update tenant1
		cache.Set(context.Background(), "tenant1", createTestTenant("tenant1-updated", true), 1*time.Hour)

		// Both should still exist
		tenant1, ok1 := cache.Get(context.Background(), "tenant1")
		_, ok2 := cache.Get(context.Background(), "tenant2")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, "tenant1-updated", tenant1.Subdomain)
	})
}

// TestInMemoryCache_Internal tests the unexported inMemoryCache type directly
func TestInMemoryCache_Internal(t *testing.T) {
	t.Parallel()

	t.Run("context cancellation stops cleanup goroutine", func(t *testing.T) {
		t.Parallel()

		// Context cancellation should stop cleanup goroutine gracefully
		ctx, cancel := context.WithCancel(context.Background())
		cache := tenant.NewInMemoryCache(ctx)

		// Add an item to verify cache is working
		cache.Set(context.Background(), "test", createTestTenant("test", true), 1*time.Hour)
		_, ok := cache.Get(context.Background(), "test")
		assert.True(t, ok)

		// Cancel context - this should stop the cleanup goroutine
		cancel()

		// Cache should still work for basic operations even after context cancellation
		cache.Set(context.Background(), "test2", createTestTenant("test2", true), 1*time.Hour)
		_, ok = cache.Get(context.Background(), "test2")
		assert.True(t, ok)
	})

	t.Run("handles nil tenant gracefully", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cache := tenant.NewInMemoryCache(ctx)

		// Setting nil tenant should work
		cache.Set(context.Background(), "nil-tenant", nil, 1*time.Hour)

		retrieved, ok := cache.Get(context.Background(), "nil-tenant")
		require.True(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("cleanup goroutine terminates on context cancellation", func(t *testing.T) {
		// Cannot run in parallel due to goroutine count checks

		// Create multiple caches to verify they clean up properly
		type cacheWithCancel struct {
			cache  tenant.Cache
			cancel context.CancelFunc
		}

		caches := make([]cacheWithCancel, 5)
		for i := range caches {
			ctx, cancel := context.WithCancel(context.Background())
			cache := tenant.NewInMemoryCache(ctx)
			caches[i] = cacheWithCancel{cache: cache, cancel: cancel}

			testTenant := createTestTenant("test", true)
			cache.Set(context.Background(), "key", testTenant, 100*time.Millisecond)
		}

		// Record goroutine count with caches running
		beforeClose := getGoroutineCount()

		// Cancel all cache contexts
		for _, cacheInfo := range caches {
			cacheInfo.cancel()
		}

		// Give some time for goroutines to finish
		time.Sleep(100 * time.Millisecond)

		// Goroutine count should decrease by at least the number of caches
		afterClose := getGoroutineCount()
		assert.Less(t, afterClose, beforeClose,
			"goroutine leak detected: before=%d, after=%d", beforeClose, afterClose)
	})
}

// Benchmark cache operations
func BenchmarkCache_Set(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache := tenant.NewInMemoryCache(ctx)
	testTenant := createTestTenant("bench", true)
	testCtx := context.Background()

	b.ResetTimer()
	for i := range b.N {
		cache.Set(testCtx, fmt.Sprintf("key%d", i%100), testTenant, 1*time.Hour)
	}
}

func BenchmarkCache_Get(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache := tenant.NewInMemoryCache(ctx)
	testTenant := createTestTenant("bench", true)
	testCtx := context.Background()

	// Pre-populate cache
	for i := range 100 {
		cache.Set(testCtx, fmt.Sprintf("key%d", i), testTenant, 1*time.Hour)
	}

	b.ResetTimer()
	for i := range b.N {
		cache.Get(testCtx, fmt.Sprintf("key%d", i%100))
	}
}

func BenchmarkCache_ConcurrentAccess(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache := tenant.NewInMemoryCache(ctx)
	testTenant := createTestTenant("bench", true)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%100)
			if i%2 == 0 {
				cache.Set(ctx, key, testTenant, 1*time.Hour)
			} else {
				cache.Get(ctx, key)
			}
			i++
		}
	})
}
