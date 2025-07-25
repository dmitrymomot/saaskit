// Package sanitizer provides a comprehensive collection of helper functions for
// cleaning, normalising and securing data of various kinds.
//
// The functions are grouped conceptually into several areas:
//
//   - Strings – trimming, case conversion, whitespace normalisation, masking and
//     conversion between common naming conventions (snake_case, kebab-case,
//     camelCase, …).
//
//   - Collections – utilities for filtering, deduplicating, transforming and
//     limiting slices and maps.
//
//   - Numeric – generic helpers for clamping, rounding and otherwise constraining
//     numeric values.
//
//   - Format – normalisation helpers for e-mail addresses, phone numbers,
//     credit-card numbers, URLs, postal codes and similar user input.
//
//   - Security – defensive routines that escape or strip dangerous content (HTML
//     tags & attributes, SQL/LDAP/shell metacharacters, path traversal, …) and
//     that mask sensitive data before it is logged or rendered.
//
// The package is completely stateless and depends only on the Go standard
// library (plus the `maps` package from Go 1.21+). All helpers are implemented
// as small, focused functions that can be freely combined.  For convenience the
// higher-order Apply and Compose helpers allow the creation of sanitisation
// pipelines:
//
//	clean := sanitizer.Compose(
//	    sanitizer.Trim,
//	    sanitizer.NormalizeWhitespace,
//	    sanitizer.ToLower,
//	)
//
//	safe := clean("  Mixed CASE   Input\n") // "mixed case input"
//
// # Usage
//
// Import the package using its module-qualified path:
//
//	import "github.com/dmitrymomot/saaskit/pkg/sanitizer"
//
// Example – e-mail address normalisation:
//
//	raw   := "  John.Doe...@Example.COM "
//	email := sanitizer.NormalizeEmail(raw)
//	// email == "john.doe@example.com"
//
// Example – securely rendering un-trusted HTML input:
//
//	safeHTML := sanitizer.PreventXSS(userInput)
//
// # Error handling
//
// None of the helpers returns an error – they always fall back to a safe result
// (usually the original input or an empty string) if sanitisation fails.
//
// # Performance
//
// All operations are implemented with efficiency in mind and allocate only what
// is necessary.  Because there is no global state the helpers are safe for use
// from multiple goroutines concurrently.
//
// See the package-level examples and individual function documentation for
// further details.
package sanitizer
