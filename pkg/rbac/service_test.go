package rbac_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/rbac"
)

func TestAuthorizer_Can(t *testing.T) {
	ctx := context.Background()
	roles := getTestRoles()
	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	tests := []struct {
		name       string
		role       string
		permission string
		wantErr    error
	}{
		{
			name:       "direct permission allowed",
			role:       "editor",
			permission: "content.write",
			wantErr:    nil,
		},
		{
			name:       "inherited permission allowed",
			role:       "editor",
			permission: "content.read",
			wantErr:    nil,
		},
		{
			name:       "wildcard permission allowed",
			role:       "admin",
			permission: "admin.users",
			wantErr:    nil,
		},
		{
			name:       "global wildcard allowed",
			role:       "superadmin",
			permission: "anything.at.all",
			wantErr:    nil,
		},
		{
			name:       "permission denied",
			role:       "viewer",
			permission: "content.write",
			wantErr:    rbac.ErrInsufficientPermissions,
		},
		{
			name:       "invalid role",
			role:       "nonexistent",
			permission: "content.read",
			wantErr:    rbac.ErrInvalidRole,
		},
		{
			name:       "empty permission denied",
			role:       "viewer",
			permission: "",
			wantErr:    rbac.ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.Can(tt.role, tt.permission)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthorizer_CanAny(t *testing.T) {
	ctx := context.Background()
	roles := getTestRoles()
	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	tests := []struct {
		name        string
		role        string
		permissions []string
		wantErr     error
	}{
		{
			name:        "has one of the permissions",
			role:        "editor",
			permissions: []string{"content.write", "admin.users"},
			wantErr:     nil,
		},
		{
			name:        "has all permissions",
			role:        "editor",
			permissions: []string{"content.read", "content.write"},
			wantErr:     nil,
		},
		{
			name:        "has none of the permissions",
			role:        "editor",
			permissions: []string{"users.read", "users.write"},
			wantErr:     rbac.ErrInsufficientPermissions,
		},
		{
			name:        "empty permissions always allowed",
			role:        "viewer",
			permissions: []string{},
			wantErr:     nil,
		},
		{
			name:        "invalid role",
			role:        "nonexistent",
			permissions: []string{"content.write"},
			wantErr:     rbac.ErrInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.CanAny(tt.role, tt.permissions...)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthorizer_CanAll(t *testing.T) {
	ctx := context.Background()
	roles := getTestRoles()
	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	tests := []struct {
		name        string
		role        string
		permissions []string
		wantErr     error
	}{
		{
			name:        "has all permissions",
			role:        "editor",
			permissions: []string{"content.read", "content.write"},
			wantErr:     nil,
		},
		{
			name:        "missing one permission",
			role:        "editor",
			permissions: []string{"content.write", "users.delete"},
			wantErr:     rbac.ErrInsufficientPermissions,
		},
		{
			name:        "admin with wildcard has all",
			role:        "admin",
			permissions: []string{"admin.users", "admin.billing", "admin.settings"},
			wantErr:     nil,
		},
		{
			name:        "empty permissions always allowed",
			role:        "viewer",
			permissions: []string{},
			wantErr:     nil,
		},
		{
			name:        "invalid role",
			role:        "nonexistent",
			permissions: []string{"content.write"},
			wantErr:     rbac.ErrInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.CanAll(tt.role, tt.permissions...)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthorizer_ContextMethods(t *testing.T) {
	ctx := context.Background()
	roles := getTestRoles()
	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	t.Run("CanFromContext with role", func(t *testing.T) {
		ctx := rbac.SetRoleToContext(context.Background(), "editor")
		err := auth.CanFromContext(ctx, "content.write")
		assert.NoError(t, err)
	})

	t.Run("CanFromContext without role", func(t *testing.T) {
		err := auth.CanFromContext(context.Background(), "content.write")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, rbac.ErrRoleNotInContext))
		assert.True(t, errors.Is(err, rbac.ErrInsufficientPermissions))
	})

	t.Run("CanAnyFromContext with role", func(t *testing.T) {
		ctx := rbac.SetRoleToContext(context.Background(), "editor")
		err := auth.CanAnyFromContext(ctx, "content.write", "users.delete")
		assert.NoError(t, err)
	})

	t.Run("CanAnyFromContext without role", func(t *testing.T) {
		err := auth.CanAnyFromContext(context.Background(), "content.write")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, rbac.ErrRoleNotInContext))
	})

	t.Run("CanAllFromContext with role", func(t *testing.T) {
		ctx := rbac.SetRoleToContext(context.Background(), "editor")
		err := auth.CanAllFromContext(ctx, "content.read", "content.write")
		assert.NoError(t, err)
	})

	t.Run("CanAllFromContext without role", func(t *testing.T) {
		err := auth.CanAllFromContext(context.Background(), "content.write")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, rbac.ErrRoleNotInContext))
	})
}

func TestAuthorizer_VerifyRole(t *testing.T) {
	ctx := context.Background()
	roles := getTestRoles()
	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	t.Run("existing role", func(t *testing.T) {
		err := auth.VerifyRole("editor")
		assert.NoError(t, err)
	})

	t.Run("nonexistent role", func(t *testing.T) {
		err := auth.VerifyRole("nonexistent")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, rbac.ErrInvalidRole))
	})
}

func TestAuthorizer_GetRoles(t *testing.T) {
	ctx := context.Background()
	roles := getTestRoles()
	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	roleNames := auth.GetRoles()
	assert.NotEmpty(t, roleNames)
	assert.Contains(t, roleNames, "viewer")
	assert.Contains(t, roleNames, "editor")
	assert.Contains(t, roleNames, "admin")
	assert.Contains(t, roleNames, "superadmin")

	// Verify ordering - base roles should come first
	viewerIdx := indexOf(roleNames, "viewer")
	editorIdx := indexOf(roleNames, "editor")
	adminIdx := indexOf(roleNames, "admin")

	assert.True(t, viewerIdx < editorIdx, "viewer should come before editor")
	assert.True(t, editorIdx < adminIdx, "editor should come before admin")
}

func TestInMemRoleSource(t *testing.T) {
	t.Run("creates defensive copy", func(t *testing.T) {
		original := map[string]rbac.Role{
			"test": {
				Permissions: []string{"read", "write"},
				Inherits:    []string{"base"},
			},
		}

		source := rbac.NewInMemRoleSource(original)

		// Modify original
		original["test"].Permissions[0] = "modified"
		original["new"] = rbac.Role{Permissions: []string{"new"}}

		// Load should return unmodified data
		loaded, err := source.Load(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "read", loaded["test"].Permissions[0])
		assert.NotContains(t, loaded, "new")
	})
}

func TestCircularInheritance(t *testing.T) {
	ctx := context.Background()

	t.Run("direct circular reference", func(t *testing.T) {
		// Create roles with direct circular inheritance
		roles := map[string]rbac.Role{
			"role1": {
				Permissions: []string{"perm1"},
				Inherits:    []string{"role2"},
			},
			"role2": {
				Permissions: []string{"perm2"},
				Inherits:    []string{"role1"}, // Direct circular reference
			},
		}

		source := rbac.NewInMemRoleSource(roles)
		_, err := rbac.NewAuthorizer(ctx, source)
		require.Error(t, err)
		assert.True(t, errors.Is(err, rbac.ErrCircularInheritance))
		assert.Contains(t, err.Error(), "circular inheritance detected")
	})

	t.Run("indirect circular reference", func(t *testing.T) {
		// Create roles with indirect circular inheritance
		roles := map[string]rbac.Role{
			"role1": {
				Permissions: []string{"perm1"},
				Inherits:    []string{"role2"},
			},
			"role2": {
				Permissions: []string{"perm2"},
				Inherits:    []string{"role3"},
			},
			"role3": {
				Permissions: []string{"perm3"},
				Inherits:    []string{"role1"}, // Indirect circular reference
			},
		}

		source := rbac.NewInMemRoleSource(roles)
		_, err := rbac.NewAuthorizer(ctx, source)
		require.Error(t, err)
		assert.True(t, errors.Is(err, rbac.ErrCircularInheritance))
		assert.Contains(t, err.Error(), "circular inheritance detected")
	})

	t.Run("self-reference", func(t *testing.T) {
		// Create role that inherits from itself
		roles := map[string]rbac.Role{
			"role1": {
				Permissions: []string{"perm1"},
				Inherits:    []string{"role1"}, // Self-reference
			},
		}

		source := rbac.NewInMemRoleSource(roles)
		_, err := rbac.NewAuthorizer(ctx, source)
		require.Error(t, err)
		assert.True(t, errors.Is(err, rbac.ErrCircularInheritance))
		assert.Contains(t, err.Error(), "circular inheritance detected")
	})

	t.Run("valid inheritance without cycles", func(t *testing.T) {
		// Create valid role hierarchy
		roles := map[string]rbac.Role{
			"base": {
				Permissions: []string{"read"},
			},
			"editor": {
				Permissions: []string{"write"},
				Inherits:    []string{"base"},
			},
			"admin": {
				Permissions: []string{"delete"},
				Inherits:    []string{"editor"},
			},
		}

		source := rbac.NewInMemRoleSource(roles)
		auth, err := rbac.NewAuthorizer(ctx, source)
		require.NoError(t, err)

		// Admin should have all permissions
		assert.NoError(t, auth.Can("admin", "read"))
		assert.NoError(t, auth.Can("admin", "write"))
		assert.NoError(t, auth.Can("admin", "delete"))
	})
}

func TestMaxInheritanceDepth(t *testing.T) {
	ctx := context.Background()

	t.Run("exceeds maximum depth", func(t *testing.T) {
		// Create a deep inheritance chain
		roles := make(map[string]rbac.Role)

		// Create a chain longer than MaxInheritanceDepth
		for i := 0; i <= rbac.MaxInheritanceDepth+2; i++ {
			roleName := fmt.Sprintf("role%d", i)
			role := rbac.Role{
				Permissions: []string{fmt.Sprintf("perm%d", i)},
			}
			if i > 0 {
				role.Inherits = []string{fmt.Sprintf("role%d", i-1)}
			}
			roles[roleName] = role
		}

		source := rbac.NewInMemRoleSource(roles)
		_, err := rbac.NewAuthorizer(ctx, source)
		require.Error(t, err)
		assert.True(t, errors.Is(err, rbac.ErrCircularInheritance))
		assert.Contains(t, err.Error(), "inheritance depth exceeds maximum allowed depth")
	})

	t.Run("at maximum depth", func(t *testing.T) {
		// Create a chain exactly at MaxInheritanceDepth
		roles := make(map[string]rbac.Role)

		for i := 0; i <= rbac.MaxInheritanceDepth; i++ {
			roleName := fmt.Sprintf("role%d", i)
			role := rbac.Role{
				Permissions: []string{fmt.Sprintf("perm%d", i)},
			}
			if i > 0 {
				role.Inherits = []string{fmt.Sprintf("role%d", i-1)}
			}
			roles[roleName] = role
		}

		source := rbac.NewInMemRoleSource(roles)
		auth, err := rbac.NewAuthorizer(ctx, source)
		require.NoError(t, err)

		// Should have all permissions from the chain
		topRole := fmt.Sprintf("role%d", rbac.MaxInheritanceDepth)
		for i := 0; i <= rbac.MaxInheritanceDepth; i++ {
			assert.NoError(t, auth.Can(topRole, fmt.Sprintf("perm%d", i)))
		}
	})
}

func TestWildcardPermissions(t *testing.T) {
	ctx := context.Background()
	roles := map[string]rbac.Role{
		"admin": {
			Permissions: []string{"admin.*", "reports.read"},
		},
		"superadmin": {
			Permissions: []string{"*"},
		},
	}

	source := rbac.NewInMemRoleSource(roles)
	auth, err := rbac.NewAuthorizer(ctx, source)
	require.NoError(t, err)

	tests := []struct {
		name       string
		role       string
		permission string
		allowed    bool
	}{
		{
			name:       "wildcard matches namespace",
			role:       "admin",
			permission: "admin.users",
			allowed:    true,
		},
		{
			name:       "wildcard matches nested namespace",
			role:       "admin",
			permission: "admin.users.delete",
			allowed:    true,
		},
		{
			name:       "wildcard doesn't match different namespace",
			role:       "admin",
			permission: "users.admin",
			allowed:    false,
		},
		{
			name:       "global wildcard matches everything",
			role:       "superadmin",
			permission: "anything.goes.here",
			allowed:    true,
		},
		{
			name:       "specific permission still works with wildcards",
			role:       "admin",
			permission: "reports.read",
			allowed:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.Can(tt.role, tt.permission)
			if tt.allowed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Helper functions

func getTestRoles() map[string]rbac.Role {
	return map[string]rbac.Role{
		"viewer": {
			Permissions: []string{"content.read"},
		},
		"editor": {
			Permissions: []string{"content.write"},
			Inherits:    []string{"viewer"},
		},
		"admin": {
			Permissions: []string{"admin.*", "users.delete"},
			Inherits:    []string{"editor"},
		},
		"superadmin": {
			Permissions: []string{"*"},
		},
	}
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
