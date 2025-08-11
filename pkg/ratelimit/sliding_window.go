package ratelimit

import (
	"context"
	"time"
)

// SlidingWindow implements a sliding window rate limiter that tracks
// individual request timestamps within a moving time window. More accurate
// than token bucket but uses more memory due to timestamp storage.
type SlidingWindow struct {
	store  SlidingWindowStore
	limit  int
	window time.Duration
}

// NewSlidingWindow creates a new sliding window rate limiter.
func NewSlidingWindow(store SlidingWindowStore, limit int, window time.Duration) (*SlidingWindow, error) {
	if store == nil {
		return nil, ErrStoreRequired
	}
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	if window <= 0 {
		return nil, ErrInvalidInterval
	}

	return &SlidingWindow{
		store:  store,
		limit:  limit,
		window: window,
	}, nil
}

// Allow checks if a single request is allowed for the given key.
func (sw *SlidingWindow) Allow(ctx context.Context, key string) (*Result, error) {
	return sw.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed for the given key.
func (sw *SlidingWindow) AllowN(ctx context.Context, key string, n int) (*Result, error) {
	if key == "" {
		return nil, ErrKeyRequired
	}
	if n <= 0 {
		n = 1
	}

	now := time.Now()

	// Use atomic check-and-record operation
	allowed, finalCount, err := sw.store.RecordTimestampIfAllowed(ctx, key, now, sw.window, sw.limit, n)
	if err != nil {
		return nil, err
	}

	remaining := sw.limit - int(finalCount)
	if !allowed {
		// If not allowed, calculate actual remaining
		count, err := sw.store.CountInWindow(ctx, key, sw.window)
		if err != nil {
			return nil, err
		}
		remaining = sw.limit - int(count)
	}

	result := &Result{
		Allowed:   allowed,
		Limit:     sw.limit,
		Remaining: max(0, remaining),
		ResetAt:   now.Add(sw.window),
	}

	return result, nil
}

// Status returns the current rate limit status without consuming tokens.
func (sw *SlidingWindow) Status(ctx context.Context, key string) (*Result, error) {
	if key == "" {
		return nil, ErrKeyRequired
	}

	count, err := sw.store.CountInWindow(ctx, key, sw.window)
	if err != nil {
		return nil, err
	}

	remaining := sw.limit - int(count)

	return &Result{
		Allowed:   remaining > 0,
		Limit:     sw.limit,
		Remaining: max(0, remaining),
		ResetAt:   time.Now().Add(sw.window),
	}, nil
}

// Reset resets the rate limit for the given key.
func (sw *SlidingWindow) Reset(ctx context.Context, key string) error {
	if key == "" {
		return ErrKeyRequired
	}

	return sw.store.Delete(ctx, key)
}
