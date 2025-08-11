package ratelimiter

import (
	"hash/fnv"
	"net/http"
	"strconv"
	"strings"
)

// maxKeyLength is the maximum allowed length for a rate limit key
// to prevent excessively long storage keys.
const maxKeyLength = 64

// KeyFunc extracts a rate limit key from the request.
type KeyFunc func(r *http.Request) string

// ErrorResponder handles error responses for rate limiting.
// If err is not nil, it indicates an internal error.
// If err is nil and result.Allowed() is false, the rate limit was exceeded.
type ErrorResponder func(w http.ResponseWriter, r *http.Request, result *Result, err error)

// middlewareConfig holds middleware configuration.
type middlewareConfig struct {
	errorResponder ErrorResponder
}

// MiddlewareOption configures the rate limiting middleware.
type MiddlewareOption func(*middlewareConfig)

// WithErrorResponder sets a custom error responder.
func WithErrorResponder(responder ErrorResponder) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.errorResponder = responder
	}
}

// Composite combines multiple key functions into one.
// Long keys (>64 chars) are hashed using FNV-1a for storage efficiency.
func Composite(keyFuncs ...KeyFunc) KeyFunc {
	return func(r *http.Request) string {
		// Collect non-empty parts
		parts := make([]string, 0, len(keyFuncs))
		for _, fn := range keyFuncs {
			if key := fn(r); key != "" {
				parts = append(parts, key)
			}
		}

		// Handle empty case
		if len(parts) == 0 {
			return ""
		}

		// Single key optimization
		if len(parts) == 1 && len(parts[0]) <= maxKeyLength {
			return parts[0]
		}

		// Join multiple parts
		combined := strings.Join(parts, ":")

		// Hash if too long using FNV-1a (fast, simple, built-in)
		if len(combined) > maxKeyLength {
			h := fnv.New64a()
			h.Write([]byte(combined))
			// Base36 encoding for compact output (~13 chars)
			return strconv.FormatUint(h.Sum64(), 36)
		}

		return combined
	}
}

// defaultErrorResponder handles all rate limiting errors by default.
func defaultErrorResponder(w http.ResponseWriter, r *http.Request, result *Result, err error) {
	// Internal error
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Rate limit exceeded
	if result != nil && !result.Allowed() {
		// Set Retry-After header
		retryAfter := int(result.RetryAfter().Seconds())
		if retryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		}
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	}
}

// Middleware creates an HTTP middleware for rate limiting.
func Middleware(limiter RateLimiter, keyFunc KeyFunc, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	// Apply default configuration
	config := &middlewareConfig{
		errorResponder: defaultErrorResponder,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			result, err := limiter.Allow(r.Context(), key)
			if err != nil {
				config.errorResponder(w, r, nil, err)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, result.Remaining)))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed() {
				config.errorResponder(w, r, result, nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
