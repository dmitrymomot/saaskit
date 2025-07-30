package rbac

import "errors"

// Domain errors for RBAC operations.
var (
	// ErrInvalidRole is returned when a role does not exist.
	ErrInvalidRole = errors.New("invalid role")

	// ErrInsufficientPermissions is returned when required permissions are not granted.
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	// ErrRoleNotInContext is returned when no role is found in the context.
	ErrRoleNotInContext = errors.New("role not found in context")

	// ErrCircularInheritance is returned when roles have circular inheritance.
	ErrCircularInheritance = errors.New("circular role inheritance detected")
)
