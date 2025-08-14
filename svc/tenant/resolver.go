package tenant

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	// MaxTenantIDLength prevents DoS attacks via very long tenant IDs and ensures DNS compatibility
	MaxTenantIDLength = 63
	MinTenantIDLength = 1
)

var (
	// pathPattern ensures safe URL path segments: alphanumeric start, allows hyphens
	pathPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)
	// subdomainPattern ensures DNS-safe subdomains: alphanumeric start, allows hyphens, no dots
	subdomainPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)
)

// Resolver extracts tenant identifier from HTTP request.
// Returns empty string if no tenant found, error if extraction failed.
type Resolver func(r *http.Request) (string, error)

func isValidPath(id string) bool {
	if id == "" || len(id) < MinTenantIDLength || len(id) > MaxTenantIDLength {
		return false
	}
	return pathPattern.MatchString(id)
}

func isValidSubdomain(id string) bool {
	if id == "" || len(id) < MinTenantIDLength || len(id) > MaxTenantIDLength {
		return false
	}
	return subdomainPattern.MatchString(id)
}

// NewSubdomainResolver extracts tenant from subdomain, optionally stripping suffix.
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
		// Skip www prefix, use next subdomain if available
		if subdomain == "www" {
			if len(parts) > 1 {
				subdomain = parts[1]
			} else {
				return "", nil
			}
		}

		// Require at least 3 parts for proper subdomain.domain.tld structure
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

// NewHeaderResolver extracts tenant from HTTP header.
// Defaults to "X-Tenant-ID" if headerName is empty.
func NewHeaderResolver(headerName string) Resolver {
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}

	return func(req *http.Request) (string, error) {
		value := req.Header.Get(headerName)
		if value == "" {
			return "", nil
		}

		value = strings.TrimSpace(value)
		if !isValidPath(value) { // Use same validation as path for consistency
			return "", fmt.Errorf("%w: header value '%s'", ErrInvalidIdentifier, value)
		}

		return value, nil
	}
}

// NewPathResolver extracts tenant from URL path segment at 1-based position.
// Position 2 extracts from /tenants/{id}/dashboard.
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
		if value == "" {
			return "", nil
		}

		value = strings.TrimSpace(value)
		if !isValidPath(value) {
			return "", fmt.Errorf("%w: path segment '%s'", ErrInvalidIdentifier, value)
		}

		return value, nil
	}
}

// NewCompositeResolver tries multiple resolvers in order, returning the first non-empty result.
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
