# RBAC Package

Role-Based Access Control (RBAC) implementation for SaasKit applications.

## Features

- **Role Inheritance**: Roles can inherit permissions from other roles
- **Wildcard Permissions**: Support for pattern matching (e.g., `admin.*`)
- **Context Integration**: Seamless integration with Go context
- **Thread-Safe**: Safe for concurrent use
- **Zero Dependencies**: Uses only standard library (except for tests)

## Usage

### Basic Setup

```go
import "github.com/saaskit/pkg/rbac"

// Define roles with permissions
roles := map[string]rbac.Role{
    "viewer": {
        Permissions: []string{"users.read", "projects.read"},
    },
    "editor": {
        Permissions: []string{"users.write", "projects.write"},
        Inherits:    []string{"viewer"}, // Inherits viewer permissions
    },
    "admin": {
        Permissions: []string{"admin.*", "billing.*"},
        Inherits:    []string{"editor"}, // Inherits editor + viewer
    },
}

// Create authorizer
source := rbac.NewInMemRoleSource(roles)
auth, err := rbac.NewAuthorizer(ctx, source)
if err != nil {
    log.Fatal(err)
}
```

### Permission Checking

```go
// Check single permission
if err := auth.Can("editor", "projects.write"); err != nil {
    // Permission denied
}

// Check any of multiple permissions
if err := auth.CanAny("editor", "users.admin", "users.write"); err != nil {
    // None of the permissions granted
}

// Check all permissions
if err := auth.CanAll("admin", "billing.view", "billing.edit"); err != nil {
    // Missing at least one permission
}
```

### Context-Based Authorization

```go
// Set role in context
ctx := rbac.SetRoleToContext(ctx, "admin")

// Check permissions from context
if err := auth.CanFromContext(ctx, "billing.manage"); err != nil {
    // Permission denied or no role in context
}
```

### Wildcard Permissions

```go
roles := map[string]rbac.Role{
    "admin": {
        // Matches admin.users, admin.billing, admin.settings, etc.
        Permissions: []string{"admin.*"},
    },
    "superadmin": {
        // Matches any permission
        Permissions: []string{"*"},
    },
}
```

## Integration with HTTP Middleware

```go
func RequirePermission(auth rbac.Authorizer, permission string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if err := auth.CanFromContext(r.Context(), permission); err != nil {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Usage
mux.Handle("/admin", RequirePermission(auth, "admin.access")(adminHandler))
```

## Custom Role Sources

Implement the `RoleSource` interface to load roles from your database:

```go
type dbRoleSource struct {
    db *sql.DB
}

func (s *dbRoleSource) Load(ctx context.Context) (map[string]rbac.Role, error) {
    // Load roles from database
    roles := make(map[string]rbac.Role)

    // Query roles and their permissions...

    return roles, nil
}
```

## Performance Considerations

- Permissions are precomputed during initialization for O(1) lookups
- The authorizer is thread-safe and can be shared across goroutines
- Role inheritance is resolved once during initialization
- In-memory role source creates defensive copies to prevent mutations
- Roles are immutable after initialization - all roles must be defined at compile time

## Best Practices

1. **Permission Naming**: Use hierarchical naming (e.g., `resource.action`)
2. **Role Hierarchy**: Keep inheritance chains shallow for clarity
3. **Wildcards**: Use sparingly and only for admin-level permissions
4. **Context Usage**: Always validate role presence when using context methods
5. **Error Handling**: Check for both `ErrInvalidRole` and `ErrInsufficientPermissions`
