package clientip

import (
	"context"
)

type clientIPContextKey struct{}

// SetIPToContext stores client IP in context for middleware chain access.
func SetIPToContext(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPContextKey{}, ip)
}

// GetIPFromContext retrieves client IP from context.
// Returns empty string if IP was not previously stored.
func GetIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPContextKey{}).(string)
	return ip
}
