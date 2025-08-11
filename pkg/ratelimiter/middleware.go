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

// Middleware creates an HTTP middleware for rate limiting.
func Middleware(tb *TokenBucket, keyFunc KeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			result, err := tb.Allow(r.Context(), key)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, result.Remaining)))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			if !result.Allowed() {
				// Set Retry-After header
				retryAfter := int(result.RetryAfter().Seconds())
				if retryAfter > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				}

				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
