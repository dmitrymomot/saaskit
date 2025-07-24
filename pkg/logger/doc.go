// Package logger provides a context-aware wrapper around Go's slog package
// adding functional options for configuration, helper attribute constructors,
// and transparent injection of values stored in context.Context.
//
// The package aims to standardise structured logging across services by
// exposing a single factory – New – that creates a *slog.Logger configured by
// a set of Option functions. These options allow you to:
//
//   • Select an output format (text or json)
//   • Set the minimum log level
//   • Supply default slog.Attr values applied to every record
//   • Register ContextExtractor callbacks that inject attributes pulled from a
//     context value (for example a request id) every time Handle is invoked.
//
// # Architecture
//
// Logger builds a decorated slog.Handler. First, New determines the concrete
// slog.Handler implementation – slog.NewTextHandler or slog.NewJSONHandler –
// based on the configured Format. It then wraps the handler with
// LogHandlerDecorator which is responsible for executing any registered
// ContextExtractor callbacks before delegating to the underlying handler.
//
// Helper constructors such as Group, Error, UserID, etc. live in attr.go and
// return commonly-used slog.Attr instances to keep attribute naming consistent
// across the codebase.
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
// # Configuration
//
// The behaviour of New can be tuned with a variety of Option helpers:
//
//   • WithDevelopment / WithStaging / WithProduction – sensible defaults per environment.
//   • WithFormat / WithTextFormatter / WithJSONFormatter – override output format.
//   • WithLevel – set a custom slog.Level.
//   • WithAttr – attach static attributes.
//   • WithContextExtractors / WithContextValue – inject attributes from context.
//
// # Error Handling
//
// Helper functions Error and Errors produce attributes only when the supplied
// error value is non-nil allowing calls like:
//
//	log.Info("operation succeeded", logger.Error(err))
//
// without an additional nil check.
//
// # Examples
//
// See the package README and example_test files for complete examples.
package logger
