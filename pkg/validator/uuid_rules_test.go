package validator_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestValidUUID(t *testing.T) {
	t.Parallel()
	t.Run("valid UUIDs", func(t *testing.T) {
		validUUIDs := []string{
			"550e8400-e29b-41d4-a716-446655440000",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"6ba7b811-9dad-11d1-80b4-00c04fd430c8",
			"00000000-0000-0000-0000-000000000000", // nil UUID but valid format
			"f47ac10b-58cc-4372-a567-0e02b2c3d479",
		}

		for _, uuidStr := range validUUIDs {
			rule := validator.ValidUUID("uuid", uuidStr)
			err := validator.Apply(rule)
			assert.NoError(t, err, "UUID should be valid: %s", uuidStr)
		}
	})

	t.Run("invalid UUIDs", func(t *testing.T) {
		invalidUUIDs := []string{
			"",
			"   ",
			"not-a-uuid",
			"550e8400-e29b-41d4-a716-44665544000",   // too short
			"550e8400-e29b-41d4-a716-4466554400000", // too long
			"550e8400-e29b-41d4-a716-44665544000g",  // invalid character
			"550e8400e29b41d4a716446655440000",      // missing hyphens
			"550e8400-e29b-41d4-a716",               // incomplete
		}

		for _, uuidStr := range invalidUUIDs {
			rule := validator.ValidUUID("uuid", uuidStr)
			err := validator.Apply(rule)
			assert.Error(t, err, "UUID should be invalid: %s", uuidStr)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.uuid", validationErr[0].TranslationKey)
		}
	})
}

func TestNonNilUUID(t *testing.T) {
	t.Parallel()
	t.Run("non-nil UUIDs", func(t *testing.T) {
		nonNilUUIDs := []uuid.UUID{
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"),
			uuid.New(), // random UUID
		}

		for _, uuidVal := range nonNilUUIDs {
			rule := validator.NonNilUUID("uuid", uuidVal)
			err := validator.Apply(rule)
			assert.NoError(t, err, "UUID should be non-nil: %s", uuidVal.String())
		}
	})

	t.Run("nil UUID", func(t *testing.T) {
		rule := validator.NonNilUUID("uuid", uuid.Nil)
		err := validator.Apply(rule)
		assert.Error(t, err, "UUID should be rejected as nil")

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.Equal(t, "validation.uuid_not_nil", validationErr[0].TranslationKey)
	})
}

func TestNonNilUUIDString(t *testing.T) {
	t.Parallel()
	t.Run("non-nil UUID strings", func(t *testing.T) {
		nonNilUUIDs := []string{
			"550e8400-e29b-41d4-a716-446655440000",
			"f47ac10b-58cc-4372-a567-0e02b2c3d479",
			uuid.New().String(),
		}

		for _, uuidStr := range nonNilUUIDs {
			rule := validator.NonNilUUIDString("uuid", uuidStr)
			err := validator.Apply(rule)
			assert.NoError(t, err, "UUID string should be non-nil: %s", uuidStr)
		}
	})

	t.Run("nil and invalid UUID strings", func(t *testing.T) {
		invalidUUIDs := []string{
			"",
			"   ",
			"00000000-0000-0000-0000-000000000000", // nil UUID
			"not-a-uuid",
			"550e8400-e29b-41d4-a716-44665544000g", // invalid format
		}

		for _, uuidStr := range invalidUUIDs {
			rule := validator.NonNilUUIDString("uuid", uuidStr)
			err := validator.Apply(rule)
			assert.Error(t, err, "UUID string should be invalid: %s", uuidStr)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.uuid_not_nil", validationErr[0].TranslationKey)
		}
	})
}

func TestValidUUIDVersion(t *testing.T) {
	t.Parallel()
	t.Run("valid UUID version 4", func(t *testing.T) {
		// Generate a few v4 UUIDs
		for range 5 {
			uuidV4 := uuid.New() // uuid.New() generates v4
			rule := validator.ValidUUIDVersion("uuid", uuidV4, 4)
			err := validator.Apply(rule)
			assert.NoError(t, err, "UUID v4 should be valid: %s", uuidV4.String())
		}
	})

	t.Run("invalid UUID version", func(t *testing.T) {
		uuidV4 := uuid.New()                                  // This is v4
		rule := validator.ValidUUIDVersion("uuid", uuidV4, 1) // Expecting v1
		err := validator.Apply(rule)
		assert.Error(t, err, "UUID should be rejected for wrong version")

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.Equal(t, "validation.uuid_version", validationErr[0].TranslationKey)
		assert.Equal(t, 1, validationErr[0].TranslationValues["version"])
	})
}

func TestValidUUIDVersionString(t *testing.T) {
	t.Parallel()
	t.Run("valid UUID version string", func(t *testing.T) {
		uuidV4Str := uuid.New().String()
		rule := validator.ValidUUIDVersionString("uuid", uuidV4Str, 4)
		err := validator.Apply(rule)
		assert.NoError(t, err, "UUID v4 string should be valid: %s", uuidV4Str)
	})

	t.Run("invalid UUID version string", func(t *testing.T) {
		testCases := []struct {
			name    string
			uuid    string
			version int
		}{
			{"wrong version", uuid.New().String(), 1},
			{"invalid format", "not-a-uuid", 4},
			{"empty string", "", 4},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := validator.ValidUUIDVersionString("uuid", tc.uuid, tc.version)
				err := validator.Apply(rule)
				assert.Error(t, err, "UUID should be invalid: %s", tc.uuid)

				validationErr := validator.ExtractValidationErrors(err)
				require.NotNil(t, validationErr)
				assert.Equal(t, "validation.uuid_version", validationErr[0].TranslationKey)
			})
		}
	})
}

func TestValidUUIDv1(t *testing.T) {
	t.Parallel()
	t.Run("UUID v1 validation", func(t *testing.T) {
		// Create a v1 UUID
		uuidV1, err := uuid.NewUUID() // This generates v1
		require.NoError(t, err)

		rule := validator.ValidUUIDv1("uuid", uuidV1)
		err = validator.Apply(rule)
		assert.NoError(t, err, "UUID v1 should be valid")
	})

	t.Run("non-v1 UUID rejection", func(t *testing.T) {
		uuidV4 := uuid.New() // This is v4
		rule := validator.ValidUUIDv1("uuid", uuidV4)
		err := validator.Apply(rule)
		assert.Error(t, err, "UUID v4 should be rejected for v1 validation")
	})
}

func TestValidUUIDv1String(t *testing.T) {
	t.Parallel()
	t.Run("UUID v1 string validation", func(t *testing.T) {
		uuidV1, err := uuid.NewUUID()
		require.NoError(t, err)

		rule := validator.ValidUUIDv1String("uuid", uuidV1.String())
		err = validator.Apply(rule)
		assert.NoError(t, err, "UUID v1 string should be valid")
	})
}

func TestValidUUIDv4(t *testing.T) {
	t.Parallel()
	t.Run("UUID v4 validation", func(t *testing.T) {
		uuidV4 := uuid.New()
		rule := validator.ValidUUIDv4("uuid", uuidV4)
		err := validator.Apply(rule)
		assert.NoError(t, err, "UUID v4 should be valid")
	})

	t.Run("non-v4 UUID rejection", func(t *testing.T) {
		uuidV1, err := uuid.NewUUID()
		require.NoError(t, err)

		rule := validator.ValidUUIDv4("uuid", uuidV1)
		err = validator.Apply(rule)
		assert.Error(t, err, "UUID v1 should be rejected for v4 validation")
	})
}

func TestValidUUIDv4String(t *testing.T) {
	t.Parallel()
	t.Run("UUID v4 string validation", func(t *testing.T) {
		uuidV4Str := uuid.New().String()
		rule := validator.ValidUUIDv4String("uuid", uuidV4Str)
		err := validator.Apply(rule)
		assert.NoError(t, err, "UUID v4 string should be valid")
	})
}

func TestValidUUIDv3(t *testing.T) {
	t.Parallel()
	t.Run("UUID v3 validation", func(t *testing.T) {
		// Create a v3 UUID
		namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
		uuidV3 := uuid.NewMD5(namespace, []byte("test"))

		rule := validator.ValidUUIDv3("uuid", uuidV3)
		err := validator.Apply(rule)
		assert.NoError(t, err, "UUID v3 should be valid")
	})

	t.Run("non-v3 UUID rejection", func(t *testing.T) {
		uuidV4 := uuid.New()
		rule := validator.ValidUUIDv3("uuid", uuidV4)
		err := validator.Apply(rule)
		assert.Error(t, err, "UUID v4 should be rejected for v3 validation")
	})
}

func TestValidUUIDv3String(t *testing.T) {
	t.Parallel()
	t.Run("UUID v3 string validation", func(t *testing.T) {
		namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
		uuidV3 := uuid.NewMD5(namespace, []byte("test"))

		rule := validator.ValidUUIDv3String("uuid", uuidV3.String())
		err := validator.Apply(rule)
		assert.NoError(t, err, "UUID v3 string should be valid")
	})
}

func TestValidUUIDv5(t *testing.T) {
	t.Parallel()
	t.Run("UUID v5 validation", func(t *testing.T) {
		// Create a v5 UUID
		namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
		uuidV5 := uuid.NewSHA1(namespace, []byte("test"))

		rule := validator.ValidUUIDv5("uuid", uuidV5)
		err := validator.Apply(rule)
		assert.NoError(t, err, "UUID v5 should be valid")
	})

	t.Run("non-v5 UUID rejection", func(t *testing.T) {
		uuidV4 := uuid.New()
		rule := validator.ValidUUIDv5("uuid", uuidV4)
		err := validator.Apply(rule)
		assert.Error(t, err, "UUID v4 should be rejected for v5 validation")
	})
}

func TestValidUUIDv5String(t *testing.T) {
	t.Parallel()
	t.Run("UUID v5 string validation", func(t *testing.T) {
		namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
		uuidV5 := uuid.NewSHA1(namespace, []byte("test"))

		rule := validator.ValidUUIDv5String("uuid", uuidV5.String())
		err := validator.Apply(rule)
		assert.NoError(t, err, "UUID v5 string should be valid")
	})
}

func TestRequiredUUID(t *testing.T) {
	t.Parallel()
	t.Run("valid required UUIDs", func(t *testing.T) {
		validUUIDs := []uuid.UUID{
			uuid.New(),
			uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		}

		for _, uuidVal := range validUUIDs {
			rule := validator.RequiredUUID("uuid", uuidVal)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Required UUID should be valid: %s", uuidVal.String())
		}
	})

	t.Run("invalid required UUIDs", func(t *testing.T) {
		invalidUUIDs := []uuid.UUID{
			uuid.Nil,
			{}, // zero value
		}

		for _, uuidVal := range invalidUUIDs {
			rule := validator.RequiredUUID("uuid", uuidVal)
			err := validator.Apply(rule)
			assert.Error(t, err, "Required UUID should be invalid: %s", uuidVal.String())

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.required", validationErr[0].TranslationKey)
		}
	})
}

func TestRequiredUUIDString(t *testing.T) {
	t.Parallel()
	t.Run("valid required UUID strings", func(t *testing.T) {
		validUUIDs := []string{
			uuid.New().String(),
			"550e8400-e29b-41d4-a716-446655440000",
		}

		for _, uuidStr := range validUUIDs {
			rule := validator.RequiredUUIDString("uuid", uuidStr)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Required UUID string should be valid: %s", uuidStr)
		}
	})

	t.Run("invalid required UUID strings", func(t *testing.T) {
		invalidUUIDs := []string{
			"",
			"   ",
			"00000000-0000-0000-0000-000000000000", // nil UUID
			"not-a-uuid",
			"550e8400-e29b-41d4-a716-44665544000g", // invalid format
		}

		for _, uuidStr := range invalidUUIDs {
			rule := validator.RequiredUUIDString("uuid", uuidStr)
			err := validator.Apply(rule)
			assert.Error(t, err, "Required UUID string should be invalid: %s", uuidStr)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.required", validationErr[0].TranslationKey)
		}
	})
}
