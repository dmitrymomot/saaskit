// Package rbac provides role-based access control for SaaS applications.
// It enables flexible permission checking with role inheritance and wildcard permissions.
//
// The package is designed to be tenant-agnostic and integrates seamlessly with
// context-based authentication systems. It supports hierarchical permissions with
// wildcard matching for scalable authorization patterns.
//
// Key concepts:
//
//   - Role: A named set of permissions that can inherit from other roles
//   - Permission: A dot-separated scope string (e.g., "users.read", "projects.write")
//   - Inheritance: Roles can inherit permissions from other roles
//   - Wildcards: Use "*" for pattern matching (e.g., "admin.*" matches all admin permissions)
//
// Basic usage:
//
//	// Define roles
//	roles := map[string]rbac.Role{
//	    "viewer": {
//	        Permissions: []string{"users.read", "projects.read"},
//	    },
//	    "editor": {
//	        Permissions: []string{"users.write", "projects.write"},
//	        Inherits:    []string{"viewer"}, // Inherits all viewer permissions
//	    },
//	    "admin": {
//	        Permissions: []string{"admin.*", "billing.*"},
//	        Inherits:    []string{"editor"}, // Inherits editor + viewer permissions
//	    },
//	}
//
//	// Create role source and authorizer
//	source := rbac.NewInMemRoleSource(roles)
//	auth, err := rbac.NewAuthorizer(ctx, source)
//
//	// Check permissions
//	if err := auth.Can("editor", "projects.write"); err != nil {
//	    // Handle permission denied
//	}
//
//	// Check from context
//	ctx = rbac.SetRoleToContext(ctx, "admin")
//	if err := auth.CanFromContext(ctx, "billing.view"); err != nil {
//	    // Handle permission denied
//	}
//
//	// Check multiple permissions
//	if err := auth.CanAll("editor", "users.write", "projects.write"); err != nil {
//	    // Handle missing permissions
//	}
package rbac
