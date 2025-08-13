# CLAUDE.md

## SaasKit Philosophy: MLP > MVP

Build Minimum Lovable Products - features users enjoy, not just tolerate. Solo developer framework for shipping quality SaaS quickly.

## Commands

```bash
make test       # Run tests before changes
make lint       # Lint before committing
make fmt        # Format code
make fmt lint   # Run after creating/updating any Go file
# Target specific: make test PKG=./pkg/validator
```

## Architecture

### Packages

- **`pkg/`** - Utilities (Errors: plain English like `"invalid input"`)
- **`handler/`** - Type-safe HTTP handlers with generics
- **`modules/`** - Feature routers (Errors: i18n keys like `"auth.failed"`)

### Handler Pattern (Core)

```go
func createUser(ctx handler.Context, req CreateUserRequest) handler.Response {
    // Type-safe, no boilerplate
    return handler.JSON(user)
}
router.Post("/users", handler.Wrap(createUser))
```

## Critical Rules

### Go 1.24+

```go
// Range over integers
for range n { }                      // NOT: for i := 0; i < n; i++

// Slices package
slices.Contains(items, target)       // NOT: custom contains loop
slices.Clone(original)               // NOT: manual slice copy
slices.Equal(s1, s2)                // NOT: custom equality check
slices.Collect(maps.Keys(m))        // Convert iterator to slice

// Maps package
maps.Clone(original)                 // NOT: manual map copy
maps.Copy(dst, src)                 // NOT: range loop copy
maps.Keys(m)                         // Iterator over keys
maps.Values(m)                       // Iterator over values

// Built-in functions
min(a, b)                           // NOT: if a < b { return a }
max(a, b)                           // NOT: if a > b { return a }
clear(slice)                        // Zero elements for GC
clear(map)                          // Clear all entries
```

### Errors

```go
// pkg/ - plain messages
errors.New("connection failed")

// modules/ - i18n keys
errors.New("auth.invalid_token")

// Wrapping - preserve chains
errors.Join(ErrValidation, err)         // ✓ Both chains
fmt.Errorf("context: %w", err)          // ✓ With context
fmt.Errorf("%w: %v", ErrFoo, err)       // ✗ NEVER - loses err chain
```

### Testing

**Test:** Business logic, payment/auth, error paths, black-box testing (80%+ coverage)
**Skip:** Getters/setters, wrappers, third-party code

## Core Constraints

- **No global state** - Inject everything
- **Type-safe generics** - Compile-time validation
- **Explicit > Magic** - Clear dependencies
- **DataStar SSE** - Real-time without JavaScript complexity

## MLP Implementation Mindset

When implementing features, prioritize:

1. User delight over feature count
2. Polish core flows before adding features
3. Type safety to prevent runtime surprises
4. Clear errors users can understand
