# randomname

Cryptographically secure random name generator with configurable patterns and word types.

## Overview

The `randomname` package generates human-readable, memorable names using various word combinations. It supports multiple patterns, custom word lists, and validation callbacks, making it suitable for generating unique identifiers for workspaces, projects, or resources.

## Internal Usage

This package is internal to the project and provides name generation capabilities for creating user-friendly identifiers across different SaaS components.

## Features

- Cryptographically secure randomness using `crypto/rand`
- Configurable patterns with multiple word types (adjectives, nouns, colors, sizes, etc.)
- Multiple suffix options for collision avoidance (hex, numeric)
- Custom word list support for any word type
- Validation callback support for external uniqueness checks
- Zero-allocation string building with `sync.Pool`
- Thread-safe implementation

## Usage

### Basic Example

```go
import "github.com/dmitrymomot/saaskit/pkg/randomname"

// Generate simple adjective-noun name
name := randomname.Simple()
// Returns: "brave-tiger"

// Generate name with hex suffix
nameWithSuffix := randomname.WithSuffix()
// Returns: "happy-dolphin-a3f21b"

// Generate colorful name (color-noun)
colorful := randomname.Colorful()
// Returns: "blue-lion"
```

### Additional Usage Scenarios

```go
// Generate with custom options
name := randomname.Generate(&randomname.Options{
    Pattern:   []randomname.WordType{randomname.Size, randomname.Color, randomname.Noun},
    Separator: "_",
    Suffix:    randomname.Numeric4,
})
// Returns: "tiny_purple_eagle_4829"

// Generate with validation callback
name := randomname.Generate(&randomname.Options{
    Suffix: randomname.Hex6,
    Validator: func(name string) bool {
        // Check if name already exists in database
        exists, _ := db.NameExists(name)
        return !exists // Accept only if name doesn't exist
    },
})
// Returns: A unique name that passes validation

// Use convenience functions for common patterns
descriptive := randomname.Descriptive() // adjective-color-noun
sized := randomname.Sized()             // size-noun
complex := randomname.Complex()         // size-adjective-noun
full := randomname.Full()               // size-adjective-color-noun
```

### Error Handling

```go
// The Generate function always returns a valid name and never returns an error
// If validation fails after 100 retries, it returns the last generated name
name := randomname.Generate(&randomname.Options{
    Validator: func(name string) bool {
        // This validator always rejects
        return false
    },
})
// Still returns a valid name after 100 attempts
```

## Best Practices

### Integration Guidelines

- Use convenience functions for common patterns to keep code concise
- Implement validation callbacks when uniqueness across external systems is required
- Keep validation functions lightweight and non-blocking

### Project-Specific Considerations

- For user-facing resources, prefer patterns with better readability (Simple, Colorful)
- For internal resources where uniqueness is critical, use suffix options
- Consider using custom word lists for domain-specific naming

## API Reference

### Configuration Variables

```go
// Word types available for name generation
const (
    Adjective WordType = iota
    Noun
    Color
    Size
    Origin
    Action
)

// Suffix types for collision avoidance
const (
    NoSuffix SuffixType = iota
    Hex6     // 6-character hexadecimal (e.g., a3f21b)
    Hex8     // 8-character hexadecimal (e.g., a3f21b9c)
    Numeric4 // 4-digit number (e.g., 4829)
)
```

### Types

```go
type WordType int

type SuffixType int

type Options struct {
    Pattern   []WordType              // Word types to use in order (default: [Adjective, Noun])
    Separator string                  // Separator between words (default: "-")
    Suffix    SuffixType              // Suffix type for collision avoidance (default: NoSuffix)
    Words     map[WordType][]string   // Custom word lists merged with defaults
    Validator func(string) bool       // Validation callback (return true to accept)
}
```

### Functions

```go
func Generate(opts *Options) string
func Simple() string      // adjective-noun
func Colorful() string    // color-noun
func Descriptive() string // adjective-color-noun
func WithSuffix() string  // adjective-noun-hex6
func Sized() string       // size-noun
func Complex() string     // size-adjective-noun
func Full() string        // size-adjective-color-noun
```

### Methods

```go
func (o *Options) merge(defaults *Options) *Options
```

### Error Types

The package does not define any error types. The `Generate` function always returns a valid name.
