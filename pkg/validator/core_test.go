package validator_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestValidationErrors_Error(t *testing.T) {
	t.Run("returns default message when no errors", func(t *testing.T) {
		var errs validator.ValidationErrors
		assert.Equal(t, "validation failed", errs.Error())
	})

	t.Run("returns formatted message with single error", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "email",
			Message: "is required",
		})
		assert.Equal(t, "validation failed: email: is required", errs.Error())
	})

	t.Run("returns formatted message with multiple errors", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "email",
			Message: "is required",
		})
		errs.Add(validator.ValidationError{
			Field:   "password",
			Message: "too short",
		})

		errorMsg := errs.Error()
		assert.Contains(t, errorMsg, "validation failed:")
		assert.Contains(t, errorMsg, "email: is required")
		assert.Contains(t, errorMsg, "password: too short")
	})

	t.Run("returns formatted message with multiple errors for same field", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "password",
			Message: "too short",
		})
		errs.Add(validator.ValidationError{
			Field:   "password",
			Message: "missing special character",
		})

		errorMsg := errs.Error()
		assert.Contains(t, errorMsg, "password: too short")
		assert.Contains(t, errorMsg, "password: missing special character")
	})
}

func TestValidationErrors_Add(t *testing.T) {
	t.Run("adds error to collection", func(t *testing.T) {
		var errs validator.ValidationErrors
		err := validator.ValidationError{
			Field:   "email",
			Message: "is required",
		}
		errs.Add(err)

		assert.True(t, errs.Has("email"))
		assert.Equal(t, []string{"is required"}, errs.Get("email"))
	})

	t.Run("adds multiple errors to same field", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "password",
			Message: "too short",
		})
		errs.Add(validator.ValidationError{
			Field:   "password",
			Message: "missing special character",
		})

		expected := []string{"too short", "missing special character"}
		assert.Equal(t, expected, errs.Get("password"))
	})
}

func TestValidationErrors_Has(t *testing.T) {
	t.Run("returns true for field with errors", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "email",
			Message: "is required",
		})

		assert.True(t, errs.Has("email"))
	})

	t.Run("returns false for field without errors", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "email",
			Message: "is required",
		})

		assert.False(t, errs.Has("password"))
	})

	t.Run("returns false for non-existent field", func(t *testing.T) {
		var errs validator.ValidationErrors

		assert.False(t, errs.Has("nonexistent"))
	})
}

func TestValidationErrors_Get(t *testing.T) {
	t.Run("returns errors for existing field", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "email",
			Message: "is required",
		})
		errs.Add(validator.ValidationError{
			Field:   "email",
			Message: "invalid format",
		})

		expected := []string{"is required", "invalid format"}
		assert.Equal(t, expected, errs.Get("email"))
	})

	t.Run("returns empty slice for non-existent field", func(t *testing.T) {
		var errs validator.ValidationErrors

		result := errs.Get("nonexistent")
		assert.Empty(t, result)
	})
}

func TestValidationErrors_GetErrors(t *testing.T) {
	t.Run("returns ValidationError objects for existing field", func(t *testing.T) {
		var errs validator.ValidationErrors
		err1 := validator.ValidationError{
			Field:             "email",
			Message:           "is required",
			TranslationKey:    "validation.required",
			TranslationValues: map[string]any{"field": "email"},
		}
		err2 := validator.ValidationError{
			Field:             "email",
			Message:           "invalid format",
			TranslationKey:    "validation.email",
			TranslationValues: map[string]any{"field": "email"},
		}
		errs.Add(err1)
		errs.Add(err2)

		result := errs.GetErrors("email")
		assert.Len(t, result, 2)
		assert.Equal(t, err1, result[0])
		assert.Equal(t, err2, result[1])
	})

	t.Run("returns empty slice for non-existent field", func(t *testing.T) {
		var errs validator.ValidationErrors

		result := errs.GetErrors("nonexistent")
		assert.Empty(t, result)
	})
}

func TestValidationErrors_Fields(t *testing.T) {
	t.Run("returns all fields with errors", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{Field: "email", Message: "is required"})
		errs.Add(validator.ValidationError{Field: "password", Message: "too short"})
		errs.Add(validator.ValidationError{Field: "name", Message: "invalid"})

		fields := errs.Fields()
		assert.Len(t, fields, 3)
		assert.Contains(t, fields, "email")
		assert.Contains(t, fields, "password")
		assert.Contains(t, fields, "name")
	})

	t.Run("returns unique fields only", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{Field: "email", Message: "is required"})
		errs.Add(validator.ValidationError{Field: "email", Message: "invalid format"})
		errs.Add(validator.ValidationError{Field: "password", Message: "too short"})

		fields := errs.Fields()
		assert.Len(t, fields, 2)
		assert.Contains(t, fields, "email")
		assert.Contains(t, fields, "password")
	})

	t.Run("returns empty slice for no errors", func(t *testing.T) {
		var errs validator.ValidationErrors

		fields := errs.Fields()
		assert.Empty(t, fields)
	})
}

func TestValidationErrors_IsEmpty(t *testing.T) {
	t.Run("returns true for empty errors", func(t *testing.T) {
		var errs validator.ValidationErrors

		assert.True(t, errs.IsEmpty())
	})

	t.Run("returns false for errors with content", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "email",
			Message: "is required",
		})

		assert.False(t, errs.IsEmpty())
	})
}

func TestValidationErrors_GetTranslatableErrors(t *testing.T) {
	t.Run("returns all errors with translation data", func(t *testing.T) {
		var errs validator.ValidationErrors
		err1 := validator.ValidationError{
			Field:             "email",
			Message:           "is required",
			TranslationKey:    "validation.required",
			TranslationValues: map[string]any{"field": "email"},
		}
		err2 := validator.ValidationError{
			Field:             "password",
			Message:           "too short",
			TranslationKey:    "validation.min_length",
			TranslationValues: map[string]any{"field": "password", "min": 8},
		}
		errs.Add(err1)
		errs.Add(err2)

		result := errs.GetTranslatableErrors()
		assert.Len(t, result, 2)
		assert.Equal(t, err1, result[0])
		assert.Equal(t, err2, result[1])
	})
}

func TestApply(t *testing.T) {
	t.Run("returns nil when all rules pass", func(t *testing.T) {
		rules := []validator.Rule{
			{
				Check: func() bool { return true },
				Error: validator.ValidationError{Field: "email", Message: "required"},
			},
			{
				Check: func() bool { return true },
				Error: validator.ValidationError{Field: "password", Message: "required"},
			},
		}

		err := validator.Apply(rules...)
		assert.NoError(t, err)
	})

	t.Run("returns ValidationErrors when rules fail", func(t *testing.T) {
		rules := []validator.Rule{
			{
				Check: func() bool { return false },
				Error: validator.ValidationError{
					Field:             "email",
					Message:           "is required",
					TranslationKey:    "validation.required",
					TranslationValues: map[string]any{"field": "email"},
				},
			},
			{
				Check: func() bool { return false },
				Error: validator.ValidationError{
					Field:             "password",
					Message:           "too short",
					TranslationKey:    "validation.min_length",
					TranslationValues: map[string]any{"field": "password", "min": 8},
				},
			},
		}

		err := validator.Apply(rules...)
		require.Error(t, err)

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.True(t, validationErr.Has("email"))
		assert.True(t, validationErr.Has("password"))
	})

	t.Run("returns ValidationErrors for mixed results", func(t *testing.T) {
		rules := []validator.Rule{
			{
				Check: func() bool { return false },
				Error: validator.ValidationError{Field: "email", Message: "is required"},
			},
			{
				Check: func() bool { return true },
				Error: validator.ValidationError{Field: "password", Message: "ok"},
			},
		}

		err := validator.Apply(rules...)
		require.Error(t, err)

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.True(t, validationErr.Has("email"))
		assert.False(t, validationErr.Has("password"))
	})

	t.Run("handles empty rules", func(t *testing.T) {
		err := validator.Apply()
		assert.NoError(t, err)
	})

	t.Run("collects multiple errors for same field", func(t *testing.T) {
		rules := []validator.Rule{
			{
				Check: func() bool { return false },
				Error: validator.ValidationError{Field: "password", Message: "too short"},
			},
			{
				Check: func() bool { return false },
				Error: validator.ValidationError{Field: "password", Message: "missing special character"},
			},
		}

		err := validator.Apply(rules...)
		require.Error(t, err)

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)

		passwordErrors := validationErr.Get("password")
		assert.Len(t, passwordErrors, 2)
		assert.Contains(t, passwordErrors, "too short")
		assert.Contains(t, passwordErrors, "missing special character")
	})
}

func TestExtractValidationErrors(t *testing.T) {
	t.Run("extracts ValidationErrors from error", func(t *testing.T) {
		var originalErrs validator.ValidationErrors
		originalErrs.Add(validator.ValidationError{
			Field:   "email",
			Message: "is required",
		})

		extractedErrs := validator.ExtractValidationErrors(originalErrs)
		require.NotNil(t, extractedErrs)
		assert.True(t, extractedErrs.Has("email"))
	})

	t.Run("returns nil for non-ValidationErrors", func(t *testing.T) {
		err := errors.New("regular error")

		extractedErrs := validator.ExtractValidationErrors(err)
		assert.Nil(t, extractedErrs)
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		extractedErrs := validator.ExtractValidationErrors(nil)
		assert.Nil(t, extractedErrs)
	})
}

func TestIsValidationError(t *testing.T) {
	t.Run("returns true for ValidationErrors", func(t *testing.T) {
		var errs validator.ValidationErrors
		errs.Add(validator.ValidationError{
			Field:   "email",
			Message: "is required",
		})

		assert.True(t, validator.IsValidationError(errs))
	})

	t.Run("returns false for regular error", func(t *testing.T) {
		err := errors.New("regular error")

		assert.False(t, validator.IsValidationError(err))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, validator.IsValidationError(nil))
	})
}

func TestRule(t *testing.T) {
	t.Run("rule structure contains expected fields", func(t *testing.T) {
		rule := validator.Rule{
			Check: func() bool { return true },
			Error: validator.ValidationError{
				Field:             "email",
				Message:           "is required",
				TranslationKey:    "validation.required",
				TranslationValues: map[string]any{"field": "email"},
			},
		}

		assert.True(t, rule.Check())
		assert.Equal(t, "email", rule.Error.Field)
		assert.Equal(t, "is required", rule.Error.Message)
		assert.Equal(t, "validation.required", rule.Error.TranslationKey)
		assert.Equal(t, map[string]any{"field": "email"}, rule.Error.TranslationValues)
	})

	t.Run("rule check function can return false", func(t *testing.T) {
		rule := validator.Rule{
			Check: func() bool { return false },
			Error: validator.ValidationError{
				Field:   "password",
				Message: "too short",
			},
		}

		assert.False(t, rule.Check())
	})
}

func TestValidationError(t *testing.T) {
	t.Run("contains all expected fields", func(t *testing.T) {
		err := validator.ValidationError{
			Field:             "email",
			Message:           "is required",
			TranslationKey:    "validation.required",
			TranslationValues: map[string]any{"field": "email"},
		}

		assert.Equal(t, "email", err.Field)
		assert.Equal(t, "is required", err.Message)
		assert.Equal(t, "validation.required", err.TranslationKey)
		assert.Equal(t, map[string]any{"field": "email"}, err.TranslationValues)
	})

	t.Run("can have complex translation values", func(t *testing.T) {
		err := validator.ValidationError{
			Field:          "password",
			Message:        "must be between 8 and 50 characters",
			TranslationKey: "validation.between",
			TranslationValues: map[string]any{
				"field": "password",
				"min":   8,
				"max":   50,
			},
		}

		assert.Equal(t, "password", err.Field)
		assert.Equal(t, "must be between 8 and 50 characters", err.Message)
		assert.Equal(t, "validation.between", err.TranslationKey)
		assert.Equal(t, 8, err.TranslationValues["min"])
		assert.Equal(t, 50, err.TranslationValues["max"])
	})
}
