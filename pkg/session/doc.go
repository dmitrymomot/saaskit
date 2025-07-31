// Package session provides flexible, high-performance session management
// for Go web applications. Features pluggable storage, multiple transports,
// automatic expiry, device fingerprinting, and efficient activity tracking
// through clean, composable interfaces.
//
// Storage-agnostic design accepts any Store implementation. Includes concurrent
// in-memory store. Session tokens delivered via pluggable Transport interface
// (cookies, headers, etc.).
//
// # Architecture
//
// Manager orchestrates session lifecycle via Transport (token handling) and
// Store (persistence). Config defines timeouts for anonymous/authenticated users
// and cleanup intervals. Background goroutine processes activity updates to
// keep hot paths allocation-free.
//
//	┌────────┐   token   ┌────────────┐
//	│ Client │ ────────► │  Transport │
//	└────────┘           └────────────┘
//	       ▲                   │
//	       │                   ▼
//	┌─────────────────────────────────┐
//	│            Manager              │
//	└─────────────────────────────────┘
//	       │   CRUD / TTL
//	       ▼
//	┌────────┐
//	│ Store  │ (memory, redis, …)
//	└────────┘
//
// # Usage
//
//	import (
//	    "github.com/dmitrymomot/saaskit/pkg/cookie"
//	    "github.com/dmitrymomot/saaskit/pkg/session"
//	)
//
//	// Cookie based sessions
//	cookieMgr, _ := cookie.New([]string{"secret-key"})
//	manager := session.New(
//	    session.WithCookieManager(cookieMgr), // adds default cookie transport
//	)
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    // Ensure returns an existing session or creates a new anonymous one
//	    sess, _ := manager.Ensure(r.Context(), w, r)
//	    sess.Set("foo", "bar")
//
//	    // Promote the session after login
//	    manager.Authenticate(r.Context(), w, r, userID)
//	}
//
// Header transport:
//
//	manager := session.New(
//	    session.WithTransport(session.NewHeaderTransport("X-Session-Token")),
//	)
//
// # Configuration
//
// Configuration via Option functions or Config struct with NewFromConfig.
// Environment variable support via DefaultConfig() for twelve-factor apps.
//
// # Error Handling
//
// Common error values returned by the package:
//
//   - ErrInvalidSession   – fingerprint mismatch
//   - ErrSessionExpired   – session has passed its expiry
//   - ErrSessionNotFound  – no session associated with token
//
// # Performance Considerations
//
// Hot paths achieve zero allocations with sub-microsecond latency.
//
// # Examples
//
// See the README and example tests in the package for additional recipes.
package session
