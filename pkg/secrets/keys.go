package secrets

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	// KeySize is the required size for both app and workspace keys
	KeySize = 32 // 256 bits for AES-256

	// saltInfo is used for HKDF key derivation to provide domain separation
	saltInfo = "go-saas-secrets-v1"
)

// ValidateKeys checks that both keys are the correct length.
// This function uses constant-time validation to prevent timing attacks.
func ValidateKeys(appKey, workspaceKey []byte) error {
	// Perform both validations to ensure constant-time behavior
	validApp := len(appKey) == KeySize
	validWorkspace := len(workspaceKey) == KeySize

	// Return appropriate error after both checks complete
	if !validApp {
		return ErrInvalidAppKey
	}
	if !validWorkspace {
		return ErrInvalidWorkspaceKey
	}
	return nil
}

// deriveKey creates a compound key from app and workspace keys using HKDF.
// The caller is responsible for clearing the returned key from memory using clearBytes()
// when it is no longer needed.
func deriveKey(appKey, workspaceKey []byte) ([]byte, error) {
	// Create HKDF reader with SHA-256
	hkdfReader := hkdf.New(sha256.New, appKey, workspaceKey, []byte(saltInfo))

	// Derive a 32-byte key
	derivedKey := make([]byte, KeySize)
	if _, err := io.ReadFull(hkdfReader, derivedKey); err != nil {
		return nil, errors.Join(ErrKeyDerivationFailed, err)
	}

	return derivedKey, nil
}

// clearBytes securely zeros out a byte slice to remove sensitive data from memory.
// This is a defense-in-depth measure to minimize the time sensitive key material
// remains in memory after use.
func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ClearBytesForTesting exposes clearBytes for testing purposes only.
// This should not be used in production code.
func ClearBytesForTesting(b []byte) {
	clearBytes(b)
}

// GenerateKey creates a new random 32-byte key suitable for encryption
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}
