// Package slug provides URL-safe string generation for use in web applications.
//
// The package offers a simple yet flexible way to convert any string into a URL-friendly
// format by replacing spaces and special characters with hyphens (or custom separators),
// normalizing Unicode characters, and optionally adding random suffixes to reduce collisions.
// This is particularly useful for creating readable URLs from user-generated content like
// blog post titles, product names, or usernames.
//
// # Features
//
// The slug generator supports several features:
//   - Unicode normalization (converts diacritics to ASCII equivalents)
//   - Configurable separators (default: hyphen)
//   - Optional lowercase conversion (enabled by default)
//   - Maximum length enforcement with proper Unicode handling
//   - Custom string replacements (e.g., "&" → "and")
//   - Character stripping for removing specific unwanted characters
//   - Random suffix generation for collision avoidance
//
// # Usage
//
// Basic usage is straightforward:
//
//	import "github.com/dmitrymomot/saaskit/pkg/slug"
//
//	// Simple slug generation
//	url := slug.Make("Hello World!")
//	// Result: "hello-world"
//
//	// With custom options
//	url := slug.Make("Price: $99.99",
//		slug.MaxLength(10),
//		slug.CustomReplace(map[string]string{"$": "usd"}),
//	)
//	// Result: "price-usd9"
//
// # Configuration Options
//
// The package uses functional options for configuration:
//
//   - MaxLength: Set maximum slug length (counts Unicode characters, not bytes)
//   - Separator: Change the separator character (default: "-")
//   - Lowercase: Enable/disable lowercase conversion (default: true)
//   - StripChars: Remove specific characters from the output
//   - CustomReplace: Apply custom string replacements before processing
//   - WithSuffix: Add a random alphanumeric suffix to reduce collisions
//
// # Unicode Support
//
// The package includes built-in support for normalizing common diacritics to their
// ASCII equivalents. For example, "café" becomes "cafe", and "naïve" becomes "naive".
// This ensures maximum compatibility with URL standards while maintaining readability.
//
// # Performance Considerations
//
// The slug generation process is optimized for performance with pre-allocated string
// builders and efficient character processing. The diacritic normalization uses a
// simple map lookup for O(1) performance per character.
//
// # Thread Safety
//
// All functions in this package are thread-safe. The random suffix generation uses
// crypto/rand for secure random number generation.
package slug
