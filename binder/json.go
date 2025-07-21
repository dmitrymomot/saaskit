package binder

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BindJSON creates a JSON binder function.
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
//		saaskit.WithBinder(binder.BindJSON()),
//	))
func BindJSON() func(r *http.Request, v any) error {
	return func(r *http.Request, v any) error {
		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			return fmt.Errorf("%w: expected application/json", ErrMissingContentType)
		}

		// Extract media type without parameters
		mediaType := contentType
		if idx := strings.Index(contentType, ";"); idx != -1 {
			mediaType = strings.TrimSpace(contentType[:idx])
		}

		if mediaType != "application/json" {
			return fmt.Errorf("%w: got %s, expected application/json", ErrUnsupportedMediaType, mediaType)
		}

		// Decode JSON
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields() // Strict mode

		if err := decoder.Decode(v); err != nil {
			// Check for common JSON errors
			switch {
			case strings.Contains(err.Error(), "cannot unmarshal"):
				return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
			case strings.Contains(err.Error(), "unexpected end of JSON"):
				return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
			case strings.Contains(err.Error(), "invalid character"):
				return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
			case err == io.EOF:
				return fmt.Errorf("%w: empty body", ErrInvalidJSON)
			default:
				return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
			}
		}

		// Ensure entire body was consumed
		var extra json.RawMessage
		if err := decoder.Decode(&extra); err != io.EOF {
			return fmt.Errorf("%w: unexpected data after JSON object", ErrInvalidJSON)
		}

		return nil
	}
}
