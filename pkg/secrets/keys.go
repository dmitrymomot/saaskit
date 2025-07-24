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

// ValidateKeys checks that both keys are the correct length
func ValidateKeys(appKey, workspaceKey []byte) error {
	if len(appKey) != KeySize {
		return ErrInvalidAppKey
	}
	if len(workspaceKey) != KeySize {
		return ErrInvalidWorkspaceKey
	}
	return nil
}

// deriveKey creates a compound key from app and workspace keys using HKDF
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

// GenerateKey creates a new random 32-byte key suitable for encryption
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}
