# slug

A URL-safe string generation package for web applications.

## Overview

The slug package provides functionality to convert arbitrary strings into URL-friendly formats by normalizing Unicode characters, replacing spaces and special characters with separators, and offering customizable options for slug generation. It's designed to handle user-generated content like blog titles, product names, or usernames with proper Unicode support and collision avoidance through optional random suffixes.

## Internal Usage

This package is internal to the project and provides URL slug generation functionality for various SaaS components such as user profiles, resource identifiers, and content URLs.

## Features

- Unicode normalization with diacritic-to-ASCII conversion
- Configurable separators and case handling
- Maximum length enforcement with proper Unicode character counting
- Custom string replacements for common patterns
- Character stripping for removing unwanted characters
- Cryptographically secure random suffix generation for collision avoidance

## Usage

### Basic Example

```go
import "github.com/dmitrymomot/saaskit/pkg/slug"

// Simple slug generation
url := slug.Make("Hello World!")
// url = "hello-world"

// With Unicode normalization
url := slug.Make("Café résumé naïve")
// url = "cafe-resume-naive"

// Product names with special characters
url := slug.Make("Price: $99.99")
// url = "price-99-99"
```

### Additional Usage Scenarios

```go
// Custom separator for different URL schemes
url := slug.Make("User Profile Page", slug.Separator("_"))
// url = "user_profile_page"

// Preserve case for specific requirements
url := slug.Make("GitHub Repository", slug.Lowercase(false))
// url = "GitHub-Repository"

// Limit length for database constraints
url := slug.Make("Very Long Product Name That Exceeds Limits", slug.MaxLength(20))
// url = "very-long-product-na"

// Custom replacements for better readability
url := slug.Make("Fish & Chips @ Home",
    slug.CustomReplace(map[string]string{
        "&": "and",
        "@": "at",
    }),
)
// url = "fish-and-chips-at-home"

// Add random suffix to prevent collisions
url := slug.Make("Common Title", slug.WithSuffix(6))
// url = "common-title-x7g3k2"

// Strip specific characters
url := slug.Make("Remove (these) [brackets]", slug.StripChars("()[]"))
// url = "remove-these-brackets"

// Combine multiple options
url := slug.Make("COMPLEX & Test @ 2024!!!",
    slug.Separator("_"),
    slug.Lowercase(false),
    slug.MaxLength(15),
    slug.StripChars("!"),
    slug.CustomReplace(map[string]string{
        "&": "AND",
        "@": "AT",
    }),
)
// url = "COMPLEX_AND_Tes"
```

### Error Handling

The Make function does not return errors. It handles all inputs gracefully:

```go
// Empty input returns empty string
url := slug.Make("")
// url = ""

// Special characters only returns empty string
url := slug.Make("!@#$%^&*()")
// url = ""

// Handles edge cases gracefully
url := slug.Make("   ", slug.WithSuffix(5))
// url = "abc12" (just the suffix)
```

## Best Practices

### Integration Guidelines

- Use consistent options across your application for uniform URL patterns
- Store both the original text and generated slug in your database
- Consider adding suffixes for user-generated content to prevent collisions
- Apply length limits that match your database schema constraints

### Project-Specific Considerations

- For user-facing URLs, prefer readable slugs without suffixes
- For internal identifiers, use suffixes to guarantee uniqueness
- Cache generated slugs to avoid regeneration on every request
- Use custom replacements for domain-specific terminology

## API Reference

### Configuration Variables

The package does not export any configuration variables. All configuration is done through functional options.

### Types

```go
// Option configures the slug generation behavior.
type Option func(*config)
```

### Functions

```go
// Make creates a URL-safe slug from the input string.
func Make(s string, opts ...Option) string

// MaxLength sets the maximum length of the generated slug.
func MaxLength(n int) Option

// Separator sets the separator character for the slug.
func Separator(s string) Option

// Lowercase controls whether the slug should be converted to lowercase.
func Lowercase(enabled bool) Option

// StripChars sets additional characters to strip from the slug.
func StripChars(chars string) Option

// CustomReplace sets custom string replacements to apply before slugification.
func CustomReplace(replacements map[string]string) Option

// WithSuffix adds a random alphanumeric suffix to reduce collision possibility.
func WithSuffix(length int) Option
```

### Methods

This package does not define any methods on exported types.

### Error Types

This package does not define any error types. The Make function handles all inputs gracefully and returns empty strings for invalid inputs.
