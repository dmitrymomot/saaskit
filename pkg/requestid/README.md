# Request ID Package

This package provides HTTP request ID tracking middleware and context utilities for tracing requests throughout the application.

## Features

- Generates unique request IDs using UUID v4
- Extracts existing request IDs from incoming requests
- Stores request IDs in context for access throughout the request lifecycle
- Sets request ID in response headers for client correlation

## Usage

### Middleware

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/dmitrymomot/saaskit/pkg/requestid"
)

r := chi.NewRouter()
r.Use(requestid.Middleware)
```

### Context Access

```go
// Get request ID from context
requestID := requestid.FromContext(ctx)

// Set request ID in context (typically done by middleware)
ctx = requestid.WithContext(ctx, requestID)
```

### HTTP Header

The middleware uses the standard `X-Request-ID` header for both incoming and outgoing requests.

## Integration

This middleware should be applied early in the middleware chain to ensure all subsequent handlers and middleware have access to the request ID.