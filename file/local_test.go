package file_test

import (
	"context"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/file"
)

func TestLocalStorage_Save(t *testing.T) {
	tempDir := t.TempDir()
	storage := file.NewLocalStorage("/files/")

	tests := []struct {
		name    string
		path    string
		content []byte
		wantErr bool
	}{
		{
			name:    "save simple file",
			path:    filepath.Join(tempDir, "test.txt"),
			content: []byte("hello world"),
			wantErr: false,
		},
		{
			name:    "save in nested directory",
			path:    filepath.Join(tempDir, "uploads", "docs", "report.pdf"),
			content: []byte("%PDF-1.4"),
			wantErr: false,
		},
		{
			name:    "invalid path traversal",
			path:    "../../../etc/passwd",
			content: []byte("malicious"),
			wantErr: true,
		},
		{
			name:    "nil file header",
			path:    filepath.Join(tempDir, "nil.txt"),
			content: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fh *multipart.FileHeader
			if tt.content != nil {
				fh = createFileHeader("test-file.txt", tt.content)
			}

			file, err := storage.Save(context.Background(), fh, tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, file)
			} else {
				require.NoError(t, err)
				require.NotNil(t, file)

				// Verify file metadata
				assert.Equal(t, "test-file.txt", file.Filename)
				assert.Equal(t, int64(len(tt.content)), file.Size)
				assert.Equal(t, ".txt", file.Extension)
				assert.Equal(t, tt.path, file.RelativePath)
				assert.NotEmpty(t, file.AbsolutePath)
				assert.NotEmpty(t, file.MIMEType)

				// Verify file was actually created
				data, err := os.ReadFile(file.AbsolutePath)
				require.NoError(t, err)
				assert.Equal(t, tt.content, data)

				// Verify file permissions
				info, err := os.Stat(file.AbsolutePath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
			}
		})
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	tempDir := t.TempDir()
	storage := file.NewLocalStorage("/files/")

	testFile := filepath.Join(tempDir, "delete-me.txt")
	err := os.WriteFile(testFile, []byte("delete me"), 0644)
	require.NoError(t, err)

	testDir := filepath.Join(tempDir, "test-dir")
	err = os.Mkdir(testDir, 0755)
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "delete existing file",
			path:    testFile,
			wantErr: false,
		},
		{
			name:    "delete non-existent file",
			path:    filepath.Join(tempDir, "not-exists.txt"),
			wantErr: true,
			errMsg:  "file not found",
		},
		{
			name:    "try to delete directory",
			path:    testDir,
			wantErr: true,
			errMsg:  "use DeleteDir instead",
		},
		{
			name:    "invalid path traversal",
			path:    "../../../etc/passwd",
			wantErr: true,
			errMsg:  "invalid path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Delete(context.Background(), tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				// Verify file was deleted
				_, err := os.Stat(tt.path)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestLocalStorage_DeleteDir(t *testing.T) {
	tempDir := t.TempDir()
	storage := file.NewLocalStorage("/files/")

	testDir := filepath.Join(tempDir, "test-dir")
	nestedDir := filepath.Join(testDir, "nested")
	err := os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(nestedDir, "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)

	singleFile := filepath.Join(tempDir, "single.txt")
	err = os.WriteFile(singleFile, []byte("single"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "delete directory with contents",
			path:    testDir,
			wantErr: false,
		},
		{
			name:    "delete non-existent directory",
			path:    filepath.Join(tempDir, "not-exists"),
			wantErr: true,
			errMsg:  "directory not found",
		},
		{
			name:    "try to delete file",
			path:    singleFile,
			wantErr: true,
			errMsg:  "not a directory",
		},
		{
			name:    "invalid path traversal",
			path:    "../../../etc",
			wantErr: true,
			errMsg:  "invalid path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.DeleteDir(context.Background(), tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				// Verify directory was deleted
				_, err := os.Stat(tt.path)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestLocalStorage_Exists(t *testing.T) {
	tempDir := t.TempDir()
	storage := file.NewLocalStorage("/files/")

	testFile := filepath.Join(tempDir, "exists.txt")
	err := os.WriteFile(testFile, []byte("I exist"), 0644)
	require.NoError(t, err)

	testDir := filepath.Join(tempDir, "existing-dir")
	err = os.Mkdir(testDir, 0755)
	require.NoError(t, err)

	tests := []struct {
		name   string
		path   string
		exists bool
	}{
		{
			name:   "existing file",
			path:   testFile,
			exists: true,
		},
		{
			name:   "existing directory",
			path:   testDir,
			exists: true,
		},
		{
			name:   "non-existent file",
			path:   filepath.Join(tempDir, "not-exists.txt"),
			exists: false,
		},
		{
			name:   "invalid path traversal",
			path:   "../../../etc/passwd",
			exists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := storage.Exists(context.Background(), tt.path)
			assert.Equal(t, tt.exists, exists)
		})
	}
}

func TestLocalStorage_List(t *testing.T) {
	tempDir := t.TempDir()
	storage := file.NewLocalStorage("/files/")

	// Create test directory structure
	testDir := filepath.Join(tempDir, "list-test")
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	subDir := filepath.Join(testDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	files := map[string][]byte{
		"file1.txt": []byte("content1"),
		"file2.pdf": []byte("%PDF-1.4"),
		"file3.jpg": []byte{0xFF, 0xD8, 0xFF},
	}

	for name, content := range files {
		err = os.WriteFile(filepath.Join(testDir, name), content, 0644)
		require.NoError(t, err)
	}

	// Create file in subdirectory (should not be listed)
	err = os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		dir         string
		wantEntries int
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "list directory contents",
			dir:         testDir,
			wantEntries: 4, // 3 files + 1 subdirectory
			wantErr:     false,
		},
		{
			name:        "list empty directory",
			dir:         filepath.Join(tempDir, "empty"),
			wantEntries: 0,
			wantErr:     true,
			errMsg:      "directory not found",
		},
		{
			name:        "list file as directory",
			dir:         filepath.Join(testDir, "file1.txt"),
			wantEntries: 0,
			wantErr:     true,
			errMsg:      "not a directory",
		},
		{
			name:        "invalid path traversal",
			dir:         "../../../etc",
			wantEntries: 0,
			wantErr:     true,
			errMsg:      "invalid path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := storage.List(context.Background(), tt.dir)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, entries, tt.wantEntries)

				// Verify entries
				for _, entry := range entries {
					assert.NotEmpty(t, entry.Name)
					assert.NotEmpty(t, entry.Path)

					if entry.IsDir {
						assert.Equal(t, int64(0), entry.Size)
						assert.Equal(t, "subdir", entry.Name)
					} else {
						assert.Greater(t, entry.Size, int64(0))
						assert.Contains(t, []string{"file1.txt", "file2.pdf", "file3.jpg"}, entry.Name)
					}
				}
			}
		})
	}
}

func TestLocalStorage_URL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{
			name:    "simple path",
			baseURL: "/files/",
			path:    "uploads/image.jpg",
			want:    "/files/uploads/image.jpg",
		},
		{
			name:    "path with leading slash",
			baseURL: "/files/",
			path:    "/uploads/image.jpg",
			want:    "/uploads/image.jpg", // Absolute paths are returned as-is
		},
		{
			name:    "base URL without trailing slash",
			baseURL: "/files",
			path:    "uploads/image.jpg",
			want:    "/files/uploads/image.jpg",
		},
		{
			name:    "empty base URL",
			baseURL: "",
			path:    "uploads/image.jpg",
			want:    "uploads/image.jpg",
		},
		{
			name:    "complex path",
			baseURL: "/static/files/",
			path:    "users/123/avatars/photo.png",
			want:    "/static/files/users/123/avatars/photo.png",
		},
		{
			name:    "path with dots",
			baseURL: "/files/",
			path:    "./uploads/../uploads/file.txt",
			want:    "/files/uploads/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := file.NewLocalStorage(tt.baseURL)
			got := storage.URL(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLocalStorage_Integration(t *testing.T) {
	// Integration test that uses multiple methods
	tempDir := t.TempDir()
	storage := file.NewLocalStorage("/files/")
	ctx := context.Background()

	content := []byte("integration test content")
	fh := createFileHeader("integration.txt", content)

	savePath := filepath.Join(tempDir, "integration", "test.txt")
	file, err := storage.Save(ctx, fh, savePath)
	require.NoError(t, err)
	require.NotNil(t, file)

	exists := storage.Exists(ctx, savePath)
	assert.True(t, exists)

	entries, err := storage.List(ctx, filepath.Join(tempDir, "integration"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "test.txt", entries[0].Name)
	assert.Equal(t, int64(len(content)), entries[0].Size)
	assert.False(t, entries[0].IsDir)

	// Get URL - since we're using absolute paths in tests, URL should return just the path
	url := storage.URL(file.RelativePath)
	assert.Equal(t, file.RelativePath, url)

	err = storage.Delete(ctx, savePath)
	require.NoError(t, err)

	exists = storage.Exists(ctx, savePath)
	assert.False(t, exists)

	err = storage.DeleteDir(ctx, filepath.Join(tempDir, "integration"))
	require.NoError(t, err)

	_, err = storage.List(ctx, filepath.Join(tempDir, "integration"))
	assert.Error(t, err)
}
