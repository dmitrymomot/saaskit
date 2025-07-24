package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"

	"github.com/dmitrymomot/saaskit/pkg/clientip"
)

// Generate creates a device fingerprint from the HTTP request.
// It combines User-Agent, Accept headers, client IP, and header order
// to create a 32-character hex string identifying the device/browser.
func Generate(r *http.Request) string {
	components := []string{
		r.UserAgent(),
		r.Header.Get("Accept-Language"),
		r.Header.Get("Accept-Encoding"),
		r.Header.Get("Accept"),
		clientip.GetIP(r),
		getHeaderOrder(r),
	}

	// Filter out empty components
	var filtered []string
	for _, comp := range components {
		if comp != "" {
			filtered = append(filtered, comp)
		}
	}

	// Create SHA256 hash of all components
	combined := strings.Join(filtered, "|")
	hash := sha256.Sum256([]byte(combined))

	// Return first 16 bytes as 32-character hex string
	return hex.EncodeToString(hash[:16])
}

// Validate compares the current request fingerprint with a stored fingerprint.
// Returns true if they match, false otherwise.
func Validate(r *http.Request, sessionFingerprint string) bool {
	currentFingerprint := Generate(r)
	return currentFingerprint == sessionFingerprint
}

// getHeaderOrder creates a fingerprint based on the order of HTTP headers.
// Different browsers and clients send headers in different orders,
// making this a useful distinguishing characteristic.
func getHeaderOrder(r *http.Request) string {
	var headerNames []string
	for name := range r.Header {
		// Skip headers that might vary in presence/absence
		// Focus on stable, commonly present headers
		switch strings.ToLower(name) {
		case "user-agent", "accept", "accept-language", "accept-encoding",
			"connection", "upgrade-insecure-requests", "sec-fetch-dest",
			"sec-fetch-mode", "sec-fetch-site", "cache-control":
			headerNames = append(headerNames, strings.ToLower(name))
		}
	}

	// Sort to ensure consistent ordering for identical header sets
	sort.Strings(headerNames)
	return strings.Join(headerNames, ",")
}
