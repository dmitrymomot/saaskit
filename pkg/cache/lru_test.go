package cache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/cache"
)

func TestLRUCache_Basic(t *testing.T) {
	t.Run("put and get", func(t *testing.T) {
		c := cache.NewLRUCache[string, int](3)

		c.Put("a", 1)
		c.Put("b", 2)
		c.Put("c", 3)

		val, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 1, val)

		val, ok = c.Get("b")
		assert.True(t, ok)
		assert.Equal(t, 2, val)

		val, ok = c.Get("c")
		assert.True(t, ok)
		assert.Equal(t, 3, val)

		assert.Equal(t, 3, c.Len())
	})

	t.Run("get non-existent", func(t *testing.T) {
		c := cache.NewLRUCache[string, int](3)

		val, ok := c.Get("missing")
		assert.False(t, ok)
		assert.Equal(t, 0, val)
	})

	t.Run("update existing", func(t *testing.T) {
		c := cache.NewLRUCache[string, int](3)

		c.Put("a", 1)
		oldVal, existed := c.Put("a", 2)

		assert.True(t, existed)
		assert.Equal(t, 1, oldVal)

		val, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 2, val)

		assert.Equal(t, 1, c.Len())
	})
}

func TestLRUCache_Eviction(t *testing.T) {
	t.Run("evict least recently used", func(t *testing.T) {
		c := cache.NewLRUCache[string, int](3)

		// Fill cache to capacity
		c.Put("a", 1)
		c.Put("b", 2)
		c.Put("c", 3)

		// Add one more - should evict "a" (least recently used)
		c.Put("d", 4)

		// "a" should be evicted
		_, ok := c.Get("a")
		assert.False(t, ok, "a should have been evicted")

		// Others should still be present
		val, ok := c.Get("b")
		assert.True(t, ok)
		assert.Equal(t, 2, val)

		val, ok = c.Get("c")
		assert.True(t, ok)
		assert.Equal(t, 3, val)

		val, ok = c.Get("d")
		assert.True(t, ok)
		assert.Equal(t, 4, val)

		assert.Equal(t, 3, c.Len())
	})

	t.Run("get updates recency", func(t *testing.T) {
		c := cache.NewLRUCache[string, int](3)

		c.Put("a", 1)
		c.Put("b", 2)
		c.Put("c", 3)

		// Access "a" to make it recently used
		c.Get("a")

		// Add "d" - should evict "b" (now least recently used)
		c.Put("d", 4)

		// "b" should be evicted
		_, ok := c.Get("b")
		assert.False(t, ok, "b should have been evicted")

		// "a" should still be present (was accessed)
		val, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 1, val)
	})

	t.Run("put updates recency", func(t *testing.T) {
		c := cache.NewLRUCache[string, int](3)

		c.Put("a", 1)
		c.Put("b", 2)
		c.Put("c", 3)

		// Update "a" to make it recently used
		c.Put("a", 10)

		// Add "d" - should evict "b" (now least recently used)
		c.Put("d", 4)

		// "b" should be evicted
		_, ok := c.Get("b")
		assert.False(t, ok, "b should have been evicted")

		// "a" should still be present (was updated)
		val, ok := c.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 10, val)
	})
}

func TestLRUCache_EvictionCallback(t *testing.T) {
	c := cache.NewLRUCache[string, int](2)

	evicted := make(map[string]int)
	c.SetEvictCallback(func(key string, value int) {
		evicted[key] = value
	})

	c.Put("a", 1)
	c.Put("b", 2)

	// Should evict "a"
	c.Put("c", 3)
	assert.Equal(t, 1, evicted["a"], "a should have been evicted with value 1")

	// Should evict "b"
	c.Put("d", 4)
	assert.Equal(t, 2, evicted["b"], "b should have been evicted with value 2")

	// Clear should evict remaining items
	c.Clear()
	assert.Equal(t, 3, evicted["c"], "c should have been evicted with value 3")
	assert.Equal(t, 4, evicted["d"], "d should have been evicted with value 4")
}

func TestLRUCache_Remove(t *testing.T) {
	c := cache.NewLRUCache[string, int](3)

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	// Remove existing
	val, ok := c.Remove("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val)
	assert.Equal(t, 2, c.Len())

	// Verify it's gone
	_, ok = c.Get("b")
	assert.False(t, ok)

	// Remove non-existent
	val, ok = c.Remove("missing")
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestLRUCache_Clear(t *testing.T) {
	c := cache.NewLRUCache[string, int](3)

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	c.Clear()

	assert.Equal(t, 0, c.Len())

	// All items should be gone
	_, ok := c.Get("a")
	assert.False(t, ok)

	_, ok = c.Get("b")
	assert.False(t, ok)

	_, ok = c.Get("c")
	assert.False(t, ok)
}

func TestLRUCache_EdgeCases(t *testing.T) {
	t.Run("capacity of 1", func(t *testing.T) {
		c := cache.NewLRUCache[string, int](1)

		c.Put("a", 1)
		c.Put("b", 2)

		// Only "b" should remain
		_, ok := c.Get("a")
		assert.False(t, ok)

		val, ok := c.Get("b")
		assert.True(t, ok)
		assert.Equal(t, 2, val)
	})

	t.Run("panic on zero capacity", func(t *testing.T) {
		assert.Panics(t, func() {
			cache.NewLRUCache[string, int](0)
		})
	})

	t.Run("panic on negative capacity", func(t *testing.T) {
		assert.Panics(t, func() {
			cache.NewLRUCache[string, int](-1)
		})
	})
}

func TestLRUCache_Concurrent(t *testing.T) {
	c := cache.NewLRUCache[int, int](100)

	// Concurrent puts
	t.Run("concurrent puts", func(t *testing.T) {
		for i := range 100 {
			go func(val int) {
				c.Put(val, val*2)
			}(i)
		}
	})

	// Concurrent gets
	t.Run("concurrent gets", func(t *testing.T) {
		for i := range 100 {
			go func(key int) {
				c.Get(key)
			}(i)
		}
	})

	// Concurrent removes
	t.Run("concurrent removes", func(t *testing.T) {
		for i := range 50 {
			go func(key int) {
				c.Remove(key)
			}(i)
		}
	})
}

func BenchmarkLRUCache_Put(b *testing.B) {
	c := cache.NewLRUCache[int, int](1000)

	b.ResetTimer()
	for i := range b.N {
		c.Put(i%2000, i)
	}
}

func BenchmarkLRUCache_Get(b *testing.B) {
	c := cache.NewLRUCache[int, int](1000)

	// Pre-fill cache
	for i := range 1000 {
		c.Put(i, i)
	}

	b.ResetTimer()
	for i := range b.N {
		c.Get(i % 1000)
	}
}

func BenchmarkLRUCache_Mixed(b *testing.B) {
	c := cache.NewLRUCache[int, int](1000)

	b.ResetTimer()
	for i := range b.N {
		if i%2 == 0 {
			c.Put(i%2000, i)
		} else {
			c.Get(i % 2000)
		}
	}
}
