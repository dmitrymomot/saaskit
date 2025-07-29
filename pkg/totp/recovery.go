package totp

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
)

// GenerateRecoveryCodes generates N recovery codes.
// Returns an error if count is less than 1.
func GenerateRecoveryCodes(count int) ([]string, error) {
	if count < 1 {
		return nil, ErrInvalidRecoveryCodeCount
	}

	codes := make([]string, count)
	for i := range count {
		codeBytes := make([]byte, 8)
		if _, err := rand.Read(codeBytes); err != nil {
			return nil, errors.Join(ErrFailedToGenerateRecoveryCode, err)
		}
		code := fmt.Sprintf("%X", codeBytes)
		codes[i] = code
	}
	return codes, nil
}

// HashRecoveryCode hashes a recovery code using SHA-256.
func HashRecoveryCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}

// VerifyRecoveryCode performs a secure constant-time comparison of a recovery code
// against its hash to prevent timing attacks. Returns true if the code is valid.
func VerifyRecoveryCode(code, hashedCode string) bool {
	// Compute hash of provided code
	computedHash := HashRecoveryCode(code)

	// Convert both hashes to byte slices for constant-time comparison
	// This prevents timing attacks by ensuring the comparison takes
	// the same amount of time regardless of where differences occur
	return subtle.ConstantTimeCompare(
		[]byte(computedHash),
		[]byte(hashedCode),
	) == 1
}
