package clientip

import (
	"context"
)

// clientIPContextKey is the key for storing client IP in context
type clientIPContextKey struct{}

// SetIPToContext stores client IP in context
func SetIPToContext(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPContextKey{}, ip)
}

// GetIPFromContext retrieves client IP from context
func GetIPFromContext(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPContextKey{}).(string)
	return ip
}
