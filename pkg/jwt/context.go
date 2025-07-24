package jwt

import (
	"context"
	"encoding/json"
	"fmt"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey struct{ name string }

// String returns the name of the context key.
func (c contextKey) String() string { return c.name }

var (
	jwtContextKey    = &contextKey{name: "jwt"}        // JWT string
	claimsContextKey = &contextKey{name: "jwt_claims"} // Parsed JWT claims
)

// SetToken sets the JWT token string in the context.
func SetToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, jwtContextKey, token)
}

// SetClaims sets the JWT claims in the context.
// It accepts any type of claims (struct or map).
func SetClaims(ctx context.Context, claims any) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// GetToken returns the JWT token string from the context.
// If no token is found, the second return value will be false.
func GetToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(jwtContextKey).(string)
	return token, ok
}

// GetClaims returns the JWT claims from the context as the specified type T.
// If no claims are found or they are of a different type, the second return value will be false.
func GetClaims[T any](ctx context.Context) (T, bool) {
	claims, ok := ctx.Value(claimsContextKey).(T)
	if !ok {
		var zero T
		return zero, false
	}
	return claims, true
}

// GetClaimsAs parses the JWT claims from the context into the specified struct.
// If no claims are found or they cannot be parsed, an error is returned.
func GetClaimsAs[T any](ctx context.Context, claims *T) error {
	if claims == nil {
		return fmt.Errorf("failed to unmarshal claims: %w", ErrInvalidClaims)
	}

	// First try direct type assertion
	v := ctx.Value(claimsContextKey)
	if v == nil {
		return ErrInvalidClaims
	}

	// If the value is already of the expected type, just assign it
	if typedClaims, ok := v.(T); ok {
		*claims = typedClaims
		return nil
	}

	// Otherwise, try to convert via JSON
	var jsonBytes []byte
	var err error

	// Handle different claim types: map[string]any or struct
	switch c := v.(type) {
	case map[string]any:
		jsonBytes, err = json.Marshal(c)
	default:
		// Try to convert any other type through JSON marshaling
		jsonBytes, err = json.Marshal(c)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal claims: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, claims); err != nil {
		return fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return nil
}
