package binder_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/binder"
)

func TestFile_PathTraversalPrevention(t *testing.T) {
	tests := []struct {
		name              string
		maliciousFilename string
		expectedFilename  string
		description       string
	}{
		{
			name:              "unix path traversal attempt",
			maliciousFilename: "../../../etc/passwd",
			expectedFilename:  "passwd",
			description:       "Should extract only the base filename",
		},
		{
			name:              "windows path traversal attempt",
			maliciousFilename: "..\\..\\..\\windows\\system32\\config\\sam",
			expectedFilename:  "sam",
			description:       "Should handle Windows-style paths",
		},
		{
			name:              "mixed path separators",
			maliciousFilename: "../..\\../etc/passwd",
			expectedFilename:  "passwd",
			description:       "Should handle mixed path separators",
		},
		{
			name:              "absolute unix path",
			maliciousFilename: "/etc/passwd",
			expectedFilename:  "passwd",
			description:       "Should remove absolute path",
		},
		{
			name:              "absolute windows path",
			maliciousFilename: "C:\\Windows\\System32\\drivers\\etc\\hosts",
			expectedFilename:  "hosts",
			description:       "Should handle Windows absolute paths",
		},
		{
			name:              "url encoded traversal",
			maliciousFilename: "..%2F..%2F..%2Fetc%2Fpasswd",
			expectedFilename:  "..%2F..%2F..%2Fetc%2Fpasswd",
			description:       "URL encoding should not bypass sanitization",
		},
		{
			name:              "filename with null-like pattern",
			maliciousFilename: "innocent.txt.exe",
			expectedFilename:  "innocent.txt.exe",
			description:       "Normal double extension should work",
		},
		{
			name:              "single space filename",
			maliciousFilename: " ",
			expectedFilename:  " ",
			description:       "Single space should be preserved",
		},
		{
			name:              "dot filename",
			maliciousFilename: ".",
			expectedFilename:  "unnamed",
			description:       "Single dot should default to 'unnamed'",
		},
		{
			name:              "double dot filename",
			maliciousFilename: "..",
			expectedFilename:  "unnamed",
			description:       "Double dot should default to 'unnamed'",
		},
		{
			name:              "hidden file",
			maliciousFilename: ".htaccess",
			expectedFilename:  ".htaccess",
			description:       "Hidden files should be preserved",
		},
		{
			name:              "filename with spaces",
			maliciousFilename: "my file.txt",
			expectedFilename:  "my file.txt",
			description:       "Spaces should be preserved",
		},
		{
			name:              "unicode filename",
			maliciousFilename: "文档.pdf",
			expectedFilename:  "文档.pdf",
			description:       "Unicode characters should be preserved",
		},
		{
			name:              "filename with encoded slashes",
			maliciousFilename: "file/with/slashes.txt",
			expectedFilename:  "slashes.txt",
			description:       "Path components should be removed, keeping only basename",
		},
		{
			name:              "filename with backslashes",
			maliciousFilename: "file\\with\\backslashes.txt",
			expectedFilename:  "backslashes.txt",
			description:       "Path components should be removed, keeping only basename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form with malicious filename
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", tt.maliciousFilename)
			require.NoError(t, err)

			testContent := []byte("test file content")
			_, err = part.Write(testContent)
			require.NoError(t, err)

			err = writer.Close()
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Test with GetFile
			file, err := binder.GetFile(req, "file")
			require.NoError(t, err)
			require.NotNil(t, file)

			assert.Equal(t, tt.expectedFilename, file.Filename, tt.description)
			assert.Equal(t, testContent, file.Content)
		})
	}
}

func TestFile_PathTraversalWithBinder(t *testing.T) {
	// Test path traversal prevention with the File binder
	type UploadRequest struct {
		Document binder.FileUpload `file:"document"`
	}

	maliciousFilename := "../../../etc/passwd"
	expectedFilename := "passwd"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("document", maliciousFilename)
	require.NoError(t, err)

	_, err = part.Write([]byte("sensitive data"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var upload UploadRequest
	fileBinder := binder.File()
	err = fileBinder(req, &upload)
	require.NoError(t, err)

	assert.Equal(t, expectedFilename, upload.Document.Filename)
}

func TestFile_SecurityEdgeCases(t *testing.T) {
	t.Run("null bytes rejected by http library", func(t *testing.T) {
		// The Go standard library's multipart parser rejects null bytes in filenames
		// This provides an additional layer of security
		filename := "file\x00.txt"
		body := createSecurityTestForm(t, "file", filename, []byte("content"))

		req := httptest.NewRequest(http.MethodPost, "/upload", body.body)
		req.Header.Set("Content-Type", body.contentType)

		file, err := binder.GetFile(req, "file")
		// The multipart parser should reject this
		require.Error(t, err)
		require.Nil(t, file)
		assert.Contains(t, err.Error(), "malformed MIME header")
	})

	t.Run("very long filename", func(t *testing.T) {
		// Create a filename that's 300 characters long
		longName := "a"
		for i := 0; i < 295; i++ {
			longName += "b"
		}
		longName += ".txt"

		body := createSecurityTestForm(t, "file", longName, []byte("content"))

		req := httptest.NewRequest(http.MethodPost, "/upload", body.body)
		req.Header.Set("Content-Type", body.contentType)

		file, err := binder.GetFile(req, "file")
		require.NoError(t, err)
		require.NotNil(t, file)

		// Long filename should be preserved
		assert.Equal(t, longName, file.Filename)
	})

	t.Run("windows reserved names", func(t *testing.T) {
		// Windows reserved names should be allowed (sanitization doesn't block them)
		// It's up to the application to handle these appropriately for the target OS
		reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "LPT1"}

		for _, name := range reservedNames {
			t.Run(name, func(t *testing.T) {
				body := createSecurityTestForm(t, "file", name, []byte("content"))

				req := httptest.NewRequest(http.MethodPost, "/upload", body.body)
				req.Header.Set("Content-Type", body.contentType)

				file, err := binder.GetFile(req, "file")
				require.NoError(t, err)
				require.NotNil(t, file)

				assert.Equal(t, name, file.Filename)
			})
		}
	})
}

func TestFile_ContentTypeDetection(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		content          []byte
		clientMimeType   string
		expectedDetected string
		description      string
	}{
		{
			name:             "JPEG with correct extension",
			filename:         "image.jpg",
			content:          []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF"),
			clientMimeType:   "image/jpeg",
			expectedDetected: "image/jpeg",
			description:      "JPEG magic bytes should be detected",
		},
		{
			name:             "JPEG with wrong extension",
			filename:         "image.txt",
			content:          []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF"),
			clientMimeType:   "text/plain",
			expectedDetected: "image/jpeg",
			description:      "Content detection should override extension",
		},
		{
			name:             "PNG file",
			filename:         "image.png",
			content:          []byte("\x89PNG\r\n\x1a\n"),
			clientMimeType:   "image/png",
			expectedDetected: "image/png",
			description:      "PNG magic bytes should be detected",
		},
		{
			name:             "PDF file",
			filename:         "document.pdf",
			content:          []byte("%PDF-1.4"),
			clientMimeType:   "application/pdf",
			expectedDetected: "application/pdf",
			description:      "PDF header should be detected",
		},
		{
			name:             "ZIP file",
			filename:         "archive.zip",
			content:          []byte("PK\x03\x04"),
			clientMimeType:   "application/zip",
			expectedDetected: "application/zip",
			description:      "ZIP magic bytes should be detected",
		},
		{
			name:             "Executable disguised as image",
			filename:         "photo.jpg",
			content:          []byte("MZ\x90\x00"), // DOS/PE executable header
			clientMimeType:   "image/jpeg",
			expectedDetected: "application/octet-stream", // Go's DetectContentType returns this for executables
			description:      "Executable should be detected despite wrong extension",
		},
		{
			name:             "Empty file",
			filename:         "empty.txt",
			content:          []byte{},
			clientMimeType:   "text/plain",
			expectedDetected: "application/octet-stream",
			description:      "Empty file should return default type",
		},
		{
			name:             "HTML file",
			filename:         "page.html",
			content:          []byte("<!DOCTYPE html><html>"),
			clientMimeType:   "text/html",
			expectedDetected: "text/html; charset=utf-8",
			description:      "HTML should be detected with charset",
		},
		{
			name:             "Plain text",
			filename:         "readme.txt",
			content:          []byte("This is a plain text file"),
			clientMimeType:   "text/plain",
			expectedDetected: "text/plain; charset=utf-8",
			description:      "Plain text should be detected with charset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, tt.filename))
			h.Set("Content-Type", tt.clientMimeType)

			part, err := writer.CreatePart(h)
			require.NoError(t, err)

			_, err = part.Write(tt.content)
			require.NoError(t, err)

			err = writer.Close()
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Get file
			file, err := binder.GetFile(req, "file")
			require.NoError(t, err)
			require.NotNil(t, file)

			// Test content type detection
			detected := file.DetectContentType()
			assert.Equal(t, tt.expectedDetected, detected, tt.description)

			// Verify that ContentType() still returns client-provided type
			assert.Equal(t, tt.clientMimeType, file.ContentType())
		})
	}
}

// Helper function to create multipart form for security tests
type securityTestForm struct {
	body        *bytes.Buffer
	contentType string
}

func createSecurityTestForm(t *testing.T, fieldName, filename string, content []byte) securityTestForm {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, filename)
	require.NoError(t, err)

	_, err = part.Write(content)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	return securityTestForm{
		body:        body,
		contentType: writer.FormDataContentType(),
	}
}
