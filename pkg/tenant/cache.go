package tenant

import (
	"context"
	"sync"
	"time"
)

// Cache is the interface for tenant caching implementations.
type Cache interface {
	// Get retrieves a tenant from cache by key.
	Get(ctx context.Context, key string) (*Tenant, bool)

	// Set stores a tenant in cache with the given TTL.
	Set(ctx context.Context, key string, tenant *Tenant, ttl time.Duration)

	// Delete removes a tenant from cache.
	Delete(ctx context.Context, key string)

	// Close releases any resources held by the cache.
	Close() error
}

// inMemoryCache is the default in-memory cache implementation.
type inMemoryCache struct {
	mu      sync.RWMutex
	items   map[string]cacheItem
	lru     []string // LRU queue for eviction
	maxSize int      // Maximum number of items
	stop    chan struct{}
	done    chan struct{}
	closed  bool
}

type cacheItem struct {
	tenant    *Tenant
	expiresAt time.Time
}

// DefaultCacheSize is the default maximum number of items in the cache.
const DefaultCacheSize = 1000

// NewInMemoryCache creates a new in-memory cache with automatic cleanup.
func NewInMemoryCache() Cache {
	return NewInMemoryCacheWithSize(DefaultCacheSize)
}

// NewInMemoryCacheWithSize creates a new in-memory cache with specified size limit.
func NewInMemoryCacheWithSize(maxSize int) Cache {
	if maxSize <= 0 {
		maxSize = DefaultCacheSize
	}

	cache := &inMemoryCache{
		items:   make(map[string]cacheItem),
		lru:     make([]string, 0, maxSize),
		maxSize: maxSize,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a tenant from cache.
func (c *inMemoryCache) Get(ctx context.Context, key string) (*Tenant, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		delete(c.items, key)
		c.removeLRU(key)
		return nil, false
	}

	// Update LRU order
	c.updateLRU(key)

	return item.tenant, true
}

// Set stores a tenant in cache.
func (c *inMemoryCache) Set(ctx context.Context, key string, tenant *Tenant, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict
	if _, exists := c.items[key]; !exists && len(c.items) >= c.maxSize {
		// Evict least recently used item
		if len(c.lru) > 0 {
			evictKey := c.lru[0]
			delete(c.items, evictKey)
			c.lru = c.lru[1:]
		}
	}

	c.items[key] = cacheItem{
		tenant:    tenant,
		expiresAt: time.Now().Add(ttl),
	}

	// Update LRU order
	c.updateLRU(key)
}

// Delete removes a tenant from cache.
func (c *inMemoryCache) Delete(ctx context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	c.removeLRU(key)
}

// cleanup periodically removes expired items from cache.
func (c *inMemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	defer close(c.done)

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stop:
			return
		}
	}
}

// removeExpired removes all expired items from the cache.
func (c *inMemoryCache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
			c.removeLRU(key)
		}
	}
}

// updateLRU moves the key to the end of the LRU queue (most recently used).
func (c *inMemoryCache) updateLRU(key string) {
	// Remove from current position
	for i, k := range c.lru {
		if k == key {
			c.lru = append(c.lru[:i], c.lru[i+1:]...)
			break
		}
	}
	// Add to end (most recently used)
	c.lru = append(c.lru, key)
}

// removeLRU removes the key from the LRU queue.
func (c *inMemoryCache) removeLRU(key string) {
	for i, k := range c.lru {
		if k == key {
			c.lru = append(c.lru[:i], c.lru[i+1:]...)
			return
		}
	}
}

// Close stops the cleanup goroutine and waits for it to finish.
func (c *inMemoryCache) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	close(c.stop)
	<-c.done
	return nil
}

// noOpCache is a cache that doesn't cache anything.
// Useful for testing or when caching should be disabled.
type noOpCache struct{}

// NewNoOpCache creates a cache that doesn't cache.
func NewNoOpCache() Cache {
	return &noOpCache{}
}

func (n *noOpCache) Get(ctx context.Context, key string) (*Tenant, bool) {
	return nil, false
}

func (n *noOpCache) Set(ctx context.Context, key string, tenant *Tenant, ttl time.Duration) {
}

func (n *noOpCache) Delete(ctx context.Context, key string) {
}

func (n *noOpCache) Close() error {
	return nil
}
