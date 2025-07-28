package rbac

import "context"

// roleCtxKey is the context key for storing role information.
type roleCtxKey struct{}

// SetRoleToContext stores the user's role in the context.
func SetRoleToContext(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleCtxKey{}, role)
}

// GetRoleFromContext retrieves the user's role from the context.
func GetRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleCtxKey{}).(string)
	return role, ok
}
