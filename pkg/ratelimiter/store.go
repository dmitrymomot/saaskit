package ratelimiter

import (
	"context"
	"time"
)

// Store defines the interface for rate limit storage backends.
type Store interface {
	// ConsumeTokens attempts to consume tokens and returns the state after consumption.
	// If tokens is 0, updates bucket state without consuming (used for status checks).
	// A negative remaining count indicates the request should be denied.
	ConsumeTokens(ctx context.Context, key string, tokens int, config Config) (remaining int, resetAt time.Time, err error)

	Reset(ctx context.Context, key string) error
}
