package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestRequiredComparable(t *testing.T) {
	t.Run("passes for non-zero int", func(t *testing.T) {
		rule := validator.RequiredComparable("id", 123)
		assert.True(t, rule.Check())
		assert.Equal(t, "id", rule.Error.Field)
		assert.Equal(t, "field is required", rule.Error.Message)
		assert.Equal(t, "validation.required", rule.Error.TranslationKey)
		assert.Equal(t, map[string]any{"field": "id"}, rule.Error.TranslationValues)
	})

	t.Run("fails for zero int", func(t *testing.T) {
		rule := validator.RequiredComparable("id", 0)
		assert.False(t, rule.Check())
	})

	t.Run("passes for negative int", func(t *testing.T) {
		rule := validator.RequiredComparable("temperature", -5)
		assert.True(t, rule.Check())
	})

	t.Run("passes for non-empty string", func(t *testing.T) {
		rule := validator.RequiredComparable("name", "John")
		assert.True(t, rule.Check())
	})

	t.Run("fails for empty string", func(t *testing.T) {
		rule := validator.RequiredComparable("name", "")
		assert.False(t, rule.Check())
	})

	t.Run("passes for true bool", func(t *testing.T) {
		rule := validator.RequiredComparable("active", true)
		assert.True(t, rule.Check())
	})

	t.Run("fails for false bool", func(t *testing.T) {
		rule := validator.RequiredComparable("active", false)
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero float64", func(t *testing.T) {
		rule := validator.RequiredComparable("price", 10.99)
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero float64", func(t *testing.T) {
		rule := validator.RequiredComparable("price", 0.0)
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero float32", func(t *testing.T) {
		rule := validator.RequiredComparable("score", float32(85.5))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero float32", func(t *testing.T) {
		rule := validator.RequiredComparable("score", float32(0.0))
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero uint", func(t *testing.T) {
		rule := validator.RequiredComparable("count", uint(42))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero uint", func(t *testing.T) {
		rule := validator.RequiredComparable("count", uint(0))
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero int64", func(t *testing.T) {
		rule := validator.RequiredComparable("timestamp", int64(1234567890))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero int64", func(t *testing.T) {
		rule := validator.RequiredComparable("timestamp", int64(0))
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero uint64", func(t *testing.T) {
		rule := validator.RequiredComparable("size", uint64(1024))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero uint64", func(t *testing.T) {
		rule := validator.RequiredComparable("size", uint64(0))
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero int8", func(t *testing.T) {
		rule := validator.RequiredComparable("level", int8(5))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero int8", func(t *testing.T) {
		rule := validator.RequiredComparable("level", int8(0))
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero uint8", func(t *testing.T) {
		rule := validator.RequiredComparable("byte", uint8(255))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero uint8", func(t *testing.T) {
		rule := validator.RequiredComparable("byte", uint8(0))
		assert.False(t, rule.Check())
	})

	t.Run("works with custom comparable type", func(t *testing.T) {
		type CustomID string
		rule := validator.RequiredComparable("customID", CustomID("test-123"))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero custom comparable type", func(t *testing.T) {
		type CustomID string
		rule := validator.RequiredComparable("customID", CustomID(""))
		assert.False(t, rule.Check())
	})

	t.Run("works with custom numeric type", func(t *testing.T) {
		type UserID int
		rule := validator.RequiredComparable("userID", UserID(123))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero custom numeric type", func(t *testing.T) {
		type UserID int
		rule := validator.RequiredComparable("userID", UserID(0))
		assert.False(t, rule.Check())
	})
}

func TestComparableRulesIntegration(t *testing.T) {
	t.Run("validates multiple comparable types", func(t *testing.T) {
		userID := 123
		username := "john_doe"
		isActive := true
		score := 95.5

		err := validator.Apply(
			validator.RequiredComparable("userID", userID),
			validator.RequiredComparable("username", username),
			validator.RequiredComparable("isActive", isActive),
			validator.RequiredComparable("score", score),
		)

		assert.NoError(t, err)
	})

	t.Run("collects multiple comparable validation errors", func(t *testing.T) {
		userID := 0
		username := ""
		isActive := false
		score := 0.0

		err := validator.Apply(
			validator.RequiredComparable("userID", userID),
			validator.RequiredComparable("username", username),
			validator.RequiredComparable("isActive", isActive),
			validator.RequiredComparable("score", score),
		)

		assert.Error(t, err)
		assert.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)
		assert.True(t, validationErr.Has("userID"))
		assert.True(t, validationErr.Has("username"))
		assert.True(t, validationErr.Has("isActive"))
		assert.True(t, validationErr.Has("score"))

		// All should have the same error message
		for _, field := range []string{"userID", "username", "isActive", "score"} {
			errors := validationErr.Get(field)
			assert.Contains(t, errors, "field is required")
		}
	})

	t.Run("validates mixed comparable and other rule types", func(t *testing.T) {
		userID := 123
		email := "user@example.com"
		tags := []string{"admin", "user"}

		err := validator.Apply(
			validator.RequiredComparable("userID", userID),
			validator.RequiredString("email", email),
			validator.RequiredSlice("tags", tags),
		)

		assert.NoError(t, err)
	})
}
