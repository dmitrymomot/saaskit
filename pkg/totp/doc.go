// Package totp provides a high-level API for generating, encrypting, validating, and managing
// Time-based One-Time Passwords (TOTP) and related recovery codes.
//
// This package bundles together everything an application needs to implement multi-factor
// authentication based on RFC 6238 including secret key creation, URI generation compatible
// with authenticator applications, one-time-password generation/validation, AES-256 encryption
// helpers for safely persisting secrets, and secure recovery-code utilities.
//
// By keeping functionality self-contained the package eliminates direct dependencies on third-party
// TOTP libraries and allows services to remain framework-agnostic while still following
// contemporary security best-practices.
//
// # Architecture
//
// Internally the package is divided into three cohesive layers.
//
//   • crypto   – helpers in aes256.go implement symmetric encryption/decryption of the secret key
//     with AES-256-GCM as well as random key generation utilities.
//
//   • totp     – functions in otp.go provide secret key generation (GenerateSecretKey), HOTP/TOTP
//     code calculation (GenerateTOTP/ValidateTOTP/GenerateHOTP) and convenient URI construction
//     (GetTOTPURI) for onboarding to Google Authenticator, 1Password and compatible apps.
//
//   • recovery – helpers in recovery.go create, hash and verify single-use recovery codes that can
//     be offered to users in case they permanently lose access to their authenticator device.
//
// Configuration such as the encryption key is loaded once per process via the env tag aware
// loader in config.go. The required environment variable name is TOTP_ENCRYPTION_KEY and it must
// contain a Base64 encoded 32-byte key suitable for AES-256.
//
// # Usage
//
// The minimal happy path for enrolling a user looks like this:
//
//	package main
//
//	import (
//	    "fmt"
//	    "github.com/dmitrymomot/saaskit/pkg/totp"
//	)
//
//	func main() {
//	    // 1. Create a brand-new secret
//	    secret, _ := totp.GenerateSecretKey()
//
//	    // 2. Persist the secret encrypted in your datastore
//	    key, _ := totp.LoadEncryptionKey()
//	    encSecret, _ := totp.EncryptSecret(secret, key)
//
//	    // 3. Display the bootstrap URI/QR code to the user
//	    uri, _ := totp.GetTOTPURI(totp.TOTPParams{
//	        Secret:      secret,
//	        AccountName: "alice@example.com",
//	        Issuer:      "Acme",
//	    })
//	    fmt.Println(uri)
//
//	    // 4. Later – validate an OTP provided by the user
//	    ok, _ := totp.ValidateTOTP(secret, "123456")
//	    fmt.Println(ok)
//	}
//
// Additional helpers exist for generating time-window agnostic codes (GenerateTOTPWithTime),
// producing export-friendly Base64 keys (GenerateEncodedEncryptionKey) and creating or verifying
// recovery codes (GenerateRecoveryCodes, HashRecoveryCode, VerifyRecoveryCode).
//
// # Error Handling
//
// Every exported operation returns a descriptive error that may be wrapped using errors.Join.
// Inspect errors with errors.Is against package level sentinels such as ErrInvalidSecret,
// ErrFailedToEncryptSecret, ErrInvalidOTP etc.
//
// # See Also
//
//   • RFC 4226 – HMAC-Based One-Time Password (HOTP) Algorithm
//   • RFC 6238 – Time-Based One-Time Password (TOTP) Algorithm
//
// To explore more usage scenarios refer to the package level examples and unit-tests.
package totp
