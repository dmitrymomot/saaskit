package feature

import "errors"

var (
	ErrFlagNotFound           = errors.New("feature flag not found")
	ErrInvalidFlag            = errors.New("invalid feature flag parameters")
	ErrProviderNotInitialized = errors.New("feature provider not initialized")
	ErrInvalidContext         = errors.New("invalid context for feature evaluation")
	ErrInvalidStrategy        = errors.New("invalid feature rollout strategy")
	ErrOperationFailed        = errors.New("feature operation failed")
)
