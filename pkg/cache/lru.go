package cache

import (
	"container/list"
	"sync"
)

type lruEntry[K comparable, V any] struct {
	key   K
	value V
}

// LRUCache is a thread-safe LRU cache implementation.
// When the cache reaches its capacity, the least recently used item is evicted.
type LRUCache[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element
	eviction *list.List
	mu       sync.Mutex
	onEvict  func(key K, value V) // Callback for cleanup when items are evicted
}

// NewLRUCache creates a new LRU cache with the specified capacity.
// The capacity must be positive, otherwise it panics.
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity <= 0 {
		panic("LRU cache capacity must be positive")
	}
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		eviction: list.New(),
	}
}

// SetEvictCallback sets a callback function that is called when items are evicted.
// This is useful for cleanup operations like closing resources.
func (c *LRUCache[K, V]) SetEvictCallback(fn func(key K, value V)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onEvict = fn
}

// Get retrieves a value from the cache and marks it as recently used.
// Returns the value and true if found, zero value and false otherwise.
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.eviction.MoveToFront(elem)
		entry := elem.Value.(*lruEntry[K, V])
		return entry.value, true
	}

	var zero V
	return zero, false
}

// Put adds or updates a value in the cache.
// If the cache is at capacity, the least recently used item is evicted.
// Returns the previous value if it existed, and a boolean indicating if it existed.
func (c *LRUCache[K, V]) Put(key K, value V) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.eviction.MoveToFront(elem)
		entry := elem.Value.(*lruEntry[K, V])
		oldValue := entry.value
		entry.value = value
		return oldValue, true
	}

	entry := &lruEntry[K, V]{key: key, value: value}
	elem := c.eviction.PushFront(entry)
	c.items[key] = elem

	if c.eviction.Len() > c.capacity {
		c.evictOldest()
	}

	var zero V
	return zero, false
}

// Remove removes an item from the cache.
// Returns the removed value and true if it existed, zero value and false otherwise.
func (c *LRUCache[K, V]) Remove(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
		entry := elem.Value.(*lruEntry[K, V])
		return entry.value, true
	}

	var zero V
	return zero, false
}

func (c *LRUCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.eviction.Len()
}

// Clear removes all items from the cache.
// If an evict callback is set, it's called for each item.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.onEvict != nil {
		for _, elem := range c.items {
			entry := elem.Value.(*lruEntry[K, V])
			c.onEvict(entry.key, entry.value)
		}
	}

	c.items = make(map[K]*list.Element)
	c.eviction.Init()
}

// Must be called with lock held.
func (c *LRUCache[K, V]) evictOldest() {
	elem := c.eviction.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// Must be called with lock held.
func (c *LRUCache[K, V]) removeElement(elem *list.Element) {
	c.eviction.Remove(elem)
	entry := elem.Value.(*lruEntry[K, V])
	delete(c.items, entry.key)

	if c.onEvict != nil {
		c.onEvict(entry.key, entry.value)
	}
}
