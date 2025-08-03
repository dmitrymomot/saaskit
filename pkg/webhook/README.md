# pkg/webhook

Low-level utility package for reliable, synchronous HTTP webhook delivery with retry logic and security features.

## Features

- **Synchronous Delivery**: Blocking HTTP POST with configurable timeouts
- **Retry Logic**: Automatic retries with exponential backoff for transient failures
- **Request Signing**: HMAC-SHA256 signatures for payload authentication
- **Circuit Breaker**: Prevents hammering of failing endpoints
- **Error Classification**: Distinguishes between retryable and permanent failures
- **Observability**: Hooks for metrics, logging, and delivery callbacks

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/webhook"
```

## Quick Start

```go
// Create sender with default config
sender := webhook.NewSender()

// Send webhook
err := sender.Send(ctx, "https://api.example.com/webhook",
    []byte(`{"event":"user.created","id":"123"}`))
```

## Advanced Usage

### With Request Signing

```go
err := sender.Send(ctx, url, payload,
    webhook.WithSignature("webhook_secret"),
    webhook.WithHeader("X-Event-Type", "user.created"),
)
```

### With Custom Retry Strategy

```go
err := sender.Send(ctx, url, payload,
    webhook.WithMaxRetries(5),
    webhook.WithBackoff(webhook.ExponentialBackoff{
        InitialInterval: 2 * time.Second,
        MaxInterval:     60 * time.Second,
        Multiplier:      2,
        JitterFactor:    0.1,
    }),
)
```

### With Circuit Breaker

```go
// Create circuit breaker (reuse for same endpoint)
cb := webhook.NewCircuitBreaker(5, 2, 30*time.Second)

// Use in sends
err := sender.Send(ctx, url, payload,
    webhook.WithCircuitBreaker(cb),
)
```

### With Delivery Tracking

```go
err := sender.Send(ctx, url, payload,
    webhook.WithOnDelivery(func(result webhook.DeliveryResult) {
        if result.Success {
            metrics.WebhookDelivered(result.Duration)
        } else {
            metrics.WebhookFailed(result.StatusCode)
            log.Printf("Webhook failed: attempt=%d status=%d error=%v",
                result.Attempt, result.StatusCode, result.Error)
        }
    }),
)
```

## Webhook Receiver Example

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    // Read body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read body", http.StatusBadRequest)
        return
    }

    // Extract signature headers
    headers, err := webhook.ExtractSignatureHeaders(headerMap(r.Header))
    if err != nil {
        http.Error(w, "Missing signature", http.StatusUnauthorized)
        return
    }

    // Verify signature (max 5 minute age)
    err = webhook.VerifySignature(webhookSecret, body, headers, 5*time.Minute)
    if err != nil {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }

    // Process webhook...
    w.WriteHeader(http.StatusOK)
}
```

## Backoff Strategies

### Exponential Backoff (Default)

```go
webhook.WithBackoff(webhook.ExponentialBackoff{
    InitialInterval: time.Second,
    MaxInterval:     30 * time.Second,
    Multiplier:      2,
    JitterFactor:    0.1, // 10% jitter
})
```

### Linear Backoff

```go
webhook.WithBackoff(webhook.LinearBackoff{
    Interval:    time.Second,
    MaxInterval: 30 * time.Second,
})
```

### Fixed Backoff

```go
webhook.WithBackoff(webhook.FixedBackoff{
    Interval: 5 * time.Second,
})
```

## Error Handling

The package classifies errors into permanent and temporary failures:

**Permanent Failures** (no retry):

- 400 Bad Request
- 401 Unauthorized
- 403 Forbidden
- 404 Not Found
- Other 4xx errors (except 408, 425, 429)

**Temporary Failures** (will retry):

- All 5xx errors
- Network errors
- Timeouts
- 408 Request Timeout
- 425 Too Early
- 429 Too Many Requests

## Quick Options

For common scenarios, use the convenience options:

```go
// Simple retry with fixed interval
webhook.WithBasicRetry(3, 5*time.Second)

// Exponential retry
webhook.WithExponentialRetry(5, time.Second, time.Minute)

// No retries
webhook.WithNoRetry()
```

## Custom HTTP Client

```go
// Use custom client (e.g., for proxy)
client := &http.Client{
    Transport: &http.Transport{
        Proxy: http.ProxyURL(proxyURL),
    },
}

err := sender.Send(ctx, url, payload,
    webhook.WithHTTPClient(client),
)
```

## Integration with modules/webhooks

This package provides the low-level delivery mechanism. For a complete webhook management solution with:

- Async delivery via queues
- Webhook endpoint management
- Delivery history and analytics
- Multi-tenant support
- REST API

See the `modules/webhooks` package which builds on top of this foundation.

## Performance Tips

1. **Reuse Senders**: Create once and reuse for better connection pooling
2. **Reuse Circuit Breakers**: One circuit breaker per endpoint
3. **Appropriate Timeouts**: Balance between reliability and resource usage
4. **Batch Webhooks**: For high volume, consider the async modules/webhooks

## Security Considerations

1. **Always use HTTPS** in production
2. **Rotate secrets** regularly
3. **Validate certificates** (default behavior)
4. **Set reasonable timeouts** to prevent resource exhaustion
5. **Use circuit breakers** to prevent cascade failures

## Example: Production Configuration

```go
sender := webhook.NewSender()

// Production-ready webhook send
err := sender.Send(ctx, webhookURL, eventPayload,
    // Security
    webhook.WithSignature(os.Getenv("WEBHOOK_SECRET")),

    // Reliability
    webhook.WithMaxRetries(5),
    webhook.WithTimeout(30 * time.Second),
    webhook.WithExponentialRetry(5, 2*time.Second, 2*time.Minute),

    // Protection
    webhook.WithCircuitBreaker(endpointCircuitBreaker),

    // Observability
    webhook.WithOnDelivery(metricsCollector),

    // Metadata
    webhook.WithHeader("X-Event-Type", event.Type),
    webhook.WithHeader("X-Event-ID", event.ID),
)
```
