package webhook

import "errors"

// Domain errors for webhook operations, designed for error wrapping and classification.
// These provide stable error identities for error handling while allowing detailed
// context to be wrapped using errors.Join() for logging and debugging.
//
// Error classification strategy:
// - Configuration errors: Invalid setup or parameters (fail fast)
// - Delivery errors: Network, timeout, or HTTP failures (may retry)
// - Circuit breaker: Protection mechanism when endpoint consistently fails
var (
	ErrWebhookDeliveryFailed = errors.New("webhook delivery failed")
	ErrInvalidConfiguration  = errors.New("invalid webhook configuration")
	ErrPermanentFailure      = errors.New("permanent webhook failure")
	ErrTemporaryFailure      = errors.New("temporary webhook failure")
	ErrCircuitOpen           = errors.New("webhook circuit breaker is open")
	ErrInvalidPayload        = errors.New("invalid webhook payload")
	ErrInvalidURL            = errors.New("invalid webhook URL")
	ErrTimeout               = errors.New("webhook request timeout")
)

// IsCircuitOpen checks if an error indicates the circuit breaker is open
func IsCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}
