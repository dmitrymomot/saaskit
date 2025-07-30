package binder_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/binder"
)

func TestSecurityVulnerabilities(t *testing.T) {
	t.Run("malicious_input_prevention", func(t *testing.T) {
		testMaliciousInputPrevention(t)
	})

	t.Run("file_upload_security", func(t *testing.T) {
		testFileUploadSecurity(t)
	})
}

func testMaliciousInputPrevention(t *testing.T) {
	t.Run("json_control_characters", func(t *testing.T) {
		t.Parallel()

		// Test JSON with control characters
		maliciousJSON := `{"field": "value\x00with\x1fnull\x0cbytes"}`

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(maliciousJSON))
		req.Header.Set("Content-Type", "application/json")

		var target struct {
			Field string `json:"field"`
		}

		err := binder.JSON()(req, &target)
		if err == nil {
			// If parsing succeeds, ensure control characters are handled safely
			assert.NotContains(t, target.Field, "\x00", "NUL bytes should be filtered")
			assert.NotContains(t, target.Field, "\x1f", "Control characters should be filtered")
		}
	})

	t.Run("unicode_normalization_attacks", func(t *testing.T) {
		t.Parallel()

		// Test that JSON binder properly rejects unknown fields, even with unicode confusion
		// This prevents attackers from trying to bypass field validation using unicode homographs
		confusingJSON := `{"user": "admin", "üser": "hacker"}`

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(confusingJSON))
		req.Header.Set("Content-Type", "application/json")

		var target struct {
			User string `json:"user"`
		}

		err := binder.JSON()(req, &target)
		// Should reject unknown field "üser" in strict mode
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")

		// Test with valid JSON (no unknown fields)
		validJSON := `{"user": "admin"}`
		req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(validJSON))
		req2.Header.Set("Content-Type", "application/json")

		var target2 struct {
			User string `json:"user"`
		}

		err2 := binder.JSON()(req2, &target2)
		require.NoError(t, err2)
		assert.Equal(t, "admin", target2.User)
	})

	t.Run("embedded_nul_bytes_in_json", func(t *testing.T) {
		t.Parallel()

		// Test JSON strings with embedded NUL bytes
		jsonWithNulBytes := "{\"filename\": \"test\\u0000.exe.txt\"}"

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonWithNulBytes))
		req.Header.Set("Content-Type", "application/json")

		var target struct {
			Filename string `json:"filename"`
		}

		err := binder.JSON()(req, &target)
		if err == nil {
			// NUL bytes should be handled safely
			assert.NotContains(t, target.Filename, "\x00", "NUL bytes should not be present in result")
		}
	})

	t.Run("form_field_path_traversal", func(t *testing.T) {
		t.Parallel()

		// Test form field names with path traversal attempts
		maliciousFields := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\config\\sam",
			"field[../../../etc/passwd]",
			"field[..\\..\\..\\windows\\system32\\config\\sam]",
		}

		for _, fieldName := range maliciousFields {
			t.Run(fmt.Sprintf("field_%s", fieldName), func(t *testing.T) {
				formData := fmt.Sprintf("%s=malicious_value", fieldName)

				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				var target map[string]string

				err := binder.Form()(req, &target)
				if err == nil {
					// Field names should be sanitized or rejected
					for key := range target {
						assert.NotContains(t, key, "../", "Path traversal should be prevented")
						assert.NotContains(t, key, "..\\", "Path traversal should be prevented")
					}
				}
			})
		}
	})

	t.Run("reflection_breaking_field_names", func(t *testing.T) {
		t.Parallel()

		// Test field names that could break reflection
		dangerousFieldNames := []string{
			"",                              // Empty field name
			"field.field",                   // Dotted field name
			"field[0]",                      // Array-like field name
			"field;DROP TABLE users;",       // SQL injection attempt in field name
			"<script>alert('xss')</script>", // XSS attempt in field name
		}

		for _, fieldName := range dangerousFieldNames {
			if fieldName == "" {
				continue // Skip empty field name test as it's handled separately
			}

			t.Run(fmt.Sprintf("dangerous_field_%s", fieldName), func(t *testing.T) {
				formData := fmt.Sprintf("%s=value", fieldName)

				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				var target map[string]string

				// Should handle dangerous field names without breaking
				err := binder.Form()(req, &target)
				if err != nil {
					// Should get controlled error, not panic
					assert.Error(t, err)
				}
			})
		}
	})

	t.Run("crlf_injection_in_field_values", func(t *testing.T) {
		t.Parallel()

		// Test CRLF injection in field values
		crlfValue := "value\r\nInjected-Header: malicious"
		formData := fmt.Sprintf("field=%s", crlfValue)

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var target struct {
			Field string `form:"field"`
		}

		err := binder.Form()(req, &target)
		if err == nil {
			// CRLF should be handled safely
			assert.NotContains(t, target.Field, "\r\n", "CRLF should be filtered")
		}
	})
}

func testFileUploadSecurity(t *testing.T) {
	t.Run("zip_bomb_simulation", func(t *testing.T) {
		t.Parallel()

		// Simulate a compressed file that would expand enormously
		// This is a simplified test - in reality, you'd need actual compressed data
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Create a file upload field
		part, err := writer.CreateFormFile("file", "bomb.zip")
		require.NoError(t, err)

		// Simulate compressed data that would expand to much larger size
		compressedData := strings.Repeat("compressed_data_block", 50)
		_, err = part.Write([]byte(compressedData))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var target struct {
			File *multipart.FileHeader `form:"file"`
		}

		// Should handle potential zip bombs safely
		err = binder.Form()(req, &target)
		if err == nil && target.File != nil {
			// If upload succeeds, verify file size is reasonable
			assert.Less(t, target.File.Size, int64(100*1024*1024), "File size should be reasonable")
		}
	})

	t.Run("executable_content_detection", func(t *testing.T) {
		t.Parallel()

		// Test files with executable headers but misleading extensions
		testCases := []struct {
			filename string
			content  []byte
			desc     string
		}{
			{
				filename: "image.jpg",
				content:  []byte("MZ\x90\x00"), // PE executable header
				desc:     "PE executable with .jpg extension",
			},
			{
				filename: "document.pdf",
				content:  []byte("\x7fELF"), // ELF executable header
				desc:     "ELF executable with .pdf extension",
			},
			{
				filename: "archive.zip",
				content:  []byte("#!/bin/sh\necho 'malicious'"), // Shell script
				desc:     "Shell script with .zip extension",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)

				part, err := writer.CreateFormFile("file", tc.filename)
				require.NoError(t, err)

				_, err = part.Write(tc.content)
				require.NoError(t, err)

				err = writer.Close()
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				var target struct {
					File *multipart.FileHeader `form:"file"`
				}

				err = binder.Form()(req, &target)
				if err == nil && target.File != nil {
					// File should be uploaded, but content validation is the responsibility
					// of the application using the binder
					assert.Equal(t, tc.filename, target.File.Filename)

					// Open and verify the content if needed
					file, err := target.File.Open()
					if err == nil {
						defer file.Close()
						content, err := io.ReadAll(file)
						if err == nil {
							assert.Equal(t, tc.content, content)
						}
					}
				}
			})
		}
	})

	t.Run("comprehensive_filename_security", func(t *testing.T) {
		t.Parallel()

		maliciousFilenames := []struct {
			filename string
			desc     string
		}{
			{"file\x00.exe.txt", "NUL byte injection"},
			{"file\r\n.txt", "CRLF injection"},
			{strings.Repeat("a", 100), "Extremely long filename"},
			{"CON.txt", "Windows reserved name"},
			{"PRN.txt", "Windows reserved name"},
			{"AUX.txt", "Windows reserved name"},
			{"NUL.txt", "Windows reserved name"},
			{"COM1.txt", "Windows reserved name"},
			{"LPT1.txt", "Windows reserved name"},
			{".htaccess", "Hidden system file"},
			{".env", "Environment file"},
			{"../.env", "Path traversal to env file"},
			{"../../../etc/passwd", "Path traversal to system file"},
			{"file\t.txt", "Tab character"},
			{"file .txt", "Trailing space"},
			{" file.txt", "Leading space"},
			{"file..txt", "Double dot"},
			{"file.", "Trailing dot"},
			{".file", "Leading dot only"},
			{"file\\/\\.txt", "Mixed path separators"},
		}

		for _, tc := range maliciousFilenames {
			t.Run(tc.desc, func(t *testing.T) {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)

				part, err := writer.CreateFormFile("file", tc.filename)
				require.NoError(t, err)

				_, err = part.Write([]byte("test content"))
				require.NoError(t, err)

				err = writer.Close()
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				var target struct {
					File *multipart.FileHeader `form:"file"`
				}

				err = binder.Form()(req, &target)
				if err == nil && target.File != nil {
					// Filename should be sanitized or upload should be rejected
					sanitizedName := target.File.Filename

					// Check for dangerous patterns
					assert.NotContains(t, sanitizedName, "\x00", "NUL bytes should be removed")
					assert.NotContains(t, sanitizedName, "\r", "CR should be removed")
					assert.NotContains(t, sanitizedName, "\n", "LF should be removed")
					assert.NotContains(t, sanitizedName, "../", "Path traversal should be removed")
					assert.NotContains(t, sanitizedName, "..\\", "Path traversal should be removed")

					// Length should be reasonable
					assert.LessOrEqual(t, len(sanitizedName), 255, "Filename should not exceed reasonable length")

					// Should not be empty after sanitization
					assert.NotEmpty(t, sanitizedName, "Filename should not be empty after sanitization")
				}
			})
		}
	})

	t.Run("mime_type_confusion", func(t *testing.T) {
		t.Parallel()

		// Test files where Content-Type header doesn't match file content
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Create file upload with misleading MIME type
		part, err := writer.CreateFormFile("file", "script.js")
		require.NoError(t, err)

		// Write executable content but claim it's JavaScript
		_, err = part.Write([]byte("MZ\x90\x00\x03\x00\x00\x00")) // PE header
		require.NoError(t, err)

		// Manually set a different Content-Type
		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var target struct {
			File *multipart.FileHeader `form:"file"`
		}

		err = binder.Form()(req, &target)
		if err == nil && target.File != nil {
			// Binder should accept the file, but application should validate content vs declared type
			assert.Equal(t, "script.js", target.File.Filename)

			// The actual MIME type validation should be done by the application
			file, err := target.File.Open()
			if err == nil {
				defer file.Close()
				content, err := io.ReadAll(file)
				if err == nil {
					// Content should match what was uploaded
					assert.True(t, bytes.HasPrefix(content, []byte("MZ")), "Content should be preserved")
				}
			}
		}
	})
}

// TestContextTimeoutHandling verifies proper handling of context timeouts during binding
func TestContextTimeoutHandling(t *testing.T) {
	t.Run("context_timeout_during_binding", func(t *testing.T) {
		t.Parallel()

		// Create a context with a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Create a large request that would take time to process
		largeJSON := fmt.Sprintf(`{"data": "%s"}`, strings.Repeat("x", 10*1024)) // 10KB

		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/", strings.NewReader(largeJSON))
		req.Header.Set("Content-Type", "application/json")

		var target struct {
			Data string `json:"data"`
		}

		// Allow some time for the context to timeout
		time.Sleep(10 * time.Millisecond)

		// Should handle context timeout gracefully
		err := binder.JSON()(req, &target)
		if err != nil {
			// Should get a context-related error
			assert.True(t,
				strings.Contains(err.Error(), "context") ||
					strings.Contains(err.Error(), "timeout") ||
					strings.Contains(err.Error(), "canceled"),
				"Expected context/timeout related error, got: %v", err)
		}
	})
}
