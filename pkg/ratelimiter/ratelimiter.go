package ratelimiter

import (
	"context"
	"fmt"
)

// RateLimiter defines the interface for rate limiting implementations.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (*Result, error)
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

func (tb *TokenBucket) Allow(ctx context.Context, key string) (*Result, error) {
	return tb.AllowN(ctx, key, 1)
}

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
	// ConsumeTokens with 0 tokens updates bucket state but doesn't actually consume
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

func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	return tb.store.Reset(ctx, key)
}

func (c Config) validate() error {
	if c.Capacity <= 0 {
		return fmt.Errorf("%w: capacity must be positive, got %d", ErrInvalidConfig, c.Capacity)
	}
	if c.RefillRate <= 0 {
		return fmt.Errorf("%w: refill rate must be positive, got %d", ErrInvalidConfig, c.RefillRate)
	}
	if c.RefillInterval <= 0 {
		return fmt.Errorf("%w: refill interval must be positive, got %v", ErrInvalidConfig, c.RefillInterval)
	}
	return nil
}
