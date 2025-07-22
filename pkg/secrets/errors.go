package secrets

import "errors"

var (
	// Key validation errors
	ErrInvalidAppKey       = errors.New("invalid app key: must be 32 bytes")
	ErrInvalidWorkspaceKey = errors.New("invalid workspace key: must be 32 bytes")

	// Encryption/decryption errors
	ErrEncryptionFailed  = errors.New("encryption failed")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")

	// Key derivation errors
	ErrKeyDerivationFailed = errors.New("key derivation failed")
)
