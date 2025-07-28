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
}

// inMemoryCache is the default in-memory cache implementation.
type inMemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	stop  chan struct{}
}

type cacheItem struct {
	tenant    *Tenant
	expiresAt time.Time
}

// NewInMemoryCache creates a new in-memory cache with automatic cleanup.
func NewInMemoryCache() Cache {
	cache := &inMemoryCache{
		items: make(map[string]cacheItem),
		stop:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a tenant from cache.
func (c *inMemoryCache) Get(ctx context.Context, key string) (*Tenant, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.tenant, true
}

// Set stores a tenant in cache.
func (c *inMemoryCache) Set(ctx context.Context, key string, tenant *Tenant, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		tenant:    tenant,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a tenant from cache.
func (c *inMemoryCache) Delete(ctx context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// cleanup periodically removes expired items from cache.
func (c *inMemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

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
		}
	}
}

// Close stops the cleanup goroutine.
func (c *inMemoryCache) Close() {
	close(c.stop)
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
