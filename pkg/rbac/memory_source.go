package rbac

import (
	"context"
	"sync"
)

// inMemRoleSource is a simple RoleSource that loads roles from memory.
// It's thread-safe and makes defensive copies to prevent external modifications.
type inMemRoleSource struct {
	mu    sync.RWMutex
	roles map[string]Role
}

// NewInMemRoleSource creates a new in-memory role source from a map of roles.
// It creates a deep copy of the input to prevent external modifications.
func NewInMemRoleSource(roles map[string]Role) RoleSource {
	// Create a deep copy of the roles map
	rolesCopy := make(map[string]Role, len(roles))
	for k, v := range roles {
		// Deep copy permissions
		permsCopy := make([]string, len(v.Permissions))
		copy(permsCopy, v.Permissions)

		// Deep copy inherits
		inheritsCopy := make([]string, len(v.Inherits))
		copy(inheritsCopy, v.Inherits)

		rolesCopy[k] = Role{
			Permissions: permsCopy,
			Inherits:    inheritsCopy,
		}
	}

	return &inMemRoleSource{
		roles: rolesCopy,
	}
}

// Load returns the map of roles.
// The returned map is safe to read but should not be modified.
func (s *inMemRoleSource) Load(ctx context.Context) (map[string]Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return the internal map directly for performance.
	// The authorizer treats this as read-only.
	return s.roles, nil
}
