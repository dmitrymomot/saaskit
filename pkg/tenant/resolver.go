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
	// pathPattern allows alphanumeric characters and hyphens only.
	// Must start with alphanumeric character.
	pathPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

	// subdomainPattern allows alphanumeric characters and hyphens only.
	// Must start with alphanumeric character. No dots allowed.
	subdomainPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

	// uuidPattern matches standard UUID format.
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// Resolver extracts tenant identifier from HTTP request.
// Returns empty string if no tenant found, error if extraction failed.
type Resolver func(r *http.Request) (string, error)

// isValidUUID validates UUID format.
func isValidUUID(id string) bool {
	return uuidPattern.MatchString(id)
}

// isValidPath validates path segment format.
func isValidPath(id string) bool {
	if id == "" {
		return false
	}

	if len(id) < MinTenantIDLength || len(id) > MaxTenantIDLength {
		return false
	}

	return pathPattern.MatchString(id)
}

// isValidSubdomain validates subdomain format.
func isValidSubdomain(id string) bool {
	if id == "" {
		return false
	}

	if len(id) < MinTenantIDLength || len(id) > MaxTenantIDLength {
		return false
	}

	return subdomainPattern.MatchString(id)
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
			subdomain = strings.TrimSpace(subdomain)
			if !isValidSubdomain(subdomain) {
				return "", fmt.Errorf("%w: subdomain '%s'", ErrInvalidIdentifier, subdomain)
			}
			return subdomain, nil
		}

		return "", nil
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
		value = strings.TrimSpace(value)
		if !isValidPath(value) { // Use same validation as path for consistency
			return "", fmt.Errorf("%w: header value '%s'", ErrInvalidIdentifier, value)
		}

		return value, nil
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
		value = strings.TrimSpace(value)
		if !isValidPath(value) {
			return "", fmt.Errorf("%w: path segment '%s'", ErrInvalidIdentifier, value)
		}

		return value, nil
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
