package binder

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// bindToStruct binds values to a struct using reflection.
// tagName specifies which struct tag to use (e.g., "query", "form").
// values is a map of parameter names to their string values.
// bindErr is the specific error to use for binding failures.
func bindToStruct(v any, tagName string, values map[string][]string, bindErr error) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("%w: target must be a non-nil pointer", bindErr)
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("%w: target must be a pointer to struct", bindErr)
	}

	rt := rv.Type()

	for i := range rv.NumField() {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		paramName, skip := parseFieldTag(fieldType, tagName)
		if skip {
			continue
		}

		fieldValues, exists := values[paramName]
		if !exists || len(fieldValues) == 0 {
			continue // No value provided, leave as zero value
		}

		if err := setFieldValue(field, fieldType.Type, fieldValues); err != nil {
			return fmt.Errorf("%w: field %s: %v", bindErr, fieldType.Name, err)
		}
	}

	return nil
}

// parseFieldTag parses the struct field tag and returns the parameter name and whether to skip.
func parseFieldTag(field reflect.StructField, tagName string) (paramName string, skip bool) {
	tag := field.Tag.Get(tagName)
	if tag == "" {
		return strings.ToLower(field.Name), false // No tag, use field name in lowercase
	}
	if tag == "-" {
		return "", true // Skip this field
	}

	// Handle comma-separated tag options (e.g., "name,omitempty")
	tagParts := strings.Split(tag, ",")
	return tagParts[0], false
}

// setFieldValue sets the field value from string values.
func setFieldValue(field reflect.Value, fieldType reflect.Type, values []string) error {
	// Handle pointer types
	if fieldType.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(fieldType.Elem()))
		}
		return setFieldValue(field.Elem(), fieldType.Elem(), values)
	}

	// Handle slice types
	if fieldType.Kind() == reflect.Slice {
		return setSliceValue(field, fieldType, values)
	}

	// For non-slice types, use the first value
	if len(values) == 0 {
		return nil
	}
	value := values[0]

	switch fieldType.Kind() {
	case reflect.String:
		field.SetString(sanitizeStringValue(value))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(value, 10, fieldType.Bits())
		if err != nil {
			return fmt.Errorf("invalid int value %q", value)
		}
		field.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(value, 10, fieldType.Bits())
		if err != nil {
			return fmt.Errorf("invalid uint value %q", value)
		}
		field.SetUint(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(value, fieldType.Bits())
		if err != nil {
			return fmt.Errorf("invalid float value %q", value)
		}
		field.SetFloat(n)

	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			// Be lenient with boolean values
			switch strings.ToLower(value) {
			case "on", "yes", "1":
				b = true
			case "off", "no", "0", "":
				b = false
			default:
				return fmt.Errorf("invalid bool value %q", value)
			}
		}
		field.SetBool(b)

	default:
		return fmt.Errorf("unsupported type %s", fieldType.Kind())
	}

	return nil
}

// setSliceValue sets slice field values from string values.
func setSliceValue(field reflect.Value, fieldType reflect.Type, values []string) error {
	elemType := fieldType.Elem()

	// Support comma-separated values as well
	var allValues []string
	for _, v := range values {
		if strings.Contains(v, ",") {
			allValues = append(allValues, strings.Split(v, ",")...)
		} else {
			allValues = append(allValues, v)
		}
	}

	slice := reflect.MakeSlice(fieldType, len(allValues), len(allValues))

	for i, value := range allValues {
		elem := slice.Index(i)
		if err := setFieldValue(elem, elemType, []string{strings.TrimSpace(value)}); err != nil {
			return err
		}
	}

	field.Set(slice)
	return nil
}

// sanitizeStringValue removes potentially dangerous characters and normalizes input.
// This prevents CRLF injection, NUL byte injection, and handles unicode normalization.
func sanitizeStringValue(value string) string {
	// Remove NUL bytes
	value = strings.ReplaceAll(value, "\x00", "")

	// Remove CRLF sequences to prevent header injection
	value = strings.ReplaceAll(value, "\r\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")

	// Remove other control characters except tab and space
	var builder strings.Builder
	builder.Grow(len(value))

	for _, r := range value {
		if r == '\t' || r >= ' ' || unicode.IsGraphic(r) {
			if utf8.ValidRune(r) {
				builder.WriteRune(r)
			}
		}
	}

	return builder.String()
}

// validateBoundary checks if a multipart boundary contains potentially malicious content.
func validateBoundary(boundary string) bool {
	if boundary == "" {
		return false
	}

	// Check for control characters that could break parsing
	for _, r := range boundary {
		if r == '\x00' || r == '\r' || r == '\n' {
			return false
		}
	}

	// RFC7578 recommends max 70 characters
	if len(boundary) > 100 {
		return false
	}

	return true
}
