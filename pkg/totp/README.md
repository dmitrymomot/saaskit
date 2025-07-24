# TOTP Package

A secure Time-based One-Time Password (TOTP) implementation with AES-256 encryption for two-factor authentication systems.

## Overview

The `totp` package provides a complete solution for implementing secure two-factor authentication (2FA) using Time-based One-Time Passwords (TOTP). It offers functionality for generating and validating TOTP codes compliant with RFC 6238, managing secrets with AES-256-GCM encryption, and handling recovery codes. The package is thread-safe and suitable for concurrent use in production applications.

## Features

- RFC 6238 compliant TOTP generation and validation
- AES-256-GCM encryption for secure storage of TOTP secrets
- Secure recovery code generation and validation
- QR code URI generation compatible with all authenticator apps
- Constant-time comparison to prevent timing attacks
- Comprehensive error handling with proper error wrapping
- Thread-safe implementation with no global state

## Usage

### Setting Up Encryption

```go
import (
	"encoding/base64"
	"fmt"
	"github.com/dmitrymomot/saaskit/pkg/totp"
)

// Generate a new encryption key (store this securely)
key, err := totp.GenerateEncryptionKey()
if err != nil {
	switch {
	case errors.Is(err, totp.ErrFailedToGenerateEncryptionKey):
		// Handle specific error
	default:
		// Handle other errors
	}
}
// Encode key for storage in environment variables
encodedKey := base64.StdEncoding.EncodeToString(key)
fmt.Println("Save this key in your secure configuration:", encodedKey)
// Returns: A base64-encoded 32-byte encryption key
```

### Using the CLI Tool

For convenience, you can use the included CLI tool to generate both a TOTP secret key and an encoded encryption key:

```bash
# Run the CLI tool from the project root
go run internal/pkg/totp/cmd/main.go
```

This will output both values:

```
Generated Encoded Encryption Key (for TOTP_ENCRYPTION_KEY env var):
———
OUVdVgolBab6kePjN3s5fUZiqTdOIydh+zL5vn0Eu30=
———
```

You can then set the encryption key in your environment:

```bash
export TOTP_ENCRYPTION_KEY="OUVdVgolBab6kePjN3s5fUZiqTdOIydh+zL5vn0Eu30="
```

### Generating and Encrypting TOTP Secrets

```go
// Generate a new TOTP secret for a user
secret, err := totp.GenerateSecretKey()
if err != nil {
	// Handle error
}
// Returns: A base32-encoded secret like "JBSWY3DPEHPK3PXP"

// Encrypt the secret before storage
encryptedSecret, err := totp.EncryptSecret(secret, key)
if err != nil {
	// Handle error
}
// Returns: A base64-encoded encrypted string

// Create a URI for QR code display
params := totp.TOTPParams{
	Secret:      secret,
	AccountName: "user@example.com",
	Issuer:      "MyApp",
}
uri, err := totp.GetTOTPURI(params)
if err != nil {
	// Handle error
}
// Returns: "otpauth://totp/MyApp:user%40example.com?algorithm=SHA1&digits=6&issuer=MyApp&period=30&secret=JBSWY3DPEHPK3PXP"

// Use this URI with a QR code generator library to display to the user
```

### Validating TOTP Codes

```go
// Retrieve encrypted secret from storage and decrypt it
key, err := totp.LoadEncryptionKey() // Load from environment
if err != nil {
	// Handle error
}

secret, err := totp.DecryptSecret(encryptedSecret, key)
if err != nil {
	// Handle error
}

// Validate a TOTP code provided by the user
userProvidedCode := "123456" // From user input

valid, err := totp.ValidateTOTP(secret, userProvidedCode)
if err != nil {
	switch {
	case errors.Is(err, totp.ErrInvalidSecret):
		// Handle invalid secret format
	case errors.Is(err, totp.ErrInvalidOTP):
		// Handle invalid OTP format
	default:
		// Handle other errors
	}
}

if valid {
	// Authentication successful, grant access
} else {
	// Authentication failed
}

// Generate a current TOTP code (useful for testing)
currentCode, err := totp.GenerateTOTP(secret)
if err != nil {
	// Handle error
}
// Returns: A 6-digit code like "123456"
```

### Recovery Codes

```go
// Generate a set of recovery codes (typically done during 2FA setup)
recoveryCodes, err := totp.GenerateRecoveryCodes(8) // Generate 8 codes
if err != nil {
	// Handle error
}
// Returns: ["1A2B3C4D", "5E6F7G8H", ...] (8 codes)

// Hash and store codes in database
var hashedCodes []string
for _, code := range recoveryCodes {
	hashedCode := totp.HashRecoveryCode(code)
	hashedCodes = append(hashedCodes, hashedCode)
	// Store hashedCode in database
}

// Provide the original unhashed codes to user for backup

// Validating a recovery code during account recovery
userProvidedCode := "1A2B3C4D" // From user input

// Check against stored hashed codes
for _, hashedCode := range hashedCodes {
	if totp.VerifyRecoveryCode(userProvidedCode, hashedCode) {
		// Valid recovery code - remove from database and reset 2FA
		break
	}
}
```

### Custom TOTP Parameters

```go
// Create custom TOTP parameters
customParams := totp.TOTPParams{
	Secret:      secret,
	AccountName: "user@example.com",
	Issuer:      "MyApp",
	Algorithm:   "SHA1",
	Digits:      8,        // 8 digits instead of default 6
	Period:      60,       // 60 second period instead of default 30
}

// Generate URI with custom parameters
customURI, err := totp.GetTOTPURI(customParams)
if err != nil {
	// Handle error
}

// ValidateTOTP uses default parameters (6 digits, 30 seconds)
// For custom validation, you would need to implement it using the GenerateHOTP function
```

## Best Practices

1. **Secret Management**:
    - Always encrypt TOTP secrets before storage
    - Store encryption keys securely (environment variables, key vaults)
    - Use different encryption keys for different environments
    - Never log or expose secrets in plaintext

2. **Authentication Security**:
    - Implement rate limiting for TOTP attempts
    - Add exponential backoff for repeated failures
    - Set up proper audit logging for authentication attempts
    - Consider browser/IP fingerprinting for additional security

3. **Recovery Codes**:
    - Only store hashed recovery codes
    - Use a one-time-use policy for recovery codes
    - Notify users when recovery codes are used
    - Allow generation of new recovery codes

4. **Implementation**:
    - Use constant-time comparison for validation
    - Follow RFC 6238 specification
    - Maintain backward compatibility when rotating TOTP algorithms
    - Test with actual authenticator apps (Google Authenticator, Authy)

## API Reference

### Types

```go
// TOTPParams contains the parameters for TOTP URI generation
type TOTPParams struct {
	Secret      string // TOTP secret key (required)
	AccountName string // Name of the account (required)
	Issuer      string // Name of the issuer (required)
	Algorithm   string // Algorithm used (optional, default "SHA1")
	Digits      int    // Number of digits (optional, default 6)
	Period      int    // Period in seconds (optional, default 30)
}
```

### TOTP Functions

```go
// GenerateSecretKey generates a new Base32-encoded secret key for TOTP
func GenerateSecretKey() (string, error)

// GetTOTPURI creates a properly encoded TOTP URI for use with authenticator apps
func GetTOTPURI(params TOTPParams) (string, error)

// ValidateTOTP validates the TOTP code provided by the user
func ValidateTOTP(secret, otp string) (bool, error)

// GenerateTOTP generates a TOTP code for the current time period
func GenerateTOTP(secret string) (string, error)

// GenerateHOTP generates an HOTP code (internal function)
func GenerateHOTP(key []byte, counter int64, digits int) int

// GenerateTOTPWithTime generates a TOTP code for a specific time
func GenerateTOTPWithTime(secret string, t time.Time) (string, error)
```

### Encryption Functions

```go
// EncryptSecret encrypts a TOTP secret using AES-256-GCM
func EncryptSecret(plainText string, key []byte) (string, error)

// DecryptSecret decrypts an encrypted TOTP secret
func DecryptSecret(cipherTextBase64 string, key []byte) (string, error)

// GenerateEncryptionKey creates a new random 32-byte key for AES-256
func GenerateEncryptionKey() ([]byte, error)

// GenerateEncodedEncryptionKey generates a base64-encoded encryption key
func GenerateEncodedEncryptionKey() (string, error)

// LoadEncryptionKey loads the encryption key from environment variables
func LoadEncryptionKey() ([]byte, error)
```

### Recovery Code Functions

```go
// GenerateRecoveryCodes generates a set of recovery codes
func GenerateRecoveryCodes(count int) ([]string, error)

// HashRecoveryCode creates a hash of a recovery code for secure storage
func HashRecoveryCode(code string) string

// VerifyRecoveryCode performs a secure constant-time comparison
func VerifyRecoveryCode(code, hashedCode string) bool
```

### Error Types

```go
var ErrFailedToEncryptSecret = errors.New("failed to encrypt TOTP secret")
var ErrFailedToDecryptSecret = errors.New("failed to decrypt TOTP secret")
var ErrInvalidCipherTooShort = errors.New("cipher text too short")
var ErrFailedToGenerateEncryptionKey = errors.New("failed to generate encryption key")
var ErrFailedToLoadEncryptionKey = errors.New("failed to load encryption key")
var ErrInvalidEncryptionKeyLength = errors.New("invalid encryption key length")
var ErrFailedToGenerateSecretKey = errors.New("failed to generate TOTP secret key")
var ErrFailedToValidateTOTP = errors.New("failed to validate TOTP")
var ErrMissingSecret = errors.New("missing secret")
var ErrInvalidSecret = errors.New("invalid secret")
var ErrMissingAccountName = errors.New("missing account name")
var ErrMissingIssuer = errors.New("missing issuer")
var ErrEncryptionKeyNotSet = errors.New("TOTP encryption key not set")
var ErrInvalidOTP = errors.New("invalid OTP format")
var ErrInvalidRecoveryCodeCount = errors.New("invalid recovery code count, must be greater than 0")
var ErrFailedToGenerateRecoveryCode = errors.New("failed to generate recovery code")
var ErrFailedToGenerateTOTP = errors.New("failed to generate TOTP")
```

### Configuration

The package uses the following environment variable:

```
TOTP_ENCRYPTION_KEY = base64-encoded 32-byte key
```
