package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptString encrypts a string using compound key from app and workspace keys.
// Returns base64-encoded ciphertext.
func EncryptString(appKey, workspaceKey []byte, plaintext string) (string, error) {
	ciphertext, err := EncryptBytes(appKey, workspaceKey, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a base64-encoded ciphertext back to string.
func DecryptString(appKey, workspaceKey []byte, ciphertext string) (string, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", errors.Join(ErrInvalidCiphertext, err)
	}

	plaintextBytes, err := DecryptBytes(appKey, workspaceKey, ciphertextBytes)
	if err != nil {
		return "", err
	}

	return string(plaintextBytes), nil
}

// EncryptBytes encrypts raw bytes using compound key from app and workspace keys.
// Returns ciphertext in format: nonce + encrypted data + tag
func EncryptBytes(appKey, workspaceKey []byte, data []byte) ([]byte, error) {
	// Validate keys
	if err := ValidateKeys(appKey, workspaceKey); err != nil {
		return nil, err
	}

	// Derive compound key
	key, err := deriveKey(appKey, workspaceKey)
	if err != nil {
		return nil, err
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Join(ErrEncryptionFailed, err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Join(ErrEncryptionFailed, err)
	}

	// Generate nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Join(ErrEncryptionFailed, err)
	}

	// Encrypt data
	// Prepend nonce to ciphertext for storage
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)

	return ciphertext, nil
}

// DecryptBytes decrypts ciphertext back to raw bytes.
// Expects ciphertext in format: nonce + encrypted data + tag
func DecryptBytes(appKey, workspaceKey []byte, ciphertext []byte) ([]byte, error) {
	// Validate keys
	if err := ValidateKeys(appKey, workspaceKey); err != nil {
		return nil, err
	}

	// Derive compound key
	key, err := deriveKey(appKey, workspaceKey)
	if err != nil {
		return nil, err
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Join(ErrDecryptionFailed, err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Join(ErrDecryptionFailed, err)
	}

	// Extract nonce
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Join(ErrDecryptionFailed, err)
	}

	return plaintext, nil
}
