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

// WithContext adds environment to context
func WithContext(ctx context.Context, env Environment) context.Context {
	return context.WithValue(ctx, contextKey{}, string(env))
}

// FromContext retrieves environment from context
func FromContext(ctx context.Context) Environment {
	if ctx == nil {
		return ""
	}
	env, _ := ctx.Value(contextKey{}).(string)
	return Environment(env)
}

// IsProduction checks if the environment from context is production
func IsProduction(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == Production || env == "prod"
}

// IsDevelopment checks if the environment from context is development
func IsDevelopment(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == Development || env == "dev"
}

// IsStaging checks if the environment from context is staging
func IsStaging(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == Staging || env == "stage"
}
