// Package randomname provides easy-to-use helpers for generating memorable
// random names composed of human-readable words such as adjectives, colors,
// sizes, and nouns. The package is useful for creating unique identifiers for
// resources (e.g. container names, S3 prefixes, database records, test data)
// that remain readable to humans while still offering a very large search
// space.
//
// A name is produced by joining a sequence of words defined by a *pattern* and
// an optional *suffix*. All words are taken from curated built-in dictionaries
// (see defaultWords in words.go) that can be extended or replaced at run time.
// The generator never returns an error: it always falls back to sensible
// defaults and makes multiple attempts to satisfy user-supplied validation
// callbacks.
//
// # Architecture
//
//   • Words are stored in-memory per WordType. `getWords` merges user-provided
//     slices with the defaults on every call, so custom dictionaries are
//     automatically respected.
//   • A tiny `sync.Pool` is used to reuse `strings.Builder` instances and keep
//     allocations per generation close to zero.
//   • Cryptographically secure randomness (`crypto/rand`) is used for word and
//     suffix selection. A time-based fallback is employed only when the crypto
//     source fails (e.g. in restricted CI sandboxes).
//   • All exported helpers such as Simple, Colorful, Complex, … are mere thin
//     wrappers around the central Generate function with preconfigured Options.
//
// # Usage
//
// Import the package:
//
//     import "github.com/dmitrymomot/saaskit/pkg/randomname"
//
// Generate a simple adjective-noun combination (e.g. "bold-otter"):
//
//     name := randomname.Simple()
//
// Customise the output by providing an Options struct:
//
//     name := randomname.Generate(&randomname.Options{
//         Pattern:   []randomname.WordType{randomname.Color, randomname.Noun},
//         Separator: "_",          // use underscore instead of dash
//         Suffix:    randomname.Hex6, // adds "-a3f21b"
//         Validator: func(s string) bool { return !strings.HasPrefix(s, "red") },
//     })
//
// # Options
//
//   • Pattern   decides the word sequence (default: adjective-noun).
//   • Separator string placed between words (default: "-").
//   • Suffix    collision-avoidance strategy (`NoSuffix`, `Hex6`, `Hex8`, `Numeric4`).
//   • Words     map lets you extend/override the dictionaries per WordType.
//   • Validator callback lets you reject generated names you don't like.
//
// # Examples
//
// See functions `Example` and friends in the *_test.go files for
// compile-checked usage demonstrations.
package randomname
