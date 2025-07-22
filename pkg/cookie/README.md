# Cookie Package

Secure cookie management with signing, encryption, and flash messages support.

## Overview

This package provides a comprehensive cookie management system with security features including HMAC-SHA256 signing, AES-GCM encryption, and automatic flash message handling. It supports secret rotation for zero-downtime key updates and uses the options pattern for flexible configuration.

## Internal Usage

This package is internal to the project and provides cookie management capabilities for HTTP handlers, session management, and temporary data storage across requests.

## Features

- Basic cookie operations with secure defaults
- HMAC-SHA256 signed cookies to prevent tampering
- AES-GCM encrypted cookies for sensitive data
- Flash messages with automatic deletion after reading
- Secret rotation support for graceful key updates
- Options pattern for per-cookie configuration
- No external dependencies - uses only Go standard library

## Usage

### Basic Example

```go
import "github.com/dmitrymomot/saaskit/pkg/cookie"

// Initialize with secrets from environment
secrets := []string{
    os.Getenv("COOKIE_SECRET"),
    os.Getenv("COOKIE_SECRET_OLD"), // Optional for rotation
}
secrets = slices.DeleteFunc(secrets, func(s string) bool { return s == "" })

cookieMgr, err := cookie.New(secrets,
    cookie.WithSecure(true),
    cookie.WithDomain(".example.com"),
    cookie.WithHTTPOnly(true),
    cookie.WithSameSite(http.SameSiteLaxMode),
)
if err != nil {
    log.Fatal(err)
}
```

### Additional Usage Scenarios

#### Basic Cookies

```go
// Set a plain cookie
err := cookieMgr.Set(w, "theme", "dark")

// Get a cookie
theme, err := cookieMgr.Get(r, "theme")

// Delete a cookie
cookieMgr.Delete(w, "theme")
```

#### Signed Cookies

Signed cookies use HMAC-SHA256 to ensure the cookie value hasn't been tampered with:

```go
// Set a signed cookie
err := cookieMgr.SetSigned(w, "user_id", "12345")

// Get and verify a signed cookie
userID, err := cookieMgr.GetSigned(r, "user_id")
if errors.Is(err, cookie.ErrInvalidSignature) {
    // Cookie was tampered with
}
```

#### Encrypted Cookies

Encrypted cookies use AES-GCM for storing sensitive data:

```go
// Set an encrypted cookie
err := cookieMgr.SetEncrypted(w, "session_data", "sensitive info")

// Get and decrypt
data, err := cookieMgr.GetEncrypted(r, "session_data")
```

#### Flash Messages

Flash messages are automatically deleted after being read:

```go
// Set a flash message (any type)
type Alert struct {
    Type    string
    Message string
}

err := cookieMgr.SetFlash(w, r, "alert", Alert{
    Type:    "success",
    Message: "Profile updated successfully",
})

// Get flash message
var alert Alert
err := cookieMgr.GetFlash(w, r, "alert", &alert)
// Cookie is automatically deleted after reading
```

#### Per-Cookie Options

Override default options for specific cookies:

```go
// Set cookie with custom options
err := cookieMgr.Set(w, "preferences", "value",
    cookie.WithMaxAge(86400*30), // 30 days
    cookie.WithPath("/app"),
    cookie.WithSecure(false),     // Override for development
)
```

### Error Handling

```go
// Handle cookie not found
value, err := cookieMgr.Get(r, "theme")
if errors.Is(err, cookie.ErrCookieNotFound) {
    // Set default value
    value = "light"
}

// Handle invalid signature
data, err := cookieMgr.GetSigned(r, "auth_token")
if errors.Is(err, cookie.ErrInvalidSignature) {
    // Cookie was tampered with, clear it
    cookieMgr.Delete(w, "auth_token")
    // Redirect to login
}

// Handle decryption failure
session, err := cookieMgr.GetEncrypted(r, "session")
if errors.Is(err, cookie.ErrDecryptionFailed) {
    // Cookie couldn't be decrypted, possibly corrupted
    cookieMgr.Delete(w, "session")
}
```

## Best Practices

### Integration Guidelines

- Initialize the cookie manager once at application startup and reuse it
- Store secrets in environment variables, never hardcode them
- Use signed cookies for data integrity (user preferences, settings)
- Use encrypted cookies for sensitive data (session IDs, tokens)
- Use flash messages for temporary notifications across redirects
- Always handle cookie errors gracefully with fallback behavior

### Project-Specific Considerations

- Coordinate cookie names across handlers to avoid conflicts
- Use consistent MaxAge values for similar cookie types
- Consider domain settings for multi-subdomain deployments
- Enable Secure flag in production, disable for local development
- Use SameSite=Strict for sensitive operations, Lax for general use

### Security Best Practices

- Rotate secrets periodically using the multi-secret support
- Monitor for ErrInvalidSignature as potential tampering attempts
- Clear invalid cookies immediately upon detection
- Use HTTPS in production to protect cookie transmission
- Limit cookie scope with Path and Domain settings

## API Reference

### Types

```go
// Manager handles cookie operations with security features
type Manager struct {
    secrets  []string
    defaults Options
}

// Options configures cookie attributes
type Options struct {
    Path     string
    Domain   string
    MaxAge   int
    Secure   bool
    HttpOnly bool
    SameSite http.SameSite
}

// Option is a functional option for configuring cookies
type Option func(*Options)
```

### Functions

```go
// New creates a new cookie manager with the provided secrets
func New(secrets []string, opts ...Option) (*Manager, error)

// Option constructors
func WithPath(path string) Option
func WithDomain(domain string) Option
func WithMaxAge(seconds int) Option
func WithSecure(secure bool) Option
func WithHTTPOnly(httpOnly bool) Option
func WithSameSite(sameSite http.SameSite) Option
```

### Methods

```go
// Basic cookie operations
func (m *Manager) Set(w http.ResponseWriter, name, value string, opts ...Option) error
func (m *Manager) Get(r *http.Request, name string) (string, error)
func (m *Manager) Delete(w http.ResponseWriter, name string)

// Signed cookie operations
func (m *Manager) SetSigned(w http.ResponseWriter, name, value string, opts ...Option) error
func (m *Manager) GetSigned(r *http.Request, name string) (string, error)

// Encrypted cookie operations
func (m *Manager) SetEncrypted(w http.ResponseWriter, name, value string, opts ...Option) error
func (m *Manager) GetEncrypted(r *http.Request, name string) (string, error)

// Flash message operations
func (m *Manager) SetFlash(w http.ResponseWriter, r *http.Request, key string, value any) error
func (m *Manager) GetFlash(w http.ResponseWriter, r *http.Request, key string, dest any) error
```

### Error Types

```go
var (
    ErrNoSecret         = errors.New("cookie.no_secret")         // No secrets provided
    ErrSecretTooShort   = errors.New("cookie.secret_too_short")  // Secret less than 32 characters
    ErrInvalidSignature = errors.New("cookie.invalid_signature") // Signature verification failed
    ErrDecryptionFailed = errors.New("cookie.decryption_failed") // Decryption failed
    ErrCookieNotFound   = errors.New("cookie.not_found")         // Cookie not present
    ErrInvalidFormat    = errors.New("cookie.invalid_format")    // Malformed cookie value
)
```
