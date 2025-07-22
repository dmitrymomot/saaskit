package validator

import (
	"fmt"
	"strings"
)

// InList validates that a value is in the allowed list of values.
func InList[T comparable](field string, value T, allowedValues []T) Rule {
	return Rule{
		Check: func() bool {
			for _, allowed := range allowedValues {
				if value == allowed {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be one of: %v", allowedValues),
			TranslationKey: "validation.in_list",
			TranslationValues: map[string]any{
				"field":          field,
				"allowed_values": allowedValues,
			},
		},
	}
}

// NotInList validates that a value is not in the forbidden list of values.
func NotInList[T comparable](field string, value T, forbiddenValues []T) Rule {
	return Rule{
		Check: func() bool {
			for _, forbidden := range forbiddenValues {
				if value == forbidden {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must not be one of: %v", forbiddenValues),
			TranslationKey: "validation.not_in_list",
			TranslationValues: map[string]any{
				"field":            field,
				"forbidden_values": forbiddenValues,
			},
		},
	}
}

// InListString validates that a string is in the allowed list of string values.
func InListString(field, value string, allowedValues []string) Rule {
	return Rule{
		Check: func() bool {
			for _, allowed := range allowedValues {
				if value == allowed {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be one of: %s", strings.Join(allowedValues, ", ")),
			TranslationKey: "validation.in_list",
			TranslationValues: map[string]any{
				"field":          field,
				"allowed_values": allowedValues,
			},
		},
	}
}

// NotInListString validates that a string is not in the forbidden list of string values.
func NotInListString(field, value string, forbiddenValues []string) Rule {
	return Rule{
		Check: func() bool {
			for _, forbidden := range forbiddenValues {
				if value == forbidden {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must not be one of: %s", strings.Join(forbiddenValues, ", ")),
			TranslationKey: "validation.not_in_list",
			TranslationValues: map[string]any{
				"field":            field,
				"forbidden_values": forbiddenValues,
			},
		},
	}
}

// InListCaseInsensitive validates that a string is in the allowed list (case-insensitive).
func InListCaseInsensitive(field, value string, allowedValues []string) Rule {
	return Rule{
		Check: func() bool {
			lowerValue := strings.ToLower(value)
			for _, allowed := range allowedValues {
				if lowerValue == strings.ToLower(allowed) {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be one of (case-insensitive): %s", strings.Join(allowedValues, ", ")),
			TranslationKey: "validation.in_list_case_insensitive",
			TranslationValues: map[string]any{
				"field":          field,
				"allowed_values": allowedValues,
			},
		},
	}
}

// NotInListCaseInsensitive validates that a string is not in the forbidden list (case-insensitive).
func NotInListCaseInsensitive(field, value string, forbiddenValues []string) Rule {
	return Rule{
		Check: func() bool {
			lowerValue := strings.ToLower(value)
			for _, forbidden := range forbiddenValues {
				if lowerValue == strings.ToLower(forbidden) {
					return false
				}
			}
			return true
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must not be one of (case-insensitive): %s", strings.Join(forbiddenValues, ", ")),
			TranslationKey: "validation.not_in_list_case_insensitive",
			TranslationValues: map[string]any{
				"field":            field,
				"forbidden_values": forbiddenValues,
			},
		},
	}
}

// OneOf validates that a value matches exactly one of the provided options.
// This is an alias for InList but with a more semantic name.
func OneOf[T comparable](field string, value T, options []T) Rule {
	return InList(field, value, options)
}

// OneOfString validates that a string matches exactly one of the provided options.
// This is an alias for InListString but with a more semantic name.
func OneOfString(field, value string, options []string) Rule {
	return InListString(field, value, options)
}

// NoneOf validates that a value does not match any of the provided options.
// This is an alias for NotInList but with a more semantic name.
func NoneOf[T comparable](field string, value T, options []T) Rule {
	return NotInList(field, value, options)
}

// NoneOfString validates that a string does not match any of the provided options.
// This is an alias for NotInListString but with a more semantic name.
func NoneOfString(field, value string, options []string) Rule {
	return NotInListString(field, value, options)
}

// ValidEnum validates that a value is a valid enum member (case-sensitive).
func ValidEnum(field, value string, enumValues []string) Rule {
	return InListString(field, value, enumValues)
}

// ValidEnumCaseInsensitive validates that a value is a valid enum member (case-insensitive).
func ValidEnumCaseInsensitive(field, value string, enumValues []string) Rule {
	return InListCaseInsensitive(field, value, enumValues)
}

// ValidStatus validates that a status string is one of the allowed status values.
func ValidStatus(field, value string, allowedStatuses []string) Rule {
	return Rule{
		Check: func() bool {
			for _, status := range allowedStatuses {
				if value == status {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("status must be one of: %s", strings.Join(allowedStatuses, ", ")),
			TranslationKey: "validation.valid_status",
			TranslationValues: map[string]any{
				"field":            field,
				"allowed_statuses": allowedStatuses,
			},
		},
	}
}

// ValidRole validates that a role string is one of the allowed role values.
func ValidRole(field, value string, allowedRoles []string) Rule {
	return Rule{
		Check: func() bool {
			for _, role := range allowedRoles {
				if value == role {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("role must be one of: %s", strings.Join(allowedRoles, ", ")),
			TranslationKey: "validation.valid_role",
			TranslationValues: map[string]any{
				"field":         field,
				"allowed_roles": allowedRoles,
			},
		},
	}
}

// ValidPermission validates that a permission string is one of the allowed permission values.
func ValidPermission(field, value string, allowedPermissions []string) Rule {
	return Rule{
		Check: func() bool {
			for _, permission := range allowedPermissions {
				if value == permission {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("permission must be one of: %s", strings.Join(allowedPermissions, ", ")),
			TranslationKey: "validation.valid_permission",
			TranslationValues: map[string]any{
				"field":               field,
				"allowed_permissions": allowedPermissions,
			},
		},
	}
}

// ValidCategory validates that a category string is one of the allowed category values.
func ValidCategory(field, value string, allowedCategories []string) Rule {
	return Rule{
		Check: func() bool {
			for _, category := range allowedCategories {
				if value == category {
					return true
				}
			}
			return false
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("category must be one of: %s", strings.Join(allowedCategories, ", ")),
			TranslationKey: "validation.valid_category",
			TranslationValues: map[string]any{
				"field":              field,
				"allowed_categories": allowedCategories,
			},
		},
	}
}
