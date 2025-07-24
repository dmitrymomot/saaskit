package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestLoginFormValidation(t *testing.T) {
	t.Parallel()
	type LoginForm struct {
		Email    string
		Password string
	}

	t.Run("validates successful login form", func(t *testing.T) {
		form := LoginForm{
			Email:    "user@example.com",
			Password: "securepassword123",
		}

		err := validator.Apply(
			validator.RequiredString("email", form.Email),
			validator.RequiredString("password", form.Password),
			validator.MinLenString("password", form.Password, 8),
			validator.MaxLenString("password", form.Password, 128),
		)

		assert.NoError(t, err)
	})

	t.Run("collects all login form validation errors", func(t *testing.T) {
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
		assert.True(t, validationErr.Has("email"))
		assert.True(t, validationErr.Has("password"))

		emailErrors := validationErr.Get("email")
		assert.Contains(t, emailErrors, "field is required")

		passwordErrors := validationErr.Get("password")
		assert.Contains(t, passwordErrors, "must be at least 8 characters long")
	})
}

func TestUserRegistrationValidation(t *testing.T) {
	t.Parallel()
	type UserRegistration struct {
		Email    string
		Password string
		Age      int
		Username string
		Bio      string
		Tags     []string
		Settings map[string]string
		Score    float64
		IsActive bool
		UserID   uint64
	}

	t.Run("validates successful user registration", func(t *testing.T) {
		reg := UserRegistration{
			Email:    "newuser@example.com",
			Password: "verysecurepassword123",
			Age:      25,
			Username: "johndoe",
			Bio:      "Software engineer passionate about Go",
			Tags:     []string{"go", "backend", "api"},
			Settings: map[string]string{"theme": "dark", "lang": "en"},
			Score:    85.5,
			IsActive: true,
			UserID:   12345,
		}

		err := validator.Apply(
			// String validations
			validator.RequiredString("email", reg.Email),
			validator.RequiredString("password", reg.Password),
			validator.RequiredString("username", reg.Username),
			validator.MinLenString("password", reg.Password, 8),
			validator.MaxLenString("password", reg.Password, 128),
			validator.MinLenString("username", reg.Username, 3),
			validator.MaxLenString("username", reg.Username, 30),
			validator.MaxLenString("bio", reg.Bio, 500),

			// Numeric validations
			validator.RequiredNum("age", reg.Age),
			validator.MinNum("age", reg.Age, 13),
			validator.MaxNum("age", reg.Age, 120),
			validator.RequiredNum("score", reg.Score),
			validator.MinNum("score", reg.Score, 0.0),
			validator.MaxNum("score", reg.Score, 100.0),
			validator.RequiredNum("userID", reg.UserID),

			// Collection validations
			validator.RequiredSlice("tags", reg.Tags),
			validator.MinLenSlice("tags", reg.Tags, 1),
			validator.MaxLenSlice("tags", reg.Tags, 10),
			validator.RequiredMap("settings", reg.Settings),
			validator.MinLenMap("settings", reg.Settings, 1),
			validator.MaxLenMap("settings", reg.Settings, 20),

			// Comparable validations
			validator.RequiredComparable("isActive", reg.IsActive),
		)

		assert.NoError(t, err)
	})

	t.Run("collects comprehensive validation errors", func(t *testing.T) {
		reg := UserRegistration{
			Email:    "",                  // Required but empty
			Password: "123",               // Too short
			Age:      0,                   // Required but zero
			Username: "ab",                // Too short
			Tags:     []string{},          // Required but empty
			Settings: map[string]string{}, // Required but empty
			Score:    150.0,               // Too high
			IsActive: false,               // Required but false
			UserID:   0,                   // Required but zero
		}

		err := validator.Apply(
			validator.RequiredString("email", reg.Email),
			validator.RequiredString("password", reg.Password),
			validator.RequiredString("username", reg.Username),
			validator.MinLenString("password", reg.Password, 8),
			validator.MinLenString("username", reg.Username, 3),
			validator.RequiredNum("age", reg.Age),
			validator.MinNum("age", reg.Age, 13),
			validator.RequiredNum("userID", reg.UserID),
			validator.MaxNum("score", reg.Score, 100.0),
			validator.RequiredSlice("tags", reg.Tags),
			validator.RequiredMap("settings", reg.Settings),
			validator.RequiredComparable("isActive", reg.IsActive),
		)

		require.Error(t, err)
		require.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)

		// Check that all expected fields have errors
		expectedFields := []string{"email", "password", "username", "age", "userID", "score", "tags", "settings", "isActive"}
		for _, field := range expectedFields {
			assert.True(t, validationErr.Has(field), "Expected field %s to have validation errors", field)
		}

		// Verify specific error messages
		emailErrors := validationErr.Get("email")
		assert.Contains(t, emailErrors, "field is required")

		passwordErrors := validationErr.Get("password")
		assert.Contains(t, passwordErrors, "must be at least 8 characters long")

		usernameErrors := validationErr.Get("username")
		assert.Contains(t, usernameErrors, "must be at least 3 characters long")

		scoreErrors := validationErr.Get("score")
		assert.Contains(t, scoreErrors, "must be at most 100")
	})
}

func TestAPIConfigValidation(t *testing.T) {
	t.Parallel()
	type APIConfig struct {
		Endpoints   []string
		Timeouts    map[string]int
		MaxRetries  int
		DefaultPort uint16
		EnableSSL   bool
		Version     string
		RateLimits  []float64
	}

	t.Run("validates complete API configuration", func(t *testing.T) {
		config := APIConfig{
			Endpoints:   []string{"/api/v1/users", "/api/v1/orders"},
			Timeouts:    map[string]int{"read": 30, "write": 60, "connect": 10},
			MaxRetries:  3,
			DefaultPort: 8080,
			EnableSSL:   true,
			Version:     "1.2.3",
			RateLimits:  []float64{100.0, 50.0, 25.0},
		}

		err := validator.Apply(
			validator.RequiredSlice("endpoints", config.Endpoints),
			validator.MinLenSlice("endpoints", config.Endpoints, 1),
			validator.MaxLenSlice("endpoints", config.Endpoints, 50),
			validator.RequiredMap("timeouts", config.Timeouts),
			validator.MinLenMap("timeouts", config.Timeouts, 1),
			validator.RequiredNum("maxRetries", config.MaxRetries),
			validator.MinNum("maxRetries", config.MaxRetries, 1),
			validator.MaxNum("maxRetries", config.MaxRetries, 10),
			validator.RequiredNum("defaultPort", config.DefaultPort),
			validator.MinNum("defaultPort", config.DefaultPort, uint16(1)),
			validator.MaxNum("defaultPort", config.DefaultPort, uint16(65535)),
			validator.RequiredComparable("enableSSL", config.EnableSSL),
			validator.RequiredString("version", config.Version),
			validator.MinLenString("version", config.Version, 1),
			validator.RequiredSlice("rateLimits", config.RateLimits),
			validator.LenSlice("rateLimits", config.RateLimits, 3),
		)

		assert.NoError(t, err)
	})

	t.Run("validates with mixed success and failures", func(t *testing.T) {
		config := APIConfig{
			Endpoints:   []string{"/api/v1/users"}, // Valid
			Timeouts:    map[string]int{},          // Invalid - empty
			MaxRetries:  15,                        // Invalid - too high
			DefaultPort: 0,                         // Invalid - zero
			EnableSSL:   true,                      // Valid
			Version:     "",                        // Invalid - empty
			RateLimits:  []float64{100.0, 50.0},    // Invalid - wrong length
		}

		err := validator.Apply(
			validator.RequiredSlice("endpoints", config.Endpoints),
			validator.RequiredMap("timeouts", config.Timeouts),
			validator.RequiredNum("maxRetries", config.MaxRetries),
			validator.MaxNum("maxRetries", config.MaxRetries, 10),
			validator.RequiredNum("defaultPort", config.DefaultPort),
			validator.RequiredComparable("enableSSL", config.EnableSSL),
			validator.RequiredString("version", config.Version),
			validator.LenSlice("rateLimits", config.RateLimits, 3),
		)

		require.Error(t, err)
		require.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)

		// Should have errors for: timeouts, maxRetries, defaultPort, version, rateLimits
		assert.True(t, validationErr.Has("timeouts"))
		assert.True(t, validationErr.Has("maxRetries"))
		assert.True(t, validationErr.Has("defaultPort"))
		assert.True(t, validationErr.Has("version"))
		assert.True(t, validationErr.Has("rateLimits"))

		// Should NOT have errors for: endpoints, enableSSL
		assert.False(t, validationErr.Has("endpoints"))
		assert.False(t, validationErr.Has("enableSSL"))
	})
}

func TestComplexNestedValidation(t *testing.T) {
	t.Parallel()
	type UserPreferences struct {
		Theme    string
		Language string
		Settings map[string]any
	}

	type ComplexUser struct {
		ID          uint64
		Email       string
		Preferences UserPreferences
		Scores      []float64
		Metadata    map[string]string
		IsVerified  bool
		Age         int8
		Balance     float32
	}

	t.Run("validates complex nested structure", func(t *testing.T) {
		user := ComplexUser{
			ID:    12345,
			Email: "complex@example.com",
			Preferences: UserPreferences{
				Theme:    "dark",
				Language: "en",
				Settings: map[string]any{"notifications": true, "autoSave": false},
			},
			Scores:     []float64{85.5, 92.3, 78.9, 95.1},
			Metadata:   map[string]string{"department": "engineering", "level": "senior"},
			IsVerified: true,
			Age:        28,
			Balance:    1250.75,
		}

		err := validator.Apply(
			// Top-level validations
			validator.RequiredNum("id", user.ID),
			validator.RequiredString("email", user.Email),
			validator.RequiredComparable("isVerified", user.IsVerified),
			validator.RequiredNum("age", user.Age),
			validator.MinNum("age", user.Age, int8(18)),
			validator.MaxNum("age", user.Age, int8(100)),
			validator.RequiredNum("balance", user.Balance),
			validator.MinNum("balance", user.Balance, float32(0.0)),

			// Nested structure validations
			validator.RequiredString("theme", user.Preferences.Theme),
			validator.RequiredString("language", user.Preferences.Language),
			validator.RequiredSlice("scores", user.Scores),
			validator.MinLenSlice("scores", user.Scores, 1),
			validator.MaxLenSlice("scores", user.Scores, 10),
			validator.RequiredMap("metadata", user.Metadata),
			validator.MinLenMap("metadata", user.Metadata, 1),
		)

		assert.NoError(t, err)
	})

	t.Run("collects validation errors from nested structure", func(t *testing.T) {
		user := ComplexUser{
			ID:    0,  // Invalid - zero
			Email: "", // Invalid - empty
			Preferences: UserPreferences{
				Theme:    "",                       // Invalid - empty
				Language: "toolongforlanguagecode", // Valid but we'll add max length check
				Settings: map[string]any{},         // Invalid - empty
			},
			Scores:     []float64{},         // Invalid - empty
			Metadata:   map[string]string{}, // Invalid - empty
			IsVerified: false,               // Invalid - false
			Age:        127,                 // Invalid - too high
			Balance:    -100.0,              // Invalid - negative
		}

		err := validator.Apply(
			validator.RequiredNum("id", user.ID),
			validator.RequiredString("email", user.Email),
			validator.RequiredString("theme", user.Preferences.Theme),
			validator.RequiredString("language", user.Preferences.Language),
			validator.MaxLenString("language", user.Preferences.Language, 5),
			validator.RequiredSlice("scores", user.Scores),
			validator.RequiredMap("metadata", user.Metadata),
			validator.RequiredComparable("isVerified", user.IsVerified),
			validator.RequiredNum("age", user.Age),
			validator.MaxNum("age", user.Age, int8(100)),
			validator.RequiredNum("balance", user.Balance),
			validator.MinNum("balance", user.Balance, float32(0.0)),
		)

		require.Error(t, err)
		require.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)

		// Verify all expected fields have errors
		expectedErrorFields := []string{
			"id", "email", "theme", "language", "scores",
			"metadata", "isVerified", "age", "balance",
		}

		for _, field := range expectedErrorFields {
			assert.True(t, validationErr.Has(field), "Expected field %s to have errors", field)
		}

		// Verify that language has multiple errors (required + too long)
		languageErrors := validationErr.Get("language")
		assert.Len(t, languageErrors, 1) // Only max length error since it's not empty
		assert.Contains(t, languageErrors, "must be at most 5 characters long")
	})
}

func TestMixedTypeConvenienceAliases(t *testing.T) {
	t.Parallel()
	t.Run("convenience aliases work together", func(t *testing.T) {
		email := "user@example.com"
		password := "securepassword123"
		age := 25
		score := 85.5

		err := validator.Apply(
			validator.Required("email", email),         // String alias
			validator.Required("password", password),   // String alias
			validator.MinLen("password", password, 8),  // String alias
			validator.MaxLen("password", password, 50), // String alias
			validator.Min("age", age, 18),              // Numeric alias
			validator.Max("age", age, 120),             // Numeric alias
			validator.Min("score", score, 0.0),         // Numeric alias
			validator.Max("score", score, 100.0),       // Numeric alias
		)

		assert.NoError(t, err)
	})

	t.Run("convenience aliases collect errors properly", func(t *testing.T) {
		email := ""
		password := "123"
		age := 15
		score := 150.0

		err := validator.Apply(
			validator.Required("email", email),
			validator.Required("password", password),
			validator.MinLen("password", password, 8),
			validator.Min("age", age, 18),
			validator.Max("score", score, 100.0),
		)

		require.Error(t, err)
		require.True(t, validator.IsValidationError(err))

		validationErr := validator.ExtractValidationErrors(err)
		assert.True(t, validationErr.Has("email"))
		assert.True(t, validationErr.Has("password"))
		assert.True(t, validationErr.Has("age"))
		assert.True(t, validationErr.Has("score"))
	})
}
