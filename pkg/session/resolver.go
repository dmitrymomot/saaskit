package session

import (
	"errors"
	"fmt"
	"net/http"
)

// TenantResolver extracts tenant identifier from session data.
// Useful for multi-tenant applications with tenant switching.
type TenantResolver struct {
	// GetSession retrieves the session from the request
	GetSession func(r *http.Request) (*Session, error)
}

// NewTenantResolver creates a new session-based tenant resolver.
func NewTenantResolver(getSession func(r *http.Request) (*Session, error)) *TenantResolver {
	return &TenantResolver{GetSession: getSession}
}

// Resolve extracts tenant from session data.
func (r *TenantResolver) Resolve(req *http.Request) (string, error) {
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

	value, ok := session.GetString("tenant_id")
	if !ok || value == "" {
		return "", nil
	}

	// Note: tenant package should validate ID format; this only extracts from session
	return value, nil
}
