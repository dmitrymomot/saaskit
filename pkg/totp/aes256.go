package totp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

const (
	AESKeySize = 32 // Required key size for AES-256 (256 bits / 8 = 32 bytes)
)

// EncryptSecret encrypts the TOTP secret using AES-256-GCM.
// Returns the ciphertext as a base64-encoded string.
func EncryptSecret(plainText string, key []byte) (string, error) {
	if len(key) != AESKeySize {
		return "", errors.Join(ErrFailedToEncryptSecret, ErrInvalidEncryptionKeyLength)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Join(ErrFailedToEncryptSecret, err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.Join(ErrFailedToEncryptSecret, err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.Join(ErrFailedToEncryptSecret, err)
	}

	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// DecryptSecret decrypts the encrypted TOTP secret.
// Expects the ciphertext as a base64-encoded string.
func DecryptSecret(cipherTextBase64 string, key []byte) (string, error) {
	if len(key) != AESKeySize {
		return "", errors.Join(ErrFailedToDecryptSecret, ErrInvalidEncryptionKeyLength)
	}

	cipherText, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", errors.Join(ErrFailedToDecryptSecret, err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Join(ErrFailedToDecryptSecret, err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.Join(ErrFailedToDecryptSecret, err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherText) < nonceSize {
		return "", errors.Join(ErrFailedToDecryptSecret, ErrInvalidCipherTooShort)
	}
	nonce, cipherText := cipherText[:nonceSize], cipherText[nonceSize:]

	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", errors.Join(ErrFailedToDecryptSecret, err)
	}

	return string(plainText), nil
}

// GenerateEncryptionKey creates a new random 32-byte key suitable for AES-256 encryption.
// Returns the generated key or an error if the random number generation fails.
func GenerateEncryptionKey() ([]byte, error) {
	key := make([]byte, AESKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, errors.Join(ErrFailedToGenerateEncryptionKey, err)
	}
	return key, nil
}

// GenerateEncodedEncryptionKey generates a new random 32-byte key suitable for AES-256 encryption.
// Returns the generated key as a base64-encoded string or an error if the random number generation fails.
// This function is useful for generating a new key and storing it in the configuration.
func GenerateEncodedEncryptionKey() (string, error) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// GetEncryptionKey decodes the encryption key from the configuration.
// The key must be a 32-byte base64-encoded string.
// Returns the decoded key bytes or an error if decoding fails or the key length is invalid.
func GetEncryptionKey(cfg Config) ([]byte, error) {
	if cfg.EncryptionKey == "" {
		return nil, errors.Join(ErrFailedToLoadEncryptionKey, ErrEncryptionKeyNotSet)
	}

	key, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
	if err != nil {
		return nil, errors.Join(ErrFailedToLoadEncryptionKey, err)
	}

	if len(key) != AESKeySize {
		return nil, errors.Join(ErrFailedToLoadEncryptionKey, ErrInvalidEncryptionKeyLength)
	}

	return key, nil
}
