package binder

import (
	"fmt"
	"net/http"
	"strings"
)

// BindForm creates a form data binder function for application/x-www-form-urlencoded content.
//
// It supports struct tags for custom field names:
//   - `form:"name"` - binds to form field "name"
//   - `form:"-"` - skips the field
//   - `form:"name,omitempty"` - same as form:"name" for parsing
//
// Supported types:
//   - Basic types: string, int, int64, uint, uint64, float32, float64, bool
//   - Slices of basic types for multi-value fields
//   - Pointers for optional fields
//
// Example:
//
//	type LoginRequest struct {
//		Username string   `form:"username"`
//		Password string   `form:"password"`
//		Remember bool     `form:"remember"`
//		Roles    []string `form:"roles"`    // Multiple checkbox values
//		Ref      *string  `form:"ref"`      // Optional field
//		Internal string   `form:"-"`        // Skipped
//	}
//
//	handler := saaskit.HandlerFunc[saaskit.Context, LoginRequest](
//		func(ctx saaskit.Context, req LoginRequest) saaskit.Response {
//			// req is populated from form data
//			return saaskit.JSONResponse(result)
//		},
//	)
//
//	http.HandleFunc("/login", saaskit.Wrap(handler,
//		saaskit.WithBinder(binder.BindForm()),
//	))
func BindForm() func(r *http.Request, v any) error {
	return func(r *http.Request, v any) error {
		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			return fmt.Errorf("%w: expected application/x-www-form-urlencoded", ErrMissingContentType)
		}

		// Extract media type without parameters
		mediaType := contentType
		if idx := strings.Index(contentType, ";"); idx != -1 {
			mediaType = strings.TrimSpace(contentType[:idx])
		}

		if mediaType != "application/x-www-form-urlencoded" {
			return fmt.Errorf("%w: got %s, expected application/x-www-form-urlencoded", ErrUnsupportedMediaType, mediaType)
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidForm, err)
		}

		// Use the shared binding logic
		return bindToStruct(v, "form", r.Form, ErrInvalidForm)
	}
}
