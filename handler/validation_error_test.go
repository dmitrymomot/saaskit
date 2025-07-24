package handler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/handler"
)

func TestValidationError(t *testing.T) {
	t.Parallel()
	t.Run("empty error", func(t *testing.T) {
		t.Parallel()
		err := handler.NewValidationError()
		assert.Equal(t, "Validation failed", err.Error())
		assert.True(t, err.IsEmpty())
	})

	t.Run("single field single error", func(t *testing.T) {
		t.Parallel()
		err := handler.NewValidationError()
		err.Add("email", "invalid format")

		assert.Equal(t, "validation error: email: invalid format", err.Error())
		assert.False(t, err.IsEmpty())
		assert.True(t, err.Has("email"))
		assert.False(t, err.Has("name"))
		assert.Equal(t, "invalid format", err.Get("email"))
	})

	t.Run("multiple fields", func(t *testing.T) {
		t.Parallel()
		err := handler.NewValidationError()
		err.Add("email", "invalid format")
		err.Add("age", "must be positive")

		// Error message should contain both fields
		errMsg := err.Error()
		assert.Contains(t, errMsg, "validation error:")
		assert.Contains(t, errMsg, "email: invalid format")
		assert.Contains(t, errMsg, "age: must be positive")
	})

	t.Run("multiple errors for same field", func(t *testing.T) {
		t.Parallel()
		err := handler.NewValidationError()
		err.Add("email", "invalid format")
		err.Add("email", "already exists")

		// Error() shows only first error
		assert.Contains(t, err.Error(), "email: invalid format")

		// But all errors are stored
		assert.Len(t, err["email"], 2)
		assert.Equal(t, "invalid format", err["email"][0])
		assert.Equal(t, "already exists", err["email"][1])
	})
}
