// Package session provides a flexible session management system for web applications.
//
// The package supports multiple storage backends (memory, Redis, database) and
// transport mechanisms (cookies, headers, JWT) through clean interfaces.
//
// Features:
//   - Pluggable storage backends via Store interface
//   - Pluggable transport mechanisms via Transport interface
//   - Automatic session expiry and cleanup
//   - Optional device fingerprinting
//   - Separate timeouts for anonymous and authenticated sessions
//   - Activity tracking with configurable update threshold
//   - Zero dependencies (except for cookie transport)
//
// Basic usage:
//
//	// Create a session manager with default settings
//	cookieMgr, _ := cookie.New([]string{"your-secret-key"})
//	manager := session.New(
//	    session.WithCookieManager(cookieMgr),
//	)
//
//	// In HTTP handler
//	sess, _ := manager.Ensure(ctx, w, r)
//	sess.Set("key", "value")
//
//	// Authenticate a session
//	manager.Authenticate(ctx, w, r, userID)
//
// With custom transport:
//
//	// API with header transport
//	manager := session.New(
//	    session.WithTransport(transport.NewHeader("X-Session-Token")),
//	)
//
// With fingerprinting:
//
//	manager := session.New(
//	    session.WithFingerprint(func(r *http.Request) string {
//	        // Custom fingerprint logic
//	        return generateFingerprint(r)
//	    }),
//	)
package session
