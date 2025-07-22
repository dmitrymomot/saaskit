# Sanitizer Package

The sanitizer package provides functions for cleaning and normalizing input data in Go applications. Unlike validation which checks if data meets criteria, sanitization transforms data into a clean, standardized format.

## Overview

This package focuses on data transformation and cleaning rather than validation. All functions are direct transformations that return cleaned data.

## Features

- **String Cleaning**: Trim whitespace, normalize case, remove unwanted characters
- **Length Management**: Truncate strings to maximum lengths
- **Content Filtering**: Remove HTML, control characters, or specific character sets
- **Format Normalization**: Convert multi-line text to single lines, normalize whitespace
- **Character Set Filtering**: Keep only alphanumeric, alphabetic, or numeric characters

## Core Functions

### Apply Pattern

The sanitizer package provides a generic `Apply` function for chaining multiple sanitization operations and a `Compose` function for creating reusable transformation pipelines.

#### Core Functions

- `Apply[T any](value T, transforms ...func(T) T) T` - Applies multiple transformations sequentially with type safety
- `Compose[T any](transforms ...func(T) T) func(T) T` - Creates reusable transformation functions from multiple transforms

### String Sanitization Functions

#### Basic String Operations

- `Trim(s string) string` - Remove leading and trailing whitespace
- `ToLower(s string) string` - Convert to lowercase
- `ToUpper(s string) string` - Convert to uppercase
- `ToTitle(s string) string` - Convert to title case
- `TrimToLower(s string) string` - Trim and convert to lowercase
- `TrimToUpper(s string) string` - Trim and convert to uppercase

#### Length and Content Management

- `MaxLength(s string, maxLen int) string` - Truncate to maximum length
- `RemoveExtraWhitespace(s string) string` - Normalize whitespace to single spaces
- `RemoveControlChars(s string) string` - Remove control characters (except \n, \r, \t)
- `StripHTML(s string) string` - Remove HTML tags and unescape entities
- `SingleLine(s string) string` - Convert multi-line text to single line

#### Character Filtering

- `RemoveChars(s, chars string) string` - Remove specified characters
- `ReplaceChars(s, old, new string) string` - Replace specified characters
- `KeepAlphanumeric(s string) string` - Keep only letters, digits, and spaces
- `KeepAlpha(s string) string` - Keep only letters and spaces
- `KeepDigits(s string) string` - Keep only numeric digits
- `ToKebabCase(s string) string` - Convert to kebab-case
- `ToSnakeCase(s string) string` - Convert to snake_case
- `ToCamelCase(s string) string` - Convert to camelCase

## Usage Examples

### Apply Pattern Usage

```go
// Apply multiple transformations in sequence
input := "  HELLO    WORLD!@#  "
clean := sanitizer.Apply(input,
    sanitizer.Trim,
    sanitizer.RemoveExtraWhitespace,
    sanitizer.ToLower,
)
// Result: "hello world!@#"

// Create reusable transformation functions
emailRule := sanitizer.Compose(
    sanitizer.Trim,
    sanitizer.ToLower,
)
cleanEmail := emailRule("  USER@EXAMPLE.COM  ") // "user@example.com"

// Use composed rules in Apply
result := sanitizer.Apply(dirtyEmail, emailRule)
```

### Basic String Cleaning

```go
// Direct function calls
input := "  Hello World!  "
clean := sanitizer.TrimToLower(input) // "hello world!"

// Normalize whitespace
messy := "hello    world\n\ntest"
normalized := sanitizer.RemoveExtraWhitespace(messy) // "hello world test"
```

### Length Management

```go
// Truncate long descriptions
description := "This is a very long description that needs to be shortened"
short := sanitizer.MaxLength(description, 20) // "This is a very long "
```

### Content Filtering

```go
// Clean HTML content
htmlContent := "<p>Hello <strong>world</strong>!</p>"
plainText := sanitizer.StripHTML(htmlContent) // "Hello world!"

// Remove unwanted characters
phone := "123-456-7890"
digits := sanitizer.KeepDigits(phone) // "1234567890"
```

### Form Input Sanitization

```go
type UserInput struct {
    Name        string
    Email       string
    Description string
    Phone       string
}

// Using direct function calls
func (u *UserInput) Sanitize() {
    u.Name = sanitizer.TrimToTitle(u.Name)
    u.Email = sanitizer.TrimToLower(u.Email)
    u.Description = sanitizer.MaxLength(sanitizer.RemoveExtraWhitespace(u.Description), 500)
    u.Phone = sanitizer.KeepDigits(u.Phone)
}

// Using Apply pattern for more complex sanitization
func (u *UserInput) SanitizeWithApply() {
    u.Name = sanitizer.Apply(u.Name,
        sanitizer.Trim,
        sanitizer.RemoveExtraWhitespace,
        sanitizer.ToTitle,
    )

    u.Email = sanitizer.Apply(u.Email,
        sanitizer.TrimToLower,
    )

    u.Description = sanitizer.Apply(u.Description,
        sanitizer.Trim,
        sanitizer.RemoveExtraWhitespace,
        func(s string) string { return sanitizer.MaxLength(s, 500) },
    )

    u.Phone = sanitizer.Apply(u.Phone,
        sanitizer.KeepDigits,
    )
}

// Using Compose for reusable transformation pipelines
func (u *UserInput) SanitizeWithCompose() {
    nameRule := sanitizer.Compose(sanitizer.Trim, sanitizer.RemoveExtraWhitespace, sanitizer.ToTitle)
    emailRule := sanitizer.Compose(sanitizer.TrimToLower)
    descRule := sanitizer.Compose(
        sanitizer.Trim,
        sanitizer.RemoveExtraWhitespace,
        func(s string) string { return sanitizer.MaxLength(s, 500) },
    )

    u.Name = nameRule(u.Name)
    u.Email = emailRule(u.Email)
    u.Description = descRule(u.Description)
    u.Phone = sanitizer.KeepDigits(u.Phone)
}
```

### Character Set Filtering

```go
// Extract only letters for name processing
input := "John123!@#"
name := sanitizer.KeepAlpha(input) // "John"

// Get alphanumeric for usernames
username := "user_name@123"
clean := sanitizer.KeepAlphanumeric(username) // "user name123"
```

### Multi-line Text Processing

```go
// Convert multi-line input to single line
multiLine := `First line
Second line
Third line`
singleLine := sanitizer.SingleLine(multiLine) // "First line Second line Third line"
```

## Best Practices

1. **Sanitize Early**: Clean input data as soon as it enters your system
2. **Use Apply for Multiple Transforms**: For multiple transformations, use `Apply` for clearer sequential processing
3. **Create Reusable Pipelines**: Use `Compose` to create reusable transformation functions
4. **Choose the Right Pattern**: Use direct functions for simple cases, Apply for sequences, Compose for reusability
5. **Preserve Intent**: Choose sanitization that preserves the intended meaning of data
6. **Document Transformations**: Be clear about what sanitization is applied to each field
7. **Test Edge Cases**: Verify behavior with empty strings, unicode, and edge cases

## Design Principles

- **Simple API**: Direct function calls, generic Apply pattern, and Compose for reusability
- **Type Safe**: Generic functions preserve input types throughout transformations
- **No Side Effects**: All functions are pure transformations
- **Unicode Safe**: Proper handling of unicode characters and runes
- **Performance Focused**: Efficient implementations using standard library
- **Composable**: Functions can be chained, combined, and reused through functional composition
- **Clean**: No unnecessary complexity like field names or structured errors

## Supported Types

Currently supports string sanitization. Future phases will include:

- Format-specific sanitization (email, phone, URL)
- Security-focused sanitization (XSS prevention)
- Numeric sanitization with generics
- Collection sanitization (slices, maps)
