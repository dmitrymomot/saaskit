package ratelimit

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// maxKeyLength is the maximum allowed length for a rate limit key
// to prevent excessively long storage keys in backends like Redis.
const maxKeyLength = 64

// KeyFunc extracts a unique identifier from an HTTP request for rate limiting.
type KeyFunc func(*http.Request) string

// Composite combines multiple key extraction functions into a single key.
// Long keys (>64 chars) are hashed to 32 hex chars using SHA256 to prevent
// storage issues while avoiding collisions.
func Composite(keyFuncs ...KeyFunc) KeyFunc {
	return func(r *http.Request) string {
		parts := make([]string, 0, len(keyFuncs))
		for _, fn := range keyFuncs {
			if key := fn(r); key != "" {
				parts = append(parts, key)
			}
		}

		if len(parts) == 0 {
			return ""
		}

		if len(parts) == 1 && len(parts[0]) <= maxKeyLength {
			return parts[0]
		}

		combined := strings.Join(parts, ":")

		if len(combined) > maxKeyLength {
			hash := sha256.Sum256([]byte(combined))
			// 128-bit hash provides sufficient collision resistance
			return hex.EncodeToString(hash[:16])
		}

		return combined
	}
}
