package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestDefaultPasswordStrength(t *testing.T) {
	t.Parallel()
	config := validator.DefaultPasswordStrength()

	assert.Equal(t, 8, config.MinLength)
	assert.Equal(t, 128, config.MaxLength)
	assert.True(t, config.RequireUppercase)
	assert.True(t, config.RequireLowercase)
	assert.True(t, config.RequireDigits)
	assert.True(t, config.RequireSpecial)
	assert.Equal(t, 3, config.MinCharClasses)
}

func TestStrongPassword(t *testing.T) {
	t.Parallel()
	config := validator.DefaultPasswordStrength()

	t.Run("valid strong passwords", func(t *testing.T) {
		validPasswords := []string{
			"StrongP@ss123",
			"MySecure#Pass1",
			"C0mplex!Password",
			"Test1234!@#$",
			"aB3$defghijklmnop", // meets all requirements
		}

		for _, password := range validPasswords {
			rule := validator.StrongPassword("password", password, config)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should be strong: %s", password)
		}
	})

	t.Run("passwords too short", func(t *testing.T) {
		shortPasswords := []string{
			"Test1!",
			"Ab1@",
			"",
		}

		for _, password := range shortPasswords {
			rule := validator.StrongPassword("password", password, config)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected as too short: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_strength", validationErr[0].TranslationKey)
		}
	})

	t.Run("passwords missing uppercase", func(t *testing.T) {
		passwords := []string{
			"lowercase123!",
			"nouppercasehere1@",
		}

		for _, password := range passwords {
			rule := validator.StrongPassword("password", password, config)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for missing uppercase: %s", password)
		}
	})

	t.Run("passwords missing lowercase", func(t *testing.T) {
		passwords := []string{
			"UPPERCASE123!",
			"NOLOWERCASEHERE1@",
		}

		for _, password := range passwords {
			rule := validator.StrongPassword("password", password, config)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for missing lowercase: %s", password)
		}
	})

	t.Run("passwords missing digits", func(t *testing.T) {
		passwords := []string{
			"NoDigitsHere!@#",
			"Password!@#$%",
		}

		for _, password := range passwords {
			rule := validator.StrongPassword("password", password, config)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for missing digits: %s", password)
		}
	})

	t.Run("passwords missing special characters", func(t *testing.T) {
		passwords := []string{
			"NoSpecialChars123",
			"Password123456",
		}

		for _, password := range passwords {
			rule := validator.StrongPassword("password", password, config)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for missing special chars: %s", password)
		}
	})

	t.Run("custom configuration", func(t *testing.T) {
		customConfig := validator.PasswordStrengthConfig{
			MinLength:        6,
			MaxLength:        20,
			RequireUppercase: false,
			RequireLowercase: true,
			RequireDigits:    true,
			RequireSpecial:   false,
			MinCharClasses:   2,
		}

		validPasswords := []string{
			"simple123",
			"test456",
		}

		for _, password := range validPasswords {
			rule := validator.StrongPassword("password", password, customConfig)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should be valid with custom config: %s", password)
		}
	})
}

func TestPasswordUppercase(t *testing.T) {
	t.Parallel()
	t.Run("passwords with uppercase", func(t *testing.T) {
		validPasswords := []string{
			"Password",
			"Test123",
			"UPPERCASE",
			"mixedCASE",
			"A",
		}

		for _, password := range validPasswords {
			rule := validator.PasswordUppercase("password", password)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should have uppercase: %s", password)
		}
	})

	t.Run("passwords without uppercase", func(t *testing.T) {
		invalidPasswords := []string{
			"lowercase",
			"123456",
			"test!@#",
			"",
		}

		for _, password := range invalidPasswords {
			rule := validator.PasswordUppercase("password", password)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for no uppercase: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_uppercase", validationErr[0].TranslationKey)
		}
	})
}

func TestPasswordLowercase(t *testing.T) {
	t.Parallel()
	t.Run("passwords with lowercase", func(t *testing.T) {
		validPasswords := []string{
			"password",
			"Test123",
			"lowercase",
			"MIXEDcase",
			"a",
		}

		for _, password := range validPasswords {
			rule := validator.PasswordLowercase("password", password)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should have lowercase: %s", password)
		}
	})

	t.Run("passwords without lowercase", func(t *testing.T) {
		invalidPasswords := []string{
			"UPPERCASE",
			"123456",
			"TEST!@#",
			"",
		}

		for _, password := range invalidPasswords {
			rule := validator.PasswordLowercase("password", password)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for no lowercase: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_lowercase", validationErr[0].TranslationKey)
		}
	})
}

func TestPasswordDigit(t *testing.T) {
	t.Parallel()
	t.Run("passwords with digits", func(t *testing.T) {
		validPasswords := []string{
			"password123",
			"Test1",
			"123456",
			"a1b2c3",
			"0",
		}

		for _, password := range validPasswords {
			rule := validator.PasswordDigit("password", password)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should have digits: %s", password)
		}
	})

	t.Run("passwords without digits", func(t *testing.T) {
		invalidPasswords := []string{
			"password",
			"UPPERCASE",
			"test!@#",
			"",
		}

		for _, password := range invalidPasswords {
			rule := validator.PasswordDigit("password", password)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for no digits: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_digit", validationErr[0].TranslationKey)
		}
	})
}

func TestPasswordSpecialChar(t *testing.T) {
	t.Parallel()
	t.Run("passwords with special characters", func(t *testing.T) {
		validPasswords := []string{
			"password!",
			"test@123",
			"#$%^&*()",
			"a-b_c+d",
			"test.email@domain.com",
		}

		for _, password := range validPasswords {
			rule := validator.PasswordSpecialChar("password", password)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should have special chars: %s", password)
		}
	})

	t.Run("passwords without special characters", func(t *testing.T) {
		invalidPasswords := []string{
			"password",
			"Test123",
			"UPPERCASE",
			"123456",
			"",
		}

		for _, password := range invalidPasswords {
			rule := validator.PasswordSpecialChar("password", password)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for no special chars: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_special", validationErr[0].TranslationKey)
		}
	})
}

func TestNotCommonPassword(t *testing.T) {
	t.Parallel()
	t.Run("non-common passwords", func(t *testing.T) {
		validPasswords := []string{
			"UniquePassword123!",
			"MySecretP@ss",
			"NotInCommonList456",
			"CustomPassword#1",
		}

		for _, password := range validPasswords {
			rule := validator.NotCommonPassword("password", password)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should not be common: %s", password)
		}
	})

	t.Run("common passwords", func(t *testing.T) {
		commonPasswords := []string{
			"password",
			"123456",
			"password123",
			"admin",
			"qwerty",
			"Password", // case insensitive check
			"PASSWORD",
		}

		for _, password := range commonPasswords {
			rule := validator.NotCommonPassword("password", password)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected as common: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_common", validationErr[0].TranslationKey)
		}
	})
}

func TestPasswordEntropy(t *testing.T) {
	t.Parallel()
	t.Run("high entropy passwords", func(t *testing.T) {
		highEntropyPasswords := []string{
			"Tr0ub4dor&3",        // Classic example
			"RandomP@ssw0rd123!", // Good mix of character types
			"C0mplex!Pa$$w0rd",   // Complex password
		}

		for _, password := range highEntropyPasswords {
			rule := validator.PasswordEntropy("password", password, 30.0) // 30 bits minimum
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should have sufficient entropy: %s", password)
		}
	})

	t.Run("low entropy passwords", func(t *testing.T) {
		lowEntropyPasswords := []string{
			"abc",
			"123",
			"aaa",
			"password", // Dictionary word
		}

		for _, password := range lowEntropyPasswords {
			rule := validator.PasswordEntropy("password", password, 50.0) // High threshold
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for low entropy: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_entropy", validationErr[0].TranslationKey)
		}
	})
}

func TestNoRepeatingChars(t *testing.T) {
	t.Parallel()
	t.Run("passwords without excessive repeating characters", func(t *testing.T) {
		validPasswords := []string{
			"password123",
			"Test1234",
			"abcdefgh",
			"Pa$$word", // Two same chars but not consecutive
		}

		for _, password := range validPasswords {
			rule := validator.NoRepeatingChars("password", password, 3)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should not have excessive repeating chars: %s", password)
		}
	})

	t.Run("passwords with excessive repeating characters", func(t *testing.T) {
		invalidPasswords := []string{
			"aaaa1234",     // 4 'a's in a row
			"password1111", // 4 '1's in a row
			"testtttt",     // 5 't's in a row
		}

		for _, password := range invalidPasswords {
			rule := validator.NoRepeatingChars("password", password, 3)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for repeating chars: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_repeating", validationErr[0].TranslationKey)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		rule := validator.NoRepeatingChars("password", "", 3)
		err := validator.Apply(rule)
		assert.NoError(t, err, "Empty password should pass repeating chars test")
	})
}

func TestNoSequentialChars(t *testing.T) {
	t.Parallel()
	t.Run("passwords without excessive sequential characters", func(t *testing.T) {
		validPasswords := []string{
			"password123",
			"Test1357", // Non-sequential numbers
			"abcZYX",   // Mixed case breaks sequence
			"Pa$$word",
		}

		for _, password := range validPasswords {
			rule := validator.NoSequentialChars("password", password, 4)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Password should not have excessive sequential chars: %s", password)
		}
	})

	t.Run("passwords with excessive sequential characters", func(t *testing.T) {
		invalidPasswords := []string{
			"abcdefgh",      // Sequential letters
			"12345678",      // Sequential numbers
			"password12345", // Sequential at end
		}

		for _, password := range invalidPasswords {
			rule := validator.NoSequentialChars("password", password, 4)
			err := validator.Apply(rule)
			assert.Error(t, err, "Password should be rejected for sequential chars: %s", password)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.password_sequential", validationErr[0].TranslationKey)
		}
	})

	t.Run("short passwords", func(t *testing.T) {
		shortPasswords := []string{
			"abc",
			"123",
			"ab",
			"",
		}

		for _, password := range shortPasswords {
			rule := validator.NoSequentialChars("password", password, 4)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Short password should pass sequential test: %s", password)
		}
	})
}

func TestPasswordValidationCombination(t *testing.T) {
	t.Parallel()
	t.Run("comprehensive password validation", func(t *testing.T) {
		password := "MySecur3P@ssw0rd!"

		err := validator.Apply(
			validator.RequiredString("password", password),
			validator.MinLenString("password", password, 8),
			validator.PasswordUppercase("password", password),
			validator.PasswordLowercase("password", password),
			validator.PasswordDigit("password", password),
			validator.PasswordSpecialChar("password", password),
			validator.NotCommonPassword("password", password),
			validator.NoRepeatingChars("password", password, 3),
			validator.NoSequentialChars("password", password, 4),
		)

		assert.NoError(t, err, "Strong password should pass all validations")
	})

	t.Run("weak password fails multiple validations", func(t *testing.T) {
		password := "password"

		err := validator.Apply(
			validator.RequiredString("password", password),
			validator.MinLenString("password", password, 8),
			validator.PasswordUppercase("password", password),
			validator.PasswordDigit("password", password),
			validator.PasswordSpecialChar("password", password),
			validator.NotCommonPassword("password", password),
		)

		assert.Error(t, err, "Weak password should fail validations")

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.True(t, len(validationErr) > 1, "Should have multiple validation errors")
	})
}
