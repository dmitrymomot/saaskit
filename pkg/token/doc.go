// Package token provides compact, signed tokens for embedding JSON payloads.
//
// Tokens use HMAC-SHA256 with truncated 8-byte signatures for balance between
// security and compactness. Suitable for email confirmations, password resets,
// and invite links. Not recommended for high-value or long-lived tokens.
//
// Token format: base64url(payload).base64url(signature)
//
// The 8-byte signature provides ~2^32 collision resistance, sufficient for
// typical short-lived application tokens but not cryptographically strong
// enough for sensitive operations.
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
//	tok, err := token.GenerateToken(Payload{"42", time.Now().Add(time.Hour).Unix()}, secret)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	var p Payload
//	if p, err = token.ParseToken[Payload](tok, secret); err != nil {
//	    log.Fatal(err)
//	}
//
// Returns ErrInvalidToken for malformed tokens and ErrSignatureInvalid
// for signature mismatches. Uses only standard library with single SHA-256
// hash per operation.
package token
