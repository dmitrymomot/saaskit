# Randomname

Thread-safe random name generator for workspace, project, and resource naming.

## Overview

The `randomname` package generates human-readable, memorable names in either "adjective-noun" or "adjective-noun-suffix" format. It's designed for creating workspace, project, or resource identifiers with built-in collision prevention. This package is thread-safe and suitable for concurrent use in applications where unique identifiers are needed.

## Features

- Thread-safe implementation with mutex protection
- Session-based uniqueness tracking to prevent duplicates
- Support for custom validation callbacks (e.g., database checks)
- Two name formats with varied collision probability:
    - "adjective-noun" (brave-tiger)
    - "adjective-noun-xxxxxx" (brave-tiger-1a2b3c)
- Rich word variety with 42 adjectives × 44 nouns (1,848 base combinations)
- With 24-bit suffix: ~16.7 million unique combinations per base name

## Usage

### Basic Generation

```go
import "github.com/dmitrymomot/saaskit/pkg/randomname"

// Generate name with suffix (e.g., "brave-tiger-1a2b3c")
name := randomname.Generate(nil)
// Returns: "brave-tiger-1a2b3c" (unique across session)

// Generate a simple name without suffix (e.g., "brave-tiger")
simpleName := randomname.GenerateSimple(nil)
// Returns: "brave-tiger" (only unique if it hasn't been used before)
```

### With Custom Validation

```go
// Generate a name that doesn't exist in the database
name := randomname.Generate(func(name string) bool {
    exists, err := db.WorkspaceExists(name)
    if err != nil {
        log.Printf("Error checking workspace name: %v", err)
        return false // Reject this candidate on error
    }
    return !exists // Accept only if name doesn't exist
})
// Returns: A name that's both session-unique and database-unique
```

### Managing Used Names

```go
// Clear the internal cache of used names when no longer needed
// Useful when starting a new naming session
randomname.Reset()
```

## Best Practices

1. **Session Management**:
    - Call `Reset()` when starting a new naming session
    - Consider session boundaries (e.g., application restart, user session)

2. **Validation Handling**:
    - Implement proper error handling in validation callbacks
    - Return `false` from callbacks on validation error to reject the name
    - Keep validation functions lightweight and non-blocking

3. **Format Selection**:
    - Use `Generate()` (with suffix) when uniqueness is critical
    - Use `GenerateSimple()` for better readability when namespace is large enough
    - Consider the potential for collisions with `GenerateSimple()` (limited to 1,848 combinations)

4. **Concurrency**:
    - The package is thread-safe, but external validation must also be thread-safe
    - Avoid long-running operations in validation callbacks while holding name reservations

## API Reference

### Functions

```go
func Generate(check func(name string) bool) string
```

Generates a random name in the format "adjective-noun-xxxxxx" with a 6-character hexadecimal suffix. Ensures uniqueness within the current session and accepts an optional validation callback.

```go
func GenerateSimple(check func(name string) bool) string
```

Generates a random name in the format "adjective-noun" without the hexadecimal suffix. Has a higher chance of collisions due to the smaller namespace.

```go
func Reset()
```

Clears the internal cache of used names, allowing previously generated names to be used again.

### Validation Callback

```go
type ValidateFunc func(name string) bool
```

Function signature for the validation callback:

- Return `true` to accept the name
- Return `false` to reject the name and generate a new one

The callback is executed after a name is reserved but before it's returned to the caller. If the callback rejects the name, the reservation is released and a new name is generated.
