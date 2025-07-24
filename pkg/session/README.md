# Session Package

A flexible session management package for Go web applications with pluggable storage and transport mechanisms.

## Features

- **Pluggable Storage**: Memory store included, easily extend with Redis, database, etc.
- **Pluggable Transport**: Cookie (default), header, and composite transports
- **Automatic Expiry**: Separate timeouts for anonymous and authenticated sessions
- **Activity Tracking**: Efficient activity updates with configurable threshold
- **Device Fingerprinting**: Optional fingerprint validation
- **Zero Dependencies**: Works out-of-box with memory store
- **Type Safe**: No reflection, compile-time safety
- **High Performance**: Zero allocations in hot paths

## Installation 

```bash
go get github.com/dmitrymomot/saaskit/pkg/session
```

## Quick Start

```go
import (
    "github.com/dmitrymomot/saaskit/pkg/cookie"
    "github.com/dmitrymomot/saaskit/pkg/session"
)

// Create cookie manager (required for default cookie transport)
cookieMgr, _ := cookie.New([]string{"your-secret-key"})

// Create session manager
manager := session.New(
    session.WithCookieManager(cookieMgr),
)

// In HTTP handler
func handler(w http.ResponseWriter, r *http.Request) {
    // Ensure session exists
    sess, err := manager.Ensure(r.Context(), w, r)

    // Store data
    sess.Set("key", "value")

    // Authenticate user
    manager.Authenticate(r.Context(), w, r, userID)
}
```

## Configuration

### Environment Variables

The session package supports configuration via environment variables:

```bash
SESSION_COOKIE_NAME=sid                    # Session cookie name
SESSION_ANON_IDLE_TIMEOUT=30m             # Anonymous session idle timeout
SESSION_ANON_MAX_LIFETIME=24h             # Anonymous session max lifetime
SESSION_AUTH_IDLE_TIMEOUT=2h              # Authenticated session idle timeout
SESSION_AUTH_MAX_LIFETIME=720h            # Authenticated session max lifetime (30 days)
SESSION_ACTIVITY_UPDATE_THRESHOLD=5m      # Min time between activity updates
SESSION_CLEANUP_INTERVAL=5m               # Cleanup interval (0 to disable)
```

Example using environment config:

```go
import (
    "github.com/dmitrymomot/saaskit/pkg/config"
    "github.com/dmitrymomot/saaskit/pkg/session"
)

// Load config from environment
var cfg session.Config
config.Load(&cfg)

// Create manager from config
manager := session.NewFromConfig(cfg,
    session.WithCookieManager(cookieMgr),
)
```

### Session Options

```go
manager := session.New(
    // Set cookie name (default: "sid")
    session.WithCookieName("my-session"),

    // Set timeouts
    session.WithIdleTimeout(30*time.Minute, 2*time.Hour), // anon, auth
    session.WithMaxLifetime(24*time.Hour, 30*24*time.Hour), // anon, auth

    // Activity update threshold
    session.WithActivityUpdateThreshold(5*time.Minute),

    // Cleanup interval (0 to disable)
    session.WithCleanupInterval(5*time.Minute),
)
```

### Custom Store

```go
// Implement the Store interface
type MyStore struct{}

func (s *MyStore) Create(ctx context.Context, session *Session) error { ... }
func (s *MyStore) Get(ctx context.Context, token string) (*Session, error) { ... }
func (s *MyStore) Update(ctx context.Context, session *Session) error { ... }
func (s *MyStore) UpdateActivity(ctx context.Context, token string, lastActivity time.Time) error { ... }
func (s *MyStore) Delete(ctx context.Context, token string) error { ... }
func (s *MyStore) DeleteExpired(ctx context.Context) error { ... }

// Use custom store
manager := session.New(
    session.WithStore(&MyStore{}),
)
```

### Transport Options

#### Header Transport (for APIs)

```go
manager := session.New(
    session.WithTransport(session.NewHeaderTransport("X-Session-Token")),
)

// Optional: custom prefix
manager := session.New(
    session.WithTransport(session.NewHeaderTransport(
        "Authorization",
        session.WithHeaderPrefix("Session "),
    )),
)
```

#### Composite Transport (multiple methods)

```go
// Try cookie first, then header
manager := session.New(
    session.WithTransport(session.NewCompositeTransport(
        session.NewCookieTransport(cookieMgr, "sid"),
        session.NewHeaderTransport("X-Session-Token"),
    )),
)
```

### Device Fingerprinting

```go
// Basic fingerprinting
manager := session.New(
    session.WithFingerprint(func(r *http.Request) string {
        return r.Header.Get("User-Agent")
    }),
)

// Using the fingerprint package
import "github.com/dmitrymomot/saaskit/pkg/fingerprint"

manager := session.New(
    session.WithFingerprint(fingerprint.Generate),
)

// Disable fingerprinting
manager := session.New(
    session.WithFingerprint(nil),
)
```

## Middleware

### Basic Middleware

```go
// Adds session to context if valid
mux.Use(manager.Middleware)

// In handler
sess, ok := session.FromContext(r.Context())
```

### Ensure Session Middleware

```go
// Always ensures a session exists
mux.Use(manager.EnsureSession)

// In handler
sess := session.MustFromContext(r.Context())
```

### Require Auth Middleware

```go
// Protected routes
mux.Handle("/admin", manager.RequireAuth(adminHandler))
```

## Session Operations

### Data Management

```go
// Set values
sess.Set("username", "john")
sess.Set("count", 42)
sess.Set("active", true)

// Get values with type helpers
username, ok := sess.GetString("username")
count, ok := sess.GetInt("count")
active, ok := sess.GetBool("active")

// Generic get
val, ok := sess.Get("key")

// Delete value
sess.Delete("temporary")

// Clear all data
sess.Clear()
```

### Authentication

```go
// Login
err := manager.Authenticate(ctx, w, r, userID)

// Check authentication
if sess.IsAuthenticated() {
    userID := sess.UserID
}

// Logout
err := manager.Destroy(ctx, w, r)
```

### Session Refresh

```go
// Extend session expiry
err := manager.Refresh(ctx, w, r)
```

## Examples

### Complete Web Application

```go
package main

import (
    "net/http"
    "github.com/dmitrymomot/saaskit/pkg/cookie"
    "github.com/dmitrymomot/saaskit/pkg/session"
)

func main() {
    // Setup
    cookieMgr, _ := cookie.New([]string{"secret-key"})
    sessionMgr := session.New(
        session.WithCookieManager(cookieMgr),
        session.WithFingerprint(customFingerprint),
    )

    // Routes
    mux := http.NewServeMux()

    // Public routes with optional session
    mux.Handle("/", sessionMgr.Middleware(homeHandler))

    // Routes that need session
    mux.Handle("/cart", sessionMgr.EnsureSession(cartHandler))

    // Protected routes
    mux.Handle("/account", sessionMgr.RequireAuth(accountHandler))

    http.ListenAndServe(":8080", mux)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    sess, ok := session.FromContext(r.Context())
    if ok && sess.IsAuthenticated() {
        // Show personalized content
    }
}

func customFingerprint(r *http.Request) string {
    // Custom logic
    return r.Header.Get("User-Agent") + r.Header.Get("Accept-Language")
}
```

### API with Header Transport

```go
manager := session.New(
    session.WithTransport(session.NewHeaderTransport("X-API-Key")),
    session.WithStore(redisStore),
)

// API endpoint
func apiHandler(w http.ResponseWriter, r *http.Request) {
    sess, err := manager.Get(r.Context(), r)
    if err != nil {
        http.Error(w, "Invalid session", 401)
        return
    }

    // Process API request
}
```

### Testing

```go
func TestMyHandler(t *testing.T) {
    // Setup manager with memory store
    cookieMgr, _ := cookie.New([]string{"test-secret"})
    manager := session.New(session.WithCookieManager(cookieMgr))

    // Create test request
    w := httptest.NewRecorder()
    r := httptest.NewRequest("GET", "/", nil)

    // Create session
    sess, _ := manager.Ensure(context.Background(), w, r)
    sess.Set("test", "value")

    // Add cookie to next request
    r2 := httptest.NewRequest("GET", "/", nil)
    for _, c := range w.Result().Cookies() {
        r2.AddCookie(c)
    }

    // Test with session
    handler := manager.Middleware(myHandler)
    handler.ServeHTTP(httptest.NewRecorder(), r2)
}
```

## Performance

The package is designed for high performance:

- Zero allocations in hot paths
- Efficient memory store with concurrent access
- Minimal overhead for session operations
- Benchmark results (on typical hardware):
    - Get session: ~200ns
    - Set value: ~300ns
    - Middleware: ~500ns

## Security Considerations

1. **Token Generation**: Uses crypto/rand for secure tokens
2. **Token Rotation**: Tokens are rotated on authentication
3. **Fingerprinting**: Optional device fingerprint validation
4. **Cookie Security**: HTTPOnly, Secure, SameSite settings
5. **Timing Attacks**: Constant-time string comparisons

## Best Practices

1. Always use HTTPS in production
2. Set appropriate session timeouts
3. Enable fingerprinting for sensitive applications
4. Rotate sessions on privilege changes
5. Clear sessions on logout
6. Monitor session storage size

## License

See the [LICENSE](../../LICENSE) file for details.
