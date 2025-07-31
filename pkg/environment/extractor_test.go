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

			ctx := environment.WithContext(context.Background(), tt.env)
			extractor := environment.LoggerExtractor()
			attr, ok := extractor(ctx)

			assert.True(t, ok)
			assert.Equal(t, "env", attr.Key)
			assert.Equal(t, tt.expected, attr.Value.String())
		})
	}
}

func TestLoggerExtractor_NoEnvironmentInContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	extractor := environment.LoggerExtractor()
	attr, ok := extractor(ctx)

	assert.False(t, ok)
	assert.Equal(t, slog.Attr{}, attr)
}

func TestLoggerExtractor_EmptyEnvironment(t *testing.T) {
	t.Parallel()

	ctx := environment.WithContext(context.Background(), environment.Environment(""))
	extractor := environment.LoggerExtractor()
	attr, ok := extractor(ctx)

	assert.False(t, ok)
	assert.Equal(t, slog.Attr{}, attr)
}
