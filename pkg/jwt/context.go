package jwt

import (
	"context"
	"encoding/json"
	"fmt"
)

// contextKey prevents context key collisions by using a private type.
// This follows Go best practices for context keys to avoid conflicts with other packages.
type contextKey struct{ name string }

func (c contextKey) String() string { return c.name }

var (
	jwtContextKey    = &contextKey{name: "jwt"}        // Raw JWT token string
	claimsContextKey = &contextKey{name: "jwt_claims"} // Parsed and validated claims
)

// SetToken stores the raw JWT token string in the context.
func SetToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, jwtContextKey, token)
}

// SetClaims stores validated JWT claims in the context.
// Accepts any claims type - typically map[string]any or custom structs.
func SetClaims(ctx context.Context, claims any) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// GetToken retrieves the JWT token string from the context.
func GetToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(jwtContextKey).(string)
	return token, ok
}

// GetClaims retrieves JWT claims from context with type safety.
// Returns zero value and false if claims are missing or of wrong type.
func GetClaims[T any](ctx context.Context) (T, bool) {
	claims, ok := ctx.Value(claimsContextKey).(T)
	if !ok {
		var zero T
		return zero, false
	}
	return claims, true
}

// GetClaimsAs converts JWT claims from context to the specified type.
// Uses direct type assertion first, then JSON marshaling for type conversion.
// This flexibility supports both map[string]any and custom struct claims.
func GetClaimsAs[T any](ctx context.Context, claims *T) error {
	if claims == nil {
		return fmt.Errorf("failed to unmarshal claims: %w", ErrInvalidClaims)
	}

	v := ctx.Value(claimsContextKey)
	if v == nil {
		return ErrInvalidClaims
	}

	// Fast path: direct type match
	if typedClaims, ok := v.(T); ok {
		*claims = typedClaims
		return nil
	}

	// Fallback: convert via JSON for type flexibility
	var jsonBytes []byte
	var err error

	switch c := v.(type) {
	case map[string]any:
		jsonBytes, err = json.Marshal(c)
	default:
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
