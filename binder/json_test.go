package binder_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/binder"
)

func TestBindJSON(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	t.Run("valid JSON binding", func(t *testing.T) {
		jsonData := `{"name":"John Doe","age":30,"email":"john@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "John Doe", result.Name)
		assert.Equal(t, 30, result.Age)
		assert.Equal(t, "john@example.com", result.Email)
	})

	t.Run("content type with charset", func(t *testing.T) {
		jsonData := `{"name":"Jane","age":25}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Jane", result.Name)
		assert.Equal(t, 25, result.Age)
	})

	t.Run("missing content type", func(t *testing.T) {
		jsonData := `{"name":"Test"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		// Don't set Content-Type

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrMissingContentType))
		assert.Contains(t, err.Error(), "expected application/json")
	})

	t.Run("wrong content type", func(t *testing.T) {
		jsonData := `{"name":"Test"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "text/plain")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrUnsupportedMediaType))
		assert.Contains(t, err.Error(), "got text/plain")
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(""))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrInvalidJSON))
		assert.Contains(t, err.Error(), "empty body")
	})

	t.Run("invalid JSON syntax", func(t *testing.T) {
		jsonData := `{"name":"Test"` // Missing closing brace
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrInvalidJSON))
		assert.Contains(t, err.Error(), "unexpected EOF")
	})

	t.Run("invalid character in JSON", func(t *testing.T) {
		jsonData := `{name:"Test"}` // Invalid JSON - unquoted key
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrInvalidJSON))
		assert.Contains(t, err.Error(), "invalid character")
	})

	t.Run("type mismatch", func(t *testing.T) {
		jsonData := `{"name":"Test","age":"not a number"}` // age should be int
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrInvalidJSON))
		assert.Contains(t, err.Error(), "cannot unmarshal")
	})

	t.Run("unknown fields rejected", func(t *testing.T) {
		jsonData := `{"name":"Test","age":25,"unknown_field":"value"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrInvalidJSON))
		// The error message varies by Go version, but it should mention the unknown field
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("extra data after valid JSON", func(t *testing.T) {
		jsonData := `{"name":"Test","age":25}{"extra":"data"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrInvalidJSON))
		assert.Contains(t, err.Error(), "unexpected data after JSON object")
	})

	t.Run("null values", func(t *testing.T) {
		jsonData := `{"name":null,"age":null,"email":null}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Name)  // zero value for string
		assert.Equal(t, 0, result.Age)    // zero value for int
		assert.Equal(t, "", result.Email) // zero value for string
	})

	t.Run("partial data", func(t *testing.T) {
		jsonData := `{"name":"Partial"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Partial", result.Name)
		assert.Equal(t, 0, result.Age)    // zero value
		assert.Equal(t, "", result.Email) // zero value
	})

	t.Run("nested structs", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}
		type Person struct {
			Name    string  `json:"name"`
			Address Address `json:"address"`
		}

		jsonData := `{"name":"John","address":{"street":"123 Main St","city":"NYC"}}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result Person
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "John", result.Name)
		assert.Equal(t, "123 Main St", result.Address.Street)
		assert.Equal(t, "NYC", result.Address.City)
	})

	t.Run("arrays", func(t *testing.T) {
		type Items struct {
			Names []string `json:"names"`
			Nums  []int    `json:"nums"`
		}

		jsonData := `{"names":["Alice","Bob"],"nums":[1,2,3]}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result Items
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"Alice", "Bob"}, result.Names)
		assert.Equal(t, []int{1, 2, 3}, result.Nums)
	})

	t.Run("pointer fields", func(t *testing.T) {
		type OptionalFields struct {
			Name     *string `json:"name"`
			Age      *int    `json:"age"`
			Required string  `json:"required"`
		}

		jsonData := `{"name":"Test","required":"value"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var result OptionalFields
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.NotNil(t, result.Name)
		assert.Equal(t, "Test", *result.Name)
		assert.Nil(t, result.Age) // Not provided, should be nil
		assert.Equal(t, "value", result.Required)
	})

	t.Run("content type with multiple parameters", func(t *testing.T) {
		jsonData := `{"name":"Test"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(jsonData))
		req.Header.Set("Content-Type", "application/json; charset=utf-8; boundary=something")

		var result testStruct
		bindFunc := binder.BindJSON()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
	})
}
