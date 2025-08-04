# audit

Comprehensive audit logging for SaaS applications with compliance features, pluggable storage backends, and async support.

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/audit
```

## Key Features

- **Pluggable Storage**: Interface-based design supports any storage backend
- **Context Extractors**: Automatic tenant, user, session, request ID, IP, and user agent extraction
- **Async Support**: Optional async storage with configurable buffer sizes
- **Cursor Pagination**: Efficient pagination with cursor-based navigation
- **Compliance Ready**: Immutable timestamps, actor identification, and structured metadata
- **High Performance**: Goroutine-safe with minimal allocations and batch processing
- **Functional Options**: Rich metadata attachment using type-safe option patterns

## Architecture Overview

The audit system uses clean separation of concerns across four main components:

```
┌────────┐   extract   ┌────────┐
│   Context   │ ──────► │  Extractors │
└────────┘             └────────┘
       │                           │
       ▼                           ▼
┌─────────────────────────┐
│                   Logger                 │
└─────────────────────────┘
       │   create/store
       ▼
┌────────┐   persist   ┌────────┐
│    Event    │ ──────► │   Storage   │
└────────┘             └────────┘
```

- **Logger**: Orchestrates audit logging with configurable context extractors
- **Storage**: Pluggable interface for persisting audit events to any backend
- **Extractors**: Context-aware functions for extracting tenant, user, and session IDs
- **Event**: Immutable audit record structure with common fields (RequestID, IP, UserAgent)

## Usage

### Basic Logger Setup

```go
package main

import (
    "context"
    "github.com/dmitrymomot/saaskit/pkg/audit"
)

func main() {
    // Create storage (implement audit.Storage interface)
    storage := NewMemoryStorage() // your storage implementation

    // Configure logger with context extractors
    logger := audit.NewLogger(storage,
        audit.WithTenantIDExtractor(extractTenantID),
        audit.WithUserIDExtractor(extractUserID),
        audit.WithSessionIDExtractor(extractSessionID),
        audit.WithRequestIDExtractor(extractRequestID),
        audit.WithIPExtractor(extractIP),
        audit.WithUserAgentExtractor(extractUserAgent),
        audit.WithAsync(1000), // Enable async with 1000 event buffer
    )

    // Context with user information
    ctx := context.WithValue(context.Background(), "user_id", "user-123")

    // Log successful action
    err := logger.Log(ctx, "user.login",
        audit.WithResource("users", "user-123"),
        audit.WithMetadata("ip", "192.168.1.1"),
        audit.WithMetadata("user_agent", "Mozilla/5.0..."),
    )
    if err != nil {
        // handle error
    }
}
```

### Context Extractors Configuration

```go
// Extract tenant ID from request context
func extractTenantID(ctx context.Context) (string, bool) {
    if tenantID, ok := ctx.Value("tenant_id").(string); ok {
        return tenantID, true
    }
    return "", false
}

// Extract user ID from JWT claims or session
func extractUserID(ctx context.Context) (string, bool) {
    if userID, ok := ctx.Value("user_id").(string); ok {
        return userID, true
    }
    return "", false
}

// Extract session ID from context
func extractSessionID(ctx context.Context) (string, bool) {
    if sessionID, ok := ctx.Value("session_id").(string); ok {
        return sessionID, true
    }
    return "", false
}

// Extract request ID for tracing
func extractRequestID(ctx context.Context) (string, bool) {
    if reqID, ok := ctx.Value("request_id").(string); ok {
        return reqID, true
    }
    return "", false
}

// Extract client IP address
func extractIP(ctx context.Context) (string, bool) {
    if ip, ok := ctx.Value("client_ip").(string); ok {
        return ip, true
    }
    return "", false
}

// Extract user agent string
func extractUserAgent(ctx context.Context) (string, bool) {
    if ua, ok := ctx.Value("user_agent").(string); ok {
        return ua, true
    }
    return "", false
}
```

### Logging Actions and Errors

```go
// Log successful operations
err := logger.Log(ctx, "document.update",
    audit.WithResource("documents", "doc-789"),
    audit.WithMetadata("fields_changed", []string{"title", "content"}),
    audit.WithMetadata("previous_version", 3),
    audit.WithMetadata("new_version", 4),
)

// Log failures with error details
err = logger.LogError(ctx, "payment.process", someError,
    audit.WithResource("payments", "pay-456"),
    audit.WithMetadata("amount", 2500),
    audit.WithMetadata("currency", "USD"),
)
```

### Using Functional Options

```go
// Rich metadata with various data types
err := logger.Log(ctx, "api.access",
    audit.WithResource("endpoints", "/api/v1/users"),
    audit.WithMetadata("method", "POST"),
    audit.WithMetadata("response_time_ms", 245),
    audit.WithMetadata("response_size", 1024),
    audit.WithMetadata("rate_limited", false),
)
```

### Querying Audit Logs

```go
// Create reader for querying
reader := audit.NewReader(storage)

// Query with filters
events, err := reader.Find(ctx, audit.Criteria{
    TenantID:  "tenant-123",
    UserID:    "user-456",
    Action:    "user.login",
    StartTime: time.Now().Add(-24 * time.Hour),
    EndTime:   time.Now(),
    Limit:     100,
})

// Query with cursor-based pagination
events, nextCursor, err := reader.FindWithCursor(ctx, audit.Criteria{
    TenantID: "tenant-123",
    Limit:    20,
}, cursor)

// Get count of matching events
count, err := reader.Count(ctx, audit.Criteria{
    TenantID: "tenant-123",
    Action:   "user.login",
})

if err != nil {
    // handle error
}

for _, event := range events {
    fmt.Printf("Action: %s, Result: %s, Time: %v\n",
        event.Action, event.Result, event.CreatedAt)
}
```

### Async Storage

```go
// Enable async storage with a 5000 event buffer
logger := audit.NewLogger(storage,
    audit.WithAsync(5000),
    audit.WithTenantIDExtractor(extractTenantID),
)

// Events are written asynchronously in batches
err := logger.Log(ctx, "high.volume.operation",
    audit.WithResource("requests", "req-123"),
    audit.WithMetadata("processing_time_ms", 15),
)

// Note: Async storage falls back to synchronous writes

// Remember to close the logger for graceful shutdown
if closer, ok := logger.(io.Closer); ok {
    defer closer.Close()
}
```

**Important Async Behavior**: 
- When the async buffer is full, the logger automatically falls back to synchronous writes to prevent event loss
- Events are never dropped - reliability is prioritized over performance
- Events are batched (up to 100 events or 100ms timeout) for efficient database writes
- Always close the logger to ensure pending events are flushed
// when the buffer is full to prevent event loss
```

## Storage Implementation Guide

Implement the `audit.Storage` interface for your preferred backend:

```go
type MemoryStorage struct {
    events []audit.Event
    mu     sync.RWMutex
}

func (s *MemoryStorage) Store(ctx context.Context, events ...audit.Event) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    for _, event := range events {
        // Events already have ID and timestamp from logger
        s.events = append(s.events, event)
    }
    return nil
}

func (s *MemoryStorage) Query(ctx context.Context, criteria audit.Criteria) ([]audit.Event, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var results []audit.Event
    
    // Apply offset
    start := criteria.Offset
    if start >= len(s.events) {
        return results, nil
    }
    
    for i := start; i < len(s.events); i++ {
        event := s.events[i]

        // Apply filters
        if criteria.TenantID != "" && event.TenantID != criteria.TenantID {
            continue
        }
        if criteria.UserID != "" && event.UserID != criteria.UserID {
            continue
        }
        if criteria.Action != "" && event.Action != criteria.Action {
            continue
        }
        // Apply time filters
        if !criteria.StartTime.IsZero() && event.CreatedAt.Before(criteria.StartTime) {
            continue
        }
        if !criteria.EndTime.IsZero() && event.CreatedAt.After(criteria.EndTime) {
            continue
        }

        results = append(results, event)

        // Apply limit
        if criteria.Limit > 0 && len(results) >= criteria.Limit {
            break
        }
    }

    return results, nil
}
```

## Context Integration

The audit package integrates seamlessly with Go's context system:

```go
// Middleware to add audit context
func auditMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Extract from JWT, session, headers, etc.
        if tenantID := extractFromJWT(r); tenantID != "" {
            ctx = context.WithValue(ctx, "tenant_id", tenantID)
        }

        if userID := extractFromSession(r); userID != "" {
            ctx = context.WithValue(ctx, "user_id", userID)
        }

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Error Handling

The package defines specific error types for different failure modes:

```go
import "errors"

// Check for specific audit errors
if err := logger.Log(ctx, "action"); err != nil {
    if errors.Is(err, audit.ErrStorageNotAvailable) {
        // Storage backend is down
        log.Error("audit storage unavailable, events may be lost")
    } else if errors.Is(err, audit.ErrInvalidEvent) {
        // Event validation failed
        log.Error("invalid audit event data")
    }
}
```

## Testing Approach

The package is designed for easy testing with mock storage implementations:

```bash
go test ./pkg/audit -v -race -cover
```

Key testing patterns:

- Mock storage implementations for unit tests
- Context extractor testing with various context values
- Async storage behavior with buffer overflow scenarios
- Concurrent logging stress tests
- Error condition simulation

## License

MIT License - see LICENSE file for details.
