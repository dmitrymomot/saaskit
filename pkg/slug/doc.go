// Package slug provides URL-safe string generation for web applications.
//
// Converts strings into URL-friendly format by replacing spaces and special characters
// with separators, normalizing Unicode diacritics to ASCII, and optionally adding
// random suffixes for collision avoidance.
//
// Basic usage:
//
//	slug.Make("Hello World!") // "hello-world"
//
// With options:
//
//	slug.Make("Price: $99.99",
//		slug.MaxLength(15),
//		slug.CustomReplace(map[string]string{"$": "usd"}),
//	) // "price-usd99-99"
//
// Key features:
//   - Unicode normalization (café → cafe)
//   - Configurable separators and max length
//   - Custom string replacements
//   - Random suffix generation
//   - Thread-safe with crypto/rand for suffixes
package slug
