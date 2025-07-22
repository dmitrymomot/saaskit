package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestRequiredString(t *testing.T) {
	t.Run("passes for non-empty string", func(t *testing.T) {
		rule := validator.RequiredString("email", "test@example.com")
		assert.True(t, rule.Check())
		assert.Equal(t, "email", rule.Error.Field)
		assert.Equal(t, "field is required", rule.Error.Message)
		assert.Equal(t, "validation.required", rule.Error.TranslationKey)
		assert.Equal(t, map[string]any{"field": "email"}, rule.Error.TranslationValues)
	})

	t.Run("fails for empty string", func(t *testing.T) {
		rule := validator.RequiredString("email", "")
		assert.False(t, rule.Check())
	})

	t.Run("fails for whitespace-only string", func(t *testing.T) {
		rule := validator.RequiredString("email", "   ")
		assert.False(t, rule.Check())
	})

	t.Run("passes for string with leading/trailing whitespace but content", func(t *testing.T) {
		rule := validator.RequiredString("name", "  John  ")
		assert.True(t, rule.Check())
	})
}

func TestMinLenString(t *testing.T) {
	t.Run("passes when string equals minimum length", func(t *testing.T) {
		rule := validator.MinLenString("password", "12345", 5)
		assert.True(t, rule.Check())
		assert.Equal(t, "password", rule.Error.Field)
		assert.Equal(t, "must be at least 5 characters long", rule.Error.Message)
		assert.Equal(t, "validation.min_length", rule.Error.TranslationKey)
		expectedValues := map[string]any{
			"field": "password",
			"min":   5,
		}
		assert.Equal(t, expectedValues, rule.Error.TranslationValues)
	})

	t.Run("passes when string exceeds minimum length", func(t *testing.T) {
		rule := validator.MinLenString("password", "123456", 5)
		assert.True(t, rule.Check())
	})

	t.Run("fails when string is shorter than minimum", func(t *testing.T) {
		rule := validator.MinLenString("password", "1234", 5)
		assert.False(t, rule.Check())
	})

	t.Run("handles zero minimum length", func(t *testing.T) {
		rule := validator.MinLenString("text", "", 0)
		assert.True(t, rule.Check())
	})

	t.Run("handles large minimum length", func(t *testing.T) {
		rule := validator.MinLenString("text", "short", 100)
		assert.False(t, rule.Check())
		assert.Equal(t, "must be at least 100 characters long", rule.Error.Message)
		assert.Equal(t, 100, rule.Error.TranslationValues["min"])
	})
}

func TestMaxLenString(t *testing.T) {
	t.Run("passes when string equals maximum length", func(t *testing.T) {
		rule := validator.MaxLenString("username", "12345", 5)
		assert.True(t, rule.Check())
		assert.Equal(t, "username", rule.Error.Field)
		assert.Equal(t, "must be at most 5 characters long", rule.Error.Message)
		assert.Equal(t, "validation.max_length", rule.Error.TranslationKey)
		expectedValues := map[string]any{
			"field": "username",
			"max":   5,
		}
		assert.Equal(t, expectedValues, rule.Error.TranslationValues)
	})

	t.Run("passes when string is shorter than maximum", func(t *testing.T) {
		rule := validator.MaxLenString("username", "1234", 5)
		assert.True(t, rule.Check())
	})

	t.Run("fails when string exceeds maximum length", func(t *testing.T) {
		rule := validator.MaxLenString("username", "123456", 5)
		assert.False(t, rule.Check())
	})

	t.Run("handles zero maximum length", func(t *testing.T) {
		rule := validator.MaxLenString("text", "", 0)
		assert.True(t, rule.Check())
	})

	t.Run("fails for any content when max is zero", func(t *testing.T) {
		rule := validator.MaxLenString("text", "a", 0)
		assert.False(t, rule.Check())
	})
}

func TestLenString(t *testing.T) {
	t.Run("passes when string equals exact length", func(t *testing.T) {
		rule := validator.LenString("code", "12345", 5)
		assert.True(t, rule.Check())
		assert.Equal(t, "code", rule.Error.Field)
		assert.Equal(t, "must be exactly 5 characters long", rule.Error.Message)
		assert.Equal(t, "validation.exact_length", rule.Error.TranslationKey)
		expectedValues := map[string]any{
			"field":  "code",
			"length": 5,
		}
		assert.Equal(t, expectedValues, rule.Error.TranslationValues)
	})

	t.Run("fails when string is shorter", func(t *testing.T) {
		rule := validator.LenString("code", "1234", 5)
		assert.False(t, rule.Check())
	})

	t.Run("fails when string is longer", func(t *testing.T) {
		rule := validator.LenString("code", "123456", 5)
		assert.False(t, rule.Check())
	})

	t.Run("handles zero length requirement", func(t *testing.T) {
		rule := validator.LenString("empty", "", 0)
		assert.True(t, rule.Check())
	})

	t.Run("fails for non-empty string when zero length required", func(t *testing.T) {
		rule := validator.LenString("empty", "a", 0)
		assert.False(t, rule.Check())
	})
}

func TestStringConvenienceAliases(t *testing.T) {
	t.Run("Required alias works for strings", func(t *testing.T) {
		rule := validator.Required("email", "test@example.com")
		assert.True(t, rule.Check())
		assert.Equal(t, "email", rule.Error.Field)
		assert.Equal(t, "field is required", rule.Error.Message)
		assert.Equal(t, "validation.required", rule.Error.TranslationKey)
	})

	t.Run("Required alias fails for empty strings", func(t *testing.T) {
		rule := validator.Required("email", "")
		assert.False(t, rule.Check())
	})

	t.Run("MinLen alias works for strings", func(t *testing.T) {
		rule := validator.MinLen("password", "12345", 5)
		assert.True(t, rule.Check())
		assert.Equal(t, "password", rule.Error.Field)
		assert.Equal(t, "must be at least 5 characters long", rule.Error.Message)
		assert.Equal(t, "validation.min_length", rule.Error.TranslationKey)
	})

	t.Run("MaxLen alias works for strings", func(t *testing.T) {
		rule := validator.MaxLen("username", "12345", 10)
		assert.True(t, rule.Check())
		assert.Equal(t, "username", rule.Error.Field)
		assert.Equal(t, "must be at most 10 characters long", rule.Error.Message)
		assert.Equal(t, "validation.max_length", rule.Error.TranslationKey)
	})

	t.Run("Len alias works for strings", func(t *testing.T) {
		rule := validator.Len("code", "12345", 5)
		assert.True(t, rule.Check())
		assert.Equal(t, "code", rule.Error.Field)
		assert.Equal(t, "must be exactly 5 characters long", rule.Error.Message)
		assert.Equal(t, "validation.exact_length", rule.Error.TranslationKey)
	})
}

func TestStringRulesIntegration(t *testing.T) {
	t.Run("validates complete string input", func(t *testing.T) {
		email := "user@example.com"
		password := "securepassword123"

		err := validator.Apply(
			validator.RequiredString("email", email),
			validator.RequiredString("password", password),
			validator.MinLenString("password", password, 8),
			validator.MaxLenString("password", password, 50),
		)

		assert.NoError(t, err)
	})

	t.Run("collects multiple string validation errors", func(t *testing.T) {
		email := ""
		password := "123"

		err := validator.Apply(
			validator.RequiredString("email", email),
			validator.RequiredString("password", password),
			validator.MinLenString("password", password, 8),
		)

		assert.Error(t, err)
		assert.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)
		assert.True(t, validationErr.Has("email"))
		assert.True(t, validationErr.Has("password"))

		emailErrors := validationErr.Get("email")
		assert.Contains(t, emailErrors, "field is required")

		passwordErrors := validationErr.Get("password")
		assert.Contains(t, passwordErrors, "must be at least 8 characters long")
	})

	t.Run("validates translation data in errors", func(t *testing.T) {
		email := ""
		password := "123"

		err := validator.Apply(
			validator.RequiredString("email", email),
			validator.MinLenString("password", password, 8),
		)

		assert.Error(t, err)
		validationErr := validator.ExtractValidationErrors(err)

		emailErrors := validationErr.GetErrors("email")
		assert.Len(t, emailErrors, 1)
		assert.Equal(t, "validation.required", emailErrors[0].TranslationKey)
		assert.Equal(t, "email", emailErrors[0].TranslationValues["field"])

		passwordErrors := validationErr.GetErrors("password")
		assert.Len(t, passwordErrors, 1)
		assert.Equal(t, "validation.min_length", passwordErrors[0].TranslationKey)
		assert.Equal(t, "password", passwordErrors[0].TranslationValues["field"])
		assert.Equal(t, 8, passwordErrors[0].TranslationValues["min"])
	})
}
