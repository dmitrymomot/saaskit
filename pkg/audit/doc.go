// Package audit provides comprehensive audit logging capabilities for tracking
// user actions and system events in SaaS applications. It supports compliance,
// security monitoring, and forensic analysis through structured logging with
// optional tamper detection via hash chaining.
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
//   - Record – immutable audit event structure with optional hash chaining
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
//     │   Record    │ ──────────► │   Storage   │
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
//	logger := audit.New(
//	    storage,  // implements audit.Storage interface
//	    audit.WithTenantExtractor(extractTenantID),
//	    audit.WithUserExtractor(extractUserID),
//	    audit.WithSessionExtractor(extractSessionID),
//	)
//
//	// Log successful actions
//	ctx := context.WithValue(context.Background(), "user_id", "user-123")
//	err := logger.LogSuccess(ctx, "user.login", "User logged in successfully",
//	    audit.WithIP("192.168.1.1"),
//	    audit.WithUserAgent("Mozilla/5.0..."),
//	    audit.WithResource("users", "user-123"),
//	)
//
//	// Log failures with error details
//	err = logger.LogError(ctx, "user.login", "Invalid credentials",
//	    someError,
//	    audit.WithIP("192.168.1.1"),
//	    audit.WithAttemptCount(3),
//	)
//
//	// Query audit logs
//	records, err := storage.Query(ctx, audit.QueryOptions{
//	    TenantID: "tenant-123",
//	    UserID:   "user-456",
//	    Action:   "user.login",
//	    Since:    time.Now().Add(-24 * time.Hour),
//	    Limit:    100,
//	})
//
// # Storage Interface
//
// Custom storage implementations must satisfy the Storage interface:
//
//	type Storage interface {
//	    Store(ctx context.Context, record Record) error
//	    Query(ctx context.Context, opts QueryOptions) ([]Record, error)
//	}
//
// Example in-memory storage:
//
//	type MemoryStorage struct {
//	    records []audit.Record
//	    mu      sync.RWMutex
//	}
//
//	func (s *MemoryStorage) Store(ctx context.Context, record audit.Record) error {
//	    s.mu.Lock()
//	    defer s.mu.Unlock()
//	    s.records = append(s.records, record)
//	    return nil
//	}
//
// # Context Extractors
//
// Extractors pull metadata from request context for automatic inclusion:
//
//	func extractTenantID(ctx context.Context) string {
//	    if tenantID, ok := ctx.Value("tenant_id").(string); ok {
//	        return tenantID
//	    }
//	    return ""
//	}
//
//	logger := audit.New(storage,
//	    audit.WithTenantExtractor(extractTenantID),
//	    audit.WithUserExtractor(extractUserID),
//	    audit.WithSessionExtractor(extractSessionID),
//	)
//
// # Hash Chaining for Tamper Detection
//
// Optional hash chaining creates cryptographic links between audit records:
//
//	logger := audit.New(storage,
//	    audit.WithHashChaining(true),
//	)
//
// Each record includes the hash of the previous record, enabling detection
// of missing or modified entries. The chain uses SHA-256 for cryptographic
// integrity.
//
// # Functional Options for Metadata
//
// Rich metadata can be attached to audit records using functional options:
//
//	err := logger.LogSuccess(ctx, "document.update", "Document updated",
//	    audit.WithResource("documents", "doc-789"),
//	    audit.WithIP("203.0.113.1"),
//	    audit.WithUserAgent("API-Client/1.0"),
//	    audit.WithMetadata(map[string]interface{}{
//	        "fields_changed": []string{"title", "content"},
//	        "previous_version": 3,
//	        "new_version": 4,
//	    }),
//	)
//
// # Error Handling
//
// The package defines specific error types for different failure modes:
//
//   - ErrStorageUnavailable – storage backend is unreachable
//   - ErrInvalidRecord      – record validation failed
//   - ErrHashChainBroken    – tamper detection found inconsistency
//   - ErrQueryLimitExceeded – query would return too many results
//
// # Performance Considerations
//
// The logger is optimized for high-throughput environments:
//
//   - Context extraction happens once per log call
//   - Record creation uses struct literals to minimize allocations
//   - Storage operations can be made asynchronous via background workers
//   - Hash computation uses efficient SHA-256 implementation
//
// For maximum performance, consider batching storage operations or using
// an asynchronous storage implementation that buffers writes.
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
//   - Optional tamper detection via hash chaining
//
// # Security Considerations
//
// - Audit records should be stored in append-only systems when possible
// - Consider encrypting audit storage for sensitive environments
// - Hash chaining provides tamper detection but not prevention
// - IP addresses and user agents may contain personally identifiable information
// - Implement retention policies to comply with data protection regulations
//
// # Examples
//
// See the package tests for additional usage patterns and storage implementations.
package audit
