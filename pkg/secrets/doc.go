// Package secrets provides high-level helpers for encrypting and decrypting
// tenant secrets in multi-tenant SaaS applications.
//
// The package derives a compound 32-byte key from an application key and a
// workspace (tenant) key using HKDF-SHA-256. The derived key is then used with
// AES-256 in GCM mode to protect arbitrary byte slices or UTF-8 strings.
//
// On successful encryption the nonce is prepended to the ciphertext so that
// all necessary data is self-contained. All operations are constant-time with
// respect to secret material.
//
// # Architecture
//
//  1. Key validation – both `appKey` and `workspaceKey` must be exactly 32 bytes
//     (256 bits). Convenience helper `ValidateKeys` is provided.
//  2. Key derivation – HKDF(SHA-256) with `saltInfo = "go-saas-secrets-v1"`
//     yields the compound key. Errors are wrapped with `ErrKeyDerivationFailed`.
//  3. Encryption / Decryption – AES-GCM is used via the standard library. High
//     level helpers accept either raw byte slices (`EncryptBytes`, `DecryptBytes`)
//     or strings that are transparently base64-encoded/decoded
//     (`EncryptString`, `DecryptString`).
//
// # Usage
//
//	import "github.com/dmitrymomot/saaskit/pkg/secrets"
//
//	// Generate keys once and store securely
//	appKey, _ := secrets.GenerateKey()
//	workspaceKey, _ := secrets.GenerateKey()
//
//	// Encrypt
//	ct, err := secrets.EncryptString(appKey, workspaceKey, "super-secret")
//	if err != nil {
//	    // handle error
//	}
//
//	// Decrypt
//	plain, err := secrets.DecryptString(appKey, workspaceKey, ct)
//	if err != nil {
//	    // handle error
//	}
//
// # Error Handling
//
// All public functions return rich errors that wrap a sentinel package error
// such as `ErrEncryptionFailed` or `ErrInvalidCiphertext`. Use `errors.Is` to
// match against these sentinels.
//
// # Performance Considerations
//
// AES-GCM is hardware-accelerated on modern CPUs. Library allocations are
// limited to the minimum nonce and tag size plus the ciphertext payload.
package secrets
