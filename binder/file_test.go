package binder_test

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/binder"
)

func TestFileUpload_ContentType(t *testing.T) {
	tests := []struct {
		name     string
		upload   binder.FileUpload
		expected string
	}{
		{
			name: "content type from header",
			upload: binder.FileUpload{
				Filename: "test.txt",
				Header: textproto.MIMEHeader{
					"Content-Type": []string{"application/pdf"},
				},
			},
			expected: "application/pdf",
		},
		{
			name: "content type from header with params",
			upload: binder.FileUpload{
				Filename: "test.txt",
				Header: textproto.MIMEHeader{
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
			},
			expected: "text/plain",
		},
		{
			name: "content type from file extension",
			upload: binder.FileUpload{
				Filename: "document.pdf",
				Header:   textproto.MIMEHeader{},
			},
			expected: "application/pdf",
		},
		{
			name: "no content type or extension",
			upload: binder.FileUpload{
				Filename: "noext",
				Header:   textproto.MIMEHeader{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.upload.ContentType())
		})
	}
}

func TestFile(t *testing.T) {
	type testFile struct {
		Avatar   binder.FileUpload    `file:"avatar"`
		Document *binder.FileUpload   `file:"document"`
		Gallery  []binder.FileUpload  `file:"gallery"`
		Photos   []*binder.FileUpload `file:"photos"`
		Skip     binder.FileUpload    `file:"-"`
		NoTag    binder.FileUpload
		private  binder.FileUpload `file:"private"`
	}

	t.Run("single file upload", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"avatar": {{filename: "avatar.jpg", content: []byte("avatar data")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "avatar.jpg", result.Avatar.Filename)
		assert.Equal(t, int64(11), result.Avatar.Size)
		assert.Equal(t, []byte("avatar data"), result.Avatar.Content)
	})

	t.Run("optional file upload present", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"document": {{filename: "doc.pdf", content: []byte("pdf content")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.NotNil(t, result.Document)
		assert.Equal(t, "doc.pdf", result.Document.Filename)
		assert.Equal(t, []byte("pdf content"), result.Document.Content)
	})

	t.Run("optional file upload missing", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"avatar": {{filename: "avatar.jpg", content: []byte("data")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Nil(t, result.Document)
	})

	t.Run("multiple files", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"gallery": {
				{filename: "img1.jpg", content: []byte("image1")},
				{filename: "img2.jpg", content: []byte("image2")},
			},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.Len(t, result.Gallery, 2)
		assert.Equal(t, "img1.jpg", result.Gallery[0].Filename)
		assert.Equal(t, "img2.jpg", result.Gallery[1].Filename)
	})

	t.Run("multiple files with pointers", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"photos": {
				{filename: "photo1.png", content: []byte("photo1")},
				{filename: "photo2.png", content: []byte("photo2")},
			},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.Len(t, result.Photos, 2)
		assert.Equal(t, "photo1.png", result.Photos[0].Filename)
		assert.Equal(t, "photo2.png", result.Photos[1].Filename)
	})

	t.Run("skip non-multipart request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("not multipart"))
		req.Header.Set("Content-Type", "application/json")

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Avatar.Filename)
	})

	t.Run("skip request without content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("data"))

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Avatar.Filename)
	})

	t.Run("error on nil pointer", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{})
		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		bindFunc := binder.File()
		err := bindFunc(req, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a non-nil pointer")
	})

	t.Run("error on non-pointer", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{})
		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a non-nil pointer")
	})

	t.Run("error on non-struct", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{})
		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result string
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a pointer to struct")
	})

	t.Run("skip fields with dash tag", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"skip": {{filename: "skip.txt", content: []byte("should not bind")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result testFile
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Skip.Filename)
	})

	t.Run("unsupported field type", func(t *testing.T) {
		type invalidStruct struct {
			File string `file:"file"`
		}

		body, contentType := createMultipartForm(t, map[string][]fileData{
			"file": {{filename: "test.txt", content: []byte("data")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result invalidStruct
		bindFunc := binder.File()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type for file field")
	})
}

func TestGetFile(t *testing.T) {
	t.Run("get single file", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"avatar": {{filename: "avatar.jpg", content: []byte("avatar data")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		file, err := binder.GetFile(req, "avatar")
		require.NoError(t, err)
		require.NotNil(t, file)
		assert.Equal(t, "avatar.jpg", file.Filename)
		assert.Equal(t, []byte("avatar data"), file.Content)
	})

	t.Run("missing file returns nil", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		file, err := binder.GetFile(req, "missing")
		require.NoError(t, err)
		assert.Nil(t, file)
	})

	t.Run("multiple files returns first", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"file": {
				{filename: "first.txt", content: []byte("first")},
				{filename: "second.txt", content: []byte("second")},
			},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		file, err := binder.GetFile(req, "file")
		require.NoError(t, err)
		require.NotNil(t, file)
		assert.Equal(t, "first.txt", file.Filename)
	})
}

func TestGetFiles(t *testing.T) {
	t.Run("get multiple files", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"photos": {
				{filename: "photo1.jpg", content: []byte("photo1")},
				{filename: "photo2.jpg", content: []byte("photo2")},
				{filename: "photo3.jpg", content: []byte("photo3")},
			},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		files, err := binder.GetFiles(req, "photos")
		require.NoError(t, err)
		require.Len(t, files, 3)
		assert.Equal(t, "photo1.jpg", files[0].Filename)
		assert.Equal(t, "photo2.jpg", files[1].Filename)
		assert.Equal(t, "photo3.jpg", files[2].Filename)
	})

	t.Run("no files returns empty slice", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		files, err := binder.GetFiles(req, "missing")
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

func TestGetAllFiles(t *testing.T) {
	t.Run("get all files from form", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"avatar": {{filename: "avatar.jpg", content: []byte("avatar")}},
			"gallery": {
				{filename: "img1.jpg", content: []byte("img1")},
				{filename: "img2.jpg", content: []byte("img2")},
			},
			"document": {{filename: "doc.pdf", content: []byte("pdf")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		files, err := binder.GetAllFiles(req)
		require.NoError(t, err)
		require.Len(t, files, 3)

		assert.Len(t, files["avatar"], 1)
		assert.Len(t, files["gallery"], 2)
		assert.Len(t, files["document"], 1)

		assert.Equal(t, "avatar.jpg", files["avatar"][0].Filename)
		assert.Equal(t, "img1.jpg", files["gallery"][0].Filename)
		assert.Equal(t, "doc.pdf", files["document"][0].Filename)
	})

	t.Run("empty form returns empty map", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		files, err := binder.GetAllFiles(req)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

func TestStreamFile(t *testing.T) {
	t.Run("stream file content", func(t *testing.T) {
		expectedContent := []byte("streamed content")
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"video": {{filename: "video.mp4", content: expectedContent}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var receivedContent []byte
		var receivedHeader *binder.FileHeader

		err := binder.StreamFile(req, "video", func(reader io.Reader, header *binder.FileHeader) error {
			receivedHeader = header
			content, err := io.ReadAll(reader)
			if err != nil {
				return err
			}
			receivedContent = content
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, expectedContent, receivedContent)
		require.NotNil(t, receivedHeader)
		assert.Equal(t, "video.mp4", receivedHeader.Filename)
		assert.Equal(t, int64(len(expectedContent)), receivedHeader.Size)
	})

	t.Run("handler error propagation", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"file": {{filename: "test.txt", content: []byte("data")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		expectedErr := errors.New("handler error")
		err := binder.StreamFile(req, "file", func(reader io.Reader, header *binder.FileHeader) error {
			return expectedErr
		})

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("missing file error", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		err := binder.StreamFile(req, "missing", func(reader io.Reader, header *binder.FileHeader) error {
			return nil
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get file")
	})
}

func TestGetFileWithLimit(t *testing.T) {
	t.Run("file within limit", func(t *testing.T) {
		body, contentType := createMultipartForm(t, map[string][]fileData{
			"file": {{filename: "small.txt", content: []byte("small file")}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		file, err := binder.GetFileWithLimit(req, "file", 1<<20) // 1MB limit
		require.NoError(t, err)
		require.NotNil(t, file)
		assert.Equal(t, "small.txt", file.Filename)
		assert.Equal(t, []byte("small file"), file.Content)
	})

	t.Run("custom limit applied", func(t *testing.T) {
		// Create a large file that would exceed a small limit
		largeContent := make([]byte, 1024) // 1KB
		for i := range largeContent {
			largeContent[i] = 'a'
		}

		body, contentType := createMultipartForm(t, map[string][]fileData{
			"file": {{filename: "large.txt", content: largeContent}},
		})

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		// This should still work as multipart parsing happens with the limit
		file, err := binder.GetFileWithLimit(req, "file", 10<<20) // 10MB limit
		require.NoError(t, err)
		require.NotNil(t, file)
		assert.Len(t, file.Content, 1024)
	})
}

// Helper types and functions

type fileData struct {
	filename string
	content  []byte
}

func createMultipartForm(t *testing.T, files map[string][]fileData) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for fieldName, fieldFiles := range files {
		for _, file := range fieldFiles {
			part, err := writer.CreateFormFile(fieldName, file.filename)
			require.NoError(t, err)
			_, err = part.Write(file.content)
			require.NoError(t, err)
		}
	}

	err := writer.Close()
	require.NoError(t, err)

	return body, writer.FormDataContentType()
}
