package ratelimit

import (
	"context"
	"time"
)

// Result contains the result of a rate limit check.
type Result struct {
	// Allowed indicates whether the request is allowed.
	Allowed bool

	// Limit is the maximum number of requests allowed in the window.
	Limit int

	// Remaining is the number of requests remaining in the current window.
	Remaining int

	// ResetAt is the time when the rate limit window resets.
	ResetAt time.Time
}

// RetryAfter returns how long to wait before the next request is allowed.
// Returns 0 if the current request was allowed.
func (r *Result) RetryAfter() time.Duration {
	if r.Allowed {
		return 0
	}
	return time.Until(r.ResetAt)
}

// Limiter defines the interface for rate limiting implementations.
type Limiter interface {
	// Allow checks if a single request is allowed for the given key.
	// If allowed, it consumes one token/slot.
	Allow(ctx context.Context, key string) (*Result, error)

	// AllowN checks if n requests are allowed for the given key.
	// If allowed, it consumes n tokens/slots.
	AllowN(ctx context.Context, key string, n int) (*Result, error)

	// Status returns the current rate limit status for the given key
	// without consuming any tokens/slots.
	Status(ctx context.Context, key string) (*Result, error)

	// Reset resets the rate limit for the given key.
	Reset(ctx context.Context, key string) error
}

// Store defines the interface for rate limit storage backends.
type Store interface {
	// IncrementAndGet atomically increments the counter for the given key
	// and returns the new value along with the TTL.
	IncrementAndGet(ctx context.Context, key string, incr int, window time.Duration) (current int64, ttl time.Duration, err error)

	// Get returns the current counter value and TTL for the given key.
	Get(ctx context.Context, key string) (current int64, ttl time.Duration, err error)

	// Delete removes the given key from the store.
	Delete(ctx context.Context, key string) error

	// ConsumeTokens atomically checks and consumes tokens if available.
	// For new buckets, initializes with burst capacity.
	// Returns (allowed, remaining, ttl, error).
	ConsumeTokens(ctx context.Context, key string, n int, burst int, window time.Duration) (allowed bool, remaining int64, ttl time.Duration, err error)
}

// SlidingWindowStore extends Store with sliding window specific operations.
type SlidingWindowStore interface {
	Store

	// RecordTimestamp adds a timestamp to the sliding window for the given key.
	RecordTimestamp(ctx context.Context, key string, timestamp time.Time, window time.Duration) error

	// CountInWindow returns the number of timestamps within the sliding window.
	CountInWindow(ctx context.Context, key string, window time.Duration) (int64, error)

	// CleanupExpired removes expired timestamps from the sliding window.
	CleanupExpired(ctx context.Context, key string, window time.Duration) error

	// RecordTimestampIfAllowed atomically checks if recording is allowed and records if so.
	// Returns whether the timestamp was recorded and the final count.
	RecordTimestampIfAllowed(ctx context.Context, key string, timestamp time.Time, window time.Duration, limit int, n int) (bool, int64, error)
}
