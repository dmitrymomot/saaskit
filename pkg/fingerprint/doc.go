// Package fingerprint provides utilities for generating and validating
// a deterministic device/browser fingerprint from an incoming HTTP request.
//
// It combines several relatively stable request attributes—including
// the User-Agent string, `Accept*` headers, client IP address, and a
// canonicalised ordering of common header names—and feeds them into a
// SHA-256 hash. The first 16 bytes of that hash are returned as a
// 32-character hexadecimal string which can be stored in a session or
// database to recognise subsequent requests from the same device.
//
// The package also offers helper functions and middleware for working
// with fingerprints in typical HTTP server setups.
//
// # Architecture
//
// The package is intentionally lightweight and framework-agnostic:
//
//   - Generate – pure function that produces the fingerprint string.
//   - Validate – convenience wrapper that compares a stored fingerprint
//     with the newly generated one.
//   - Middleware – standard `net/http` middleware that injects the
//     fingerprint into the request context so that downstream handlers
//     can retrieve it via `GetFingerprintFromContext`.
//   - Context helpers – `SetFingerprintToContext` /
//     `GetFingerprintFromContext` allow manual manipulation when the
//     middleware is not used.
//
// The only external dependency is the sibling `clientip` package which
// extracts the real client IP address from a request.
//
// # Usage
//
// Import the package:
//
//	import "github.com/dmitrymomot/saaskit/pkg/fingerprint"
//
// Basic generation example:
//
//	fp := fingerprint.Generate(r) // *http.Request
//	log.Printf("client fingerprint: %s", fp)
//
// Validating against a stored value:
//
//	if !fingerprint.Validate(r, storedFP) {
//	    http.Error(w, "fingerprint mismatch", http.StatusUnauthorized)
//	    return
//	}
//
// Using the provided middleware:
//
//	http.Handle("/", fingerprint.Middleware(yourHandler))
//
// Within `yourHandler` you can later retrieve the value:
//
//	fp := fingerprint.GetFingerprintFromContext(r.Context())
//
// # Error Handling
//
// All functions are side-effect-free and do not return errors; the hash
// algorithm is deterministic. Make sure to handle the case where the
// generated fingerprint does not match the stored one.
//
// # Performance Considerations
//
// Generating a SHA-256 hash for the small amount of header data is fast
// and should not be a bottleneck. Benchmarks show fingerprint generation
// completes in approximately 1.6 microseconds per operation, making it
// suitable for high-traffic applications. The most expensive step is usually
// extracting the client IP—delegated to the `clientip` package—so cache
// or memoise that if you call `Generate` multiple times per request.
//
// See the unit tests in `fingerprint_test.go` for additional examples.
package fingerprint
