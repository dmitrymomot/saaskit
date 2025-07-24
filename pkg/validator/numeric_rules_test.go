package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestRequiredNum(t *testing.T) {
	t.Parallel()
	t.Run("passes for non-zero int", func(t *testing.T) {
		rule := validator.RequiredNum("age", 25)
		assert.True(t, rule.Check())
		assert.Equal(t, "age", rule.Error.Field)
		assert.Equal(t, "field is required", rule.Error.Message)
		assert.Equal(t, "validation.required", rule.Error.TranslationKey)
		assert.Equal(t, map[string]any{"field": "age"}, rule.Error.TranslationValues)
	})

	t.Run("fails for zero int", func(t *testing.T) {
		rule := validator.RequiredNum("age", 0)
		assert.False(t, rule.Check())
	})

	t.Run("passes for positive int", func(t *testing.T) {
		rule := validator.RequiredNum("count", 1)
		assert.True(t, rule.Check())
	})

	t.Run("passes for negative int", func(t *testing.T) {
		rule := validator.RequiredNum("temperature", -10)
		assert.True(t, rule.Check())
	})

	t.Run("passes for non-zero float32", func(t *testing.T) {
		rule := validator.RequiredNum("price", float32(10.5))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero float32", func(t *testing.T) {
		rule := validator.RequiredNum("price", float32(0.0))
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero float64", func(t *testing.T) {
		rule := validator.RequiredNum("score", 85.7)
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero float64", func(t *testing.T) {
		rule := validator.RequiredNum("score", 0.0)
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero uint", func(t *testing.T) {
		rule := validator.RequiredNum("id", uint(123))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero uint", func(t *testing.T) {
		rule := validator.RequiredNum("id", uint(0))
		assert.False(t, rule.Check())
	})

	t.Run("passes for non-zero int64", func(t *testing.T) {
		rule := validator.RequiredNum("timestamp", int64(1234567890))
		assert.True(t, rule.Check())
	})

	t.Run("fails for zero int64", func(t *testing.T) {
		rule := validator.RequiredNum("timestamp", int64(0))
		assert.False(t, rule.Check())
	})
}

func TestMinNum(t *testing.T) {
	t.Parallel()
	t.Run("passes when int value equals minimum", func(t *testing.T) {
		rule := validator.MinNum("age", 18, 18)
		assert.True(t, rule.Check())
		assert.Equal(t, "age", rule.Error.Field)
		assert.Equal(t, "must be at least 18", rule.Error.Message)
		assert.Equal(t, "validation.min", rule.Error.TranslationKey)
		expectedValues := map[string]any{
			"field": "age",
			"min":   18,
		}
		assert.Equal(t, expectedValues, rule.Error.TranslationValues)
	})

	t.Run("passes when int value exceeds minimum", func(t *testing.T) {
		rule := validator.MinNum("age", 25, 18)
		assert.True(t, rule.Check())
	})

	t.Run("fails when int value is below minimum", func(t *testing.T) {
		rule := validator.MinNum("age", 16, 18)
		assert.False(t, rule.Check())
	})

	t.Run("passes when float64 value equals minimum", func(t *testing.T) {
		rule := validator.MinNum("score", 85.5, 85.5)
		assert.True(t, rule.Check())
	})

	t.Run("passes when float64 value exceeds minimum", func(t *testing.T) {
		rule := validator.MinNum("score", 90.0, 85.5)
		assert.True(t, rule.Check())
	})

	t.Run("fails when float64 value is below minimum", func(t *testing.T) {
		rule := validator.MinNum("score", 80.0, 85.5)
		assert.False(t, rule.Check())
		assert.Equal(t, "must be at least 85.5", rule.Error.Message)
		assert.Equal(t, 85.5, rule.Error.TranslationValues["min"])
	})

	t.Run("works with float32", func(t *testing.T) {
		rule := validator.MinNum("price", float32(15.99), float32(10.0))
		assert.True(t, rule.Check())
	})

	t.Run("works with uint", func(t *testing.T) {
		rule := validator.MinNum("count", uint(5), uint(3))
		assert.True(t, rule.Check())
	})

	t.Run("works with int64", func(t *testing.T) {
		rule := validator.MinNum("bytes", int64(1024), int64(512))
		assert.True(t, rule.Check())
	})

	t.Run("handles negative numbers", func(t *testing.T) {
		rule := validator.MinNum("temperature", -5, -10)
		assert.True(t, rule.Check())
	})

	t.Run("fails for negative number below negative minimum", func(t *testing.T) {
		rule := validator.MinNum("temperature", -15, -10)
		assert.False(t, rule.Check())
	})
}

func TestMaxNum(t *testing.T) {
	t.Parallel()
	t.Run("passes when int value equals maximum", func(t *testing.T) {
		rule := validator.MaxNum("age", 65, 65)
		assert.True(t, rule.Check())
		assert.Equal(t, "age", rule.Error.Field)
		assert.Equal(t, "must be at most 65", rule.Error.Message)
		assert.Equal(t, "validation.max", rule.Error.TranslationKey)
		expectedValues := map[string]any{
			"field": "age",
			"max":   65,
		}
		assert.Equal(t, expectedValues, rule.Error.TranslationValues)
	})

	t.Run("passes when int value is below maximum", func(t *testing.T) {
		rule := validator.MaxNum("age", 45, 65)
		assert.True(t, rule.Check())
	})

	t.Run("fails when int value exceeds maximum", func(t *testing.T) {
		rule := validator.MaxNum("age", 75, 65)
		assert.False(t, rule.Check())
	})

	t.Run("passes when float64 value equals maximum", func(t *testing.T) {
		rule := validator.MaxNum("score", 100.0, 100.0)
		assert.True(t, rule.Check())
	})

	t.Run("passes when float64 value is below maximum", func(t *testing.T) {
		rule := validator.MaxNum("score", 95.5, 100.0)
		assert.True(t, rule.Check())
	})

	t.Run("fails when float64 value exceeds maximum", func(t *testing.T) {
		rule := validator.MaxNum("score", 105.0, 100.0)
		assert.False(t, rule.Check())
		assert.Equal(t, "must be at most 100", rule.Error.Message)
		assert.Equal(t, 100.0, rule.Error.TranslationValues["max"])
	})

	t.Run("works with float32", func(t *testing.T) {
		rule := validator.MaxNum("price", float32(9.99), float32(10.0))
		assert.True(t, rule.Check())
	})

	t.Run("works with uint", func(t *testing.T) {
		rule := validator.MaxNum("count", uint(3), uint(5))
		assert.True(t, rule.Check())
	})

	t.Run("works with int64", func(t *testing.T) {
		rule := validator.MaxNum("bytes", int64(512), int64(1024))
		assert.True(t, rule.Check())
	})

	t.Run("handles negative numbers", func(t *testing.T) {
		rule := validator.MaxNum("temperature", -15, -10)
		assert.True(t, rule.Check())
	})

	t.Run("fails for negative number above negative maximum", func(t *testing.T) {
		rule := validator.MaxNum("temperature", -5, -10)
		assert.False(t, rule.Check())
	})
}

func TestNumericConvenienceAliases(t *testing.T) {
	t.Parallel()
	t.Run("Min alias works for int", func(t *testing.T) {
		rule := validator.Min("age", 25, 18)
		assert.True(t, rule.Check())
		assert.Equal(t, "age", rule.Error.Field)
		assert.Equal(t, "must be at least 18", rule.Error.Message)
		assert.Equal(t, "validation.min", rule.Error.TranslationKey)
	})

	t.Run("Min alias works for float64", func(t *testing.T) {
		rule := validator.Min("score", 85.5, 80.0)
		assert.True(t, rule.Check())
	})

	t.Run("Max alias works for int", func(t *testing.T) {
		rule := validator.Max("age", 25, 65)
		assert.True(t, rule.Check())
		assert.Equal(t, "age", rule.Error.Field)
		assert.Equal(t, "must be at most 65", rule.Error.Message)
		assert.Equal(t, "validation.max", rule.Error.TranslationKey)
	})

	t.Run("Max alias works for float64", func(t *testing.T) {
		rule := validator.Max("score", 85.5, 100.0)
		assert.True(t, rule.Check())
	})
}

func TestNumericRulesIntegration(t *testing.T) {
	t.Parallel()
	t.Run("validates complete numeric input", func(t *testing.T) {
		age := 25
		score := 85.5
		count := uint(10)

		err := validator.Apply(
			validator.RequiredNum("age", age),
			validator.MinNum("age", age, 18),
			validator.MaxNum("age", age, 120),
			validator.RequiredNum("score", score),
			validator.MinNum("score", score, 0.0),
			validator.MaxNum("score", score, 100.0),
			validator.RequiredNum("count", count),
			validator.MinNum("count", count, uint(1)),
		)

		assert.NoError(t, err)
	})

	t.Run("collects multiple numeric validation errors", func(t *testing.T) {
		age := 0         // Required but zero
		score := 150.0   // Too high
		count := uint(0) // Required but zero

		err := validator.Apply(
			validator.RequiredNum("age", age),
			validator.MinNum("age", age, 18),
			validator.MaxNum("score", score, 100.0),
			validator.RequiredNum("count", count),
		)

		assert.Error(t, err)
		assert.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)
		assert.True(t, validationErr.Has("age"))
		assert.True(t, validationErr.Has("score"))
		assert.True(t, validationErr.Has("count"))

		ageErrors := validationErr.Get("age")
		assert.Contains(t, ageErrors, "field is required")

		scoreErrors := validationErr.Get("score")
		assert.Contains(t, scoreErrors, "must be at most 100")

		countErrors := validationErr.Get("count")
		assert.Contains(t, countErrors, "field is required")
	})

	t.Run("validates mixed positive and negative numbers", func(t *testing.T) {
		temperature := -5
		balance := -100.50

		err := validator.Apply(
			validator.RequiredNum("temperature", temperature),
			validator.MinNum("temperature", temperature, -20),
			validator.MaxNum("temperature", temperature, 50),
			validator.RequiredNum("balance", balance),
			validator.MinNum("balance", balance, -1000.0),
		)

		assert.NoError(t, err)
	})
}
