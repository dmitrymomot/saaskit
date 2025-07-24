// Package jwt provides utilities for generating, parsing, and validating
// JSON Web Tokens (JWT) as well as HTTP middleware and context helpers for Go
// services.
//
// The implementation focuses on the HS256 (HMAC-SHA256) algorithm. A high-level
// Service type wraps signing and verification while accepting any JSON-
// serialisable claims structure. StandardClaims is provided as a convenient
// struct mirroring the RFC 7519 registered fields.
//
// Context helper functions make it easy to attach a token and its claims to a
// context.Context and retrieve them later in the request lifecycle.
//
// # Architecture
//
//   • Service – signs and verifies tokens.
//   • context.go – helper functions for working with context.
//   • middleware.go – HTTP middleware that extracts a token (from header,
//     cookie, query, or custom header) and injects verified claims into the
//     request context.
//   • errors.go – sentinel error values returned by the package.
//
// # Usage
//
// import "github.com/dmitrymomot/saaskit/pkg/jwt"
//
// // Initialise the service.
// svc, err := jwt.NewFromString("super-secret")
// if err != nil {
//     // handle error
// }
//
// // Generate a token.
// claims := jwt.StandardClaims{
//     Subject:   "123",
//     ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
// }
// token, err := svc.Generate(claims)
//
// // Parse the token back.
// var parsed jwt.StandardClaims
// if err := svc.Parse(token, &parsed); err != nil {
//     // handle invalid / expired token
// }
//
// // Use middleware in an http.Handler chain.
// http.Handle("/api", jwt.Middleware(svc)(yourHandler))
//
// # Error Handling
//
// Errors such as ErrExpiredToken or ErrInvalidSignature are returned as
// sentinel variables and can be compared using errors.Is.
//
// # Performance Considerations
//
// The package uses only the Go standard library, avoiding external
// dependencies and allocations where possible. Signing keys are kept in memory
// only. No reflection is used during normal operation.
package jwt
