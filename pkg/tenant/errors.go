package tenant

import "errors"

var (
	// ErrTenantNotFound is returned when a tenant cannot be found.
	ErrTenantNotFound = errors.New("tenant not found")

	// ErrInvalidIdentifier is returned when the identifier format is invalid.
	ErrInvalidIdentifier = errors.New("invalid tenant identifier")

	// ErrNoTenantInContext is returned when no tenant is found in context.
	ErrNoTenantInContext = errors.New("no tenant in context")

	// ErrInactiveTenant is returned when trying to use an inactive tenant.
	ErrInactiveTenant = errors.New("tenant is inactive")
)
