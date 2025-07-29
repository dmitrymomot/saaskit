package rbac

import "errors"

// Domain errors for RBAC operations.
var (
	// ErrInvalidRole is returned when a role does not exist.
	ErrInvalidRole = errors.New("rbac.invalid_role")

	// ErrInsufficientPermissions is returned when required permissions are not granted.
	ErrInsufficientPermissions = errors.New("rbac.insufficient_permissions")

	// ErrRoleNotInContext is returned when no role is found in the context.
	ErrRoleNotInContext = errors.New("rbac.role_not_in_context")

	// ErrCircularInheritance is returned when roles have circular inheritance.
	ErrCircularInheritance = errors.New("rbac.circular_inheritance")
)
