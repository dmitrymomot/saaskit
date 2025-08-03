package webhook

import (
	"net/http"
	"time"
)

// DeliveryResult contains information about a webhook delivery attempt
type DeliveryResult struct {
	Success    bool
	StatusCode int
	Attempt    int
	Duration   time.Duration
	Error      error
}

// DeliveryHook is called after each delivery attempt
type DeliveryHook func(result DeliveryResult)

// sendOptions contains all configurable options for a webhook send operation
type sendOptions struct {
	timeout    time.Duration
	headers    map[string]string
	httpClient *http.Client

	maxRetries      int
	backoffStrategy BackoffStrategy

	signatureSecret string

	circuitBreaker *CircuitBreaker

	onDelivery DeliveryHook
}

// defaultSendOptions returns options with sensible defaults
func defaultSendOptions() *sendOptions {
	return &sendOptions{
		timeout:         10 * time.Second,
		headers:         make(map[string]string),
		maxRetries:      3,
		backoffStrategy: DefaultBackoffStrategy(),
	}
}

// SendOption is a functional option for configuring webhook sends
type SendOption func(*sendOptions)

// WithTimeout sets the HTTP request timeout.
// Default is 10 seconds if not specified.
func WithTimeout(timeout time.Duration) SendOption {
	return func(o *sendOptions) {
		if timeout > 0 {
			o.timeout = timeout
		}
	}
}

// WithHeader adds a custom header to the webhook request.
// Standard headers like Content-Type are set automatically.
func WithHeader(key, value string) SendOption {
	return func(o *sendOptions) {
		if key != "" && value != "" {
			o.headers[key] = value
		}
	}
}

// WithHeaders adds multiple custom headers to the webhook request.
func WithHeaders(headers map[string]string) SendOption {
	return func(o *sendOptions) {
		for k, v := range headers {
			if k != "" && v != "" {
				o.headers[k] = v
			}
		}
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
// Default is 3 if not specified. Set to 0 to disable retries.
func WithMaxRetries(n int) SendOption {
	return func(o *sendOptions) {
		if n >= 0 {
			o.maxRetries = n
		}
	}
}

// WithBackoff sets the backoff strategy for retries.
// Default is exponential backoff with jitter.
func WithBackoff(strategy BackoffStrategy) SendOption {
	return func(o *sendOptions) {
		if strategy != nil {
			o.backoffStrategy = strategy
		}
	}
}

// WithSignature enables HMAC-SHA256 request signing with the given secret.
// Adds X-Webhook-Signature, X-Webhook-Timestamp, and X-Webhook-ID headers.
func WithSignature(secret string) SendOption {
	return func(o *sendOptions) {
		o.signatureSecret = secret
	}
}

// WithHTTPClient sets a custom HTTP client for the request.
// Useful for custom transports, proxies, or testing.
func WithHTTPClient(client *http.Client) SendOption {
	return func(o *sendOptions) {
		if client != nil {
			o.httpClient = client
		}
	}
}

// WithCircuitBreaker enables circuit breaker protection for the endpoint.
// Reuse the same instance per endpoint to track failure state across requests.
func WithCircuitBreaker(cb *CircuitBreaker) SendOption {
	return func(o *sendOptions) {
		o.circuitBreaker = cb
	}
}

// WithOnDelivery sets a callback that's invoked after each delivery attempt.
// Useful for logging, metrics, or custom retry logic.
func WithOnDelivery(hook DeliveryHook) SendOption {
	return func(o *sendOptions) {
		o.onDelivery = hook
	}
}

// WithBasicRetry configures simple retry behavior with fixed intervals.
// Suitable for testing or when you need predictable retry timing.
func WithBasicRetry(attempts int, interval time.Duration) SendOption {
	return func(o *sendOptions) {
		o.maxRetries = attempts
		o.backoffStrategy = FixedBackoff{Interval: interval}
	}
}

// WithExponentialRetry configures exponential backoff with jitter.
// Recommended for production use to prevent thundering herd problems.
func WithExponentialRetry(attempts int, initialInterval, maxInterval time.Duration) SendOption {
	return func(o *sendOptions) {
		o.maxRetries = attempts
		o.backoffStrategy = ExponentialBackoff{
			InitialInterval: initialInterval,
			MaxInterval:     maxInterval,
			Multiplier:      2,
			JitterFactor:    0.1,
		}
	}
}

// WithNoRetry disables all retry attempts
func WithNoRetry() SendOption {
	return func(o *sendOptions) {
		o.maxRetries = 0
	}
}
