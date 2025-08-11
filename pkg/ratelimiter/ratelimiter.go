package ratelimiter

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrInvalidTokenCount is returned when the token count is invalid.
	ErrInvalidTokenCount = errors.New("invalid token count")
)

// RateLimiter defines the interface for rate limiting implementations.
type RateLimiter interface {
	// Allow checks if a single request is allowed.
	Allow(ctx context.Context, key string) (*Result, error)
	// AllowN checks if n requests are allowed.
	AllowN(ctx context.Context, key string, n int) (*Result, error)
}

// TokenBucket implements a token bucket rate limiter.
type TokenBucket struct {
	store  Store
	config Config
}

// NewTokenBucket creates a new token bucket rate limiter.
func NewTokenBucket(store Store, config Config) (*TokenBucket, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &TokenBucket{
		store:  store,
		config: config,
	}, nil
}

// Allow checks if a single request is allowed.
func (tb *TokenBucket) Allow(ctx context.Context, key string) (*Result, error) {
	return tb.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed.
func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int) (*Result, error) {
	if n <= 0 {
		return nil, fmt.Errorf("%w: must be positive, got %d", ErrInvalidTokenCount, n)
	}

	remaining, resetAt, err := tb.store.ConsumeTokens(ctx, key, n, tb.config)
	if err != nil {
		return nil, err
	}

	return &Result{
		Limit:     tb.config.Capacity,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}

// Status returns the current state without consuming tokens.
func (tb *TokenBucket) Status(ctx context.Context, key string) (*Result, error) {
	// Check status by consuming 0 tokens
	remaining, resetAt, err := tb.store.ConsumeTokens(ctx, key, 0, tb.config)
	if err != nil {
		return nil, err
	}

	return &Result{
		Limit:     tb.config.Capacity,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}

// Reset resets the rate limit for the given key.
func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	return tb.store.Reset(ctx, key)
}

func (c Config) validate() error {
	if c.Capacity <= 0 {
		return fmt.Errorf("capacity must be positive, got %d", c.Capacity)
	}
	if c.RefillRate <= 0 {
		return fmt.Errorf("refill rate must be positive, got %d", c.RefillRate)
	}
	if c.RefillInterval <= 0 {
		return fmt.Errorf("refill interval must be positive, got %v", c.RefillInterval)
	}
	return nil
}
