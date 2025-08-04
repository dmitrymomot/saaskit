# audit

Comprehensive audit logging system for SaaS applications with flexible storage backends and high-throughput asynchronous processing.

## Features

- Synchronous and asynchronous audit logging with batching optimization
- Automatic context extraction from HTTP requests and other sources
- Pluggable storage backends via Writer and BatchWriter interfaces
- Complete event metadata capture (tenant, user, session, IP, user agent)
- Structured error handling with domain-specific error types

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/audit
```

## Usage

```go
package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/dmitrymomot/saaskit/pkg/audit"
	_ "github.com/lib/pq"
)

// DatabaseWriter implements the Writer interface for PostgreSQL
type DatabaseWriter struct {
	db *sql.DB
}

func (w *DatabaseWriter) Store(ctx context.Context, event audit.Event) error {
	query := `INSERT INTO audit_events (id, action, tenant_id, user_id, created_at)
	          VALUES ($1, $2, $3, $4, $5)`
	_, err := w.db.ExecContext(ctx, query, event.ID, event.Action,
		event.TenantID, event.UserID, event.CreatedAt)
	return err
}

func main() {
	db, err := sql.Open("postgres", "postgres://user:pass@localhost/db?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create logger with context extractors
	writer := &DatabaseWriter{db: db}
	logger := audit.NewLogger(writer,
		audit.WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
			if tenantID := ctx.Value("tenant_id"); tenantID != nil {
				return tenantID.(string), true
			}
			return "", false
		}),
		audit.WithUserIDExtractor(func(ctx context.Context) (string, bool) {
			if userID := ctx.Value("user_id"); userID != nil {
				return userID.(string), true
			}
			return "", false
		}),
	)

	// Create context with audit information
	ctx := context.Background()
	ctx = context.WithValue(ctx, "tenant_id", "tenant-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")

	// Log successful action
	err = logger.Log(ctx, "user.login",
		audit.WithResource("user", "456"),
		audit.WithMetadata("login_method", "password"),
	)
	if err != nil {
		log.Printf("Failed to log audit event: %v", err)
	}
}
```

## Common Operations

### Synchronous Logging

```go
// Log successful action
err := logger.Log(ctx, "user.create",
	audit.WithResource("user", userID),
	audit.WithMetadata("email", user.Email),
)

// Log failed action
err := logger.LogError(ctx, "user.login", loginErr,
	audit.WithResource("user", userID),
	audit.WithMetadata("failure_reason", "invalid_password"),
)
```

### High-Throughput Asynchronous Logging

```go
// BatchDatabaseWriter implements BatchWriter for bulk operations
type BatchDatabaseWriter struct {
	db *sql.DB
}

func (w *BatchDatabaseWriter) StoreBatch(ctx context.Context, events []audit.Event) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO audit_events (id, action, tenant_id, user_id, created_at)
		 VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, event := range events {
		_, err := stmt.ExecContext(ctx, event.ID, event.Action,
			event.TenantID, event.UserID, event.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Create async logger
batchWriter := &BatchDatabaseWriter{db: db}
logger, cleanup := audit.NewAsyncLogger(batchWriter, 1000)
defer cleanup(context.Background())

// Usage is identical to synchronous logger
err := logger.Log(ctx, "data.export",
	audit.WithResource("dataset", "789"),
	audit.WithMetadata("export_format", "csv"),
)
```

## Error Handling

```go
err := logger.Log(ctx, "user.create", audit.WithResource("user", userID))
if err != nil {
	switch {
	case errors.Is(err, audit.ErrEventValidation):
		log.Printf("Invalid audit event: %v", err)
	case errors.Is(err, audit.ErrStorageNotAvailable):
		log.Printf("Audit storage unavailable: %v", err)
	case errors.Is(err, audit.ErrStorageTimeout):
		log.Printf("Audit storage timeout: %v", err)
	default:
		log.Printf("Audit logging failed: %v", err)
	}
}
```

## Configuration

### Async Writer Options

```go
// High-volume configuration
asyncWriter, cleanup := audit.NewAsyncWriter(batchWriter, audit.AsyncOptions{
	BufferSize:     10000,                    // Large buffer for burst capacity
	BatchSize:      500,                      // Large batches for efficiency
	BatchTimeout:   500 * time.Millisecond,  // Higher latency acceptable
	StorageTimeout: 10 * time.Second,        // Allow slow storage operations
})

// Low-latency configuration
asyncWriter, cleanup := audit.NewAsyncWriter(batchWriter, audit.AsyncOptions{
	BufferSize:     1000,                    // Smaller buffer
	BatchSize:      50,                      // Smaller batches
	BatchTimeout:   50 * time.Millisecond,  // Low latency requirement
	StorageTimeout: 2 * time.Second,        // Quick storage operations
})
```

### Event Customization

```go
// Multiple metadata entries
err := logger.Log(ctx, "api.request",
	audit.WithResource("endpoint", "/api/v1/users"),
	audit.WithMetadata("method", "POST"),
	audit.WithMetadata("response_time_ms", 234),
	audit.WithMetadata("status_code", 201),
	audit.WithResult(audit.ResultSuccess), // Override default result
)
```

## API Documentation

For detailed API documentation:

```bash
go doc -all ./pkg/audit
```

Or visit [pkg.go.dev](https://pkg.go.dev/github.com/dmitrymomot/saaskit/pkg/audit) for online documentation.

## Notes

- Only the Action field is required for audit events - all other fields are optional
- AsyncLogger falls back to synchronous writes when buffer is full to prevent event loss
- Always call the cleanup function returned by NewAsyncLogger during application shutdown
- Storage operations in async mode use background context to prevent client timeout cascades
- Events are JSON-serializable for compliance reporting and analysis
