package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestInList(t *testing.T) {
	t.Run("valid values in list", func(t *testing.T) {
		allowedInts := []int{1, 2, 3, 4, 5}
		validInts := []int{1, 3, 5}

		for _, value := range validInts {
			rule := validator.InList("field", value, allowedInts)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should be in list: %d", value)
		}
	})

	t.Run("invalid values not in list", func(t *testing.T) {
		allowedInts := []int{1, 2, 3, 4, 5}
		invalidInts := []int{0, 6, 10, -1}

		for _, value := range invalidInts {
			rule := validator.InList("field", value, allowedInts)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should not be in list: %d", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.in_list", validationErr[0].TranslationKey)
		}
	})

	t.Run("string values in list", func(t *testing.T) {
		allowedStrings := []string{"apple", "banana", "cherry"}

		rule := validator.InList("fruit", "banana", allowedStrings)
		err := validator.Apply(rule)
		assert.NoError(t, err, "Value should be in list")
	})

	t.Run("boolean values in list", func(t *testing.T) {
		allowedBools := []bool{true}

		rule := validator.InList("flag", true, allowedBools)
		err := validator.Apply(rule)
		assert.NoError(t, err, "Value should be in list")

		rule = validator.InList("flag", false, allowedBools)
		err = validator.Apply(rule)
		assert.Error(t, err, "Value should not be in list")
	})
}

func TestNotInList(t *testing.T) {
	t.Run("valid values not in forbidden list", func(t *testing.T) {
		forbiddenInts := []int{1, 2, 3}
		validInts := []int{4, 5, 6, 0, -1}

		for _, value := range validInts {
			rule := validator.NotInList("field", value, forbiddenInts)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Value should not be in forbidden list: %d", value)
		}
	})

	t.Run("invalid values in forbidden list", func(t *testing.T) {
		forbiddenInts := []int{1, 2, 3}
		invalidInts := []int{1, 2, 3}

		for _, value := range invalidInts {
			rule := validator.NotInList("field", value, forbiddenInts)
			err := validator.Apply(rule)
			assert.Error(t, err, "Value should be rejected: %d", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.not_in_list", validationErr[0].TranslationKey)
		}
	})
}

func TestInListString(t *testing.T) {
	t.Run("valid strings in list", func(t *testing.T) {
		allowedValues := []string{"red", "green", "blue"}
		validValues := []string{"red", "blue"}

		for _, value := range validValues {
			rule := validator.InListString("color", value, allowedValues)
			err := validator.Apply(rule)
			assert.NoError(t, err, "String should be in list: %s", value)
		}
	})

	t.Run("invalid strings not in list", func(t *testing.T) {
		allowedValues := []string{"red", "green", "blue"}
		invalidValues := []string{"yellow", "purple", "orange", ""}

		for _, value := range invalidValues {
			rule := validator.InListString("color", value, allowedValues)
			err := validator.Apply(rule)
			assert.Error(t, err, "String should not be in list: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.in_list", validationErr[0].TranslationKey)
		}
	})
}

func TestNotInListString(t *testing.T) {
	t.Run("valid strings not in forbidden list", func(t *testing.T) {
		forbiddenValues := []string{"admin", "root", "system"}
		validValues := []string{"user", "guest", "member", "moderator"}

		for _, value := range validValues {
			rule := validator.NotInListString("username", value, forbiddenValues)
			err := validator.Apply(rule)
			assert.NoError(t, err, "String should not be in forbidden list: %s", value)
		}
	})

	t.Run("invalid strings in forbidden list", func(t *testing.T) {
		forbiddenValues := []string{"admin", "root", "system"}
		invalidValues := []string{"admin", "root", "system"}

		for _, value := range invalidValues {
			rule := validator.NotInListString("username", value, forbiddenValues)
			err := validator.Apply(rule)
			assert.Error(t, err, "String should be rejected: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.not_in_list", validationErr[0].TranslationKey)
		}
	})
}

func TestInListCaseInsensitive(t *testing.T) {
	t.Run("valid case insensitive matches", func(t *testing.T) {
		allowedValues := []string{"Red", "Green", "Blue"}
		validValues := []string{"red", "RED", "Green", "BLUE", "blue"}

		for _, value := range validValues {
			rule := validator.InListCaseInsensitive("color", value, allowedValues)
			err := validator.Apply(rule)
			assert.NoError(t, err, "String should match case insensitive: %s", value)
		}
	})

	t.Run("invalid case insensitive non-matches", func(t *testing.T) {
		allowedValues := []string{"Red", "Green", "Blue"}
		invalidValues := []string{"yellow", "PURPLE", "Orange", ""}

		for _, value := range invalidValues {
			rule := validator.InListCaseInsensitive("color", value, allowedValues)
			err := validator.Apply(rule)
			assert.Error(t, err, "String should not match: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.in_list_case_insensitive", validationErr[0].TranslationKey)
		}
	})
}

func TestNotInListCaseInsensitive(t *testing.T) {
	t.Run("valid case insensitive non-matches", func(t *testing.T) {
		forbiddenValues := []string{"Admin", "Root", "System"}
		validValues := []string{"user", "GUEST", "Member", "moderator"}

		for _, value := range validValues {
			rule := validator.NotInListCaseInsensitive("username", value, forbiddenValues)
			err := validator.Apply(rule)
			assert.NoError(t, err, "String should not match forbidden list: %s", value)
		}
	})

	t.Run("invalid case insensitive matches", func(t *testing.T) {
		forbiddenValues := []string{"Admin", "Root", "System"}
		invalidValues := []string{"admin", "ROOT", "System", "ADMIN"}

		for _, value := range invalidValues {
			rule := validator.NotInListCaseInsensitive("username", value, forbiddenValues)
			err := validator.Apply(rule)
			assert.Error(t, err, "String should be rejected: %s", value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.not_in_list_case_insensitive", validationErr[0].TranslationKey)
		}
	})
}

func TestOneOf(t *testing.T) {
	t.Run("valid one of values", func(t *testing.T) {
		options := []string{"small", "medium", "large"}

		rule := validator.OneOf("size", "medium", options)
		err := validator.Apply(rule)
		assert.NoError(t, err, "Value should be one of the options")
	})

	t.Run("invalid one of values", func(t *testing.T) {
		options := []string{"small", "medium", "large"}

		rule := validator.OneOf("size", "extra-large", options)
		err := validator.Apply(rule)
		assert.Error(t, err, "Value should not be in options")
	})
}

func TestOneOfString(t *testing.T) {
	t.Run("valid one of string values", func(t *testing.T) {
		options := []string{"GET", "POST", "PUT", "DELETE"}

		rule := validator.OneOfString("method", "GET", options)
		err := validator.Apply(rule)
		assert.NoError(t, err, "String should be one of the options")
	})

	t.Run("invalid one of string values", func(t *testing.T) {
		options := []string{"GET", "POST", "PUT", "DELETE"}

		rule := validator.OneOfString("method", "PATCH", options)
		err := validator.Apply(rule)
		assert.Error(t, err, "String should not be in options")
	})
}

func TestNoneOf(t *testing.T) {
	t.Run("valid none of values", func(t *testing.T) {
		forbiddenOptions := []int{1, 2, 3}

		rule := validator.NoneOf("number", 5, forbiddenOptions)
		err := validator.Apply(rule)
		assert.NoError(t, err, "Value should not be in forbidden options")
	})

	t.Run("invalid none of values", func(t *testing.T) {
		forbiddenOptions := []int{1, 2, 3}

		rule := validator.NoneOf("number", 2, forbiddenOptions)
		err := validator.Apply(rule)
		assert.Error(t, err, "Value should be rejected")
	})
}

func TestNoneOfString(t *testing.T) {
	t.Run("valid none of string values", func(t *testing.T) {
		forbiddenOptions := []string{"spam", "adult", "violence"}

		rule := validator.NoneOfString("content", "family-friendly", forbiddenOptions)
		err := validator.Apply(rule)
		assert.NoError(t, err, "String should not be in forbidden options")
	})

	t.Run("invalid none of string values", func(t *testing.T) {
		forbiddenOptions := []string{"spam", "adult", "violence"}

		rule := validator.NoneOfString("content", "spam", forbiddenOptions)
		err := validator.Apply(rule)
		assert.Error(t, err, "String should be rejected")
	})
}

func TestValidEnum(t *testing.T) {
	t.Run("valid enum values", func(t *testing.T) {
		enumValues := []string{"PENDING", "PROCESSING", "COMPLETED", "FAILED"}
		validValues := []string{"PENDING", "COMPLETED"}

		for _, value := range validValues {
			rule := validator.ValidEnum("status", value, enumValues)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Enum value should be valid: %s", value)
		}
	})

	t.Run("invalid enum values", func(t *testing.T) {
		enumValues := []string{"PENDING", "PROCESSING", "COMPLETED", "FAILED"}
		invalidValues := []string{"UNKNOWN", "pending", "Invalid", ""}

		for _, value := range invalidValues {
			rule := validator.ValidEnum("status", value, enumValues)
			err := validator.Apply(rule)
			assert.Error(t, err, "Enum value should be invalid: %s", value)
		}
	})
}

func TestValidEnumCaseInsensitive(t *testing.T) {
	t.Run("valid case insensitive enum values", func(t *testing.T) {
		enumValues := []string{"PENDING", "PROCESSING", "COMPLETED", "FAILED"}
		validValues := []string{"pending", "COMPLETED", "Processing", "failed"}

		for _, value := range validValues {
			rule := validator.ValidEnumCaseInsensitive("status", value, enumValues)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Enum value should be valid: %s", value)
		}
	})

	t.Run("invalid case insensitive enum values", func(t *testing.T) {
		enumValues := []string{"PENDING", "PROCESSING", "COMPLETED", "FAILED"}
		invalidValues := []string{"unknown", "Invalid", "cancelled", ""}

		for _, value := range invalidValues {
			rule := validator.ValidEnumCaseInsensitive("status", value, enumValues)
			err := validator.Apply(rule)
			assert.Error(t, err, "Enum value should be invalid: %s", value)
		}
	})
}

func TestValidStatus(t *testing.T) {
	t.Run("valid status values", func(t *testing.T) {
		allowedStatuses := []string{"active", "inactive", "pending", "suspended"}
		validStatuses := []string{"active", "pending"}

		for _, status := range validStatuses {
			rule := validator.ValidStatus("status", status, allowedStatuses)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Status should be valid: %s", status)
		}
	})

	t.Run("invalid status values", func(t *testing.T) {
		allowedStatuses := []string{"active", "inactive", "pending", "suspended"}
		invalidStatuses := []string{"deleted", "unknown", "archived", ""}

		for _, status := range invalidStatuses {
			rule := validator.ValidStatus("status", status, allowedStatuses)
			err := validator.Apply(rule)
			assert.Error(t, err, "Status should be invalid: %s", status)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.valid_status", validationErr[0].TranslationKey)
		}
	})
}

func TestValidRole(t *testing.T) {
	t.Run("valid role values", func(t *testing.T) {
		allowedRoles := []string{"admin", "user", "moderator", "guest"}
		validRoles := []string{"admin", "user"}

		for _, role := range validRoles {
			rule := validator.ValidRole("role", role, allowedRoles)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Role should be valid: %s", role)
		}
	})

	t.Run("invalid role values", func(t *testing.T) {
		allowedRoles := []string{"admin", "user", "moderator", "guest"}
		invalidRoles := []string{"superadmin", "owner", "invalid", ""}

		for _, role := range invalidRoles {
			rule := validator.ValidRole("role", role, allowedRoles)
			err := validator.Apply(rule)
			assert.Error(t, err, "Role should be invalid: %s", role)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.valid_role", validationErr[0].TranslationKey)
		}
	})
}

func TestValidPermission(t *testing.T) {
	t.Run("valid permission values", func(t *testing.T) {
		allowedPermissions := []string{"read", "write", "delete", "admin"}
		validPermissions := []string{"read", "write"}

		for _, permission := range validPermissions {
			rule := validator.ValidPermission("permission", permission, allowedPermissions)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Permission should be valid: %s", permission)
		}
	})

	t.Run("invalid permission values", func(t *testing.T) {
		allowedPermissions := []string{"read", "write", "delete", "admin"}
		invalidPermissions := []string{"execute", "modify", "invalid", ""}

		for _, permission := range invalidPermissions {
			rule := validator.ValidPermission("permission", permission, allowedPermissions)
			err := validator.Apply(rule)
			assert.Error(t, err, "Permission should be invalid: %s", permission)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.valid_permission", validationErr[0].TranslationKey)
		}
	})
}

func TestValidCategory(t *testing.T) {
	t.Run("valid category values", func(t *testing.T) {
		allowedCategories := []string{"technology", "sports", "entertainment", "news"}
		validCategories := []string{"technology", "sports"}

		for _, category := range validCategories {
			rule := validator.ValidCategory("category", category, allowedCategories)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Category should be valid: %s", category)
		}
	})

	t.Run("invalid category values", func(t *testing.T) {
		allowedCategories := []string{"technology", "sports", "entertainment", "news"}
		invalidCategories := []string{"politics", "science", "invalid", ""}

		for _, category := range invalidCategories {
			rule := validator.ValidCategory("category", category, allowedCategories)
			err := validator.Apply(rule)
			assert.Error(t, err, "Category should be invalid: %s", category)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.valid_category", validationErr[0].TranslationKey)
		}
	})
}

func TestChoiceValidationCombination(t *testing.T) {
	t.Run("comprehensive choice validation", func(t *testing.T) {
		status := "active"
		role := "user"
		permission := "read"
		category := "technology"

		err := validator.Apply(
			validator.ValidStatus("status", status, []string{"active", "inactive"}),
			validator.ValidRole("role", role, []string{"admin", "user", "guest"}),
			validator.ValidPermission("permission", permission, []string{"read", "write", "delete"}),
			validator.ValidCategory("category", category, []string{"technology", "sports", "news"}),
		)

		assert.NoError(t, err, "Valid choice data should pass all validations")
	})

	t.Run("invalid choice data fails multiple validations", func(t *testing.T) {
		status := "unknown"
		role := "superuser"
		permission := "execute"
		category := "politics"

		err := validator.Apply(
			validator.ValidStatus("status", status, []string{"active", "inactive"}),
			validator.ValidRole("role", role, []string{"admin", "user", "guest"}),
			validator.ValidPermission("permission", permission, []string{"read", "write", "delete"}),
			validator.ValidCategory("category", category, []string{"technology", "sports", "news"}),
		)

		assert.Error(t, err, "Invalid choice data should fail validations")

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.True(t, len(validationErr) > 1, "Should have multiple validation errors")
	})

	t.Run("mixed case insensitive validation", func(t *testing.T) {
		value := "Technology"
		allowedValues := []string{"technology", "sports", "entertainment"}

		// Case sensitive should fail
		rule1 := validator.InListString("category", value, allowedValues)
		err1 := validator.Apply(rule1)
		assert.Error(t, err1, "Case sensitive validation should fail")

		// Case insensitive should pass
		rule2 := validator.InListCaseInsensitive("category", value, allowedValues)
		err2 := validator.Apply(rule2)
		assert.NoError(t, err2, "Case insensitive validation should pass")
	})

	t.Run("empty lists", func(t *testing.T) {
		emptyList := []string{}

		rule := validator.InListString("field", "any", emptyList)
		err := validator.Apply(rule)
		assert.Error(t, err, "Any value should fail validation against empty list")
	})
}
