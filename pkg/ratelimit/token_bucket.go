package ratelimit

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter.
// It allows bursts of traffic while maintaining an average rate.
type TokenBucket struct {
	store    Store
	rate     int           // Tokens added per interval
	interval time.Duration // Interval for token refill
	burst    int           // Maximum bucket capacity

	// Track token refill times per key
	mu         sync.RWMutex
	lastRefill map[string]time.Time
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
		store:      store,
		rate:       rate,
		interval:   interval,
		burst:      rate, // Default burst equals rate
		lastRefill: make(map[string]time.Time),
	}

	for _, opt := range opts {
		opt(tb)
	}

	// Ensure burst is at least equal to rate
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

	tb.mu.RLock()
	lastRefill, exists := tb.lastRefill[key]
	tb.mu.RUnlock()

	if !exists {
		lastRefill = now
		tb.mu.Lock()
		tb.lastRefill[key] = now
		tb.mu.Unlock()
	}

	elapsed := now.Sub(lastRefill)
	intervals := elapsed / tb.interval
	tokensToAdd := int(intervals) * tb.rate

	if tokensToAdd > 0 {
		tb.mu.Lock()
		tb.lastRefill[key] = now
		tb.mu.Unlock()
	}

	// Get current token count
	current, ttl, err := tb.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Calculate available tokens (can't exceed burst)
	available := int(current)
	if tokensToAdd > 0 {
		available = min(tb.burst, available+tokensToAdd)
		// Reset the counter with new value
		if available > int(current) {
			current, ttl, err = tb.store.IncrementAndGet(ctx, key, available-int(current), tb.interval)
			if err != nil {
				return nil, err
			}
			available = int(current)
		}
	}

	// Check if request is allowed
	allowed := available >= n
	remaining := available

	if allowed {
		// Consume tokens
		consumed, _, err := tb.store.IncrementAndGet(ctx, key, -n, tb.interval)
		if err != nil {
			return nil, err
		}
		remaining = int(consumed)
	}

	// Calculate when next token will be available
	var resetAt time.Time
	if ttl > 0 {
		resetAt = now.Add(ttl)
	} else {
		resetAt = now.Add(tb.interval)
	}

	return &Result{
		Allowed:   allowed,
		Limit:     tb.burst,
		Remaining: max(0, remaining),
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

	if !exists {
		lastRefill = now
	}

	elapsed := now.Sub(lastRefill)
	intervals := elapsed / tb.interval
	tokensToAdd := int(intervals) * tb.rate

	// Get current token count
	current, ttl, err := tb.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Calculate available tokens (can't exceed burst)
	available := min(tb.burst, int(current)+tokensToAdd)

	// Calculate when next token will be available
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
	tb.mu.Unlock()

	return tb.store.Delete(ctx, key)
}
