package tenant

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Validation constants
const (
	// MaxTenantIDLength defines the maximum allowed length for tenant identifiers.
	// This prevents DoS attacks via very long tenant IDs and ensures DNS compatibility.
	MaxTenantIDLength = 63

	// MinTenantIDLength defines the minimum allowed length for tenant identifiers.
	MinTenantIDLength = 1
)

// Common validation patterns
var (
	// tenantIDPattern allows alphanumeric characters, hyphens, and underscores.
	// Must start with alphanumeric character.
	tenantIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_]*$`)

	// uuidPattern matches standard UUID format.
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	// dangerousCharsPattern matches potentially dangerous characters.
	dangerousCharsPattern = regexp.MustCompile(`[\x00-\x1f\x7f-\x9f\/\\<>:"|\?\*\.]`)
)

// Resolver extracts tenant identifier from HTTP request.
// Returns empty string if no tenant found, error if extraction failed.
type Resolver func(r *http.Request) (string, error)

// sanitizeTenantID cleans and preprocesses tenant identifier input.
// It trims whitespace and removes potentially dangerous characters.
func sanitizeTenantID(id string) string {
	id = strings.TrimSpace(id)
	id = dangerousCharsPattern.ReplaceAllString(id, "")
	return id
}

// isValidTenantID validates tenant identifier format after sanitization.
func isValidTenantID(id string) bool {
	id = sanitizeTenantID(id)

	if id == "" {
		return false
	}

	if len(id) < MinTenantIDLength || len(id) > MaxTenantIDLength {
		return false
	}

	if uuidPattern.MatchString(id) {
		return true
	}

	return tenantIDPattern.MatchString(id)
}

// NewSubdomainResolver creates a resolver that extracts tenant from subdomain.
// Suffix parameter allows stripping domain suffix (e.g., ".saas.com").
// Returns empty string for base domain (no subdomain).
func NewSubdomainResolver(suffix string) Resolver {
	return func(req *http.Request) (string, error) {
		host := req.Host

		// Remove port if present
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}

		originalParts := strings.Split(host, ".")

		if suffix != "" && strings.HasSuffix(host, suffix) && len(host) > len(suffix) {
			host = host[:len(host)-len(suffix)]
		}

		parts := strings.Split(host, ".")
		if len(parts) == 0 || parts[0] == "" {
			return "", nil
		}

		subdomain := parts[0]
		if subdomain == "www" {
			if len(parts) > 1 {
				subdomain = parts[1]
			} else {
				return "", nil
			}
		}

		// Need at least 3 parts for subdomain.domain.tld
		if len(originalParts) < 3 {
			return "", nil
		}

		if subdomain != "" {
			sanitized := sanitizeTenantID(subdomain)
			if !isValidTenantID(subdomain) {
				return "", fmt.Errorf("%w: subdomain '%s'", ErrInvalidIdentifier, sanitized)
			}
			return sanitized, nil
		}

		return subdomain, nil
	}
}

// NewHeaderResolver creates a resolver that extracts tenant from HTTP header.
// Default header name is "X-Tenant-ID" if empty string provided.
func NewHeaderResolver(headerName string) Resolver {
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}

	return func(req *http.Request) (string, error) {
		value := req.Header.Get(headerName)

		// Return empty if no header value
		if value == "" {
			return "", nil
		}

		// Validate the header value
		sanitized := sanitizeTenantID(value)
		if !isValidTenantID(value) {
			return "", fmt.Errorf("%w: header value '%s'", ErrInvalidIdentifier, sanitized)
		}

		return sanitized, nil
	}
}

// NewPathResolver creates a resolver that extracts tenant from URL path segment.
// Position is 1-based (e.g., 2 for /tenants/{id}/...).
// Returns error if position < 1.
func NewPathResolver(position int) Resolver {
	return func(req *http.Request) (string, error) {
		if position < 1 {
			return "", fmt.Errorf("invalid path position: %d", position)
		}

		path := strings.TrimPrefix(req.URL.Path, "/")
		path = strings.TrimSuffix(path, "/")

		if path == "" {
			return "", nil
		}

		parts := strings.Split(path, "/")
		if position > len(parts) {
			return "", nil
		}

		value := parts[position-1]

		// Return empty if no value at position
		if value == "" {
			return "", nil
		}

		// Validate the path segment
		sanitized := sanitizeTenantID(value)
		if !isValidTenantID(value) {
			return "", fmt.Errorf("%w: path segment '%s'", ErrInvalidIdentifier, sanitized)
		}

		return sanitized, nil
	}
}

// NewCompositeResolver creates a resolver that tries multiple resolvers in order.
// Returns the first non-empty tenant ID found.
// If all resolvers return empty, returns empty string.
// Aggregates errors from all resolvers for debugging.
func NewCompositeResolver(resolvers ...Resolver) Resolver {
	return func(r *http.Request) (string, error) {
		var errs []error

		for _, resolver := range resolvers {
			id, err := resolver(r)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if id != "" {
				return id, nil
			}
		}

		if len(errs) > 0 {
			return "", fmt.Errorf("composite resolver errors: %w", errors.Join(errs...))
		}

		return "", nil
	}
}
