// Package audit provides comprehensive audit logging capabilities for tracking
// user actions and system events in SaaS applications. It supports compliance,
// security monitoring, and forensic analysis through structured logging with
// async support and cursor-based pagination.
//
// The package is designed as a pure utility with no infrastructure dependencies,
// using pluggable storage backends and configurable context extractors for
// tenant, user, and session identification. All components are goroutine-safe
// and optimized for high-throughput environments.
//
// # Architecture
//
// The audit system uses a clean separation of concerns:
//
//   - Logger – orchestrates audit logging with configurable extractors
//
//   - Storage – pluggable interface for persisting audit records
//
//   - Extractor – context-aware functions for extracting metadata
//
//   - Event – immutable audit event structure with common request metadata
//
//   - Query – flexible interface for retrieving historical audit data
//
//     ┌─────────────┐   extract   ┌─────────────┐
//     │   Context   │ ──────────► │ Extractors  │
//     └─────────────┘             └─────────────┘
//     │                           │
//     ▼                           ▼
//     ┌─────────────────────────────────────────┐
//     │               Logger                    │
//     └─────────────────────────────────────────┘
//     │   create/store
//     ▼
//     ┌─────────────┐   persist   ┌─────────────┐
//     │   Event     │ ──────────► │   Storage   │
//     └─────────────┘             └─────────────┘
//
// # Usage
//
//	import (
//	    "context"
//	    "github.com/dmitrymomot/saaskit/pkg/audit"
//	)
//
//	// Basic logger setup
//	logger := audit.NewLogger(storage,
//	    audit.WithTenantIDExtractor(extractTenantID),
//	    audit.WithUserIDExtractor(extractUserID),
//	    audit.WithSessionIDExtractor(extractSessionID),
//	    audit.WithRequestIDExtractor(extractRequestID),
//	    audit.WithIPExtractor(extractIP),
//	    audit.WithUserAgentExtractor(extractUserAgent),
//	    audit.WithAsync(1000), // Enable async with 1000 event buffer
//	)
//
//	// Log successful actions
//	ctx := context.WithValue(context.Background(), "user_id", "user-123")
//	err := logger.Log(ctx, "user.login",
//	    audit.WithResource("users", "user-123"),
//	    audit.WithMetadata("method", "oauth"),
//	)
//
//	// Log failures with error details
//	err = logger.LogError(ctx, "user.login", someError,
//	    audit.WithResource("users", "user-123"),
//	    audit.WithMetadata("attempt_count", 3),
//	)
//
//	// Query audit logs
//	reader := audit.NewReader(storage)
//	events, err := reader.Find(ctx, audit.Criteria{
//	    TenantID:  "tenant-123",
//	    UserID:    "user-456",
//	    Action:    "user.login",
//	    StartTime: time.Now().Add(-24 * time.Hour),
//	    EndTime:   time.Now(),
//	    Limit:     100,
//	})
//
//	// Query with cursor pagination
//	events, nextCursor, err := reader.FindWithCursor(ctx, audit.Criteria{
//	    TenantID: "tenant-123",
//	    Limit:    20,
//	}, cursor)
//
// # Storage Interface
//
// Custom storage implementations must satisfy the Storage interface:
//
//	type Storage interface {
//	    Store(ctx context.Context, events ...Event) error
//	    Query(ctx context.Context, criteria Criteria) ([]Event, error)
//	}
//
// Example in-memory storage:
//
//	type MemoryStorage struct {
//	    records []audit.Record
//	    mu      sync.RWMutex
//	}
//
//	func (s *MemoryStorage) Store(ctx context.Context, events ...audit.Event) error {
//	    s.mu.Lock()
//	    defer s.mu.Unlock()
//	    s.records = append(s.records, events...)
//	    return nil
//	}
//
// # Context Extractors
//
// Extractors pull metadata from request context for automatic inclusion:
//
//	func extractTenantID(ctx context.Context) (string, bool) {
//	    if tenantID, ok := ctx.Value("tenant_id").(string); ok {
//	        return tenantID, true
//	    }
//	    return "", false
//	}
//
//	logger := audit.NewLogger(storage,
//	    audit.WithTenantIDExtractor(extractTenantID),
//	    audit.WithUserIDExtractor(extractUserID),
//	    audit.WithSessionIDExtractor(extractSessionID),
//	)
//
// # Async Storage
//
// Enable async storage for high-throughput scenarios:
//
//	logger := audit.NewLogger(storage,
//	    audit.WithAsync(5000), // 5000 event buffer
//	)
//
// Events are batched and written asynchronously to improve performance.
// When the buffer is full, the system falls back to synchronous writes
// to prevent event loss.
//
// # Functional Options for Metadata
//
// Rich metadata can be attached to audit records using functional options:
//
//	err := logger.Log(ctx, "document.update",
//	    audit.WithResource("documents", "doc-789"),
//	    audit.WithMetadata("fields_changed", []string{"title", "content"}),
//	    audit.WithMetadata("previous_version", 3),
//	    audit.WithMetadata("new_version", 4),
//	)
//
// # Error Handling
//
// The package defines specific error types for different failure modes:
//
//   - ErrStorageNotAvailable – storage backend is unavailable
//   - ErrInvalidEvent        – event data is invalid
//
// # Performance Considerations
//
// The logger is optimized for high-throughput environments:
//
//   - Context extraction happens once per log call
//   - Event creation uses struct literals to minimize allocations
//   - Async storage batches writes for improved throughput
//   - Cursor pagination enables efficient large dataset navigation
//
// For maximum performance, use WithAsync option to enable batched writes.
//
// # Compliance Features
//
// Audit records include fields commonly required for compliance frameworks:
//
//   - Immutable timestamps with nanosecond precision
//   - Actor identification (user, session, IP address)
//   - Resource identification (type and ID)
//   - Action classification (success/failure/error)
//   - Detailed context and metadata
//   - Request tracking via RequestID, IP, and UserAgent fields
//
// # Security Considerations
//
// - Audit records should be stored in append-only systems when possible
// - Consider encrypting audit storage for sensitive environments
// - Async storage may lose events on ungraceful shutdown - use sync for critical logs
// - IP addresses and user agents may contain personally identifiable information
// - Implement retention policies to comply with data protection regulations
//
// # Examples
//
// See the package tests for additional usage patterns and storage implementations.
package audit
