package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"strings"
)

// ParseToken verifies token signature and decodes payload.
// Returns ErrInvalidToken for malformed tokens, ErrSignatureInvalid for tampering.
func ParseToken[T any](token string, secret string) (T, error) {
	var payload T
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return payload, ErrInvalidToken
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return payload, ErrInvalidToken
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return payload, ErrInvalidToken
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	expectedSig := h.Sum(nil)[:8] // Match truncated signature from generation

	// Constant-time comparison prevents timing attacks
	if subtle.ConstantTimeCompare(sig, expectedSig) != 1 {
		return payload, ErrSignatureInvalid
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, err
	}

	return payload, nil
}
