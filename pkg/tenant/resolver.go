package tenant

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Resolver extracts tenant identifier from HTTP requests.
type Resolver interface {
	// Resolve extracts the tenant identifier from the request.
	// Returns empty string if no tenant identifier is found.
	// Returns error if the extraction fails.
	Resolve(r *http.Request) (string, error)
}

// SubdomainResolver extracts tenant identifier from request subdomain.
type SubdomainResolver struct {
	// Suffix to strip from the host (e.g., ".saas.com")
	// If empty, only the first subdomain part is used.
	Suffix string
}

// NewSubdomainResolver creates a new subdomain resolver.
func NewSubdomainResolver(suffix string) *SubdomainResolver {
	return &SubdomainResolver{Suffix: suffix}
}

// Resolve extracts tenant from subdomain (e.g., "acme" from "acme.app.com").
func (r *SubdomainResolver) Resolve(req *http.Request) (string, error) {
	host := req.Host

	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Count dots to determine if we have a subdomain
	originalParts := strings.Split(host, ".")

	// Strip suffix if configured
	if r.Suffix != "" && strings.HasSuffix(host, r.Suffix) {
		// Make sure we're not stripping the entire domain
		if len(host) > len(r.Suffix) {
			host = host[:len(host)-len(r.Suffix)]
		}
	}

	// Split by dots after suffix removal
	parts := strings.Split(host, ".")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil
	}

	// Skip www prefix
	subdomain := parts[0]
	if subdomain == "www" {
		if len(parts) > 1 {
			subdomain = parts[1]
		} else {
			return "", nil
		}
	}

	// Don't treat base domain as tenant
	// For a valid subdomain, we need:
	// - Without suffix: at least 3 parts (subdomain.domain.tld)
	// - With suffix: depends on what's left after stripping

	if r.Suffix != "" {
		// With suffix, check if we have a real subdomain
		if len(originalParts) < 3 {
			// Not enough parts for subdomain.domain.tld
			return "", nil
		}
	} else {
		// Without suffix, need at least 3 parts for subdomain.domain.tld
		if len(originalParts) < 3 {
			return "", nil
		}
	}

	return subdomain, nil
}

// HeaderResolver extracts tenant identifier from HTTP header.
type HeaderResolver struct {
	// HeaderName is the name of the header to read (e.g., "X-Tenant-ID")
	HeaderName string
}

// NewHeaderResolver creates a new header resolver.
func NewHeaderResolver(headerName string) *HeaderResolver {
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}
	return &HeaderResolver{HeaderName: headerName}
}

// Resolve extracts tenant from the configured header.
func (r *HeaderResolver) Resolve(req *http.Request) (string, error) {
	value := req.Header.Get(r.HeaderName)
	return value, nil
}

// PathResolver extracts tenant identifier from URL path segment.
type PathResolver struct {
	// Position is the 1-based position in the path (e.g., 2 for /tenants/{id}/...)
	Position int
}

// NewPathResolver creates a new path resolver.
func NewPathResolver(position int) *PathResolver {
	return &PathResolver{Position: position}
}

// Resolve extracts tenant from the specified path position.
func (r *PathResolver) Resolve(req *http.Request) (string, error) {
	if r.Position < 1 {
		return "", errors.New("invalid path position")
	}

	path := strings.TrimPrefix(req.URL.Path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return "", nil
	}

	parts := strings.Split(path, "/")
	if r.Position > len(parts) {
		return "", nil
	}

	return parts[r.Position-1], nil
}

// CompositeResolver tries multiple resolvers in order until one succeeds.
type CompositeResolver struct {
	Resolvers []Resolver
}

// NewCompositeResolver creates a new composite resolver.
func NewCompositeResolver(resolvers ...Resolver) *CompositeResolver {
	return &CompositeResolver{Resolvers: resolvers}
}

// Resolve tries each resolver in order, returning the first non-empty result.
func (c *CompositeResolver) Resolve(r *http.Request) (string, error) {
	var errs []error

	for _, resolver := range c.Resolvers {
		id, err := resolver.Resolve(r)
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

// SessionResolver extracts tenant identifier from session data.
// This is useful for applications where users can switch between tenants.
type SessionResolver struct {
	// GetSession retrieves the session from the request
	GetSession func(r *http.Request) (SessionData, error)
}

// SessionData represents the minimal session interface needed by the resolver.
type SessionData interface {
	GetString(key string) string
}

// NewSessionResolver creates a new session resolver.
func NewSessionResolver(getSession func(r *http.Request) (SessionData, error)) *SessionResolver {
	return &SessionResolver{GetSession: getSession}
}

// Resolve extracts tenant from session data.
func (r *SessionResolver) Resolve(req *http.Request) (string, error) {
	if r.GetSession == nil {
		return "", errors.New("session resolver: GetSession function not configured")
	}

	session, err := r.GetSession(req)
	if err != nil {
		return "", fmt.Errorf("session resolver: %w", err)
	}

	if session == nil {
		return "", nil
	}

	return session.GetString("tenant_id"), nil
}

// ResolverFunc is an adapter to allow the use of ordinary functions as Resolvers.
type ResolverFunc func(r *http.Request) (string, error)

// Resolve calls the function.
func (f ResolverFunc) Resolve(r *http.Request) (string, error) {
	return f(r)
}
