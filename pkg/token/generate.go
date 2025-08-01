package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
)

// GenerateToken creates a signed token from a JSON-encodable payload.
// Uses 8-byte truncated HMAC-SHA256 for compact tokens suitable for URLs.
func GenerateToken[T any](payload T, secret string) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	payloadEnc := base64.RawURLEncoding.EncodeToString(data)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	sig := h.Sum(nil)[:8] // Truncate to 8 bytes for compactness
	sigEnc := base64.RawURLEncoding.EncodeToString(sig)

	return payloadEnc + "." + sigEnc, nil
}
