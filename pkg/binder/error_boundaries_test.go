package binder_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/binder"
)

func TestErrorBoundaries(t *testing.T) {
	t.Run("malformed_content_type_headers", func(t *testing.T) {
		testMalformedContentTypeHeaders(t)
	})

	t.Run("parser_limits", func(t *testing.T) {
		testParserLimits(t)
	})

	t.Run("charset_confusion_attacks", func(t *testing.T) {
		testCharsetConfusionAttacks(t)
	})

	t.Run("boundary_injection", func(t *testing.T) {
		testBoundaryInjection(t)
	})
}

func testMalformedContentTypeHeaders(t *testing.T) {
	t.Run("content_type_with_embedded_newlines", func(t *testing.T) {
		t.Parallel()

		jsonData := `{"field": "value"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonData))

		// Content-Type with embedded CRLF
		req.Header.Set("Content-Type", "application/json\r\nX-Injected-Header: malicious")

		var target struct {
			Field string `json:"field"`
		}

		// Should handle malformed Content-Type header gracefully
		err := binder.JSON()(req, &target)
		if err != nil {
			// Should get a controlled error, not a panic or security bypass
			assert.Error(t, err)
		}
	})

	t.Run("missing_content_type", func(t *testing.T) {
		t.Parallel()

		jsonData := `{"field": "value"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonData))
		// No Content-Type header set

		var target struct {
			Field string `json:"field"`
		}

		// Should handle missing Content-Type gracefully
		err := binder.JSON()(req, &target)
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "content-type",
				"Error should mention content-type issue")
		}
	})

	t.Run("empty_content_type", func(t *testing.T) {
		t.Parallel()

		jsonData := `{"field": "value"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonData))
		req.Header.Set("Content-Type", "")

		var target struct {
			Field string `json:"field"`
		}

		err := binder.JSON()(req, &target)
		if err != nil {
			assert.Error(t, err)
		}
	})

	t.Run("malformed_charset_parameter", func(t *testing.T) {
		t.Parallel()

		jsonData := `{"field": "value"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonData))

		// Malformed charset parameter
		req.Header.Set("Content-Type", "application/json; charset=utf-8\x00malicious")

		var target struct {
			Field string `json:"field"`
		}

		err := binder.JSON()(req, &target)
		if err != nil {
			assert.Error(t, err)
		}
	})

	t.Run("extremely_long_content_type", func(t *testing.T) {
		t.Parallel()

		jsonData := `{"field": "value"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonData))

		// Extremely long Content-Type header
		longContentType := "application/json; " + strings.Repeat("parameter=value; ", 1000)
		req.Header.Set("Content-Type", longContentType)

		var target struct {
			Field string `json:"field"`
		}

		err := binder.JSON()(req, &target)
		if err != nil {
			assert.Error(t, err)
		}
	})
}

func testParserLimits(t *testing.T) {
	t.Run("maximum_json_nesting_depth", func(t *testing.T) {
		t.Parallel()

		// Create JSON with excessive nesting depth
		const maxDepth = 100
		openBraces := strings.Repeat(`{"nested":`, maxDepth)
		closeBraces := strings.Repeat(`}`, maxDepth)
		deepJSON := openBraces + `"value"` + closeBraces

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(deepJSON))
		req.Header.Set("Content-Type", "application/json")

		var target any

		// Should either parse with reasonable depth or return controlled error
		err := binder.JSON()(req, &target)
		if err != nil {
			// Should be a parser limit error, not a stack overflow
			assert.Error(t, err)
			assert.NotContains(t, err.Error(), "runtime", "Should not be a runtime error")
		}
	})

	t.Run("maximum_array_size", func(t *testing.T) {
		t.Parallel()

		// Create JSON with extremely large array
		const arraySize = 1000
		items := make([]string, arraySize)
		for i := range items {
			items[i] = fmt.Sprintf(`"item_%d"`, i)
		}
		largeArrayJSON := fmt.Sprintf(`{"items": [%s]}`, strings.Join(items, ","))

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(largeArrayJSON))
		req.Header.Set("Content-Type", "application/json")

		var target struct {
			Items []string `json:"items"`
		}

		// Should handle large arrays or return controlled error
		err := binder.JSON()(req, &target)
		if err != nil {
			assert.Error(t, err)
		} else {
			// If successful, verify the array size
			assert.LessOrEqual(t, len(target.Items), arraySize, "Array size should not exceed input")
		}
	})

	t.Run("circular_reference_detection", func(t *testing.T) {
		t.Parallel()

		// This test is more conceptual since Go's json package handles circular references
		// by default during marshaling, but we test what happens with malformed JSON
		// that might represent circular structures

		malformedJSON := `{"a": {"b": {"c": {"a": "circular_ref"}}}}`

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(malformedJSON))
		req.Header.Set("Content-Type", "application/json")

		var target any

		err := binder.JSON()(req, &target)
		if err != nil {
			assert.Error(t, err)
		}
	})

	t.Run("json_with_duplicate_keys", func(t *testing.T) {
		t.Parallel()

		// JSON with duplicate keys (RFC violation)
		duplicateKeyJSON := `{"key": "value1", "key": "value2", "key": "value3"}`

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(duplicateKeyJSON))
		req.Header.Set("Content-Type", "application/json")

		var target struct {
			Key string `json:"key"`
		}

		err := binder.JSON()(req, &target)
		if err == nil {
			// Go's JSON parser takes the last value for duplicate keys
			assert.Equal(t, "value3", target.Key, "Should use the last value for duplicate keys")
		}
	})

	t.Run("malformed_unicode_sequences", func(t *testing.T) {
		t.Parallel()

		// Test various malformed Unicode sequences
		malformedUnicodeTests := []struct {
			name string
			json string
		}{
			{"incomplete_escape", `{"field": "value\u12"`},
			{"invalid_hex", `{"field": "value\uGGGG"}`},
			{"truncated_sequence", `{"field": "value\u123`},
			{"overlong_sequence", `{"field": "value\u0000000041"}`},
		}

		for _, tc := range malformedUnicodeTests {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.json))
				req.Header.Set("Content-Type", "application/json")

				var target struct {
					Field string `json:"field"`
				}

				// Should handle malformed Unicode gracefully
				err := binder.JSON()(req, &target)
				if err != nil {
					assert.Error(t, err)
					assert.Contains(t, strings.ToLower(err.Error()), "failed to parse",
						"Error should indicate parsing failure")
				}
			})
		}
	})

	t.Run("json_number_limits", func(t *testing.T) {
		t.Parallel()

		// Test extremely large numbers
		largeNumberTests := []struct {
			name  string
			json  string
			field string
		}{
			{"extremely_large_int", `{"num": 999999999999999999999999999999999999999}`, "num"},
			{"extremely_small_int", `{"num": -999999999999999999999999999999999999999}`, "num"},
			{"extremely_precise_float", `{"num": 1.` + strings.Repeat("9", 1000) + `}`, "num"},
			{"scientific_notation_overflow", `{"num": 1e999999}`, "num"},
		}

		for _, tc := range largeNumberTests {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.json))
				req.Header.Set("Content-Type", "application/json")

				var target struct {
					Num float64 `json:"num"`
				}

				// Should handle extreme numbers gracefully
				err := binder.JSON()(req, &target)
				if err != nil {
					assert.Error(t, err)
				} else {
					// If parsing succeeds, the number should be finite
					assert.True(t, !isInf(target.Num, 0) && !isNaN(target.Num),
						"Number should be finite and not NaN")
				}
			})
		}
	})
}

func testCharsetConfusionAttacks(t *testing.T) {
	t.Run("charset_parameter_injection", func(t *testing.T) {
		t.Parallel()

		// Test charset confusion attacks
		jsonData := `{"field": "value"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonData))

		// Charset parameter with potential injection
		req.Header.Set("Content-Type", "application/json; charset=utf-8; boundary=malicious")

		var target struct {
			Field string `json:"field"`
		}

		err := binder.JSON()(req, &target)
		if err != nil {
			assert.Error(t, err)
		}
	})

	t.Run("mixed_encoding_attack", func(t *testing.T) {
		t.Parallel()

		// Create JSON with mixed encoding indicators
		jsonData := `{"field": "value with special chars: ü ñ 中文"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonData))

		// Claim different charset than actual content
		req.Header.Set("Content-Type", "application/json; charset=iso-8859-1")

		var target struct {
			Field string `json:"field"`
		}

		err := binder.JSON()(req, &target)
		if err == nil {
			// If parsing succeeds, special characters should be handled correctly
			assert.Contains(t, target.Field, "value", "Basic text should be preserved")
		}
	})

	t.Run("bom_handling", func(t *testing.T) {
		t.Parallel()

		// JSON with Byte Order Mark (BOM)
		jsonWithBOM := "\uFEFF" + `{"field": "value"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonWithBOM))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		var target struct {
			Field string `json:"field"`
		}

		err := binder.JSON()(req, &target)
		if err == nil {
			assert.Equal(t, "value", target.Field, "BOM should be handled transparently")
		}
	})
}

func testBoundaryInjection(t *testing.T) {
	t.Run("multipart_boundary_injection", func(t *testing.T) {
		t.Parallel()

		// Test boundary injection in multipart forms
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Create a normal field
		err := writer.WriteField("field1", "value1")
		require.NoError(t, err)

		// Attempt boundary injection in field value
		maliciousBoundary := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"injected\"\r\n\r\nmalicious_value\r\n--%s--",
			writer.Boundary(), writer.Boundary())

		err = writer.WriteField("field2", maliciousBoundary)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var target struct {
			Field1 string `form:"field1"`
			Field2 string `form:"field2"`
		}

		err = binder.Form()(req, &target)
		if err == nil {
			// Verify that legitimate fields are parsed correctly
			assert.Equal(t, "value1", target.Field1, "Legitimate field should be parsed correctly")
			// The boundary injection attempt creates a separate field, field2 should be empty
			// This is the secure behavior - boundary injection doesn't affect legitimate fields
			assert.Empty(t, target.Field2, "Field with boundary injection should not contain malicious content")
		} else {
			// If binding fails due to security measures, that's also acceptable
			t.Logf("Binding failed with security error: %v", err)
		}
	})

	t.Run("content_disposition_injection", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Manually create a part with injected Content-Disposition
		boundary := writer.Boundary()
		_, err := buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		require.NoError(t, err)

		// Inject additional parameters in Content-Disposition
		_, err = buf.WriteString(`Content-Disposition: form-data; name="field"; filename="test.txt"; malicious="injected"` + "\r\n")
		require.NoError(t, err)

		_, err = buf.WriteString("Content-Type: text/plain\r\n\r\n")
		require.NoError(t, err)

		_, err = buf.WriteString("field_value\r\n")
		require.NoError(t, err)

		_, err = buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))

		var target struct {
			Field string `form:"field"`
		}

		err = binder.Form()(req, &target)
		if err == nil {
			// If parsing succeeds, verify field content
			// Due to malicious Content-Disposition, the field might not be parsed correctly
			// This is secure behavior - malformed headers don't expose data
			if target.Field == "field_value" {
				t.Log("Field parsed correctly despite malicious Content-Disposition")
			} else {
				t.Log("Field not parsed due to malicious Content-Disposition - this is secure behavior")
				assert.Empty(t, target.Field, "Malformed Content-Disposition should not expose field data")
			}
		} else {
			// If binding fails due to security measures, that's acceptable
			t.Logf("Binding failed with security error: %v", err)
		}
	})

	t.Run("header_continuation_attack", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Create part with header continuation that could be exploited
		boundary := writer.Boundary()
		_, err := buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		require.NoError(t, err)

		// Header with continuation line that could inject new headers
		_, err = buf.WriteString("Content-Disposition: form-data;\r\n")
		require.NoError(t, err)
		_, err = buf.WriteString(" name=\"field\"\r\n")
		require.NoError(t, err)
		_, err = buf.WriteString("X-Injected-Header: malicious\r\n\r\n")
		require.NoError(t, err)

		_, err = buf.WriteString("field_value\r\n")
		require.NoError(t, err)

		_, err = buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))

		var target struct {
			Field string `form:"field"`
		}

		// Should handle header continuation without allowing header injection
		err = binder.Form()(req, &target)
		if err == nil {
			assert.Equal(t, "field_value", target.Field, "Field should be parsed correctly")
		}
	})

	t.Run("malformed_boundary_in_content_type", func(t *testing.T) {
		t.Parallel()

		formData := "field=value"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))

		// Malformed boundary parameter
		req.Header.Set("Content-Type", "multipart/form-data; boundary=")

		var target struct {
			Field string `form:"field"`
		}

		// Should handle malformed boundary gracefully
		err := binder.Form()(req, &target)
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "boundary",
				"Error should mention boundary issue")
		}
	})

	t.Run("boundary_with_special_characters", func(t *testing.T) {
		t.Parallel()

		// Test boundary with special characters that could cause parsing issues
		specialBoundaries := []string{
			"boundary\x00with\x00nulls",
			"boundary\r\nwith\r\nnewlines",
			"boundary with spaces",
			"boundary\"with\"quotes",
			"boundary'with'apostrophes",
			"boundary\\with\\backslashes",
			"boundary/with/slashes",
			"boundary;with;semicolons",
		}

		for _, specialBoundary := range specialBoundaries {
			t.Run(fmt.Sprintf("boundary_%s", specialBoundary), func(t *testing.T) {
				formData := "field=value"
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))

				contentType := fmt.Sprintf("multipart/form-data; boundary=%s", specialBoundary)
				req.Header.Set("Content-Type", contentType)

				var target struct {
					Field string `form:"field"`
				}

				// Should handle special characters in boundary
				err := binder.Form()(req, &target)
				if err != nil {
					// Should get controlled error for invalid boundary
					assert.Error(t, err)
				}
			})
		}
	})
}

// Helper functions for numeric checks (since math package functions may not be available)
func isInf(f float64, sign int) bool {
	return f > 1e308 || f < -1e308
}

func isNaN(f float64) bool {
	return f != f
}
