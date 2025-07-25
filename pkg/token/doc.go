// Package token provides small, dependency-free helpers for creating and
// verifying signed tokens that embed an arbitrary JSON-encoded payload.
//
// A token has the following shape:
//
//	base64url(payload).base64url(signature)
//
// where
//
//   - payload  – raw JSON-encoded bytes of the generic payload value
//   - signature – first 8 bytes of a HMAC-SHA256 digest calculated over
//     the payload bytes with the supplied secret.
//
// The shortened 8-byte signature keeps the token compact while still
// providing a sufficient collision resistance for typical application
// usage such as e-mail confirmations, password resets, invite links, etc.
// Do not use it for high-value or long-lived tokens where a full MAC
// would be more appropriate.
//
// # Usage
//
//	import "github.com/dmitrymomot/saaskit/pkg/token"
//
//	type Payload struct {
//	    UserID string `json:"uid"`
//	    Exp    int64  `json:"exp"`
//	}
//
//	const secret = "my-very-strong-secret"
//
//	// generate
//	tok, err := token.GenerateToken(Payload{"42", time.Now().Add(time.Hour).Unix()}, secret)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// parse & verify
//	var p Payload
//	if p, err = token.ParseToken[Payload](tok, secret); err != nil {
//	    log.Fatal(err)
//	}
//
// # Error Handling
//
// The package returns ErrInvalidToken when the token is malformed and
// ErrSignatureInvalid when the embedded signature does not match.
//
// # Performance
//
// The implementation relies only on the Go standard library and performs
// a single SHA-256 hash per call, which is usually negligible compared
// with network round-trips.
//
// Example usage can be found in the package tests.
package token
