// Package webhook provides reliable HTTP webhook delivery with automatic retries,
// exponential backoff, request signing, and circuit breaker protection.
//
// This is a low-level utility package that handles the mechanics of webhook delivery
// without business logic or persistence. For a complete webhook management solution
// with async delivery and persistence, see the modules/webhooks package which builds
// on top of this foundation.
//
// # Key Features
//
// - Synchronous HTTP POST delivery with configurable timeouts
// - Automatic retry logic with exponential backoff and jitter
// - HMAC-SHA256 request signing for payload authentication
// - Circuit breaker to prevent hammering failed endpoints
// - Flexible error classification (permanent vs temporary failures)
// - Delivery hooks for metrics, logging, and custom handling
//
// # Basic Usage
//
//	import (
//	    "context"
//	    "github.com/dmitrymomot/saaskit/pkg/webhook"
//	)
//
//	// Create a sender
//	sender := webhook.NewSender()
//
//	// Send a webhook
//	err := sender.Send(ctx, "https://api.example.com/webhook",
//	    []byte(`{"event":"user.created","id":"123"}`))
//
// # Advanced Usage
//
// The Send method accepts functional options to customize behavior:
//
//	err := sender.Send(ctx, url, payload,
//	    // Security
//	    webhook.WithSignature("webhook_secret"),
//
//	    // Custom headers
//	    webhook.WithHeader("X-Event-Type", "user.created"),
//
//	    // Retry configuration
//	    webhook.WithMaxRetries(5),
//	    webhook.WithBackoff(webhook.ExponentialBackoff{
//	        InitialInterval: 2 * time.Second,
//	        MaxInterval:     60 * time.Second,
//	        Multiplier:      2,
//	        JitterFactor:    0.1,
//	    }),
//
//	    // Timeouts
//	    webhook.WithTimeout(30 * time.Second),
//
//	    // Circuit breaker (reuse for same endpoint)
//	    webhook.WithCircuitBreaker(circuitBreaker),
//
//	    // Delivery hooks
//	    webhook.WithOnDelivery(func(result webhook.DeliveryResult) {
//	        log.Printf("Webhook delivery: success=%v status=%d duration=%v",
//	            result.Success, result.StatusCode, result.Duration)
//	    }),
//	)
//
// # Request Signing
//
// When WithSignature is used, the package adds standard webhook headers:
//
//	X-Webhook-Signature: HMAC-SHA256 hex-encoded signature
//	X-Webhook-Timestamp: Unix timestamp when signature was created
//	X-Webhook-ID: Unique identifier for this webhook event
//
// The signature is calculated as: HMAC-SHA256(secret, timestamp + "." + payload)
//
// Receivers can verify signatures using the VerifySignature function:
//
//	headers := webhook.ExtractSignatureHeaders(httpHeaders)
//	err := webhook.VerifySignature(secret, payload, headers, 5*time.Minute)
//
// # Retry Logic
//
// The package distinguishes between permanent and temporary failures:
//
// Permanent failures (no retry):
// - 4xx status codes (except 408, 425, 429)
// - Invalid URLs or payloads
//
// Temporary failures (will retry):
// - 5xx status codes
// - Network errors
// - Timeouts
// - 408 Request Timeout
// - 425 Too Early
// - 429 Too Many Requests
//
// # Backoff Strategies
//
// Three backoff strategies are provided:
//
// ExponentialBackoff (default):
//   - Exponentially increasing delays with jitter
//   - Prevents thundering herd problem
//   - Formula: InitialInterval * (Multiplier ^ attempt) * (1 ± JitterFactor)
//
// LinearBackoff:
//   - Linearly increasing delays
//   - Formula: Interval * attempt
//
// FixedBackoff:
//   - Constant delay between retries
//
// # Circuit Breaker
//
// The circuit breaker prevents hammering of consistently failing endpoints:
//
//	cb := webhook.NewCircuitBreaker(
//	    5,                    // Failure threshold
//	    2,                    // Success threshold to close
//	    30 * time.Second,     // Recovery timeout
//	)
//
//	// Reuse the same circuit breaker for the same endpoint
//	err := sender.Send(ctx, url, payload, webhook.WithCircuitBreaker(cb))
//
// States:
// - Closed: Normal operation, requests pass through
// - Open: Too many failures, requests blocked
// - Half-Open: Testing if service recovered
//
// # Performance Considerations
//
// - The default HTTP client reuses connections with proper pooling
// - Payload signing uses HMAC-SHA256 which is fast and secure
// - Circuit breakers should be reused per endpoint, not created per request
// - For high-volume webhooks, consider using the async modules/webhooks package
//
// # Integration Points
//
// This package is designed as a building block:
//
// - Use directly for simple, synchronous webhook needs
// - Combine with pkg/queue for async delivery in modules/webhooks
// - Add pkg/storage for webhook persistence and history
// - Integrate with pkg/tenant for multi-tenant webhook management
//
// See the README.md file for more examples and the modules/webhooks package
// for a complete webhook management solution.
package webhook
