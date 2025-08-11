package ratelimit

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter that refills tokens
// at a constant rate and allows bursts up to the bucket capacity.
// Memory-efficient as it only stores counters, not individual timestamps.
type TokenBucket struct {
	store    Store
	rate     int           // Tokens added per interval
	interval time.Duration // Interval for token refill
	burst    int           // Maximum bucket capacity

	mu          sync.RWMutex
	lastRefill  map[string]time.Time
	initialized map[string]bool // Track if bucket has been initialized
}

// TokenBucketOption configures a TokenBucket.
type TokenBucketOption func(*TokenBucket)

// WithBurst sets the maximum burst size (bucket capacity).
func WithBurst(burst int) TokenBucketOption {
	return func(tb *TokenBucket) {
		if burst > 0 {
			tb.burst = burst
		}
	}
}

// NewTokenBucket creates a new token bucket rate limiter.
func NewTokenBucket(store Store, rate int, interval time.Duration, opts ...TokenBucketOption) (*TokenBucket, error) {
	if store == nil {
		return nil, ErrStoreRequired
	}
	if rate <= 0 {
		return nil, ErrInvalidLimit
	}
	if interval <= 0 {
		return nil, ErrInvalidInterval
	}

	tb := &TokenBucket{
		store:       store,
		rate:        rate,
		interval:    interval,
		burst:       rate, // Default burst equals rate
		lastRefill:  make(map[string]time.Time),
		initialized: make(map[string]bool),
	}

	for _, opt := range opts {
		opt(tb)
	}

	if tb.burst < tb.rate {
		tb.burst = tb.rate
	}

	return tb, nil
}

// Allow checks if a single request is allowed for the given key.
func (tb *TokenBucket) Allow(ctx context.Context, key string) (*Result, error) {
	return tb.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed for the given key.
func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int) (*Result, error) {
	if key == "" {
		return nil, ErrKeyRequired
	}
	if n <= 0 {
		n = 1
	}

	now := time.Now()

	// Check if this is a brand new bucket or needs refill
	tb.mu.Lock()
	isNew := !tb.initialized[key]
	lastRefill, hasRefill := tb.lastRefill[key]

	if isNew {
		// Mark as initialized
		tb.initialized[key] = true
		tb.lastRefill[key] = now
		tb.mu.Unlock()

		// Initialize with burst capacity and consume
		allowed, remaining, ttl, err := tb.store.ConsumeTokens(ctx, key, n, tb.burst, tb.interval)
		if err != nil {
			return nil, err
		}

		var resetAt time.Time
		if ttl > 0 {
			resetAt = now.Add(ttl)
		} else {
			resetAt = now.Add(tb.interval)
		}

		return &Result{
			Allowed:   allowed,
			Limit:     tb.burst,
			Remaining: max(0, int(remaining)),
			ResetAt:   resetAt,
		}, nil
	}

	// Not new - handle refill
	if hasRefill {
		elapsed := now.Sub(lastRefill)
		intervals := int(elapsed / tb.interval)
		tokensToAdd := intervals * tb.rate

		if tokensToAdd > 0 {
			// Update last refill time
			tb.lastRefill[key] = lastRefill.Add(time.Duration(intervals) * tb.interval)
			tb.mu.Unlock()

			// Get current tokens
			current, _, err := tb.store.Get(ctx, key)
			if err != nil {
				return nil, err
			}

			// Add tokens (up to burst capacity)
			newTotal := min(tb.burst, int(current)+tokensToAdd)
			if newTotal > int(current) {
				_, _, err = tb.store.IncrementAndGet(ctx, key, newTotal-int(current), tb.interval)
				if err != nil {
					return nil, err
				}
			}
		} else {
			tb.mu.Unlock()
		}
	} else {
		tb.mu.Unlock()
	}

	// Try to consume tokens
	allowed, remaining, ttl, err := tb.store.ConsumeTokens(ctx, key, n, tb.burst, tb.interval)
	if err != nil {
		return nil, err
	}

	var resetAt time.Time
	if ttl > 0 {
		resetAt = now.Add(ttl)
	} else {
		resetAt = now.Add(tb.interval)
	}

	return &Result{
		Allowed:   allowed,
		Limit:     tb.burst,
		Remaining: max(0, int(remaining)),
		ResetAt:   resetAt,
	}, nil
}

// Status returns the current rate limit status without consuming tokens.
func (tb *TokenBucket) Status(ctx context.Context, key string) (*Result, error) {
	if key == "" {
		return nil, ErrKeyRequired
	}

	now := time.Now()

	tb.mu.RLock()
	lastRefill, exists := tb.lastRefill[key]
	tb.mu.RUnlock()

	current, ttl, err := tb.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// For new keys, consider full burst capacity available
	if !exists && current == 0 {
		available := tb.burst
		var resetAt time.Time
		if ttl > 0 {
			resetAt = now.Add(ttl)
		} else {
			resetAt = now.Add(tb.interval)
		}
		return &Result{
			Allowed:   available > 0,
			Limit:     tb.burst,
			Remaining: available,
			ResetAt:   resetAt,
		}, nil
	} else if !exists {
		lastRefill = now
	}

	elapsed := now.Sub(lastRefill)
	intervals := elapsed / tb.interval
	tokensToAdd := int(intervals) * tb.rate

	available := min(tb.burst, int(current)+tokensToAdd)

	var resetAt time.Time
	if ttl > 0 {
		resetAt = now.Add(ttl)
	} else {
		resetAt = now.Add(tb.interval)
	}

	return &Result{
		Allowed:   available > 0,
		Limit:     tb.burst,
		Remaining: max(0, available),
		ResetAt:   resetAt,
	}, nil
}

// Reset resets the rate limit for the given key.
func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	if key == "" {
		return ErrKeyRequired
	}

	tb.mu.Lock()
	delete(tb.lastRefill, key)
	delete(tb.initialized, key)
	tb.mu.Unlock()

	// Delete the key from store
	if err := tb.store.Delete(ctx, key); err != nil {
		return err
	}

	// Will be reinitialized with burst capacity on next access
	return nil
}
