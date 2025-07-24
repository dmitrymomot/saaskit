# environment

Context-aware environment management for HTTP handlers and logging.

## Overview

This package provides utilities for managing environment information (development, staging, production) throughout the application lifecycle. It enables environment-aware logging and request handling by storing environment data in the context.

## Internal Usage

This package is internal to the project and provides environment context propagation for HTTP handlers and structured logging components.

## Features

- Store and retrieve environment information from context
- HTTP middleware for automatic environment context injection
- Production environment detection helper
- Structured logger attribute extraction for environment-aware logging
- Zero external dependencies
- Thread-safe context operations

## Usage

### Basic Example

```go
import "github.com/dmitrymomot/saaskit/pkg/environment"

// Store environment in context
ctx := environment.WithContext(context.Background(), "production")

// Retrieve environment from context
env := environment.FromContext(ctx) // "production"

// Check if running in production
if environment.IsProduction(ctx) {
    // Production-specific logic
}
```

### HTTP Middleware Integration

```go
// In your router setup
router := chi.NewRouter()

// Add environment middleware
router.Use(environment.Middleware("production"))

// All subsequent handlers will have environment in context
router.Get("/", func(w http.ResponseWriter, r *http.Request) {
    env := environment.FromContext(r.Context()) // "production"
})
```

### Structured Logging Integration

```go
// Configure slog with environment extractor
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    AddSource: true,
}))

// Use with context-aware logger
extractor := environment.LoggerExtractor()
ctx := environment.WithContext(context.Background(), "staging")

// When logging with context, environment will be included
attr, ok := extractor(ctx) // Returns slog.String("environment", "staging"), true
```

## Best Practices

### Integration Guidelines

- Apply the middleware early in your middleware chain to ensure all handlers have access to environment context
- Use consistent environment naming across your application ("production", "staging", "development")
- Leverage `IsProduction()` for environment-specific behavior rather than string comparisons

### Project-Specific Considerations

- The middleware should be configured with the actual runtime environment from your configuration
- Environment context is automatically propagated to child contexts
- Empty context or missing environment data returns empty string (safe default)

## API Reference

### Functions

```go
func WithContext(ctx context.Context, env string) context.Context
func FromContext(ctx context.Context) string
func IsProduction(ctx context.Context) bool
func Middleware(env string) func(http.Handler) http.Handler
func LoggerExtractor() func(ctx context.Context) (slog.Attr, bool)
```

### Methods

This package does not export any methods on types.

### Error Types

This package does not define any error types.