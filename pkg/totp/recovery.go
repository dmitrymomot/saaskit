package totp

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
)

// GenerateRecoveryCodes creates cryptographically secure backup codes for account recovery.
// Each code is a 16-character hexadecimal string (64 bits of entropy).
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

// HashRecoveryCode creates a SHA-256 hash for secure storage of recovery codes.
func HashRecoveryCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}

// VerifyRecoveryCode performs constant-time comparison to prevent timing attacks.
// Essential for security: comparison time must not reveal information about where differences occur.
func VerifyRecoveryCode(code, hashedCode string) bool {
	computedHash := HashRecoveryCode(code)

	// Use constant-time comparison to prevent timing-based side-channel attacks
	return subtle.ConstantTimeCompare(
		[]byte(computedHash),
		[]byte(hashedCode),
	) == 1
}
