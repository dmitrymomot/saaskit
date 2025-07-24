# Token Package

A lightweight, secure token generation and validation library with HMAC signatures.

## Overview

The `token` package provides a simple way to create and validate secure tokens with HMAC-SHA256 signatures. It's designed for applications that need a lightweight token solution without the complexity of JWT. The package is thread-safe and can be used concurrently without any issues.

## Features

- Type-safe payload handling with Go generics
- HMAC-SHA256 signatures for security
- Base64URL encoding for URL-safe tokens
- Simple, intuitive API with minimal code
- Zero external dependencies
- Constant-time signature comparison for security
- Thread-safe implementation for concurrent usage

## Usage

### Generating Tokens

```go
import "github.com/dmitrymomot/saaskit/pkg/token"

// Define your payload structure
type UserPayload struct {
    ID    int    `json:"id"`
    Email string `json:"email"`
    Role  string `json:"role"`
}

// Create a payload
payload := UserPayload{
    ID:    123,
    Email: "user@example.com",
    Role:  "admin",
}

// Your secret key for signing tokens
secret := "your-secret-key"

// Generate a token
tokenStr, err := token.GenerateToken(payload, secret)
if err != nil {
    // Handle error
}
// tokenStr = "eyJpZCI6MTIzLCJlbWFpbCI6InVzZXJAZXhhbXBsZS5jb20iLCJyb2xlIjoiYWRtaW4ifQ.I2SuLRl4BbY"
```

### Validating Tokens

```go
import (
    "errors"
    "github.com/dmitrymomot/saaskit/pkg/token"
)

// Parse and validate the token
var parsedPayload UserPayload
parsedPayload, err = token.ParseToken[UserPayload](tokenStr, secret)
if err != nil {
    switch {
    case errors.Is(err, token.ErrInvalidToken):
        // Handle invalid token format
    case errors.Is(err, token.ErrSignatureInvalid):
        // Handle invalid signature
    default:
        // Handle other errors
    }
    return
}

// Use the validated payload
// parsedPayload.ID = 123
// parsedPayload.Email = "user@example.com"
// parsedPayload.Role = "admin"
```

### With Expiration Time

```go
import (
    "time"
    "github.com/dmitrymomot/saaskit/pkg/token"
)

// Add expiration time to your payload
type TokenWithExpiry struct {
    UserID int       `json:"user_id"`
    Exp    time.Time `json:"exp"`
}

// Create token with expiration
expToken := TokenWithExpiry{
    UserID: 123,
    Exp:    time.Now().Add(24 * time.Hour), // Expires in 24 hours
}

token, err := token.GenerateToken(expToken, secret)
if err != nil {
    // Handle error
}
// token = "eyJ1c2VyX2lkIjoxMjMsImV4cCI6IjIwMjUtMDUtMDJUMTg6MzM6MTUrMDM6MDAifQ.xKQ_r8j0Ew"

// When validating, check expiration
var parsed TokenWithExpiry
parsed, err = token.ParseToken[TokenWithExpiry](token, secret)
if err != nil {
    // Handle token validation error
}

if time.Now().After(parsed.Exp) {
    // Token has expired
}
```

## Best Practices

1. **Secret Management**:
    - Store secrets securely, not in code or version control
    - Use environment variables or a secret management service
    - Use different secrets for different environments

2. **Token Design**:
    - Include expiration time in your payload for time-limited tokens
    - Consider adding a unique token ID for revocation capability
    - Include only necessary data in the payload to keep tokens compact

3. **Security Considerations**:
    - Validate all tokens before trusting their contents
    - Always use HTTPS when transmitting tokens
    - Consider token rotation for long-lived sessions

4. **Error Handling**:
    - Use `errors.Is()` for checking specific error types
    - Don't reveal detailed error information to clients
    - Implement appropriate logging for failed validation attempts

## API Reference

### Functions

```go
func GenerateToken[T any](payload T, secret string) (string, error)
```

Creates a token by JSON encoding the payload and appending an 8-byte truncated HMAC-SHA256 signature.

```go
func ParseToken[T any](token string, secret string) (T, error)
```

Verifies the token's signature and decodes the JSON payload into the generic type.

### Error Types

```go
var ErrInvalidToken = errors.New("invalid token format")
var ErrSignatureInvalid = errors.New("signature mismatch")
```

## Implementation Details

Tokens are generated in the format `payload.signature` where:

1. `payload` is the Base64URL-encoded JSON representation of your data
2. `signature` is the truncated (8 bytes) HMAC-SHA256 signature of the payload, also Base64URL-encoded

This creates compact tokens that can be safely used in URLs and cookies.
