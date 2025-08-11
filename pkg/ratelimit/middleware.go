package ratelimit

import (
	"fmt"
	"net/http"
	"strconv"
)

// Middleware creates an HTTP middleware for rate limiting.
// It uses the provided Limiter and KeyFunc to enforce rate limits.
// The keyFunc parameter is required and defines how to extract the rate limit key from requests.
func Middleware(limiter Limiter, keyFunc KeyFunc) func(http.Handler) http.Handler {
	if keyFunc == nil {
		panic("ratelimit.Middleware: keyFunc is required")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract rate limit key from request
			key := keyFunc(r)
			if key == "" {
				// If no key can be extracted, allow the request
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			result, err := limiter.Allow(r.Context(), key)
			if err != nil {
				// Allow request on error (fail open)
				next.ServeHTTP(w, r)
				return
			}

			// Always set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			// Check if request is allowed
			if !result.Allowed {
				// Calculate retry after in seconds
				retryAfter := result.RetryAfter().Seconds()
				if retryAfter < 1 {
					retryAfter = 1
				}

				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter)))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			// Request allowed, continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// MiddlewareOption configures middleware behavior.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	keyFunc        KeyFunc
	onLimitReached func(w http.ResponseWriter, r *http.Request, result *Result)
	skipFunc       func(r *http.Request) bool
}

// WithKeyFunc sets a custom key extraction function.
func WithKeyFunc(fn KeyFunc) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.keyFunc = fn
	}
}

// WithOnLimitReached sets a custom handler for rate limit exceeded.
func WithOnLimitReached(fn func(w http.ResponseWriter, r *http.Request, result *Result)) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.onLimitReached = fn
	}
}

// WithSkipFunc sets a function to determine if rate limiting should be skipped.
func WithSkipFunc(fn func(r *http.Request) bool) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.skipFunc = fn
	}
}

// MiddlewareWithOptions creates an HTTP middleware with custom options.
// The keyFunc must be provided either as the first parameter or via WithKeyFunc option.
func MiddlewareWithOptions(limiter Limiter, keyFunc KeyFunc, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	config := &middlewareConfig{
		keyFunc: keyFunc,
		onLimitReached: func(w http.ResponseWriter, r *http.Request, result *Result) {
			retryAfter := result.RetryAfter().Seconds()
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter)))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	if config.keyFunc == nil {
		panic("ratelimit.MiddlewareWithOptions: keyFunc is required")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if rate limiting should be skipped
			if config.skipFunc != nil && config.skipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract rate limit key from request
			key := config.keyFunc(r)
			if key == "" {
				// If no key can be extracted, allow the request
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			result, err := limiter.Allow(r.Context(), key)
			if err != nil {
				// Allow request on error (fail open)
				next.ServeHTTP(w, r)
				return
			}

			// Always set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			// Check if request is allowed
			if !result.Allowed {
				config.onLimitReached(w, r, result)
				return
			}

			// Request allowed, continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// HandlerFunc wraps a handler function with rate limiting.
// This is useful for applying rate limits to specific endpoints.
func HandlerFunc(limiter Limiter, keyFunc KeyFunc, handler http.HandlerFunc) http.HandlerFunc {
	middleware := Middleware(limiter, keyFunc)
	return middleware(handler).ServeHTTP
}

// PerEndpoint creates a middleware that applies different rate limits per endpoint.
type EndpointConfig struct {
	Path    string
	Limiter Limiter
	KeyFunc KeyFunc
}

// PerEndpoint creates a middleware with different rate limits for different endpoints.
// Each endpoint must have its own KeyFunc defined.
func PerEndpoint(configs []EndpointConfig, defaultLimiter Limiter, defaultKeyFunc KeyFunc) func(http.Handler) http.Handler {
	// Create a map for faster lookups
	configMap := make(map[string]EndpointConfig)
	for _, cfg := range configs {
		if cfg.KeyFunc == nil {
			panic(fmt.Sprintf("ratelimit.PerEndpoint: KeyFunc is required for path %s", cfg.Path))
		}
		configMap[cfg.Path] = cfg
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var limiter Limiter
			var keyFunc KeyFunc

			// Check if there's a specific config for this path
			if cfg, ok := configMap[r.URL.Path]; ok {
				limiter = cfg.Limiter
				keyFunc = cfg.KeyFunc
			} else {
				limiter = defaultLimiter
				keyFunc = defaultKeyFunc
			}

			// If no limiter is configured, skip rate limiting
			if limiter == nil || keyFunc == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Apply rate limiting
			key := keyFunc(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			result, err := limiter.Allow(r.Context(), key)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Set headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed {
				retryAfter := result.RetryAfter().Seconds()
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter)))
				http.Error(w, fmt.Sprintf("Rate limit exceeded for %s", r.URL.Path), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
