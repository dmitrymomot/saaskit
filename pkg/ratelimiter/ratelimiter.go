package ratelimiter

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("invalid rate limiter configuration")
)

// TokenBucket implements a token bucket rate limiter.
type TokenBucket struct {
	store  Store
	config Config
}

// NewTokenBucket creates a new token bucket rate limiter.
func NewTokenBucket(store Store, config Config) (*TokenBucket, error) {
	if err := config.Validate(); err != nil {
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

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.Capacity <= 0 {
		return fmt.Errorf("%w: capacity must be greater than 0, got %d", ErrInvalidConfig, c.Capacity)
	}
	if c.RefillRate <= 0 {
		return fmt.Errorf("%w: refill rate must be greater than 0, got %d", ErrInvalidConfig, c.RefillRate)
	}
	if c.RefillInterval <= 0 {
		return fmt.Errorf("%w: refill interval must be greater than 0, got %v", ErrInvalidConfig, c.RefillInterval)
	}
	return nil
}