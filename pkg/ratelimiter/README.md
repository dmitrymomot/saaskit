# ratelimiter

Token bucket rate limiter with in-memory storage, HTTP middleware, and configurable options.

## Features

- Token bucket algorithm with configurable capacity and refill rate
- In-memory store with automatic cleanup of stale buckets
- HTTP middleware with standard rate limit headers
- Composite key functions for complex rate limiting scenarios
- Thread-safe operations with proper error handling

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/ratelimiter
```

## Usage

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/dmitrymomot/saaskit/pkg/ratelimiter"
)

func main() {
    // Create rate limiter configuration
    config := ratelimiter.Config{
        Capacity:       100,           // 100 requests max burst
        RefillRate:     10,            // 10 tokens per interval
        RefillInterval: time.Second,   // Refill every second
    }

    // Create memory store with cleanup
    store := ratelimiter.NewMemoryStore()
    defer store.Close()

    // Create token bucket limiter
    limiter, err := ratelimiter.NewBucket(store, config)
    if err != nil {
        log.Fatal(err)
    }

    // Use with HTTP middleware
    keyFunc := func(r *http.Request) string {
        return r.RemoteAddr
    }

    middleware := ratelimiter.Middleware(limiter, keyFunc)

    handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    }))

    http.ListenAndServe(":8080", handler)
}
```

## Common Operations

### Direct Rate Limiting

```go
// Check if request is allowed
result, err := limiter.Allow(ctx, "user:123")
if err != nil {
    // Handle error
}

if !result.Allowed() {
    // Rate limit exceeded
    retryAfter := result.RetryAfter()
}
```

### Bulk Token Consumption

```go
// Consume multiple tokens at once
result, err := limiter.AllowN(ctx, "user:123", 5)
if err != nil {
    // Handle error
}
```

### Status Check Without Consumption

```go
// Check current bucket state without consuming tokens
result, err := limiter.Status(ctx, "user:123")
if err != nil {
    // Handle error
}

remaining := result.Remaining
resetAt := result.ResetAt
```

### Composite Key Functions

```go
// Combine multiple key extractors
keyFunc := ratelimiter.Composite(
    func(r *http.Request) string { return r.Header.Get("X-API-Key") },
    func(r *http.Request) string { return r.RemoteAddr },
)

middleware := ratelimiter.Middleware(limiter, keyFunc)
```

## Error Handling

```go
import "errors"

result, err := limiter.Allow(ctx, key)
if err != nil {
    if errors.Is(err, ratelimiter.ErrInvalidTokenCount) {
        // Invalid token count
    } else if errors.Is(err, ratelimiter.ErrInvalidConfig) {
        // Invalid configuration
    } else if errors.Is(err, ratelimiter.ErrStoreUnavailable) {
        // Store backend unavailable
    }
}
```

## Configuration

### Memory Store Options

```go
store := ratelimiter.NewMemoryStore(
    ratelimiter.WithCleanupInterval(10 * time.Minute), // Custom cleanup interval
)

// Disable cleanup
store := ratelimiter.NewMemoryStore(
    ratelimiter.WithCleanupInterval(0),
)
```

### Custom Error Responder

```go
errorResponder := func(w http.ResponseWriter, r *http.Request, result *ratelimiter.Result, err error) {
    if err != nil {
        http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
        return
    }

    if result != nil && !result.Allowed() {
        w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter().Seconds())))
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
    }
}

middleware := ratelimiter.Middleware(limiter, keyFunc,
    ratelimiter.WithErrorResponder(errorResponder),
)
```

## API Documentation

For detailed API documentation:

```bash
go doc -all ./pkg/ratelimiter
```

Or visit [pkg.go.dev](https://pkg.go.dev/github.com/dmitrymomot/saaskit/pkg/ratelimiter) for online documentation.

## Notes

- Memory store automatically cleans up stale buckets (default: 5 minutes interval, 1 hour threshold)
- Keys are automatically hashed when they exceed 64 characters to prevent unbounded storage growth
- HTTP middleware adds standard rate limit headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
- Thread-safe for concurrent use across multiple goroutines
