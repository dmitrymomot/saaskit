package validator

import (
	"fmt"
	"strings"
)

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

// Semantic aliases for choice validation

func OneOf[T comparable](field string, value T, options []T) Rule {
	return InList(field, value, options)
}

func OneOfString(field, value string, options []string) Rule {
	return InListString(field, value, options)
}

func NoneOf[T comparable](field string, value T, options []T) Rule {
	return NotInList(field, value, options)
}

func NoneOfString(field, value string, options []string) Rule {
	return NotInListString(field, value, options)
}

func ValidEnum(field, value string, enumValues []string) Rule {
	return InListString(field, value, enumValues)
}

func ValidEnumCaseInsensitive(field, value string, enumValues []string) Rule {
	return InListCaseInsensitive(field, value, enumValues)
}

// Domain-specific validation helpers - these could use InListString but provide
// more semantic error messages for common business concepts.

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
