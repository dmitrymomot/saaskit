package rbac

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/dmitrymomot/saaskit/pkg/scopes"
)

// Authorizer provides role-based access control by mapping roles to permissions.
// It supports role inheritance and wildcard permissions for flexible authorization.
type Authorizer interface {
	// Can checks if a role has the specified permission (direct or inherited).
	Can(roleName, permission string) error

	// CanAny checks if a role has any of the provided permissions.
	CanAny(roleName string, permissions ...string) error

	// CanAll checks if a role has all of the provided permissions.
	CanAll(roleName string, permissions ...string) error

	// CanFromContext checks if the role in context has the specified permission.
	CanFromContext(ctx context.Context, permission string) error

	// CanAnyFromContext checks if the role in context has any of the specified permissions.
	CanAnyFromContext(ctx context.Context, permissions ...string) error

	// CanAllFromContext checks if the role in context has all specified permissions.
	CanAllFromContext(ctx context.Context, permissions ...string) error

	// VerifyRole returns an error if the given role does not exist.
	VerifyRole(role string) error

	// GetRoles returns all role names sorted by inheritance (base roles first).
	GetRoles() []string
}

// RoleSource defines the interface for providing role data.
type RoleSource interface {
	// Load returns a map of all roles.
	Load(ctx context.Context) (map[string]Role, error)
}

// authorizer implements the Authorizer interface.
type authorizer struct {
	// rolePermissions contains all permissions (direct and inherited) for each role.
	// Using map[string]struct{} for O(1) permission lookups and zero memory overhead.
	// This map is treated as immutable after initialization for thread safety.
	rolePermissions map[string]map[string]struct{}
	// sortedRoles lists all roles sorted by inheritance (base roles first).
	sortedRoles []string
}

// NewAuthorizer creates a new Authorizer that loads roles from the provided source.
// It precomputes all permissions (including inherited ones) for efficient runtime checks.
func NewAuthorizer(ctx context.Context, source RoleSource) (Authorizer, error) {
	roles, err := source.Load(ctx)
	if err != nil {
		return nil, err
	}

	if roles == nil {
		roles = make(map[string]Role)
	}

	// Validate role inheritance for circular dependencies
	if err := validateRoleInheritance(roles); err != nil {
		return nil, err
	}

	// Precompute all permissions for each role as sets for O(1) lookups
	rolePermissions := make(map[string]map[string]struct{})
	for roleName := range roles {
		allPermissions := getAllPermissions(roleName, roles, make(map[string]bool), 0)
		normalizedPermissions := scopes.NormalizeScopes(allPermissions)
		
		// Convert slice to set (map[string]struct{})
		permissionSet := make(map[string]struct{}, len(normalizedPermissions))
		for _, perm := range normalizedPermissions {
			permissionSet[perm] = struct{}{}
		}
		rolePermissions[roleName] = permissionSet
	}

	// Sort roles by inheritance
	sortedRoles := sortRolesByInheritance(roles)

	return &authorizer{
		rolePermissions: rolePermissions,
		sortedRoles:     sortedRoles,
	}, nil
}

// Can checks if a role has the specified permission (direct or inherited).
func (a *authorizer) Can(roleName, permission string) error {
	permissionSet, exists := a.rolePermissions[roleName]
	if !exists {
		return ErrInvalidRole
	}

	// Check for exact permission match (O(1) lookup)
	if _, hasPermission := permissionSet[permission]; hasPermission {
		return nil
	}

	// Check for wildcard permissions using the existing scopes logic
	// Convert back to slice for compatibility with scopes package
	permissions := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		permissions = append(permissions, perm)
	}
	
	if !scopes.HasScope(permissions, permission) {
		return ErrInsufficientPermissions
	}

	return nil
}

// CanAny checks if a role has any of the provided permissions.
func (a *authorizer) CanAny(roleName string, permissions ...string) error {
	if len(permissions) == 0 {
		return nil
	}

	permissionSet, exists := a.rolePermissions[roleName]
	if !exists {
		return ErrInvalidRole
	}

	// First try exact matches (O(1) for each)
	for _, permission := range permissions {
		if _, hasPermission := permissionSet[permission]; hasPermission {
			return nil // Found at least one exact match
		}
	}

	// If no exact matches, fall back to wildcard checking
	rolePermissions := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		rolePermissions = append(rolePermissions, perm)
	}

	if !scopes.HasAnyScopes(rolePermissions, permissions) {
		return ErrInsufficientPermissions
	}

	return nil
}

// CanAll checks if a role has all of the provided permissions.
func (a *authorizer) CanAll(roleName string, permissions ...string) error {
	if len(permissions) == 0 {
		return nil
	}

	permissionSet, exists := a.rolePermissions[roleName]
	if !exists {
		return ErrInvalidRole
	}

	// Check each permission individually for efficiency
	exactMatches := 0
	var remainingPermissions []string
	
	for _, permission := range permissions {
		if _, hasPermission := permissionSet[permission]; hasPermission {
			exactMatches++
		} else {
			remainingPermissions = append(remainingPermissions, permission)
		}
	}

	// If all permissions have exact matches, we're done
	if exactMatches == len(permissions) {
		return nil
	}

	// For remaining permissions, check wildcards
	if len(remainingPermissions) > 0 {
		rolePermissions := make([]string, 0, len(permissionSet))
		for perm := range permissionSet {
			rolePermissions = append(rolePermissions, perm)
		}

		if !scopes.HasAllScopes(rolePermissions, remainingPermissions) {
			return ErrInsufficientPermissions
		}
	}

	return nil
}

// CanFromContext checks if the role in context has the specified permission.
func (a *authorizer) CanFromContext(ctx context.Context, permission string) error {
	role, ok := GetRoleFromContext(ctx)
	if !ok {
		return errors.Join(ErrRoleNotInContext, ErrInsufficientPermissions)
	}

	return a.Can(role, permission)
}

// CanAnyFromContext checks if the role in context has any of the specified permissions.
func (a *authorizer) CanAnyFromContext(ctx context.Context, permissions ...string) error {
	role, ok := GetRoleFromContext(ctx)
	if !ok {
		return errors.Join(ErrRoleNotInContext, ErrInsufficientPermissions)
	}

	return a.CanAny(role, permissions...)
}

// CanAllFromContext checks if the role in context has all specified permissions.
func (a *authorizer) CanAllFromContext(ctx context.Context, permissions ...string) error {
	role, ok := GetRoleFromContext(ctx)
	if !ok {
		return errors.Join(ErrRoleNotInContext, ErrInsufficientPermissions)
	}

	return a.CanAll(role, permissions...)
}

// VerifyRole returns an error if the given role does not exist.
func (a *authorizer) VerifyRole(role string) error {
	if _, exists := a.rolePermissions[role]; !exists {
		return ErrInvalidRole
	}
	return nil
}

// GetRoles returns all role names sorted by inheritance (base roles first).
func (a *authorizer) GetRoles() []string {
	return a.sortedRoles
}

// getAllPermissions recursively collects all permissions for a role, including inherited ones.
func getAllPermissions(roleName string, roles map[string]Role, visited map[string]bool, depth int) []string {
	// Check maximum depth
	if depth > MaxInheritanceDepth {
		return nil
	}

	// Prevent infinite recursion in case of circular inheritance
	if visited[roleName] {
		return nil
	}
	visited[roleName] = true

	role, exists := roles[roleName]
	if !exists {
		return nil
	}

	// Start with direct permissions
	result := make([]string, len(role.Permissions))
	copy(result, role.Permissions)

	// Add inherited permissions
	for _, inheritedRole := range role.Inherits {
		inheritedPerms := getAllPermissions(inheritedRole, roles, visited, depth+1)
		result = append(result, inheritedPerms...)
	}

	return result
}

// sortRolesByInheritance returns role names sorted by inheritance depth.
func sortRolesByInheritance(roles map[string]Role) []string {
	// Calculate inheritance depth for each role
	depths := make(map[string]int)
	visited := make(map[string]bool)

	for roleName := range roles {
		if !visited[roleName] {
			calculateRoleDepth(roleName, roles, depths, visited, make(map[string]bool))
		}
	}

	// Create sorted slice
	result := make([]string, 0, len(roles))
	for roleName := range roles {
		result = append(result, roleName)
	}

	// Sort by depth (base roles first)
	slices.SortFunc(result, func(a, b string) int {
		return depths[a] - depths[b]
	})

	return result
}

// calculateRoleDepth computes the inheritance depth of a role using DFS.
func calculateRoleDepth(roleName string, roles map[string]Role, depths map[string]int, visited, inProcess map[string]bool) int {
	if visited[roleName] {
		return depths[roleName]
	}

	if inProcess[roleName] {
		return 0 // Circular dependency detected
	}

	inProcess[roleName] = true

	role, exists := roles[roleName]
	if !exists {
		depths[roleName] = 0
		visited[roleName] = true
		inProcess[roleName] = false
		return 0
	}

	if len(role.Inherits) == 0 {
		depths[roleName] = 0
		visited[roleName] = true
		inProcess[roleName] = false
		return 0
	}

	maxDepth := 0
	for _, inheritedRole := range role.Inherits {
		depth := calculateRoleDepth(inheritedRole, roles, depths, visited, inProcess) + 1
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	depths[roleName] = maxDepth
	visited[roleName] = true
	inProcess[roleName] = false
	return maxDepth
}

// validateRoleInheritance checks for circular dependencies and excessive depth in role inheritance.
func validateRoleInheritance(roles map[string]Role) error {
	// Check each role for circular dependencies
	for roleName := range roles {
		if err := checkCircularInheritance(roleName, roles, make(map[string]bool), []string{roleName}); err != nil {
			return err
		}
	}

	// Check maximum inheritance depth
	depths := make(map[string]int)
	visited := make(map[string]bool)
	for roleName := range roles {
		if !visited[roleName] {
			depth := calculateRoleDepth(roleName, roles, depths, visited, make(map[string]bool))
			if depth > MaxInheritanceDepth {
				return errors.Join(ErrCircularInheritance,
					fmt.Errorf("inheritance depth exceeds maximum allowed depth of %d", MaxInheritanceDepth))
			}
		}
	}

	return nil
}

// checkCircularInheritance performs DFS to detect circular dependencies in role inheritance.
func checkCircularInheritance(roleName string, roles map[string]Role, visited map[string]bool, path []string) error {
	visited[roleName] = true
	defer func() { visited[roleName] = false }()

	role, exists := roles[roleName]
	if !exists {
		return nil
	}

	for _, inheritedRole := range role.Inherits {
		// Check if we've seen this role in the current path
		if slices.Contains(path, inheritedRole) {
			return errors.Join(ErrCircularInheritance,
				fmt.Errorf("circular inheritance detected: %s -> %s", roleName, inheritedRole))
		}

		// Continue DFS
		newPath := append(path, inheritedRole)
		if err := checkCircularInheritance(inheritedRole, roles, visited, newPath); err != nil {
			return err
		}
	}

	return nil
}
