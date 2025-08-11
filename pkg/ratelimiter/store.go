package ratelimiter

import (
	"context"
	"time"
)

// Store defines the interface for rate limit storage backends.
type Store interface {
	// ConsumeTokens attempts to consume the specified number of tokens.
	// Returns the remaining tokens and reset time.
	// If remaining is negative, the request should be denied.
	ConsumeTokens(ctx context.Context, key string, tokens int, config Config) (remaining int, resetAt time.Time, err error)

	// Reset clears the rate limit state for the given key.
	Reset(ctx context.Context, key string) error
}
