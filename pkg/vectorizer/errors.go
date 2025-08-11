package vectorizer

import "errors"

// Domain errors for vectorization operations.
// These are designed to be wrapped with internal errors using errors.Join()
// to provide both user-facing messages and detailed logging context.
var (
	ErrProviderNotSet        = errors.New("vectorization provider not set")
	ErrChunkerNotSet         = errors.New("chunker not set")
	ErrEmptyText             = errors.New("text cannot be empty")
	ErrInvalidChunkSize      = errors.New("invalid chunk size")
	ErrVectorizationFailed   = errors.New("failed to vectorize text")
	ErrInvalidDimensions     = errors.New("invalid vector dimensions")
	ErrAPIKeyRequired        = errors.New("API key is required")
	ErrInvalidModel          = errors.New("invalid model name")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrContextLengthExceeded = errors.New("text exceeds maximum context length")
)
