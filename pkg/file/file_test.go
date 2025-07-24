package file_test

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"hash"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/file"
)

func createFileHeader(filename string, content []byte) *multipart.FileHeader {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil
	}

	if _, err := part.Write(content); err != nil {
		return nil
	}

	if err := writer.Close(); err != nil {
		return nil
	}

	req := &http.Request{
		Method: "POST",
		Header: http.Header{
			"Content-Type": []string{writer.FormDataContentType()},
		},
		Body: io.NopCloser(body),
	}

	if err := req.ParseMultipartForm(32 << 20); err != nil {
		return nil
	}

	if req.MultipartForm != nil && req.MultipartForm.File != nil {
		if files, ok := req.MultipartForm.File["file"]; ok && len(files) > 0 {
			return files[0]
		}
	}

	return nil
}

func TestIsImage(t *testing.T) {
	t.Parallel()
	t.Run("jpeg image", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.jpg", []byte{0xFF, 0xD8, 0xFF})
		got := file.IsImage(fh)
		assert.True(t, got)
	})

	t.Run("png image", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		got := file.IsImage(fh)
		assert.True(t, got)
	})

	t.Run("text file", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.txt", []byte("hello world"))
		got := file.IsImage(fh)
		assert.False(t, got)
	})

	t.Run("gif image", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.gif", []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61})
		got := file.IsImage(fh)
		assert.True(t, got)
	})

	t.Run("nil file header", func(t *testing.T) {
		t.Parallel()
		var fh *multipart.FileHeader
		got := file.IsImage(fh)
		assert.False(t, got)
	})
}

func TestIsVideo(t *testing.T) {
	t.Parallel()
	t.Run("webm video", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.webm", []byte{0x1A, 0x45, 0xDF, 0xA3})
		got := file.IsVideo(fh)
		assert.True(t, got)
	})

	t.Run("not a video", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.jpg", []byte{0xFF, 0xD8, 0xFF})
		got := file.IsVideo(fh)
		assert.False(t, got)
	})

	t.Run("nil file header", func(t *testing.T) {
		t.Parallel()
		var fh *multipart.FileHeader
		got := file.IsVideo(fh)
		assert.False(t, got)
	})
}

func TestIsAudio(t *testing.T) {
	t.Parallel()
	t.Run("not audio", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.txt", []byte("hello"))
		got := file.IsAudio(fh)
		assert.False(t, got)
	})

	t.Run("nil file header", func(t *testing.T) {
		t.Parallel()
		var fh *multipart.FileHeader
		got := file.IsAudio(fh)
		assert.False(t, got)
	})
}

func TestIsPDF(t *testing.T) {
	t.Parallel()
	t.Run("pdf file", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.pdf", []byte("%PDF-1.4"))
		got := file.IsPDF(fh)
		assert.True(t, got)
	})

	t.Run("pdf by extension", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.pdf", []byte("not really pdf"))
		got := file.IsPDF(fh)
		assert.True(t, got)
	})

	t.Run("not pdf", func(t *testing.T) {
		t.Parallel()
		fh := createFileHeader("test.doc", []byte("word doc"))
		got := file.IsPDF(fh)
		assert.False(t, got)
	})
}

func TestGetExtension(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "jpeg extension",
			filename: "photo.jpg",
			want:     ".jpg",
		},
		{
			name:     "no extension",
			filename: "README",
			want:     "",
		},
		{
			name:     "multiple dots",
			filename: "archive.tar.gz",
			want:     ".gz",
		},
		{
			name:     "hidden file",
			filename: ".gitignore",
			want:     ".gitignore",
		},
		{
			name:     "nil file header",
			filename: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var fh *multipart.FileHeader
			if tt.filename != "" {
				fh = createFileHeader(tt.filename, []byte("content"))
			}
			got := file.GetExtension(fh)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetMIMEType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		filename    string
		content     []byte
		contentType string
		wantPrefix  string
		wantErr     bool
	}{
		{
			name:        "jpeg image",
			filename:    "test.jpg",
			content:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46},
			contentType: "image/jpeg",
			wantPrefix:  "image/jpeg",
			wantErr:     false,
		},
		{
			name:        "png image",
			filename:    "test.png",
			content:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			contentType: "image/png",
			wantPrefix:  "image/png",
			wantErr:     false,
		},
		{
			name:        "plain text",
			filename:    "test.txt",
			content:     []byte("Hello, World!"),
			contentType: "text/plain",
			wantPrefix:  "text/plain",
			wantErr:     false,
		},
		{
			name:        "pdf file",
			filename:    "test.pdf",
			content:     []byte("%PDF-1.4"),
			contentType: "application/pdf",
			wantPrefix:  "application/pdf",
			wantErr:     false,
		},
		{
			name:        "nil file header",
			filename:    "",
			content:     nil,
			contentType: "",
			wantPrefix:  "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var fh *multipart.FileHeader
			if tt.content != nil {
				fh = createFileHeader(tt.filename, tt.content)
			}
			got, err := file.GetMIMEType(fh)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Contains(t, got, tt.wantPrefix)
			}
		})
	}
}

func TestValidateSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		content  []byte
		maxBytes int64
		wantErr  bool
	}{
		{
			name:     "within limit",
			content:  []byte("small file"),
			maxBytes: 1024,
			wantErr:  false,
		},
		{
			name:     "exactly at limit",
			content:  []byte("exact"),
			maxBytes: 5,
			wantErr:  false,
		},
		{
			name:     "exceeds limit",
			content:  []byte("too large file"),
			maxBytes: 5,
			wantErr:  true,
		},
		{
			name:     "nil file header",
			content:  nil,
			maxBytes: 1024,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var fh *multipart.FileHeader
			if tt.content != nil {
				fh = createFileHeader("test.txt", tt.content)
			}
			err := file.ValidateSize(fh, tt.maxBytes)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMIMEType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		filename     string
		content      []byte
		allowedTypes []string
		wantErr      bool
	}{
		{
			name:         "allowed jpeg",
			filename:     "test.jpg",
			content:      []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46},
			allowedTypes: []string{"image/jpeg", "image/png"},
			wantErr:      false,
		},
		{
			name:         "not allowed type",
			filename:     "test.txt",
			content:      []byte("text content"),
			allowedTypes: []string{"image/jpeg", "image/png"},
			wantErr:      true,
		},
		{
			name:         "no restrictions",
			filename:     "test.txt",
			content:      []byte("any content"),
			allowedTypes: []string{},
			wantErr:      false,
		},
		{
			name:         "nil file header",
			content:      nil,
			allowedTypes: []string{"image/jpeg"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var fh *multipart.FileHeader
			if tt.content != nil {
				fh = createFileHeader(tt.filename, tt.content)
			}
			err := file.ValidateMIMEType(fh, tt.allowedTypes...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReadAll(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content []byte
		wantErr bool
	}{
		{
			name:    "read content",
			content: []byte("file content here"),
			wantErr: false,
		},
		{
			name:    "empty file",
			content: []byte{},
			wantErr: false,
		},
		{
			name:    "nil file header",
			content: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var fh *multipart.FileHeader
			if tt.content != nil {
				fh = createFileHeader("test.txt", tt.content)
			}
			got, err := file.ReadAll(fh)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.content, got)
			}
		})
	}
}

func TestHash(t *testing.T) {
	content := []byte("hello world")

	tests := []struct {
		name     string
		content  []byte
		hashFunc hash.Hash
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "sha256 hash",
			content:  content,
			hashFunc: sha256.New(),
			wantLen:  64,
			wantErr:  false,
		},
		{
			name:     "md5 hash",
			content:  content,
			hashFunc: md5.New(),
			wantLen:  32,
			wantErr:  false,
		},
		{
			name:     "default sha256",
			content:  content,
			hashFunc: nil,
			wantLen:  64,
			wantErr:  false,
		},
		{
			name:     "nil file header",
			content:  nil,
			hashFunc: sha256.New(),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var fh *multipart.FileHeader
			if tt.content != nil {
				fh = createFileHeader("test.txt", tt.content)
			}

			got, err := file.Hash(fh, tt.hashFunc)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, tt.wantLen)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "normal filename",
			filename: "document.pdf",
			want:     "document.pdf",
		},
		{
			name:     "path traversal attempt",
			filename: "../../../etc/passwd",
			want:     "passwd",
		},
		{
			name:     "windows path",
			filename: "C:\\Windows\\System32\\config.sys",
			want:     "config.sys",
		},
		{
			name:     "null bytes",
			filename: "file\x00name.txt",
			want:     "filename.txt",
		},
		{
			name:     "empty string",
			filename: "",
			want:     "unnamed",
		},
		{
			name:     "dot only",
			filename: ".",
			want:     "unnamed",
		},
		{
			name:     "double dot",
			filename: "..",
			want:     "unnamed",
		},
		{
			name:     "forward slash",
			filename: "/",
			want:     "unnamed",
		},
		{
			name:     "complex path",
			filename: "/var/www/../../../etc/hosts",
			want:     "hosts",
		},
		{
			name:     "mixed separators",
			filename: "path\\to/file\\document.txt",
			want:     "document.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := file.SanitizeFilename(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}
