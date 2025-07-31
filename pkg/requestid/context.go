package requestid

import "context"

// contextKey uses an empty struct to ensure uniqueness and zero memory overhead
type contextKey struct{}

func WithContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, contextKey{}, requestID)
}

func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, ok := ctx.Value(contextKey{}).(string)
	if !ok {
		return ""
	}
	return requestID
}
