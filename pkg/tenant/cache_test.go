package tenant_test

import (
	"context"
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

		cache := tenant.NewInMemoryCache()
		testTenant := createTestTenant("acme", true)

		cache.Set(context.Background(), "key1", testTenant, 1*time.Hour)

		retrieved, ok := cache.Get(context.Background(), "key1")
		require.True(t, ok)
		assert.Equal(t, testTenant, retrieved)
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		t.Parallel()

		cache := tenant.NewInMemoryCache()

		retrieved, ok := cache.Get(context.Background(), "missing")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("respects TTL expiration", func(t *testing.T) {
		t.Parallel()

		cache := tenant.NewInMemoryCache()
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

		cache := tenant.NewInMemoryCache()
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

		cache := tenant.NewInMemoryCache()
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

		cache := tenant.NewInMemoryCache()
		var wg sync.WaitGroup
		iterations := 100

		// Concurrent writes
		for i := range iterations {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				testTenant := createTestTenant("tenant"+string(rune(i)), true)
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

		cache := tenant.NewInMemoryCache()

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

		cache := tenant.NewInMemoryCache()
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

		cache := tenant.NewInMemoryCache()
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

		cache := tenant.NewInMemoryCache()
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

// TestInMemoryCache_Internal tests the unexported inMemoryCache type directly
func TestInMemoryCache_Internal(t *testing.T) {
	t.Parallel()

	t.Run("close stops cleanup goroutine", func(t *testing.T) {
		t.Parallel()

		// Type assertion to access Close method
		cache := tenant.NewInMemoryCache()
		if closeable, ok := cache.(interface{ Close() }); ok {
			// Should not panic
			closeable.Close()
		}
	})
}

// Benchmark cache operations
func BenchmarkCache_Set(b *testing.B) {
	cache := tenant.NewInMemoryCache()
	testTenant := createTestTenant("bench", true)
	ctx := context.Background()

	b.ResetTimer()
	for i := range b.N {
		cache.Set(ctx, "key"+string(rune(i%100)), testTenant, 1*time.Hour)
	}
}

func BenchmarkCache_Get(b *testing.B) {
	cache := tenant.NewInMemoryCache()
	testTenant := createTestTenant("bench", true)
	ctx := context.Background()

	// Pre-populate cache
	for i := range 100 {
		cache.Set(ctx, "key"+string(rune(i)), testTenant, 1*time.Hour)
	}

	b.ResetTimer()
	for i := range b.N {
		cache.Get(ctx, "key"+string(rune(i%100)))
	}
}

func BenchmarkCache_ConcurrentAccess(b *testing.B) {
	cache := tenant.NewInMemoryCache()
	testTenant := createTestTenant("bench", true)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key" + string(rune(i%100))
			if i%2 == 0 {
				cache.Set(ctx, key, testTenant, 1*time.Hour)
			} else {
				cache.Get(ctx, key)
			}
			i++
		}
	})
}
