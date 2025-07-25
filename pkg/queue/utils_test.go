package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/queue"
)

// Test types for qualified name extraction
type simpleStruct struct {
	Field string
}

type nestedStruct struct {
	Inner simpleStruct
}

// TestQualifiedStructName tests the internal qualifiedStructName function indirectly
// through the handler creation which uses it to generate task names
func TestQualifiedStructName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "simple struct",
			input:    simpleStruct{},
			expected: "queue_test.simpleStruct",
		},
		{
			name:     "pointer to struct",
			input:    &simpleStruct{},
			expected: "queue_test.simpleStruct",
		},
		{
			name:     "nested struct",
			input:    nestedStruct{},
			expected: "queue_test.nestedStruct",
		},
		{
			name:     "pointer to nested struct",
			input:    &nestedStruct{},
			expected: "queue_test.nestedStruct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test through handler creation
			switch tt.input.(type) {
			case simpleStruct, *simpleStruct:
				handler := queue.NewTaskHandler(func(ctx context.Context, payload simpleStruct) error {
					return nil
				})
				assert.Equal(t, tt.expected, handler.Name())
			case nestedStruct, *nestedStruct:
				handler := queue.NewTaskHandler(func(ctx context.Context, payload nestedStruct) error {
					return nil
				})
				assert.Equal(t, tt.expected, handler.Name())
			}
		})
	}
}

// TestQualifiedStructName_ThroughEnqueuer tests name generation through the enqueuer
func TestQualifiedStructName_ThroughEnqueuer(t *testing.T) {
	t.Parallel()

	type testPayload struct {
		Value string
	}

	repo := &mockEnqueuerRepo{}
	enqueuer, err := queue.NewEnqueuer(repo)
	require.NoError(t, err)

	tests := []struct {
		name         string
		payload      any
		expectedName string
	}{
		{
			name:         "struct payload",
			payload:      testPayload{Value: "test"},
			expectedName: "queue_test.testPayload",
		},
		{
			name:         "pointer payload",
			payload:      &testPayload{Value: "test"},
			expectedName: "queue_test.testPayload",
		},
		{
			name:         "map payload",
			payload:      map[string]string{"key": "value"},
			expectedName: "map[string]string",
		},
		{
			name:         "slice payload",
			payload:      []string{"a", "b", "c"},
			expectedName: "[]string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo.tasks = nil // Clear tasks

			err := enqueuer.Enqueue(context.Background(), tt.payload)
			require.NoError(t, err)

			require.Len(t, repo.tasks, 1)
			assert.Equal(t, tt.expectedName, repo.tasks[0].TaskName)
		})
	}
}

// Test with types from different packages
func TestQualifiedStructName_ExternalPackages(t *testing.T) {
	t.Parallel()

	t.Run("time.Time through handler", func(t *testing.T) {
		t.Parallel()
		handler := queue.NewTaskHandler(func(ctx context.Context, payload time.Time) error {
			return nil
		})
		assert.Equal(t, "time.Time", handler.Name())
	})

	t.Run("uuid.UUID through handler", func(t *testing.T) {
		t.Parallel()
		handler := queue.NewTaskHandler(func(ctx context.Context, payload uuid.UUID) error {
			return nil
		})
		assert.Equal(t, "uuid.UUID", handler.Name())
	})
}

// Test anonymous struct handling
func TestQualifiedStructName_AnonymousStruct(t *testing.T) {
	t.Parallel()

	repo := &mockEnqueuerRepo{}
	enqueuer, err := queue.NewEnqueuer(repo)
	require.NoError(t, err)

	// Anonymous struct
	payload := struct {
		Name  string
		Value int
	}{
		Name:  "test",
		Value: 42,
	}

	err = enqueuer.Enqueue(context.Background(), payload)
	require.NoError(t, err)

	require.Len(t, repo.tasks, 1)
	// Anonymous structs will have a complex type name
	assert.Contains(t, repo.tasks[0].TaskName, "struct")
}
