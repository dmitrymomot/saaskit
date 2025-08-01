package environment

import "context"

// Environment represents application environment.
type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
	Staging     Environment = "staging"
)

type contextKey struct{}

// WithContext attaches the environment to a context, preserving type safety
// for compile-time checking and consistent API usage throughout the application.
func WithContext(ctx context.Context, env Environment) context.Context {
	return context.WithValue(ctx, contextKey{}, env)
}

// FromContext retrieves the environment from a context, returning empty Environment
// for nil contexts or missing values to ensure zero-allocation error handling.
func FromContext(ctx context.Context) Environment {
	if ctx == nil {
		return ""
	}
	env, _ := ctx.Value(contextKey{}).(Environment)
	return env
}

// IsProduction checks if the environment is production.
// Accepts both "production" (constant) and "prod" (common alias) for flexibility.
func IsProduction(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == Production || env == "prod"
}

// IsDevelopment checks if the environment is development.
// Accepts both "development" (constant) and "dev" (common alias) for flexibility.
func IsDevelopment(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == Development || env == "dev"
}

// IsStaging checks if the environment is staging.
// Accepts both "staging" (constant) and "stage" (common alias) for flexibility.
func IsStaging(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == Staging || env == "stage"
}
