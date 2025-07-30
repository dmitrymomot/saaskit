package binder

import (
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
)

// DefaultMaxMemory is the default maximum memory used for parsing multipart forms (10MB).
const DefaultMaxMemory = 10 << 20 // 10 MB

// Form creates a unified binder for both form data and file uploads.
// It handles application/x-www-form-urlencoded and multipart/form-data content types.
//
// Supported struct tags:
//   - `form:"name"` - binds to form field "name"
//   - `form:"-"`    - skips the field
//   - `file:"name"` - binds to uploaded file "name"
//   - `file:"-"`    - skips the field
//
// Supported types for form fields:
//   - Basic types: string, int, int64, uint, uint64, float32, float64, bool
//   - Slices of basic types for multi-value fields
//   - Pointers for optional fields
//
// Supported types for file fields:
//   - *multipart.FileHeader - single file
//   - []*multipart.FileHeader - multiple files
//
// Example:
//
//	type UploadRequest struct {
//		Title    string                  `form:"title"`
//		Category string                  `form:"category"`
//		Tags     []string                `form:"tags"`     // Multi-value field
//		Avatar   *multipart.FileHeader   `file:"avatar"`   // Optional file
//		Gallery  []*multipart.FileHeader `file:"gallery"`  // Multiple files
//		Internal string                  `form:"-"`        // Skipped
//	}
//
//	handler := saaskit.HandlerFunc[saaskit.Context, UploadRequest](
//		func(ctx saaskit.Context, req UploadRequest) saaskit.Response {
//			if req.Avatar != nil {
//				file, err := req.Avatar.Open()
//				if err != nil {
//					return saaskit.Error(http.StatusBadRequest, "Failed to open file")
//				}
//				defer file.Close()
//				// Process file...
//			}
//			return saaskit.JSONResponse(result)
//		},
//	)
//
//	http.HandleFunc("/upload", saaskit.Wrap(handler,
//		saaskit.WithBinder(binder.Form()),
//	))
func Form() func(r *http.Request, v any) error {
	return func(r *http.Request, v any) error {
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			return fmt.Errorf("%w: missing content-type header, expected application/x-www-form-urlencoded or multipart/form-data", ErrMissingContentType)
		}

		// Extract media type without parameters
		mediaType := contentType
		if idx := strings.Index(contentType, ";"); idx != -1 {
			mediaType = strings.TrimSpace(contentType[:idx])
		}

		var values map[string][]string
		var files map[string][]*multipart.FileHeader

		switch {
		case mediaType == "application/x-www-form-urlencoded":
			if err := r.ParseForm(); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidForm, err)
			}
			values = r.Form

		case strings.HasPrefix(mediaType, "multipart/form-data"):
			// Validate multipart content type and boundary for security
			_, params, err := mime.ParseMediaType(contentType)
			if err != nil {
				return fmt.Errorf("%w: malformed content type with boundary", ErrInvalidForm)
			}

			boundary, ok := params["boundary"]
			if !ok || boundary == "" {
				return fmt.Errorf("%w: missing boundary in content type", ErrInvalidForm)
			}

			if !validateBoundary(boundary) {
				return fmt.Errorf("%w: invalid boundary parameter", ErrInvalidForm)
			}

			// Note: Request size limits should be handled at server/middleware level
			if err := r.ParseMultipartForm(DefaultMaxMemory); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidForm, err)
			}

			if r.MultipartForm != nil {
				values = r.MultipartForm.Value
				files = r.MultipartForm.File
			} else {
				values = make(map[string][]string)
			}

		default:
			return fmt.Errorf("%w: got %s, expected application/x-www-form-urlencoded or multipart/form-data", ErrUnsupportedMediaType, mediaType)
		}

		// Note: Multipart form cleanup should be handled at server/middleware level
		// to maintain standard Go behavior and allow access to r.MultipartForm after binding
		return bindFormAndFiles(v, values, files, ErrInvalidForm)
	}
}

// bindFormAndFiles binds both form values and files to a struct.
func bindFormAndFiles(v any, values map[string][]string, files map[string][]*multipart.FileHeader, bindErr error) error {
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

		formTag := fieldType.Tag.Get("form")
		fileTag := fieldType.Tag.Get("file")

		// Skip if both tags are missing
		if formTag == "" && fileTag == "" {
			continue
		}

		// Handle form tag
		if formTag != "" {
			if formTag == "-" {
				continue // Skip explicitly ignored fields
			}

			// Extract parameter name from tag
			paramName := formTag
			if idx := strings.Index(formTag, ","); idx != -1 {
				paramName = formTag[:idx]
			}

			// Skip empty parameter names
			if paramName == "" {
				continue
			}

			if fieldValues, exists := values[paramName]; exists && len(fieldValues) > 0 {
				if err := setFieldValue(field, fieldType.Type, fieldValues); err != nil {
					return fmt.Errorf("%w: field %s: %v", bindErr, fieldType.Name, err)
				}
			}
		}

		// Handle file tag
		if fileTag != "" && fileTag != "-" && files != nil {
			// Skip empty file tag values
			if fileTag == "" {
				continue
			}

			if fileHeaders, exists := files[fileTag]; exists && len(fileHeaders) > 0 {
				if err := setFileField(field, fieldType.Type, fileHeaders); err != nil {
					return fmt.Errorf("%w: field %s: %v", bindErr, fieldType.Name, err)
				}
			}
		}
	}

	return nil
}

// setFileField sets file values to struct fields.
func setFileField(field reflect.Value, fieldType reflect.Type, fileHeaders []*multipart.FileHeader) error {
	// Sanitize filenames
	for _, fh := range fileHeaders {
		fh.Filename = sanitizeFilename(fh.Filename)
	}

	if fieldType.Kind() == reflect.Slice {
		elemType := fieldType.Elem()
		if elemType != reflect.TypeOf((*multipart.FileHeader)(nil)) {
			return fmt.Errorf("unsupported slice element type for file field: %v", elemType)
		}

		slice := reflect.MakeSlice(fieldType, len(fileHeaders), len(fileHeaders))
		for i, fh := range fileHeaders {
			slice.Index(i).Set(reflect.ValueOf(fh))
		}
		field.Set(slice)
		return nil
	}

	if fieldType == reflect.TypeOf((*multipart.FileHeader)(nil)) {
		if len(fileHeaders) > 0 {
			field.Set(reflect.ValueOf(fileHeaders[0]))
		}
		return nil
	}

	return fmt.Errorf("unsupported type for file field: %v (expected *multipart.FileHeader or []*multipart.FileHeader)", fieldType)
}

// sanitizeFilename removes any path components and dangerous characters from a filename
// to prevent path traversal attacks and other security issues.
func sanitizeFilename(filename string) string {
	// Replace backslashes with forward slashes to normalize paths
	// This ensures filepath.Base works correctly on Windows-style paths
	filename = strings.ReplaceAll(filename, "\\", "/")

	// Remove any directory components - handles both Unix and Windows paths
	filename = filepath.Base(filename)

	// Remove null bytes and other potentially dangerous characters
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Ensure the filename is not empty or a special directory reference
	if filename == "." || filename == ".." || filename == "" || filename == "/" {
		filename = "unnamed"
	}

	return filename
}
