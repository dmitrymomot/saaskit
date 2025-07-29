package environment_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/environment"
)

func TestLoggerExtractor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      environment.Environment
		expected string
	}{
		{
			name:     "development environment",
			env:      environment.Development,
			expected: "development",
		},
		{
			name:     "production environment",
			env:      environment.Production,
			expected: "production",
		},
		{
			name:     "staging environment",
			env:      environment.Staging,
			expected: "staging",
		},
		{
			name:     "custom environment",
			env:      environment.Environment("custom"),
			expected: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create context with environment
			ctx := environment.WithContext(context.Background(), tt.env)

			// Create logger extractor
			extractor := environment.LoggerExtractor()

			// Extract attribute
			attr, ok := extractor(ctx)

			// Should extract environment attribute
			assert.True(t, ok)
			assert.Equal(t, "env", attr.Key)
			assert.Equal(t, tt.expected, attr.Value.String())
		})
	}
}

func TestLoggerExtractor_NoEnvironmentInContext(t *testing.T) {
	t.Parallel()

	// Create context without environment
	ctx := context.Background()

	// Create logger extractor
	extractor := environment.LoggerExtractor()

	// Extract attribute
	attr, ok := extractor(ctx)

	// Should not extract attribute when no environment is set
	assert.False(t, ok)
	assert.Equal(t, slog.Attr{}, attr)
}

func TestLoggerExtractor_EmptyEnvironment(t *testing.T) {
	t.Parallel()

	// Create context with empty environment
	ctx := environment.WithContext(context.Background(), environment.Environment(""))

	// Create logger extractor
	extractor := environment.LoggerExtractor()

	// Extract attribute
	attr, ok := extractor(ctx)

	// Should not extract attribute when environment is empty
	assert.False(t, ok)
	assert.Equal(t, slog.Attr{}, attr)
}
