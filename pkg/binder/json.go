package binder

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

// DefaultMaxJSONSize is the default maximum size for JSON request bodies (1MB).
const DefaultMaxJSONSize = 1 << 20 // 1 MB

// JSON creates a JSON binder function.
//
// Example:
//
//	handler := saaskit.HandlerFunc[saaskit.Context, CreateUserRequest](
//		func(ctx saaskit.Context, req CreateUserRequest) saaskit.Response {
//			// req is populated from JSON body
//			return saaskit.JSONResponse(user)
//		},
//	)
//
//	http.HandleFunc("/users", saaskit.Wrap(handler,
//		saaskit.WithBinder(binder.JSON()),
//	))
func JSON() func(r *http.Request, v any) error {
	return func(r *http.Request, v any) error {
		// Check for context timeout
		ctx := r.Context()
		if ctx != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("%w: context timeout", ErrFailedToParseJSON)
			default:
			}
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			return fmt.Errorf("%w: missing content-type header, expected application/json", ErrMissingContentType)
		}

		// Extract media type without parameters
		mediaType := contentType
		if idx := strings.Index(contentType, ";"); idx != -1 {
			mediaType = strings.TrimSpace(contentType[:idx])
		}

		if mediaType != "application/json" {
			return fmt.Errorf("%w: got %s, expected application/json", ErrUnsupportedMediaType, mediaType)
		}

		// Read the entire body with size limit
		limitedReader := io.LimitReader(r.Body, DefaultMaxJSONSize+1)
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return fmt.Errorf("%w: failed to read request body: %v", ErrFailedToParseJSON, err)
		}

		// Check if body exceeded size limit
		if len(body) > DefaultMaxJSONSize {
			return fmt.Errorf("%w: request body too large (max %d bytes)", ErrFailedToParseJSON, DefaultMaxJSONSize)
		}

		decoder := json.NewDecoder(strings.NewReader(string(body)))
		decoder.DisallowUnknownFields() // Always use strict mode

		if err := decoder.Decode(v); err != nil {
			switch {
			case strings.Contains(err.Error(), "cannot unmarshal"):
				return fmt.Errorf("%w: %v", ErrFailedToParseJSON, err)
			case strings.Contains(err.Error(), "unexpected end of JSON"):
				return fmt.Errorf("%w: %v", ErrFailedToParseJSON, err)
			case strings.Contains(err.Error(), "invalid character"):
				return fmt.Errorf("%w: %v", ErrFailedToParseJSON, err)
			case err == io.EOF:
				return fmt.Errorf("%w: empty body", ErrFailedToParseJSON)
			default:
				return fmt.Errorf("%w: %v", ErrFailedToParseJSON, err)
			}
		}

		// Ensure entire body was consumed
		var extra json.RawMessage
		if err := decoder.Decode(&extra); err != io.EOF {
			return fmt.Errorf("%w: unexpected data after JSON object", ErrFailedToParseJSON)
		}

		// Sanitize all string fields in the decoded struct
		if err := sanitizeJSONStruct(v); err != nil {
			return fmt.Errorf("%w: failed to sanitize input: %v", ErrFailedToParseJSON, err)
		}

		return nil
	}
}

// sanitizeJSONStruct recursively sanitizes all string fields in a struct.
func sanitizeJSONStruct(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil
	}

	rv = rv.Elem()
	return sanitizeReflectValue(rv)
}

// sanitizeReflectValue recursively sanitizes reflect.Value.
func sanitizeReflectValue(rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.String:
		if rv.CanSet() {
			sanitized := sanitizeStringValue(rv.String())
			rv.SetString(sanitized)
		}

	case reflect.Struct:
		for i := range rv.NumField() {
			field := rv.Field(i)
			if field.CanSet() {
				if err := sanitizeReflectValue(field); err != nil {
					return err
				}
			}
		}

	case reflect.Slice, reflect.Array:
		for i := range rv.Len() {
			elem := rv.Index(i)
			if err := sanitizeReflectValue(elem); err != nil {
				return err
			}
		}

	case reflect.Map:
		for _, key := range rv.MapKeys() {
			value := rv.MapIndex(key)
			if value.CanSet() {
				if err := sanitizeReflectValue(value); err != nil {
					return err
				}
			}
		}

	case reflect.Ptr:
		if !rv.IsNil() {
			if err := sanitizeReflectValue(rv.Elem()); err != nil {
				return err
			}
		}

	case reflect.Interface:
		if !rv.IsNil() {
			if err := sanitizeReflectValue(rv.Elem()); err != nil {
				return err
			}
		}
	}

	return nil
}
