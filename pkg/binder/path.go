package binder

import (
	"fmt"
	"net/http"
	"reflect"
)

// Path creates a path parameter binder function using the provided extractor.
// The extractor function is called for each struct field to get its path parameter value.
//
// It supports struct tags for custom parameter names:
//   - `path:"name"` - binds to path parameter "name"
//   - `path:"-"` - skips the field
//
// Supported types:
//   - Basic types: string, int, int64, uint, uint64, float32, float64, bool
//   - Pointers for optional fields
//
// Example with chi router:
//
//	type ProfileRequest struct {
//		UserID   string `path:"id"`
//		Username string `path:"username"`
//		Name     string `form:"name"`     // From form data
//		Expand   bool   `query:"expand"`  // From query string
//	}
//
//	handler := saaskit.HandlerFunc[saaskit.Context, ProfileRequest](
//		func(ctx saaskit.Context, req ProfileRequest) saaskit.Response {
//			// req.UserID and req.Username are populated from path
//			return saaskit.JSONResponse(profile)
//		},
//	)
//
//	r := chi.NewRouter()
//	r.Get("/users/{id}/profile/{username}", saaskit.Wrap(handler,
//		saaskit.WithBinders(
//			binder.Path(chi.URLParam),
//			binder.Query(),
//			binder.Form(),
//		),
//	))
//
// Example with gorilla/mux:
//
//	muxExtractor := func(r *http.Request, fieldName string) string {
//		vars := mux.Vars(r)
//		return vars[fieldName]
//	}
//
//	router := mux.NewRouter()
//	router.HandleFunc("/users/{id}/profile/{username}", saaskit.Wrap(handler,
//		saaskit.WithBinders(
//			binder.Path(muxExtractor),
//			binder.Query(),
//			binder.Form(),
//		),
//	))
func Path(extractor func(r *http.Request, fieldName string) string) func(r *http.Request, v any) error {
	return func(r *http.Request, v any) error {
		if extractor == nil {
			return fmt.Errorf("%w: extractor function is nil", ErrInvalidPath)
		}

		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			return fmt.Errorf("%w: target must be a non-nil pointer", ErrInvalidPath)
		}

		rv = rv.Elem()
		if rv.Kind() != reflect.Struct {
			return fmt.Errorf("%w: target must be a pointer to struct", ErrInvalidPath)
		}

		rt := rv.Type()

		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)
			fieldType := rt.Field(i)

			// Skip unexported fields
			if !field.CanSet() {
				continue
			}

			// Parse field tag
			paramName, skip := parseFieldTag(fieldType, "path")
			if skip {
				continue
			}

			// Get value using extractor
			value := extractor(r, paramName)
			if value == "" {
				// No value provided, leave as zero value
				continue
			}

			// Set field value based on type
			if err := setFieldValue(field, fieldType.Type, []string{value}); err != nil {
				return fmt.Errorf("%w: field %s: %v", ErrInvalidPath, fieldType.Name, err)
			}
		}

		return nil
	}
}
