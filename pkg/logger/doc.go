// Package logger provides a context-aware wrapper around Go's slog package
// with zero-config defaults and framework-level consistency for SaaS applications.
//
// # Design Philosophy
//
// This package prioritizes explicitness over magic, with fail-fast initialization
// to prevent misconfigured logging from reaching production. The decorator pattern
// minimizes allocations in hot paths while enabling automatic context injection.
//
// # Architecture
//
// Logger uses a decorator pattern to wrap slog.Handler implementations:
//
//  1. New() selects TextHandler (development) or JSONHandler (production)
//  2. LogHandlerDecorator wraps the handler to inject context attributes
//  3. Context extraction happens per-log-call to ensure fresh request-scoped values
//
// This design adds ~50ns overhead per log call but eliminates manual context
// attribute extraction in application code.
//
// # Standardized Attributes
//
// Helper constructors in attr.go enforce consistent naming across microservices:
// user_id, workspace_id, request_id, etc. These helpers return empty slog.Attr
// for nil values, enabling clean logging without explicit nil checks.
//
// # Environment Conventions
//
// Production environments default to JSON format for log aggregation compatibility
// and INFO level to reduce noise. Development uses human-readable text format
// with DEBUG level for detailed troubleshooting.
//
// # Usage
//
//	import "github.com/dmitrymomot/saaskit/pkg/logger"
//
//	func main() {
//	    log := logger.New(
//	        logger.WithDevelopment("billing-service"),
//	        logger.WithContextValue("request_id", ctxKeyRequestID),
//	    )
//	    logger.SetAsDefault(log)
//
//	    ctx := context.WithValue(context.Background(), ctxKeyRequestID, "abc-123")
//	    log.InfoContext(ctx, "processed request",
//	        logger.UserID(42),
//	        logger.Duration(time.Since(start)),
//	    )
//	}
//
// # Configuration Options
//
//   - WithDevelopment/WithStaging/WithProduction – environment-specific defaults
//   - WithFormat/WithTextFormatter/WithJSONFormatter – output format control
//   - WithLevel – custom log level threshold
//   - WithAttr – static attributes added to all records
//   - WithContextExtractors/WithContextValue – dynamic context injection
//
// # Nil-Safe Error Attributes
//
// Error helpers produce attributes only for non-nil errors:
//
//	log.Info("operation succeeded", logger.Error(err)) // safe even if err is nil
//
// This eliminates conditional logging code throughout the application.
package logger
