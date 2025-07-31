package environment_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/environment"
)

func TestWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  environment.Environment
	}{
		{
			name: "development environment",
			env:  environment.Development,
		},
		{
			name: "production environment",
			env:  environment.Production,
		},
		{
			name: "staging environment",
			env:  environment.Staging,
		},
		{
			name: "custom environment",
			env:  environment.Environment("custom"),
		},
		{
			name: "empty environment",
			env:  environment.Environment(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctxWithEnv := environment.WithContext(ctx, tt.env)

			assert.NotNil(t, ctxWithEnv)
			assert.NotEqual(t, ctx, ctxWithEnv)

			retrievedEnv := environment.FromContext(ctxWithEnv)
			assert.Equal(t, tt.env, retrievedEnv)
		})
	}
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	t.Run("context with environment", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		originalEnv := environment.Production
		ctxWithEnv := environment.WithContext(ctx, originalEnv)

		retrievedEnv := environment.FromContext(ctxWithEnv)
		assert.Equal(t, originalEnv, retrievedEnv)
	})

	t.Run("context without environment", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		retrievedEnv := environment.FromContext(ctx)

		assert.Equal(t, environment.Environment(""), retrievedEnv)
	})

	t.Run("nil context", func(t *testing.T) {
		t.Parallel()

		retrievedEnv := environment.FromContext(context.TODO())

		assert.Equal(t, environment.Environment(""), retrievedEnv)
	})
}

func TestIsProduction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      environment.Environment
		expected bool
	}{
		{
			name:     "production environment",
			env:      environment.Production,
			expected: true,
		},
		{
			name:     "development environment",
			env:      environment.Development,
			expected: false,
		},
		{
			name:     "staging environment",
			env:      environment.Staging,
			expected: false,
		},
		{
			name:     "empty environment",
			env:      environment.Environment(""),
			expected: false,
		},
		{
			name:     "prod alias",
			env:      environment.Environment("prod"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := environment.WithContext(context.Background(), tt.env)
			result := environment.IsProduction(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDevelopment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      environment.Environment
		expected bool
	}{
		{
			name:     "development environment",
			env:      environment.Development,
			expected: true,
		},
		{
			name:     "production environment",
			env:      environment.Production,
			expected: false,
		},
		{
			name:     "staging environment",
			env:      environment.Staging,
			expected: false,
		},
		{
			name:     "empty environment",
			env:      environment.Environment(""),
			expected: false,
		},
		{
			name:     "dev alias",
			env:      environment.Environment("dev"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := environment.WithContext(context.Background(), tt.env)
			result := environment.IsDevelopment(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStaging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      environment.Environment
		expected bool
	}{
		{
			name:     "staging environment",
			env:      environment.Staging,
			expected: true,
		},
		{
			name:     "development environment",
			env:      environment.Development,
			expected: false,
		},
		{
			name:     "production environment",
			env:      environment.Production,
			expected: false,
		},
		{
			name:     "empty environment",
			env:      environment.Environment(""),
			expected: false,
		},
		{
			name:     "stage alias",
			env:      environment.Environment("stage"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := environment.WithContext(context.Background(), tt.env)
			result := environment.IsStaging(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}
