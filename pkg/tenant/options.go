package tenant

import (
	"errors"
	"log/slog"
	"net/http"
)

// ErrorHandler handles errors that occur during tenant resolution.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// config holds middleware configuration.
type config struct {
	cache         Cache
	errorHandler  ErrorHandler
	skipPaths     []string
	requireActive bool
	logger        *slog.Logger
}

// Option configures the middleware.
type Option func(*config)

// WithCache sets a custom cache implementation.
func WithCache(cache Cache) Option {
	return func(c *config) {
		c.cache = cache
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

// WithLogger sets a custom logger for the middleware.
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

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
