package ratelimit

import "errors"

var (
	// Common rate limiting errors.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrInvalidLimit      = errors.New("invalid limit")
	ErrInvalidInterval   = errors.New("invalid interval")
	ErrKeyRequired       = errors.New("key is required")
	ErrStoreRequired     = errors.New("store is required")
)
