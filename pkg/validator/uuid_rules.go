package validator

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidUUID validates standard UUID format with pre-validation to avoid expensive parsing.
func ValidUUID(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			// Fast rejection: check length and hyphen positions before parsing
			if len(value) != 36 {
				return false
			}

			if value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
				return false
			}

			_, err := uuid.Parse(value)
			return err == nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "must be a valid UUID",
			TranslationKey: "validation.uuid",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func NonNilUUID(field string, value uuid.UUID) Rule {
	return Rule{
		Check: func() bool {
			return value != uuid.Nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "UUID cannot be nil",
			TranslationKey: "validation.uuid_not_nil",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func NonNilUUIDString(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
				return false
			}

			parsedUUID, err := uuid.Parse(value)
			if err != nil {
				return false
			}
			return parsedUUID != uuid.Nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "UUID cannot be nil",
			TranslationKey: "validation.uuid_not_nil",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func ValidUUIDVersion(field string, value uuid.UUID, version int) Rule {
	return Rule{
		Check: func() bool {
			return value.Version() == uuid.Version(version)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be a UUID version %d", version),
			TranslationKey: "validation.uuid_version",
			TranslationValues: map[string]any{
				"field":   field,
				"version": version,
			},
		},
	}
}

func ValidUUIDVersionString(field, value string, version int) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
				return false
			}

			parsedUUID, err := uuid.Parse(value)
			if err != nil {
				return false
			}
			return parsedUUID.Version() == uuid.Version(version)
		},
		Error: ValidationError{
			Field:          field,
			Message:        fmt.Sprintf("must be a UUID version %d", version),
			TranslationKey: "validation.uuid_version",
			TranslationValues: map[string]any{
				"field":   field,
				"version": version,
			},
		},
	}
}

// Version-specific validation helpers

func ValidUUIDv1(field string, value uuid.UUID) Rule {
	return ValidUUIDVersion(field, value, 1)
}

func ValidUUIDv1String(field, value string) Rule {
	return ValidUUIDVersionString(field, value, 1)
}

func ValidUUIDv3(field string, value uuid.UUID) Rule {
	return ValidUUIDVersion(field, value, 3)
}

func ValidUUIDv3String(field, value string) Rule {
	return ValidUUIDVersionString(field, value, 3)
}

func ValidUUIDv4(field string, value uuid.UUID) Rule {
	return ValidUUIDVersion(field, value, 4)
}

func ValidUUIDv4String(field, value string) Rule {
	return ValidUUIDVersionString(field, value, 4)
}

func ValidUUIDv5(field string, value uuid.UUID) Rule {
	return ValidUUIDVersion(field, value, 5)
}

func ValidUUIDv5String(field, value string) Rule {
	return ValidUUIDVersionString(field, value, 5)
}

func RequiredUUID(field string, value uuid.UUID) Rule {
	return Rule{
		Check: func() bool {
			return value != uuid.UUID{} && value != uuid.Nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "UUID is required",
			TranslationKey: "validation.required",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}

func RequiredUUIDString(field, value string) Rule {
	return Rule{
		Check: func() bool {
			if strings.TrimSpace(value) == "" {
				return false
			}

			if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
				return false
			}

			parsedUUID, err := uuid.Parse(value)
			if err != nil {
				return false
			}
			return parsedUUID != uuid.Nil
		},
		Error: ValidationError{
			Field:          field,
			Message:        "UUID is required",
			TranslationKey: "validation.required",
			TranslationValues: map[string]any{
				"field": field,
			},
		},
	}
}
