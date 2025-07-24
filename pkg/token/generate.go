package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
)

// GenerateToken creates a token by JSON encoding the payload and appending an 8-byte truncated HMAC-SHA256 signature.
func GenerateToken[T any](payload T, secret string) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	payloadEnc := base64.RawURLEncoding.EncodeToString(data)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	sig := h.Sum(nil)[:8]
	sigEnc := base64.RawURLEncoding.EncodeToString(sig)

	return payloadEnc + "." + sigEnc, nil
}
