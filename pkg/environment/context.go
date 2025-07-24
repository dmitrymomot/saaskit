package environment

import "context"

// Environment represents application environment.
type Environment string

const (
	// Development for development environment.
	Development Environment = "development"
	// Production for production environment.
	Production Environment = "production"
	// Staging for staging environment.
	Staging Environment = "staging"
)

type contextKey struct{}

// WithContext adds environment to context
func WithContext(ctx context.Context, env string) context.Context {
	return context.WithValue(ctx, contextKey{}, env)
}

// FromContext retrieves environment from context
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	env, _ := ctx.Value(contextKey{}).(string)
	return env
}

// IsProduction checks if the environment from context is production
func IsProduction(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == string(Production) || env == "prod"
}

// IsDevelopment checks if the environment from context is development
func IsDevelopment(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == string(Development) || env == "dev"
}

// IsStaging checks if the environment from context is staging
func IsStaging(ctx context.Context) bool {
	env := FromContext(ctx)
	return env == string(Staging) || env == "stage"
}
