package rbac

import "github.com/dmitrymomot/saaskit/pkg/scopes"

// MaxInheritanceDepth is the maximum allowed depth of role inheritance
// to prevent excessive nesting and potential performance issues.
const MaxInheritanceDepth = 10

// Permission represents a string-based permission scope.
// Permissions can be hierarchical using dots (e.g., "users.read")
// and support wildcards (e.g., "admin.*").
type Permission string

// Role represents a set of permissions with optional inheritance.
// Roles can inherit permissions from other roles, creating a hierarchy.
type Role struct {
	// Permissions directly granted to this role.
	Permissions []string

	// Inherits lists role names this role inherits from.
	// All permissions from inherited roles are included.
	Inherits []string
}

// Can checks if the role has the specified permission directly.
// This does not check inherited permissions.
func (r *Role) Can(permission string) bool {
	return scopes.HasScope(r.Permissions, permission)
}
