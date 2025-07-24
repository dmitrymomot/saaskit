// Package cookie provides a secure and convenient HTTP cookie manager for Go applications.
//
// It wraps Go's net/http `http.Cookie` type with higher-level helpers for creating, reading,
// deleting and upgrading cookies as well as advanced capabilities such as cryptographic
// signing, authenticated encryption and one-time flash messages.
//
// # Overview
//
// The `Manager` type is the entry point. It is initialised with one or more secret keys and
// a set of default cookie `Options`. Secrets are used for both HMAC-SHA256 signatures and
// AES-GCM encryption ensuring tamper detection and confidentiality.
//
// Once created you can:
//
//   • Set(), Get(), Delete() – plain cookies
//   • SetSigned(), GetSigned() – signed cookies (integrity only)
//   • SetEncrypted(), GetEncrypted() – encrypted cookies (integrity + privacy)
//   • SetFlash(), GetFlash() – single-use JSON-encoded flash messages
//
// # Architecture
//
// Signing uses `crypto/hmac` with SHA-256 over the base64 encoded value. Encryption uses
// AES-256 in GCM mode with a randomly generated nonce that is prepended to the ciphertext.
// Multiple secrets are supported to enable key rotation – the first is used for writing,
// the rest for reading.
//
// # Usage
//
//	import "github.com/dmitrymomot/saaskit/pkg/cookie"
//
//	// secrets must be at least 32 bytes
//	man, err := cookie.New([]string{os.Getenv("COOKIE_SECRET")})
//	if err != nil { log.Fatal(err) }
//
//	http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
//	    _ = man.SetSigned(w, "session", "user-id")
//	})
//
//	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
//	    id, err := man.GetSigned(r, "session")
//	    _ = id
//	    _ = err
//	})
//
// # Configuration
//
// The `Config` struct allows the manager to be constructed from environment variables via
// github.com/caarlos0/env. Only non-zero fields are applied.
//
//	cfg := cookie.DefaultConfig()
//	_ = env.Parse(&cfg)
//	man, _ := cookie.NewFromConfig(cfg)
//
// # Error Handling
//
// Package-level sentinel errors are returned for common failure scenarios such as
// `ErrCookieNotFound`, `ErrInvalidSignature` and `ErrDecryptionFailed` so callers can use
// `errors.Is`.
//
// # Performance Considerations
//
// All operations are performed in memory and rely on Go's standard library cryptography
// primitives which are highly optimised. Encryption incurs an allocation for the ciphertext;
// keep values small.
//
// # See Also
//
//   • net/http – underlying cookie implementation.
package cookie
