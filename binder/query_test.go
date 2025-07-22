package binder_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/binder"
)

func TestQuery(t *testing.T) {
	type basicStruct struct {
		Name     string  `query:"name"`
		Age      int     `query:"age"`
		Height   float64 `query:"height"`
		Active   bool    `query:"active"`
		Page     uint    `query:"page"`
		Internal string  `query:"-"` // Should be skipped
	}

	t.Run("valid query binding with all types", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?name=John&age=30&height=5.9&active=true&page=2", nil)

		var result basicStruct
		bindFunc := binder.Query()
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
		req := httptest.NewRequest(http.MethodGet, "/test?name=Test&internal=secret", nil)

		var result basicStruct
		result.Internal = "original" // Set a value that should not be overwritten
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, "original", result.Internal) // Should not be changed
	})

	t.Run("empty query parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Name)      // zero value
		assert.Equal(t, 0, result.Age)        // zero value
		assert.Equal(t, 0.0, result.Height)   // zero value
		assert.Equal(t, false, result.Active) // zero value
		assert.Equal(t, uint(0), result.Page) // zero value
	})

	t.Run("partial query parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?name=Jane&age=25", nil)

		var result basicStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Jane", result.Name)
		assert.Equal(t, 25, result.Age)
		assert.Equal(t, 0.0, result.Height)   // zero value
		assert.Equal(t, false, result.Active) // zero value
	})

	t.Run("invalid int value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?age=notanumber", nil)

		var result basicStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid int value")
		assert.Contains(t, err.Error(), "Age")
	})

	t.Run("invalid uint value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?page=-1", nil)

		var result basicStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid uint value")
	})

	t.Run("invalid float value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?height=tall", nil)

		var result basicStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid float value")
		assert.Contains(t, err.Error(), "Height")
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

			// Empty value
			{"", false},
		}

		for _, tt := range tests {
			t.Run(tt.value, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/test?active="+tt.value, nil)

				var result basicStruct
				bindFunc := binder.Query()
				err := bindFunc(req, &result)

				require.NoError(t, err)
				assert.Equal(t, tt.expected, result.Active)
			})
		}
	})

	t.Run("invalid boolean value", func(t *testing.T) {
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
				req := httptest.NewRequest(http.MethodGet, "/test?active="+value, nil)

				var result basicStruct
				bindFunc := binder.Query()
				err := bindFunc(req, &result)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid bool value")
				assert.Contains(t, err.Error(), value)
			})
		}
	})

	t.Run("slice parameters multiple values", func(t *testing.T) {
		type sliceStruct struct {
			Tags []string `query:"tags"`
			IDs  []int    `query:"ids"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?tags=go&tags=web&tags=api&ids=1&ids=2&ids=3", nil)

		var result sliceStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "web", "api"}, result.Tags)
		assert.Equal(t, []int{1, 2, 3}, result.IDs)
	})

	t.Run("slice parameters comma separated", func(t *testing.T) {
		type sliceStruct struct {
			Tags   []string  `query:"tags"`
			Scores []float64 `query:"scores"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?tags=go,web,api&scores=1.5,2.0,3.5", nil)

		var result sliceStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "web", "api"}, result.Tags)
		assert.Equal(t, []float64{1.5, 2.0, 3.5}, result.Scores)
	})

	t.Run("slice parameters mixed format", func(t *testing.T) {
		type sliceStruct struct {
			Tags []string `query:"tags"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?tags=go,web&tags=api&tags=backend,frontend", nil)

		var result sliceStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "web", "api", "backend", "frontend"}, result.Tags)
	})

	t.Run("boolean slice parameters", func(t *testing.T) {
		type boolSliceStruct struct {
			Flags []bool `query:"flags"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?flags=true&flags=false&flags=1&flags=0&flags=yes&flags=no&flags=on&flags=off", nil)

		var result boolSliceStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []bool{true, false, true, false, true, false, true, false}, result.Flags)
	})

	t.Run("boolean slice comma separated", func(t *testing.T) {
		type boolSliceStruct struct {
			Settings []bool `query:"settings"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?settings=true,false,yes,no,1,0", nil)

		var result boolSliceStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []bool{true, false, true, false, true, false}, result.Settings)
	})

	t.Run("pointer fields", func(t *testing.T) {
		type pointerStruct struct {
			Name     *string  `query:"name"`
			Age      *int     `query:"age"`
			Active   *bool    `query:"active"`
			Score    *float64 `query:"score"`
			Required string   `query:"required"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?name=John&active=true&required=value", nil)

		var result pointerStruct
		bindFunc := binder.Query()
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
		type noTagStruct struct {
			Name  string
			Count int
		}

		req := httptest.NewRequest(http.MethodGet, "/test?name=Test&count=5", nil)

		var result noTagStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, 5, result.Count)
	})

	t.Run("special characters in values", func(t *testing.T) {
		type specialStruct struct {
			Email   string `query:"email"`
			URL     string `query:"url"`
			Message string `query:"msg"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?email=user%40example.com&url=https%3A%2F%2Fexample.com&msg=Hello%20World%21", nil)

		var result specialStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "user@example.com", result.Email)
		assert.Equal(t, "https://example.com", result.URL)
		assert.Equal(t, "Hello World!", result.Message)
	})

	t.Run("non-pointer target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result basicStruct
		bindFunc := binder.Query()
		err := bindFunc(req, result) // Pass by value, not pointer

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a non-nil pointer")
	})

	t.Run("nil pointer target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result *basicStruct
		bindFunc := binder.Query()
		err := bindFunc(req, result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a non-nil pointer")
	})

	t.Run("non-struct target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		var result string
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a pointer to struct")
	})

	t.Run("tags with options", func(t *testing.T) {
		type tagOptionsStruct struct {
			Name     string `query:"name,omitempty"`
			Optional string `query:"opt,omitempty"`
			Count    int    `query:"count,omitempty"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?name=Test&count=10", nil)

		var result tagOptionsStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, "", result.Optional) // Not provided, zero value
		assert.Equal(t, 10, result.Count)
	})

	t.Run("various numeric types", func(t *testing.T) {
		type numericStruct struct {
			Int8    int8    `query:"int8"`
			Int16   int16   `query:"int16"`
			Int32   int32   `query:"int32"`
			Int64   int64   `query:"int64"`
			Uint8   uint8   `query:"uint8"`
			Uint16  uint16  `query:"uint16"`
			Uint32  uint32  `query:"uint32"`
			Uint64  uint64  `query:"uint64"`
			Float32 float32 `query:"float32"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?int8=127&int16=32767&int32=2147483647&int64=9223372036854775807&uint8=255&uint16=65535&uint32=4294967295&uint64=18446744073709551615&float32=3.14", nil)

		var result numericStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, int8(127), result.Int8)
		assert.Equal(t, int16(32767), result.Int16)
		assert.Equal(t, int32(2147483647), result.Int32)
		assert.Equal(t, int64(9223372036854775807), result.Int64)
		assert.Equal(t, uint8(255), result.Uint8)
		assert.Equal(t, uint16(65535), result.Uint16)
		assert.Equal(t, uint32(4294967295), result.Uint32)
		assert.Equal(t, uint64(18446744073709551615), result.Uint64)
		assert.InDelta(t, float32(3.14), result.Float32, 0.001)
	})

	t.Run("unexported fields are skipped", func(t *testing.T) {
		type mixedStruct struct {
			Public  string `query:"public"`
			private string `query:"private"` // unexported
		}

		req := httptest.NewRequest(http.MethodGet, "/test?public=visible&private=hidden", nil)

		var result mixedStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "visible", result.Public)
		assert.Equal(t, "", result.private) // Should remain empty
	})

	t.Run("trimmed values in comma-separated slices", func(t *testing.T) {
		type sliceStruct struct {
			Tags []string `query:"tags"`
		}

		req := httptest.NewRequest(http.MethodGet, "/test?tags=go%20,%20web%20,%20%20api", nil)

		var result sliceStruct
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "web", "api"}, result.Tags)
	})
}
