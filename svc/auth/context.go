package auth

import (
	"context"
)

type userContextKey struct{}

// SetUserToContext stores authenticated user in context for middleware chain access.
func SetUserToContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// GetUserFromContext retrieves authenticated user from context.
// Returns nil if user was not previously stored.
func GetUserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey{}).(*User)
	return user
}
