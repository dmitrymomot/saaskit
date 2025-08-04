// Package audit provides a comprehensive audit logging system for SaaS applications
// with flexible storage backends, automatic context extraction, and high-throughput
// asynchronous processing capabilities.
//
// # Architecture
//
// The package is built around several key components that work together to provide
// flexible and efficient audit logging:
//
//   - Event: Core audit event structure with comprehensive metadata
//   - Logger: Synchronous logger with context extraction capabilities
//   - AsyncLogger: High-throughput asynchronous logger with batching
//   - Writer interfaces: Pluggable storage backends (writer, batchWriter)
//   - MetadataFilter: Configurable PII and sensitive data filtering system
//   - AsyncOptions: Configuration for batching and buffering behavior
//   - Result constants: Standard result values (ResultSuccess, ResultFailure, ResultError)
//
// The design emphasizes flexibility and performance while maintaining audit integrity.
// Only the Action field is required for events, allowing applications to adopt audit
// logging incrementally without major architectural changes.
//
// # Basic Usage
//
// Create a logger with a storage backend and start logging events:
//
//	// Implement the writer interface for your storage backend
//	type DatabaseWriter struct {
//		db *sql.DB
//	}
//
//	func (w *DatabaseWriter) Store(ctx context.Context, event audit.Event) error {
//		query := `INSERT INTO audit_events (id, action, tenant_id, user_id, created_at)
//		          VALUES ($1, $2, $3, $4, $5)`
//		_, err := w.db.ExecContext(ctx, query, event.ID, event.Action,
//			event.TenantID, event.UserID, event.CreatedAt)
//		return err
//	}
//
//	// Create logger with context extractors
//	writer := &DatabaseWriter{db: db}
//	logger := audit.NewLogger(writer,
//		audit.WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
//			if tenantID := ctx.Value("tenant_id"); tenantID != nil {
//				return tenantID.(string), true
//			}
//			return "", false
//		}),
//		audit.WithUserIDExtractor(func(ctx context.Context) (string, bool) {
//			if userID := ctx.Value("user_id"); userID != nil {
//				return userID.(string), true
//			}
//			return "", false
//		}),
//	)
//
//	// Log successful actions
//	err := logger.Log(ctx, "user.login",
//		audit.WithResource("user", "123"),
//		audit.WithMetadata("login_method", "password"),
//	)
//
//	// Log failed actions
//	err := logger.LogError(ctx, "user.login", loginErr,
//		audit.WithResource("user", "123"),
//		audit.WithMetadata("failure_reason", "invalid_password"),
//	)
//
// # High-Throughput Scenarios
//
// For applications with high audit event volumes, use AsyncLogger with batch processing:
//
//	// Implement batch writer for bulk operations
//	type BatchDatabaseWriter struct {
//		db *sql.DB
//	}
//
//	func (w *BatchDatabaseWriter) StoreBatch(ctx context.Context, events []audit.Event) error {
//		tx, err := w.db.BeginTx(ctx, nil)
//		if err != nil {
//			return err
//		}
//		defer tx.Rollback()
//
//		stmt, err := tx.PrepareContext(ctx,
//			`INSERT INTO audit_events (id, action, tenant_id, user_id, created_at)
//			 VALUES ($1, $2, $3, $4, $5)`)
//		if err != nil {
//			return err
//		}
//		defer stmt.Close()
//
//		for _, event := range events {
//			_, err := stmt.ExecContext(ctx, event.ID, event.Action,
//				event.TenantID, event.UserID, event.CreatedAt)
//			if err != nil {
//				return err
//			}
//		}
//
//		return tx.Commit()
//	}
//
//	// Create async logger with cleanup function
//	batchWriter := &BatchDatabaseWriter{db: db}
//	logger, cleanup := audit.NewAsyncLogger(batchWriter, 1000, /* options */)
//	defer cleanup(context.Background()) // Always call during shutdown
//
//	// Usage remains the same as synchronous logger
//	err := logger.Log(ctx, "data.export",
//		audit.WithResource("dataset", "456"),
//		audit.WithMetadata("export_format", "csv"),
//	)
//
// # Context Integration
//
// The package integrates seamlessly with Go's context.Context to automatically
// extract audit metadata from HTTP requests or other contextual information:
//
//	// HTTP middleware to populate audit context
//	func auditMiddleware(next http.Handler) http.Handler {
//		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			ctx := r.Context()
//
//			// Extract tenant from JWT or session
//			if tenantID := extractTenantID(r); tenantID != "" {
//				ctx = context.WithValue(ctx, "tenant_id", tenantID)
//			}
//
//			// Extract user from authentication
//			if userID := extractUserID(r); userID != "" {
//				ctx = context.WithValue(ctx, "user_id", userID)
//			}
//
//			// Add request metadata
//			ctx = context.WithValue(ctx, "request_id", r.Header.Get("X-Request-ID"))
//			ctx = context.WithValue(ctx, "ip", r.RemoteAddr)
//			ctx = context.WithValue(ctx, "user_agent", r.UserAgent())
//
//			next.ServeHTTP(w, r.WithContext(ctx))
//		})
//	}
//
//	// Configure logger to extract from context
//	logger := audit.NewLogger(writer,
//		audit.WithTenantIDExtractor(func(ctx context.Context) (string, bool) {
//			val := ctx.Value("tenant_id")
//			if val == nil {
//				return "", false
//			}
//			return val.(string), true
//		}),
//		audit.WithUserIDExtractor(func(ctx context.Context) (string, bool) {
//			val := ctx.Value("user_id")
//			if val == nil {
//				return "", false
//			}
//			return val.(string), true
//		}),
//		audit.WithRequestIDExtractor(func(ctx context.Context) (string, bool) {
//			val := ctx.Value("request_id")
//			if val == nil {
//				return "", false
//			}
//			return val.(string), true
//		}),
//		audit.WithIPExtractor(func(ctx context.Context) (string, bool) {
//			val := ctx.Value("ip")
//			if val == nil {
//				return "", false
//			}
//			return val.(string), true
//		}),
//		audit.WithUserAgentExtractor(func(ctx context.Context) (string, bool) {
//			val := ctx.Value("user_agent")
//			if val == nil {
//				return "", false
//			}
//			return val.(string), true
//		}),
//	)
//
// # Event Customization
//
// Events can be enriched with metadata and resource information using EventOption functions:
//
//	// Track file operations with metadata
//	err := logger.Log(ctx, "file.upload",
//		audit.WithResource("file", fileID),
//		audit.WithMetadata("filename", "document.pdf"),
//		audit.WithMetadata("size_bytes", 2048576),
//		audit.WithMetadata("content_type", "application/pdf"),
//	)
//
//	// Override default result for business logic
//	err := logger.Log(ctx, "payment.process",
//		audit.WithResource("payment", paymentID),
//		audit.WithResult(audit.ResultFailure), // Override success default
//		audit.WithMetadata("reason", "insufficient_funds"),
//	)
//
//	// Multiple metadata entries
//	err := logger.Log(ctx, "api.request",
//		audit.WithResource("endpoint", "/api/v1/users"),
//		audit.WithMetadata("method", "POST"),
//		audit.WithMetadata("response_time_ms", 234),
//		audit.WithMetadata("status_code", 201),
//	)
//
// # Metadata Filtering and Security
//
// The package provides built-in protection against logging sensitive data through the
// MetadataFilter system. By default, common PII and security-sensitive fields are
// automatically filtered using configurable actions:
//
//	// Create filter with default PII protection
//	filter := audit.NewMetadataFilter()
//
//	// Customize filtering behavior
//	filter := audit.NewMetadataFilter(
//		audit.WithCustomField("internal_id", audit.FilterActionHash),
//		audit.WithAllowedField("email"), // Override default email hashing
//		audit.WithoutPIIDefaults(),     // Disable all default filtering
//	)
//
//	// Apply filter to logger
//	logger := audit.NewLogger(writer, audit.WithMetadataFilter(filter))
//
// The filter supports three actions:
//
//   - FilterActionRemove: Completely removes the field (for secrets, passwords)
//   - FilterActionHash: SHA-256 hashes the value (for searchable PII like emails)
//   - FilterActionMask: Masks the value with asterisks (for display purposes)
//
// Default PII fields include passwords, tokens, SSNs, credit cards, and other
// sensitive data commonly found in application logs.
//
// # Error Handling
//
// The package provides structured error handling for different failure scenarios:
//
//	err := logger.Log(ctx, "user.create", audit.WithResource("user", userID))
//	if err != nil {
//		switch {
//		case errors.Is(err, audit.ErrEventValidation):
//			// Handle validation errors (missing required fields)
//			log.Printf("Invalid audit event: %v", err)
//		case errors.Is(err, audit.ErrStorageNotAvailable):
//			// Handle storage backend unavailable
//			log.Printf("Audit storage unavailable: %v", err)
//		case errors.Is(err, audit.ErrStorageTimeout):
//			// Handle storage operation timeouts
//			log.Printf("Audit storage timeout: %v", err)
//		default:
//			// Handle other storage errors
//			log.Printf("Audit logging failed: %v", err)
//		}
//	}
//
// # Performance Considerations
//
// The async logger is designed for high-throughput scenarios with several optimizations:
//
//   - Batching: Events are collected and written in batches to reduce I/O operations
//   - Buffering: In-memory buffer prevents blocking on temporary storage slowdowns
//   - Fallback: When buffer is full, operations fall back to synchronous writes to prevent event loss
//   - Isolation: Storage operations use background context to prevent client timeout cascades
//
// Configure AsyncOptions based on your workload characteristics:
//
//	// High-volume, latency-tolerant workload
//	asyncWriter, cleanup := audit.NewAsyncWriter(batchWriter, audit.AsyncOptions{
//		BufferSize:     10000,             // Large buffer for burst capacity
//		BatchSize:      500,               // Large batches for efficiency
//		BatchTimeout:   500 * time.Millisecond, // Higher latency acceptable
//		StorageTimeout: 10 * time.Second,  // Allow slow storage operations
//	})
//
//	// Low-latency, moderate-volume workload
//	asyncWriter, cleanup := audit.NewAsyncWriter(batchWriter, audit.AsyncOptions{
//		BufferSize:     1000,              // Smaller buffer
//		BatchSize:      50,                // Smaller batches
//		BatchTimeout:   50 * time.Millisecond,  // Low latency requirement
//		StorageTimeout: 2 * time.Second,   // Quick storage operations
//	})
//
// # Compliance and Security
//
// The audit system is designed with compliance and security requirements in mind:
//
//   - Immutable Events: Once created, events should not be modified by application code
//   - Complete Context: Automatic extraction ensures consistent metadata collection
//   - Failure Handling: Async writer falls back to sync writes to prevent audit event loss
//   - Structured Data: JSON-serializable events support compliance reporting and analysis
//
// For compliance scenarios, ensure your storage backend provides:
//
//   - Tamper-proof storage (append-only logs, write-once storage)
//   - Retention policies aligned with regulatory requirements
//   - Access controls preventing unauthorized modification
//   - Backup and recovery procedures for audit data
//
// # Integration Patterns
//
// Common integration patterns for different application architectures:
//
// Web Applications with Middleware:
//
//	// Global audit middleware
//	app.Use(auditMiddleware(logger))
//
//	// Handler-specific logging
//	func createUserHandler(logger *audit.Logger) http.HandlerFunc {
//		return func(w http.ResponseWriter, r *http.Request) {
//			// ... business logic ...
//			logger.Log(r.Context(), "user.create",
//				audit.WithResource("user", user.ID),
//				audit.WithMetadata("email", user.Email),
//			)
//		}
//	}
//
// Microservices with gRPC:
//
//	// gRPC interceptor for audit logging
//	func auditInterceptor(logger *audit.Logger) grpc.UnaryServerInterceptor {
//		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
//			handler grpc.UnaryHandler) (any, error) {
//
//			resp, err := handler(ctx, req)
//
//			if err != nil {
//				logger.LogError(ctx, info.FullMethod, err,
//					audit.WithMetadata("grpc_method", info.FullMethod),
//				)
//			} else {
//				logger.Log(ctx, info.FullMethod,
//					audit.WithMetadata("grpc_method", info.FullMethod),
//				)
//			}
//
//			return resp, err
//		}
//	}
//
// Background Jobs and Workers:
//
//	func processJob(ctx context.Context, logger *audit.Logger, job Job) error {
//		// Add job context for audit logging
//		ctx = context.WithValue(ctx, "tenant_id", job.TenantID)
//		ctx = context.WithValue(ctx, "user_id", job.UserID)
//
//		err := job.Execute(ctx)
//		if err != nil {
//			logger.LogError(ctx, "job.failed", err,
//				audit.WithResource("job", job.ID),
//				audit.WithMetadata("job_type", job.Type),
//			)
//			return err
//		}
//
//		logger.Log(ctx, "job.completed",
//			audit.WithResource("job", job.ID),
//			audit.WithMetadata("job_type", job.Type),
//			audit.WithMetadata("duration_ms", job.Duration().Milliseconds()),
//		)
//
//		return nil
//	}
package audit
