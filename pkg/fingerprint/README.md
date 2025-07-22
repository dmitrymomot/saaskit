# Device Fingerprint Package

Server-side device fingerprinting to prevent session hijacking by binding sessions to specific browser/device combinations.

## Overview

This package provides HTTP request fingerprinting functionality to detect potential session hijacking attempts. It analyzes various request headers and client information to create a unique device identifier that can be validated across requests.

## Internal Usage

This package is internal to the project and provides device fingerprinting capabilities for session security and authentication handlers.

## Features

- **Request fingerprinting** - Creates unique 32-character hex identifiers from HTTP requests
- **Context integration** - Store and retrieve fingerprints from request context
- **Middleware support** - Ready-to-use middleware for automatic fingerprint generation
- **Sub-millisecond performance** - ~1.6μs per operation
- **Proxy-aware** - Uses internal clientip package for accurate IP detection
- **Header analysis** - User-Agent, Accept headers, IP, header order

## Usage

```go
import "github.com/dmitrymomot/saaskit/pkg/fingerprint"

// Generate fingerprint during login
fp := fingerprint.Generate(r)
session.Fingerprint = fp

// Validate on each request
if !fingerprint.Validate(r, session.Fingerprint) {
    // Session hijacking detected
    invalidateSession(session.ID)
    http.Error(w, "Unauthorized", 401)
    return
}
```

## API Reference

### Functions

```go
func Generate(r *http.Request) string
```
Creates a 32-character hex fingerprint from request headers and IP.

```go
func Validate(r *http.Request, sessionFingerprint string) bool
```
Compares current request fingerprint with stored value.

```go
func Middleware(next http.Handler) http.Handler
```
HTTP middleware that generates fingerprint and stores it in request context.

```go
func SetFingerprintToContext(ctx context.Context, fingerprint string) context.Context
```
Stores fingerprint in context for later retrieval.

```go
func GetFingerprintFromContext(ctx context.Context) string
```
Retrieves fingerprint from context, returns empty string if not found.

## Components Analyzed

- User-Agent, Accept-Language, Accept-Encoding, Accept headers
- Client IP (via internal clientip package)
- Header order pattern for stable headers: user-agent, accept, accept-language, accept-encoding, connection, upgrade-insecure-requests, sec-fetch-*, cache-control

## Additional Usage Scenarios

### Using the Built-in Middleware

```go
// Automatically generates and stores fingerprint in context
router.Use(fingerprint.Middleware)

// Later retrieve from context in handlers
func handler(w http.ResponseWriter, r *http.Request) {
    fp := fingerprint.GetFingerprintFromContext(r.Context())
    // fp contains the fingerprint for this request
}
```

### Custom Session Validation Middleware

```go
func sessionValidationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session := getSession(r)
        if session != nil && !fingerprint.Validate(r, session.Fingerprint) {
            invalidateSession(session.ID)
            http.Redirect(w, r, "/login", http.StatusFound)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Database schema
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    fingerprint VARCHAR(32) NOT NULL,
    expires_at TIMESTAMP NOT NULL
);
```

## Best Practices

### Integration Guidelines

- Use the provided `Middleware` for automatic fingerprint generation
- Store fingerprints in session data during authentication
- Validate fingerprints on sensitive operations
- Consider using context-based fingerprint retrieval for cleaner code

### Project-Specific Considerations

- Integrates with internal `clientip` package for accurate IP detection
- Designed to work with the project's session management system
- Performance overhead is minimal (~1.6μs per operation)

## Limitations

- **False positives**: Browser updates, language changes, network switches
- **Mitigation**: Graceful degradation (re-auth instead of logout) or fingerprint rotation