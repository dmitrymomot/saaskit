// Package session provides a flexible, high-performance session management
// system for Go web applications. It offers pluggable storage back-ends,
// multiple transport mechanisms, automatic expiry, device fingerprinting and
// efficient activity tracking — all exposed through clean, composable
// interfaces.
//
// The package is storage-agnostic: any datastore that satisfies the Store
// interface can be plugged in. A concurrent in-memory implementation ships out
// of the box. Likewise, session tokens can be delivered through different
// transports such as HTTP cookies or custom headers via the Transport
// interface.
//
// # Architecture
//
// A Manager orchestrates the session life-cycle. It relies on a Transport to
// extract / set the session token on every request and on a Store to persist
// session state. A Config struct defines idle / max timeouts for anonymous and
// authenticated users as well as the cleanup interval. An internal goroutine
// processes activity updates so that hot paths remain allocation-free.
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
// Most knobs are exposed via Option functions (e.g. WithIdleTimeout) or by
// passing a Config struct to NewFromConfig. Twelve-factor applications can
// populate the same fields from environment variables through DefaultConfig().
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
// Hot paths (token lookup + map read) perform zero allocations and exhibit
// sub-microsecond latency in benchmarks on typical hardware.
//
// # Examples
//
// See the README and example tests in the package for additional recipes.
package session
