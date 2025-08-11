package ratelimiter

import "errors"

// Package-level error definitions for rate limiter operations.
var (
	// ErrInvalidConfig indicates that the provided configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrInvalidTokenCount indicates that the requested token count is invalid.
	ErrInvalidTokenCount = errors.New("invalid token count")

	// ErrContextCancelled indicates that the operation was cancelled due to context.
	ErrContextCancelled = errors.New("context cancelled")

	// ErrStoreUnavailable indicates that the store backend is unavailable.
	ErrStoreUnavailable = errors.New("store unavailable")
)
