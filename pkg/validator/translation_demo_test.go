package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

// MockTranslator simulates a translation function
func MockTranslator(key string, values map[string]any) string {
	translations := map[string]string{
		"validation.required":     "The {{field}} field is required.",
		"validation.min_length":   "The {{field}} must be at least {{min}} characters long.",
		"validation.max_length":   "The {{field}} must not exceed {{max}} characters.",
		"validation.exact_length": "The {{field}} must be exactly {{length}} characters long.",
		"validation.min":          "The {{field}} must be at least {{min}}.",
		"validation.max":          "The {{field}} must not exceed {{max}}.",
		"validation.min_items":    "The {{field}} must contain at least {{min}} items.",
		"validation.max_items":    "The {{field}} must not contain more than {{max}} items.",
		"validation.exact_items":  "The {{field}} must contain exactly {{count}} items.",
	}

	template := translations[key]
	if template == "" {
		return key
	}

	result := template
	for placeholder, value := range values {
		token := "{{" + placeholder + "}}"
		if str, ok := value.(string); ok {
			result = replaceAll(result, token, str)
		} else {
			result = replaceAll(result, token, toString(value))
		}
	}
	return result
}

func replaceAll(s, old, new string) string {
	for {
		newS := ""
		for i := 0; i < len(s); i++ {
			if i <= len(s)-len(old) && s[i:i+len(old)] == old {
				newS += new
				i += len(old) - 1
			} else {
				newS += string(s[i])
			}
		}
		if newS == s {
			break
		}
		s = newS
	}
	return s
}

func toString(v any) string {
	switch val := v.(type) {
	case int:
		return intToString(val)
	case string:
		return val
	default:
		return "unknown"
	}
}

func intToString(i int) string {
	if i == 0 {
		return "0"
	}

	negative := i < 0
	if negative {
		i = -i
	}

	digits := ""
	for i > 0 {
		digit := i % 10
		digits = string(rune('0'+digit)) + digits
		i /= 10
	}

	if negative {
		digits = "-" + digits
	}

	return digits
}

func TestTranslationWorkflow(t *testing.T) {
	t.Run("demonstrates basic translation workflow", func(t *testing.T) {
		type LoginForm struct {
			Email    string
			Password string
		}

		form := LoginForm{
			Email:    "",
			Password: "123",
		}

		err := validator.Apply(
			validator.RequiredString("email", form.Email),
			validator.RequiredString("password", form.Password),
			validator.MinLenString("password", form.Password, 8),
		)

		require.Error(t, err)
		require.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)
		translatableErrors := validationErr.GetTranslatableErrors()

		translatedMessages := make(map[string][]string)
		for _, errInfo := range translatableErrors {
			translatedMsg := MockTranslator(errInfo.TranslationKey, errInfo.TranslationValues)
			translatedMessages[errInfo.Field] = append(translatedMessages[errInfo.Field], translatedMsg)
		}

		expectedTranslations := map[string][]string{
			"email":    {"The email field is required."},
			"password": {"The password must be at least 8 characters long."},
		}

		assert.Equal(t, expectedTranslations, translatedMessages)
	})

	t.Run("demonstrates complex validation with multiple translation values", func(t *testing.T) {
		username := "ab"
		tags := []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6"}
		age := 15

		err := validator.Apply(
			validator.MinLenString("username", username, 3),
			validator.MaxLenString("username", username, 20),
			validator.MaxLenSlice("tags", tags, 5),
			validator.MinNum("age", age, 18),
		)

		require.Error(t, err)
		validationErr := validator.ExtractValidationErrors(err)

		translatedErrors := make(map[string]string)
		for _, errInfo := range validationErr.GetTranslatableErrors() {
			translatedMsg := MockTranslator(errInfo.TranslationKey, errInfo.TranslationValues)
			translatedErrors[errInfo.Field+"_"+errInfo.TranslationKey] = translatedMsg
		}

		expectedTranslations := map[string]string{
			"username_validation.min_length": "The username must be at least 3 characters long.",
			"tags_validation.max_items":      "The tags must not contain more than 5 items.",
			"age_validation.min":             "The age must be at least 18.",
		}

		for key, expected := range expectedTranslations {
			assert.Equal(t, expected, translatedErrors[key])
		}
	})

	t.Run("demonstrates field-specific translation data extraction", func(t *testing.T) {
		email := ""
		password := "weak"

		err := validator.Apply(
			validator.RequiredString("email", email),
			validator.MinLenString("password", password, 8),
			validator.MaxLenString("password", password, 128),
		)

		require.Error(t, err)
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

	t.Run("demonstrates translation key consistency across rule types", func(t *testing.T) {
		stringField := ""
		sliceField := []string{}
		mapField := map[string]string{}
		numField := 0

		err := validator.Apply(
			validator.RequiredString("stringField", stringField),
			validator.RequiredSlice("sliceField", sliceField),
			validator.RequiredMap("mapField", mapField),
			validator.RequiredNum("numField", numField),
		)

		require.Error(t, err)
		validationErr := validator.ExtractValidationErrors(err)

		allErrors := validationErr.GetTranslatableErrors()
		for _, errInfo := range allErrors {
			assert.Equal(t, "validation.required", errInfo.TranslationKey)
			assert.Contains(t, errInfo.TranslationValues, "field")

			translatedMsg := MockTranslator(errInfo.TranslationKey, errInfo.TranslationValues)
			assert.Contains(t, translatedMsg, "field is required")
		}
	})
}

func TestTranslationKeyStandards(t *testing.T) {
	t.Run("validates standard translation keys", func(t *testing.T) {
		tests := []struct {
			rule           validator.Rule
			expectedKey    string
			expectedValues map[string]any
		}{
			{
				rule:        validator.RequiredString("email", ""),
				expectedKey: "validation.required",
				expectedValues: map[string]any{
					"field": "email",
				},
			},
			{
				rule:        validator.MinLenString("password", "123", 8),
				expectedKey: "validation.min_length",
				expectedValues: map[string]any{
					"field": "password",
					"min":   8,
				},
			},
			{
				rule:        validator.MaxLenString("username", "verylongusername", 10),
				expectedKey: "validation.max_length",
				expectedValues: map[string]any{
					"field": "username",
					"max":   10,
				},
			},
			{
				rule:        validator.LenString("code", "1234", 6),
				expectedKey: "validation.exact_length",
				expectedValues: map[string]any{
					"field":  "code",
					"length": 6,
				},
			},
			{
				rule:        validator.MinNum("age", 15, 18),
				expectedKey: "validation.min",
				expectedValues: map[string]any{
					"field": "age",
					"min":   18,
				},
			},
			{
				rule:        validator.MaxNum("score", 105, 100),
				expectedKey: "validation.max",
				expectedValues: map[string]any{
					"field": "score",
					"max":   100,
				},
			},
		}

		for _, test := range tests {
			assert.Equal(t, test.expectedKey, test.rule.Error.TranslationKey)
			assert.Equal(t, test.expectedValues, test.rule.Error.TranslationValues)
		}
	})
}
