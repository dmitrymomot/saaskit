package binder

import (
	"fmt"
	"net/http"
	"strings"
)

// BindForm creates a form data binder function for both application/x-www-form-urlencoded
// and multipart/form-data content types. For multipart forms, it only binds non-file fields.
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
// For file uploads in multipart forms, use the File() binder with `file:` tags.
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
			return fmt.Errorf("%w: expected application/x-www-form-urlencoded or multipart/form-data", ErrMissingContentType)
		}

		// Extract media type without parameters
		mediaType := contentType
		if idx := strings.Index(contentType, ";"); idx != -1 {
			mediaType = strings.TrimSpace(contentType[:idx])
		}

		var values map[string][]string

		switch {
		case mediaType == "application/x-www-form-urlencoded":
			// Parse URL-encoded form
			if err := r.ParseForm(); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidForm, err)
			}
			values = r.Form

		case strings.HasPrefix(mediaType, "multipart/form-data"):
			// Parse multipart form (default 10MB limit)
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidForm, err)
			}
			if r.MultipartForm != nil {
				values = r.MultipartForm.Value
			} else {
				values = make(map[string][]string)
			}

		default:
			return fmt.Errorf("%w: got %s, expected application/x-www-form-urlencoded or multipart/form-data", ErrUnsupportedMediaType, mediaType)
		}

		// Use the shared binding logic
		return bindToStruct(v, "form", values, ErrInvalidForm)
	}
}
