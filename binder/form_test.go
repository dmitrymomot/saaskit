package binder_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/binder"
)

func TestBindForm(t *testing.T) {
	type basicForm struct {
		Name     string  `form:"name"`
		Age      int     `form:"age"`
		Height   float64 `form:"height"`
		Active   bool    `form:"active"`
		Page     uint    `form:"page"`
		Internal string  `form:"-"` // Should be skipped
	}

	t.Run("valid form binding with all types", func(t *testing.T) {
		formData := url.Values{
			"name":   {"John"},
			"age":    {"30"},
			"height": {"5.9"},
			"active": {"true"},
			"page":   {"2"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "John", result.Name)
		assert.Equal(t, 30, result.Age)
		assert.Equal(t, 5.9, result.Height)
		assert.Equal(t, true, result.Active)
		assert.Equal(t, uint(2), result.Page)
		assert.Equal(t, "", result.Internal) // Should remain empty
	})

	t.Run("skips fields with dash tag", func(t *testing.T) {
		formData := url.Values{
			"name":     {"Test"},
			"internal": {"secret"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		result.Internal = "original" // Set a value that should not be overwritten
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, "original", result.Internal) // Should not be changed
	})

	t.Run("content type with charset", func(t *testing.T) {
		formData := url.Values{
			"name": {"Jane"},
			"age":  {"25"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Jane", result.Name)
		assert.Equal(t, 25, result.Age)
	})

	t.Run("missing content type", func(t *testing.T) {
		formData := url.Values{"name": {"Test"}}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		// Don't set Content-Type

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrMissingContentType))
		assert.Contains(t, err.Error(), "expected application/x-www-form-urlencoded")
	})

	t.Run("wrong content type", func(t *testing.T) {
		formData := url.Values{"name": {"Test"}}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/json")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrUnsupportedMediaType))
		assert.Contains(t, err.Error(), "got application/json")
	})

	t.Run("empty form data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Name)      // zero value
		assert.Equal(t, 0, result.Age)        // zero value
		assert.Equal(t, 0.0, result.Height)   // zero value
		assert.Equal(t, false, result.Active) // zero value
		assert.Equal(t, uint(0), result.Page) // zero value
	})

	t.Run("partial form data", func(t *testing.T) {
		formData := url.Values{
			"name": {"Jane"},
			"age":  {"25"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Jane", result.Name)
		assert.Equal(t, 25, result.Age)
		assert.Equal(t, 0.0, result.Height)   // zero value
		assert.Equal(t, false, result.Active) // zero value
	})

	t.Run("invalid int value", func(t *testing.T) {
		formData := url.Values{"age": {"notanumber"}}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid int value")
		assert.Contains(t, err.Error(), "Age")
	})

	t.Run("slice parameters multiple values", func(t *testing.T) {
		type sliceForm struct {
			Tags []string `form:"tags"`
			IDs  []int    `form:"ids"`
		}

		formData := url.Values{
			"tags": {"go", "web", "api"},
			"ids":  {"1", "2", "3"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result sliceForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "web", "api"}, result.Tags)
		assert.Equal(t, []int{1, 2, 3}, result.IDs)
	})

	t.Run("slice parameters comma separated", func(t *testing.T) {
		type sliceForm struct {
			Tags   []string  `form:"tags"`
			Scores []float64 `form:"scores"`
		}

		formData := url.Values{
			"tags":   {"go,web,api"},
			"scores": {"1.5,2.0,3.5"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result sliceForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "web", "api"}, result.Tags)
		assert.Equal(t, []float64{1.5, 2.0, 3.5}, result.Scores)
	})

	t.Run("boolean slice parameters", func(t *testing.T) {
		type boolSliceForm struct {
			Flags []bool `form:"flags"`
		}

		formData := url.Values{
			"flags": {"true", "false", "1", "0", "yes", "no", "on", "off"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result boolSliceForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []bool{true, false, true, false, true, false, true, false}, result.Flags)
	})

	t.Run("boolean slice comma separated", func(t *testing.T) {
		type boolSliceForm struct {
			Settings []bool `form:"settings"`
		}

		formData := url.Values{
			"settings": {"true,false,yes,no,1,0,ON,OFF"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result boolSliceForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []bool{true, false, true, false, true, false, true, false}, result.Settings)
	})

	t.Run("pointer fields", func(t *testing.T) {
		type pointerForm struct {
			Name     *string  `form:"name"`
			Age      *int     `form:"age"`
			Active   *bool    `form:"active"`
			Score    *float64 `form:"score"`
			Required string   `form:"required"`
		}

		formData := url.Values{
			"name":     {"John"},
			"active":   {"true"},
			"required": {"value"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result pointerForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.NotNil(t, result.Name)
		assert.Equal(t, "John", *result.Name)
		assert.Nil(t, result.Age) // Not provided
		require.NotNil(t, result.Active)
		assert.Equal(t, true, *result.Active)
		assert.Nil(t, result.Score) // Not provided
		assert.Equal(t, "value", result.Required)
	})

	t.Run("no struct tag uses lowercase field name", func(t *testing.T) {
		type noTagForm struct {
			Name  string
			Count int
		}

		formData := url.Values{
			"name":  {"Test"},
			"count": {"5"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result noTagForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, 5, result.Count)
	})

	t.Run("special characters in values", func(t *testing.T) {
		type specialForm struct {
			Email   string `form:"email"`
			URL     string `form:"url"`
			Message string `form:"msg"`
		}

		formData := url.Values{
			"email": {"user@example.com"},
			"url":   {"https://example.com?q=test&p=1"},
			"msg":   {"Hello World!"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result specialForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "user@example.com", result.Email)
		assert.Equal(t, "https://example.com?q=test&p=1", result.URL)
		assert.Equal(t, "Hello World!", result.Message)
	})

	t.Run("non-pointer target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, result) // Pass by value, not pointer

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a non-nil pointer")
	})

	t.Run("boolean variations", func(t *testing.T) {
		tests := []struct {
			value    string
			expected bool
		}{
			// Standard boolean strings
			{"true", true},
			{"false", false},
			{"True", true},
			{"False", false},
			{"TRUE", true},
			{"FALSE", false},
			{"t", true},
			{"f", false},
			{"T", true},
			{"F", false},

			// Numeric strings
			{"1", true},
			{"0", false},

			// Alternative boolean strings
			{"on", true},
			{"off", false},
			{"On", true},
			{"Off", false},
			{"ON", true},
			{"OFF", false},

			{"yes", true},
			{"no", false},
			{"Yes", true},
			{"No", false},
			{"YES", true},
			{"NO", false},

			// Empty value (tested separately in "empty values in form" test)
		}

		for _, tt := range tests {
			t.Run(tt.value, func(t *testing.T) {
				formData := url.Values{"active": {tt.value}}
				req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				var result basicForm
				bindFunc := binder.BindForm()
				err := bindFunc(req, &result)

				require.NoError(t, err)
				assert.Equal(t, tt.expected, result.Active)
			})
		}
	})

	t.Run("invalid boolean values", func(t *testing.T) {
		invalidValues := []string{
			"maybe",
			"unknown",
			"y",
			"n",
			"Y",
			"N",
			"2",
			"-1",
			"10",
			"truee",
			"fals",
			"yess",
			"noo",
		}

		for _, value := range invalidValues {
			t.Run(value, func(t *testing.T) {
				formData := url.Values{"active": {value}}
				req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				var result basicForm
				bindFunc := binder.BindForm()
				err := bindFunc(req, &result)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid bool value")
				assert.Contains(t, err.Error(), value)
			})
		}
	})

	t.Run("tags with options", func(t *testing.T) {
		type tagOptionsForm struct {
			Name     string `form:"name,omitempty"`
			Optional string `form:"opt,omitempty"`
			Count    int    `form:"count,omitempty"`
		}

		formData := url.Values{
			"name":  {"Test"},
			"count": {"10"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result tagOptionsForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, "", result.Optional) // Not provided, zero value
		assert.Equal(t, 10, result.Count)
	})

	t.Run("large form data", func(t *testing.T) {
		// Create large form data
		formData := url.Values{}
		for i := 0; i < 100; i++ {
			formData.Set(strings.ToLower(string(rune('A'+i%26)))+string(rune('0'+i/26)), "value")
		}
		formData.Set("name", "LargeForm")

		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "LargeForm", result.Name)
	})

	t.Run("invalid form data", func(t *testing.T) {
		// Send malformed form data
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("%ZZ"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrInvalidForm))
	})

	t.Run("empty values in form", func(t *testing.T) {
		formData := url.Values{
			"name":   {""},
			"active": {""},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Name)
		assert.Equal(t, false, result.Active) // Empty string is treated as false
	})

	t.Run("multipart form content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=----boundary")

		var result basicForm
		bindFunc := binder.BindForm()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrUnsupportedMediaType))
		assert.Contains(t, err.Error(), "expected application/x-www-form-urlencoded")
	})
}
