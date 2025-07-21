package binder_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/binder"
)

// loadTestFile loads a file from testdata directory
func loadTestFile(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestFileUpload_RealWorldScenarios(t *testing.T) {
	t.Run("valid file uploads with real file headers", func(t *testing.T) {
		tests := []struct {
			name         string
			fieldName    string
			filename     string
			testDataFile string
			contentType  string
		}{
			{
				name:         "profile picture JPEG upload",
				fieldName:    "avatar",
				filename:     "profile.jpg",
				testDataFile: "valid.jpg",
				contentType:  "image/jpeg",
			},
			{
				name:         "logo PNG upload",
				fieldName:    "logo",
				filename:     "company-logo.png",
				testDataFile: "valid.png",
				contentType:  "image/png",
			},
			{
				name:         "document PDF upload",
				fieldName:    "document",
				filename:     "contract.pdf",
				testDataFile: "valid.pdf",
				contentType:  "application/pdf",
			},
			{
				name:         "text file upload",
				fieldName:    "notes",
				filename:     "readme.txt",
				testDataFile: "valid.txt",
				contentType:  "text/plain",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				content := loadTestFile(t, tt.testDataFile)

				type uploadStruct struct {
					File binder.FileUpload `file:"file"`
				}

				body, contentType := createMultipartFormWithField(t, tt.fieldName, fileData{
					filename: tt.filename,
					content:  content,
				})

				req := httptest.NewRequest(http.MethodPost, "/upload", body)
				req.Header.Set("Content-Type", contentType)

				file, err := binder.GetFile(req, tt.fieldName)
				require.NoError(t, err)
				require.NotNil(t, file)
				assert.Equal(t, tt.filename, file.Filename)
				assert.Equal(t, content, file.Content)
				assert.Equal(t, int64(len(content)), file.Size)
			})
		}
	})

	t.Run("invalid file uploads", func(t *testing.T) {
		tests := []struct {
			name         string
			fieldName    string
			filename     string
			testDataFile string
			description  string
		}{
			{
				name:         "broken JPEG file",
				fieldName:    "avatar",
				filename:     "broken.jpg",
				testDataFile: "broken.jpg",
				description:  "JPEG with invalid header bytes",
			},
			{
				name:         "broken PNG file",
				fieldName:    "logo",
				filename:     "broken.png",
				testDataFile: "broken.png",
				description:  "PNG with corrupted signature",
			},
			{
				name:         "malformed PDF",
				fieldName:    "document",
				filename:     "malformed.pdf",
				testDataFile: "malformed.pdf",
				description:  "PDF with random bytes",
			},
			{
				name:         "wrong extension",
				fieldName:    "image",
				filename:     "fake.jpg",
				testDataFile: "wrong_extension.jpg",
				description:  "Text file with .jpg extension",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				content := loadTestFile(t, tt.testDataFile)

				body, contentType := createMultipartFormWithField(t, tt.fieldName, fileData{
					filename: tt.filename,
					content:  content,
				})

				req := httptest.NewRequest(http.MethodPost, "/upload", body)
				req.Header.Set("Content-Type", contentType)

				// The upload itself should succeed - validation is application's responsibility
				file, err := binder.GetFile(req, tt.fieldName)
				require.NoError(t, err)
				require.NotNil(t, file)
				assert.Equal(t, tt.filename, file.Filename)
				assert.Equal(t, content, file.Content)
			})
		}
	})

	t.Run("edge case files", func(t *testing.T) {
		tests := []struct {
			name         string
			testDataFile string
			description  string
		}{
			{
				name:         "empty file",
				testDataFile: "empty.txt",
				description:  "0 byte file",
			},
			{
				name:         "file with null bytes",
				testDataFile: "null_bytes.txt",
				description:  "Text file containing null bytes",
			},
			{
				name:         "binary data in text file",
				testDataFile: "binary.txt",
				description:  "Binary content in .txt extension",
			},
			{
				name:         "hidden file",
				testDataFile: ".hidden",
				description:  "File starting with dot",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				content := loadTestFile(t, tt.testDataFile)

				body, contentType := createMultipartFormWithField(t, "file", fileData{
					filename: tt.testDataFile,
					content:  content,
				})

				req := httptest.NewRequest(http.MethodPost, "/upload", body)
				req.Header.Set("Content-Type", contentType)

				file, err := binder.GetFile(req, "file")
				require.NoError(t, err)
				require.NotNil(t, file)
				assert.Equal(t, tt.testDataFile, file.Filename)
				assert.Equal(t, content, file.Content)
				assert.Equal(t, int64(len(content)), file.Size)
			})
		}
	})

	t.Run("real-world multi-file upload scenarios", func(t *testing.T) {
		t.Run("user onboarding with avatar and documents", func(t *testing.T) {
			jpegContent := loadTestFile(t, "valid.jpg")
			pdfContent := loadTestFile(t, "valid.pdf")

			type onboardingForm struct {
				Avatar  binder.FileUpload  `file:"avatar"`
				Resume  *binder.FileUpload `file:"resume"`
				IDProof *binder.FileUpload `file:"id_proof"`
			}

			body, contentType := createMultipartForm(t, map[string][]fileData{
				"avatar":   {{filename: "profile.jpg", content: jpegContent}},
				"resume":   {{filename: "resume.pdf", content: pdfContent}},
				"id_proof": {{filename: "passport.pdf", content: pdfContent}},
			})

			req := httptest.NewRequest(http.MethodPost, "/onboarding", body)
			req.Header.Set("Content-Type", contentType)

			var result onboardingForm
			bindFunc := binder.File()
			err := bindFunc(req, &result)

			require.NoError(t, err)
			assert.Equal(t, "profile.jpg", result.Avatar.Filename)
			assert.Equal(t, jpegContent, result.Avatar.Content)

			require.NotNil(t, result.Resume)
			assert.Equal(t, "resume.pdf", result.Resume.Filename)

			require.NotNil(t, result.IDProof)
			assert.Equal(t, "passport.pdf", result.IDProof.Filename)
		})

		t.Run("product gallery upload", func(t *testing.T) {
			jpegContent := loadTestFile(t, "valid.jpg")
			pngContent := loadTestFile(t, "valid.png")

			type productForm struct {
				MainImage binder.FileUpload   `file:"main_image"`
				Gallery   []binder.FileUpload `file:"gallery"`
			}

			body, contentType := createMultipartForm(t, map[string][]fileData{
				"main_image": {{filename: "product-main.jpg", content: jpegContent}},
				"gallery": {
					{filename: "product-1.jpg", content: jpegContent},
					{filename: "product-2.png", content: pngContent},
					{filename: "product-3.jpg", content: jpegContent},
				},
			})

			req := httptest.NewRequest(http.MethodPost, "/products", body)
			req.Header.Set("Content-Type", contentType)

			var result productForm
			bindFunc := binder.File()
			err := bindFunc(req, &result)

			require.NoError(t, err)
			assert.Equal(t, "product-main.jpg", result.MainImage.Filename)
			require.Len(t, result.Gallery, 3)
			assert.Equal(t, "product-1.jpg", result.Gallery[0].Filename)
			assert.Equal(t, "product-2.png", result.Gallery[1].Filename)
			assert.Equal(t, "product-3.jpg", result.Gallery[2].Filename)
		})
	})
}

// createMultipartFormWithField is a helper for single field uploads
func createMultipartFormWithField(t *testing.T, fieldName string, file fileData) (*bytes.Buffer, string) {
	t.Helper()
	return createMultipartForm(t, map[string][]fileData{
		fieldName: {file},
	})
}
