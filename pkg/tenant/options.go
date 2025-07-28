package tenant

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// ErrorHandler handles errors that occur during tenant resolution.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// config holds middleware configuration.
type config struct {
	cache         Cache
	cacheTTL      time.Duration
	errorHandler  ErrorHandler
	skipPaths     []string
	requireActive bool
}

// Option configures the middleware.
type Option func(*config)

// WithCache sets a custom cache implementation.
func WithCache(cache Cache) Option {
	return func(c *config) {
		c.cache = cache
	}
}

// WithCacheSize sets the maximum cache size.
// This will replace the default cache with a size-limited one.
// NOTE: This requires a context for the cache lifecycle. Consider using WithCache instead.
func WithCacheSize(ctx context.Context, size int) Option {
	return func(c *config) {
		c.cache = NewInMemoryCacheWithSize(ctx, size)
	}
}

// WithCacheTTL sets the cache time-to-live.
func WithCacheTTL(ttl time.Duration) Option {
	return func(c *config) {
		c.cacheTTL = ttl
	}
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(c *config) {
		c.errorHandler = handler
	}
}

// WithSkipPaths sets paths that should skip tenant resolution.
func WithSkipPaths(paths []string) Option {
	return func(c *config) {
		c.skipPaths = paths
	}
}

// WithRequireActive ensures only active tenants are allowed.
func WithRequireActive(require bool) Option {
	return func(c *config) {
		c.requireActive = require
	}
}

// defaultErrorHandler is the default error handler.
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrTenantNotFound):
		http.Error(w, "Tenant not found", http.StatusNotFound)
	case errors.Is(err, ErrInactiveTenant):
		http.Error(w, "Tenant is inactive", http.StatusForbidden)
	case errors.Is(err, ErrInvalidIdentifier):
		http.Error(w, "Invalid tenant identifier", http.StatusBadRequest)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
