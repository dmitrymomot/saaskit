package tenant

import (
	"context"
)

// Cache is the interface for tenant caching implementations.
type Cache interface {
	// Get retrieves a tenant from cache by key.
	Get(ctx context.Context, key string) (*Tenant, bool)

	// Set stores a tenant in cache.
	Set(ctx context.Context, key string, tenant *Tenant) error

	// Delete removes a tenant from cache.
	Delete(ctx context.Context, key string) error
}

// NoOpCache disables caching, useful for testing or when caching is unwanted.
type NoOpCache struct{}

func (n *NoOpCache) Get(ctx context.Context, key string) (*Tenant, bool) {
	return nil, false
}

func (n *NoOpCache) Set(ctx context.Context, key string, tenant *Tenant) error {
	return nil
}

func (n *NoOpCache) Delete(ctx context.Context, key string) error {
	return nil
}
