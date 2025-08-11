package ratelimit

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// KeyFunc extracts a rate limit key from an HTTP request.
type KeyFunc func(*http.Request) string

// Composite combines multiple key extraction functions into a single key.
// If the combined key is too long (>64 chars), it uses SHA256 hashing
// to create a fixed-length key that avoids collisions.
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

		// For single key, return as-is if short enough
		if len(parts) == 1 && len(parts[0]) <= 64 {
			return parts[0]
		}

		// Combine all parts
		combined := strings.Join(parts, ":")

		// Hash if too long to avoid storage issues
		if len(combined) > 64 {
			hash := sha256.Sum256([]byte(combined))
			// Use first 16 bytes (32 hex chars) - enough to avoid collisions
			return hex.EncodeToString(hash[:16])
		}

		return combined
	}
}
