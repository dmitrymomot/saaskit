// Package environment provides simple helpers to propagate the current
// application environment (development, staging, production, etc.) through
// context.Context, HTTP requests and structured logs.
//
// It defines the typed string alias Environment with predefined constants
// Development, Staging and Production. These values can be attached to a
// context using WithContext, extracted with FromContext and queried with the
// convenience predicates IsDevelopment, IsStaging and IsProduction.
//
// In HTTP servers the Middleware function can be used to set the desired
// environment on every request's context, making it available across the
// request-handling pipeline and to any downstream code that consumes the
// context.
//
// For structured logging the package provides LoggerExtractor which returns a
// slog.Attr containing the environment value so it can be seamlessly injected
// into slog based loggers.
//
// # Usage
//
// Import the package:
//
//	import "github.com/dmitrymomot/saaskit/pkg/environment"
//
// Set the environment on an HTTP server:
//
//	mux := http.NewServeMux()
//	mux.Handle("/", handler)
//	envAwareMux := environment.Middleware(environment.Production)(mux)
//	http.ListenAndServe(":8080", envAwareMux)
//
// Retrieve the environment from a context:
//
//	env := environment.FromContext(ctx)
//	if environment.IsProduction(ctx) {
//	    // production-specific behaviour
//	}
//
// Add the environment to a slog logger:
//
//	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//	logger = logger.With(environment.LoggerExtractor())
//
// # Error Handling
//
// All helpers are designed to be allocation-free and never return errors.
// Missing values simply result in the zero value ("").
//
// See the function-level documentation for further details.
package environment
