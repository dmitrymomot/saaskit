package binder

import (
	"net/http"
)

// BindQuery creates a query parameter binder function.
//
// It supports struct tags for custom parameter names:
//   - `query:"name"` - binds to query parameter "name"
//   - `query:"-"` - skips the field
//   - `query:"name,omitempty"` - same as query:"name" for parsing
//
// Supported types:
//   - Basic types: string, int, int64, uint, uint64, float32, float64, bool
//   - Slices of basic types for multi-value parameters
//   - Pointers for optional fields
//
// Example:
//
//	type SearchRequest struct {
//		Query    string   `query:"q"`
//		Page     int      `query:"page"`
//		PageSize int      `query:"page_size"`
//		Tags     []string `query:"tags"`     // ?tags=go&tags=web or ?tags=go,web
//		Active   *bool    `query:"active"`   // Optional
//		Internal string   `query:"-"`        // Skipped
//	}
//
//	handler := saaskit.HandlerFunc[saaskit.Context, SearchRequest](
//		func(ctx saaskit.Context, req SearchRequest) saaskit.Response {
//			// req is populated from query parameters
//			return saaskit.JSONResponse(results)
//		},
//	)
//
//	http.HandleFunc("/search", saaskit.Wrap(handler,
//		saaskit.WithBinder(binder.BindQuery()),
//	))
func BindQuery() func(r *http.Request, v any) error {
	return func(r *http.Request, v any) error {
		return bindToStruct(v, "query", r.URL.Query(), ErrInvalidQuery)
	}
}
