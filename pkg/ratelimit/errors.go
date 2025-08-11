package ratelimit

import "errors"

var (
	// ErrRateLimitExceeded is returned when the rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrInvalidLimit is returned when the limit is invalid.
	ErrInvalidLimit = errors.New("invalid limit")

	// ErrInvalidInterval is returned when the interval is invalid.
	ErrInvalidInterval = errors.New("invalid interval")

	// ErrKeyRequired is returned when the key is empty.
	ErrKeyRequired = errors.New("key is required")

	// ErrStoreRequired is returned when the store is nil.
	ErrStoreRequired = errors.New("store is required")
)
