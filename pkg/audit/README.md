# audit

Comprehensive audit logging for SaaS applications with tamper detection, compliance features, and pluggable storage backends.

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/audit
```

## Key Features

- **Pluggable Storage**: Interface-based design supports any storage backend
- **Context Extractors**: Automatic tenant, user, and session identification from request context
- **Hash Chaining**: Optional cryptographic tamper detection using SHA-256
- **Compliance Ready**: Immutable timestamps, actor identification, and structured metadata
- **High Performance**: Goroutine-safe with minimal allocations and async-friendly design
- **Functional Options**: Rich metadata attachment using type-safe option patterns

## Architecture Overview

The audit system uses clean separation of concerns across four main components:

```
┌─────────────┐   extract   ┌─────────────┐
│   Context   │ ──────────► │ Extractors  │
└─────────────┘             └─────────────┘
       │                           │
       ▼                           ▼
┌─────────────────────────────────────────┐
│               Logger                    │
└─────────────────────────────────────────┘
       │   create/store
       ▼
┌─────────────┐   persist   ┌─────────────┐
│    Event    │ ──────────► │   Storage   │
└─────────────┘             └─────────────┘
```

- **Logger**: Orchestrates audit logging with configurable context extractors
- **Storage**: Pluggable interface for persisting audit events to any backend
- **Extractors**: Context-aware functions for extracting tenant, user, and session IDs
- **Event**: Immutable audit record structure with optional hash chaining support

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
events, err := reader.Query(ctx, audit.Criteria{
    TenantID:  "tenant-123",
    UserID:    "user-456",
    Action:    "user.login",
    StartTime: time.Now().Add(-24 * time.Hour),
    EndTime:   time.Now(),
    Limit:     100,
})

if err != nil {
    // handle error
}

for _, event := range events {
    fmt.Printf("Action: %s, Result: %s, Time: %v\n", 
        event.Action, event.Result, event.CreatedAt)
}
```

### Hash Chaining Example

```go
// Enable tamper detection with hash chaining
hasher := audit.NewSHA256Hasher()
logger := audit.NewLogger(storage,
    audit.WithHasher(hasher),
    audit.WithTenantIDExtractor(extractTenantID),
)

// Each logged event will include hash of previous event
err := logger.Log(ctx, "sensitive.operation",
    audit.WithResource("accounts", "acc-123"),
    audit.WithMetadata("balance_change", 10000),
)
```

## Storage Implementation Guide

Implement the `audit.Storage` interface for your preferred backend:

```go
type MemoryStorage struct {
    events []audit.Event
    mu     sync.RWMutex
}

func (s *MemoryStorage) Store(ctx context.Context, event *audit.Event) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Generate ID and timestamp if not set
    if event.ID == "" {
        event.ID = uuid.New().String()
    }
    if event.CreatedAt.IsZero() {
        event.CreatedAt = time.Now()
    }
    
    s.events = append(s.events, *event)
    return nil
}

func (s *MemoryStorage) Query(ctx context.Context, criteria audit.Criteria) ([]*audit.Event, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    var results []*audit.Event
    for i := range s.events {
        event := &s.events[i]
        
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
    } else if errors.Is(err, audit.ErrHashVerificationFailed) {
        // Hash chain integrity compromised
        log.Critical("audit hash chain verification failed - possible tampering")
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
- Hash chaining verification across multiple events
- Concurrent logging stress tests
- Error condition simulation

## License

MIT License - see LICENSE file for details.