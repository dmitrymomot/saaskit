# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SaasKit is a Go framework for building SaaS applications, designed for solo developers who want to ship MVPs quickly without sacrificing quality. The framework follows principles of explicitness, type safety, and convention with escape hatches.

## Key Commands

```bash
# Run commands for entire project (default)
make test       # Run tests with race detector and coverage
make lint       # Run go vet and golangci-lint before committing
make fmt        # Format code with go fmt and goimports
make bench      # Run benchmarks with memory profiling

# Run commands for specific packages
make test PKG=./pkg/scopes        # Test only pkg/scopes
make lint PKG=./modules/auth      # Lint only modules/auth
make fmt PKG=./handler            # Format only handler package
make bench PKG=./pkg/validator    # Benchmark only pkg/validator
```

## High-Level Architecture

### Package Structure

The framework uses a modular architecture organized into distinct layers:

- **`pkg/`** - Stateless, reusable utility packages (validator, email, storage, JWT, etc.)
    - Each package is self-contained with its own README.md and doc.go
    - Designed for zero dependencies between pkg components
    - Examples: validator, sanitizer, jwt, email, queue, redis, pg
    - Errors: use general english message, no i18n keys

- **`handler/`** - Type-safe HTTP request handling abstractions
    - Generic handler functions that bind requests to structs
    - Multiple response formats (JSON, Templ, Redirect)
    - First-class DataStar support for real-time UI updates
    - Built-in validation and error handling

- **`modules/`** - Plug-and-play feature modules (auth, billing, user management)
    - Self-contained chi.Routers that can be mounted with minimal config
    - Each module encapsulates routes, handlers, and business logic
    - Designed for rapid feature composition
    - Errors: use i18n key instead of message in English.

- **`middleware/`** - HTTP middleware implementations (planned)
    - Errors: use i18n key instead of message in English.

- **`decorators/`** - Handler decorators for cross-cutting concerns (planned)
    - Errors: use i18n key instead of message in English.

### Core Patterns

1. **Type Safety First**: Generic handlers with compile-time request/response validation
2. **Explicit Configuration**: No magic, all dependencies injected explicitly
3. **DataStar Integration**: Server-sent events for real-time UI without JavaScript
4. **Error Handling**: Structured errors with i18n support built-in
5. **Modular Design**: Features as self-contained, mountable routers

### Handler Pattern

The framework centers around type-safe handlers that eliminate boilerplate:

```go
type CreateUserRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

func createUserHandler(ctx handler.Context, req CreateUserRequest) handler.Response {
    // Business logic here
    return handler.JSON(user)
}

// Wrap handler explicitly in router
router.Post("/users", handler.Wrap(createUserHandler))
```

## Go Version and Standards

- **Go 1.24+** - Uses modern Go features including generics and range-over-int
- **Modern Patterns**: Always use `for range n` instead of `for i := 0; i < n; i++`
- **Error Handling**: Domain errors with i18n keys, wrapped internal errors
- **Testing**: github.com/stretchr/testify for assertions and mocking
- **No Reflection**: Avoid reflection where alternatives exist
- **Zero Allocations**: Optimize hot paths for minimal allocations

## Error Handling Standards

### Error Definition

- **pkg/ packages**: Use general English messages for errors (utility packages should be i18n-agnostic)

    ```go
    var ErrInvalidInput = errors.New("invalid input provided")
    var ErrConnectionFailed = errors.New("connection failed")
    ```

- **modules/ and handlers**: Use i18n keys for user-facing errors
    ```go
    var ErrAuthFailed = errors.New("auth.failed")
    var ErrPermissionDenied = errors.New("permission.denied")
    ```

### Error Wrapping

- **Use `errors.Join`** for combining sentinel errors with underlying errors:

    ```go
    // Correct - preserves both error chains
    return nil, errors.Join(ErrVectorizationFailed, err)
    ```

- **Use `fmt.Errorf` with %w** for adding descriptive context:

    ```go
    // Correct - adds context while preserving error chain
    return nil, fmt.Errorf("failed to connect to database: %w", err)
    ```

- **Never use** `fmt.Errorf("%w: %v", sentinel, err)` pattern:
    ```go
    // Wrong - loses error chain for err
    return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
    ```

### Error Checking

- Use `errors.Is()` for sentinel error checking
- Use `errors.As()` for typed error extraction
- Always wrap errors with appropriate context when propagating up the stack
- Maintain error chains for proper unwrapping

### Examples

```go
// Defining errors
var (
    ErrInvalidConfig = errors.New("invalid configuration")
    ErrAPIKeyMissing = errors.New("API key is missing")
)

// Wrapping with context
if err := db.Connect(); err != nil {
    return fmt.Errorf("failed to initialize database: %w", err)
}

// Combining sentinel with underlying error
if err := provider.Validate(); err != nil {
    return errors.Join(ErrInvalidConfig, err)
}

// Checking errors
if errors.Is(err, ErrInvalidConfig) {
    // Handle invalid configuration
}
```

## Development Workflow

1. **Before Implementation**: Run `make test` to ensure clean baseline
2. **Format Code**: Use `make fmt` for consistent formatting
3. **Lint Check**: Run `make lint` to catch issues early
4. **Test Coverage**: Ensure >80% coverage for business logic
5. **Benchmarks**: Run `make bench` for performance-critical code
6. **Documentation**: Update package README.md for API changes

## Testing Philosophy

### Test These:

- Business logic and algorithms
- Error handling and edge cases
- Public API endpoints and contracts
- Complex state transformations
- Payment/auth operations (high risk)
- Concurrent operations
- Integration points

### Skip These:

- Simple getters/setters
- Third-party library behavior
- Language features
- Wrapper functions with no logic
- Private methods directly

## Framework Principles

1. **MLP > MVP**: Build minimum lovable products, not just viable ones
2. **Ship Pragmatically**: Balance quality with shipping speed
3. **User Delight > Features**: Focus on core experience over feature count
4. **Explicit > Implicit**: Make dependencies and behavior obvious
5. **Simple First**: Start simple, scale when proven necessary

## Important Constraints

- **No Global State**: All dependencies must be injected
- **Router Agnostic**: Framework works with any router (chi, gorilla, etc.)
- **Minimal Dependencies**: Core has no runtime dependencies
- **Type Safe**: Leverage Go's type system for compile-time guarantees
- **i18n Ready**: All user-facing errors support translation keys
