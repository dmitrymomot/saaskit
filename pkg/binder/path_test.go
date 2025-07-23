package binder_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/binder"
)

func TestPath(t *testing.T) {
	t.Parallel()
	type basicStruct struct {
		ID       string  `path:"id"`
		Username string  `path:"username"`
		Age      int     `path:"age"`
		Height   float64 `path:"height"`
		Active   bool    `path:"active"`
		Page     uint    `path:"page"`
		Internal string  `path:"-"` // Should be skipped
	}

	t.Run("custom extractor function", func(t *testing.T) {
		t.Parallel()
		// Simulate a simple path params map
		pathParams := map[string]string{
			"id":       "123",
			"username": "john_doe",
			"age":      "30",
			"height":   "5.9",
			"active":   "true",
			"page":     "2",
		}

		// Create a custom extractor
		extractor := func(r *http.Request, fieldName string) string {
			return pathParams[fieldName]
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		bindFunc := binder.Path(extractor)
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "123", result.ID)
		assert.Equal(t, "john_doe", result.Username)
		assert.Equal(t, 30, result.Age)
		assert.Equal(t, 5.9, result.Height)
		assert.Equal(t, true, result.Active)
		assert.Equal(t, uint(2), result.Page)
		assert.Equal(t, "", result.Internal) // Should remain empty
	})

	t.Run("missing path params", func(t *testing.T) {
		t.Parallel()
		extractor := func(r *http.Request, fieldName string) string {
			return "" // Always return empty
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		result.ID = "original" // Set a value that should not be overwritten
		bindFunc := binder.Path(extractor)
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "original", result.ID) // Should not be changed
		assert.Equal(t, "", result.Username)   // Should remain empty
		assert.Equal(t, 0, result.Age)         // Should remain zero
	})

	t.Run("nil extractor", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		bindFunc := binder.Path(nil)
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "extractor function is nil")
	})

	t.Run("nil target", func(t *testing.T) {
		t.Parallel()
		extractor := func(r *http.Request, fieldName string) string {
			return "value"
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		bindFunc := binder.Path(extractor)
		err := bindFunc(req, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a non-nil pointer")
	})

	t.Run("non-pointer target", func(t *testing.T) {
		t.Parallel()
		extractor := func(r *http.Request, fieldName string) string {
			return "value"
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		bindFunc := binder.Path(extractor)
		err := bindFunc(req, result) // Pass by value, not pointer

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a non-nil pointer")
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		t.Parallel()
		extractor := func(r *http.Request, fieldName string) string {
			return "value"
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result string
		bindFunc := binder.Path(extractor)
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a pointer to struct")
	})

	t.Run("skips fields with dash tag", func(t *testing.T) {
		t.Parallel()
		pathParams := map[string]string{
			"id":       "123",
			"internal": "secret", // This should be ignored
		}

		extractor := func(r *http.Request, fieldName string) string {
			return pathParams[fieldName]
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		result.Internal = "original" // Set a value that should not be overwritten
		bindFunc := binder.Path(extractor)
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "123", result.ID)
		assert.Equal(t, "original", result.Internal) // Should not be changed
	})

	t.Run("no tag uses field name", func(t *testing.T) {
		t.Parallel()
		type noTagStruct struct {
			UserID string // No tag, should use "userid" (lowercase)
			Count  int
		}

		pathParams := map[string]string{
			"userid": "789",
			"count":  "42",
		}

		extractor := func(r *http.Request, fieldName string) string {
			return pathParams[fieldName]
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result noTagStruct
		bindFunc := binder.Path(extractor)
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "789", result.UserID)
		assert.Equal(t, 42, result.Count)
	})

	t.Run("pointer fields", func(t *testing.T) {
		t.Parallel()
		type pointerStruct struct {
			ID      *string  `path:"id"`
			Age     *int     `path:"age"`
			Height  *float64 `path:"height"`
			Active  *bool    `path:"active"`
			Missing *string  `path:"missing"` // Will not be provided
		}

		pathParams := map[string]string{
			"id":     "ptr123",
			"age":    "35",
			"height": "6.2",
			"active": "true",
		}

		extractor := func(r *http.Request, fieldName string) string {
			return pathParams[fieldName]
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result pointerStruct
		bindFunc := binder.Path(extractor)
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.NotNil(t, result.ID)
		assert.Equal(t, "ptr123", *result.ID)
		require.NotNil(t, result.Age)
		assert.Equal(t, 35, *result.Age)
		require.NotNil(t, result.Height)
		assert.Equal(t, 6.2, *result.Height)
		require.NotNil(t, result.Active)
		assert.Equal(t, true, *result.Active)
		assert.Nil(t, result.Missing) // Should remain nil
	})

	t.Run("invalid numeric values", func(t *testing.T) {
		t.Parallel()
		pathParams := map[string]string{
			"age":    "not-a-number",
			"height": "invalid",
			"page":   "-5", // Negative uint
		}

		extractor := func(r *http.Request, fieldName string) string {
			return pathParams[fieldName]
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		bindFunc := binder.Path(extractor)
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("chi router style extractor", func(t *testing.T) {
		t.Parallel()
		// Simulate chi router URL params
		urlParams := map[string]string{
			"id":       "chi123",
			"username": "chi_user",
		}

		// Chi-style extractor
		chiExtractor := func(r *http.Request, fieldName string) string {
			// In real chi, you'd use chi.URLParam(r, fieldName)
			return urlParams[fieldName]
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		bindFunc := binder.Path(chiExtractor)
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "chi123", result.ID)
		assert.Equal(t, "chi_user", result.Username)
	})

	t.Run("gorilla mux style extractor", func(t *testing.T) {
		t.Parallel()
		// Simulate gorilla/mux vars
		muxVars := map[string]string{
			"id":       "mux456",
			"username": "mux_user",
		}

		// Gorilla/mux style extractor
		muxExtractor := func(r *http.Request, fieldName string) string {
			// In real mux, you'd use mux.Vars(r)[fieldName]
			return muxVars[fieldName]
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		bindFunc := binder.Path(muxExtractor)
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "mux456", result.ID)
		assert.Equal(t, "mux_user", result.Username)
	})
}
