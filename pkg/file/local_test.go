package file_test

import (
	"context"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/file"
)

func TestLocalStorage_Save(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	storage, err := file.NewLocalStorage(tempDir, "/files/")
	require.NoError(t, err)

	t.Run("save simple file", func(t *testing.T) {
		t.Parallel()
		content := []byte("hello world")
		fh := createFileHeader("test-file.txt", content)
		path := "test.txt"

		file, err := storage.Save(context.Background(), fh, path)
		require.NoError(t, err)
		require.NotNil(t, file)

		assert.Equal(t, "test-file.txt", file.Filename)
		assert.Equal(t, int64(len(content)), file.Size)
		assert.Equal(t, ".txt", file.Extension)
		assert.Equal(t, path, file.RelativePath)
		assert.NotEmpty(t, file.AbsolutePath)
		assert.NotEmpty(t, file.MIMEType)

		data, err := os.ReadFile(file.AbsolutePath)
		require.NoError(t, err)
		assert.Equal(t, content, data)

		info, err := os.Stat(file.AbsolutePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	})

	t.Run("save in nested directory", func(t *testing.T) {
		t.Parallel()
		content := []byte("%PDF-1.4")
		fh := createFileHeader("test-file.txt", content)
		path := "uploads/docs/report.pdf"

		file, err := storage.Save(context.Background(), fh, path)
		require.NoError(t, err)
		require.NotNil(t, file)

		assert.Equal(t, "test-file.txt", file.Filename)
		assert.Equal(t, int64(len(content)), file.Size)
		assert.Equal(t, ".txt", file.Extension)
		assert.Equal(t, path, file.RelativePath)
		assert.NotEmpty(t, file.AbsolutePath)
		assert.NotEmpty(t, file.MIMEType)

		data, err := os.ReadFile(file.AbsolutePath)
		require.NoError(t, err)
		assert.Equal(t, content, data)

		info, err := os.Stat(file.AbsolutePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		t.Parallel()
		content := []byte("malicious")
		fh := createFileHeader("test-file.txt", content)
		path := "../../../etc/passwd"

		file, err := storage.Save(context.Background(), fh, path)
		assert.Error(t, err)
		assert.Nil(t, file)
	})

	t.Run("nil file header", func(t *testing.T) {
		t.Parallel()
		var fh *multipart.FileHeader
		path := "nil.txt"

		file, err := storage.Save(context.Background(), fh, path)
		assert.Error(t, err)
		assert.Nil(t, file)
	})
}

func TestLocalStorage_Delete(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	storage, err := file.NewLocalStorage(tempDir, "/files/")
	require.NoError(t, err)

	t.Run("delete existing file", func(t *testing.T) {
		t.Parallel()
		testFile := "delete-me.txt"
		filePath := filepath.Join(tempDir, testFile)
		err := os.WriteFile(filePath, []byte("delete me"), 0644)
		require.NoError(t, err)

		// Cleanup in case delete fails
		t.Cleanup(func() {
			_ = os.Remove(filePath)
		})

		err = storage.Delete(context.Background(), testFile)
		assert.NoError(t, err)

		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		t.Parallel()
		path := "not-exists.txt"

		err := storage.Delete(context.Background(), path)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrFileNotFound))
	})

	t.Run("try to delete directory", func(t *testing.T) {
		t.Parallel()
		testDir := "test-dir"
		dirPath := filepath.Join(tempDir, testDir)
		err := os.Mkdir(dirPath, 0755)
		require.NoError(t, err)

		// Cleanup directory
		t.Cleanup(func() {
			_ = os.RemoveAll(dirPath)
		})

		err = storage.Delete(context.Background(), testDir)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrIsDirectory))
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		t.Parallel()
		err := storage.Delete(context.Background(), "../../../etc/passwd")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidPath))
	})
}

func TestLocalStorage_DeleteDir(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	storage, err := file.NewLocalStorage(tempDir, "/files/")
	require.NoError(t, err)

	t.Run("delete directory with contents", func(t *testing.T) {
		t.Parallel()
		testDir := "test-dir"
		testDirAbs := filepath.Join(tempDir, testDir)
		nestedDir := filepath.Join(testDirAbs, "nested")
		err := os.MkdirAll(nestedDir, 0755)
		require.NoError(t, err)

		// Cleanup entire directory structure
		t.Cleanup(func() {
			_ = os.RemoveAll(testDirAbs)
		})

		err = os.WriteFile(filepath.Join(testDirAbs, "file1.txt"), []byte("content1"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(nestedDir, "file2.txt"), []byte("content2"), 0644)
		require.NoError(t, err)

		err = storage.DeleteDir(context.Background(), testDir)
		assert.NoError(t, err)

		_, err = os.Stat(testDirAbs)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete non-existent directory", func(t *testing.T) {
		t.Parallel()
		path := "not-exists"

		err := storage.DeleteDir(context.Background(), path)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrDirectoryNotFound))
	})

	t.Run("try to delete file", func(t *testing.T) {
		t.Parallel()
		singleFile := "single.txt"
		filePath := filepath.Join(tempDir, singleFile)
		err := os.WriteFile(filePath, []byte("single"), 0644)
		require.NoError(t, err)

		// Cleanup file
		t.Cleanup(func() {
			_ = os.Remove(filePath)
		})

		err = storage.DeleteDir(context.Background(), singleFile)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrNotDirectory))
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		t.Parallel()
		err := storage.DeleteDir(context.Background(), "../../../etc")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidPath))
	})
}

func TestLocalStorage_Exists(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	storage, err := file.NewLocalStorage(tempDir, "/files/")
	require.NoError(t, err)

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()
		testFile := "exists.txt"
		filePath := filepath.Join(tempDir, testFile)
		err := os.WriteFile(filePath, []byte("I exist"), 0644)
		require.NoError(t, err)

		// Cleanup file
		t.Cleanup(func() {
			_ = os.Remove(filePath)
		})

		exists := storage.Exists(context.Background(), testFile)
		assert.True(t, exists)
	})

	t.Run("existing directory", func(t *testing.T) {
		t.Parallel()
		testDir := "existing-dir"
		dirPath := filepath.Join(tempDir, testDir)
		err := os.Mkdir(dirPath, 0755)
		require.NoError(t, err)

		// Cleanup directory
		t.Cleanup(func() {
			_ = os.RemoveAll(dirPath)
		})

		exists := storage.Exists(context.Background(), testDir)
		assert.True(t, exists)
	})

	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()
		path := "not-exists.txt"

		exists := storage.Exists(context.Background(), path)
		assert.False(t, exists)
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		t.Parallel()
		exists := storage.Exists(context.Background(), "../../../etc/passwd")
		assert.False(t, exists)
	})
}

func TestLocalStorage_List(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	storage, err := file.NewLocalStorage(tempDir, "/files/")
	require.NoError(t, err)

	t.Run("list directory contents", func(t *testing.T) {
		t.Parallel()
		testDir := "list-test"
		testDirAbs := filepath.Join(tempDir, testDir)
		err := os.MkdirAll(testDirAbs, 0755)
		require.NoError(t, err)

		// Cleanup entire directory structure
		t.Cleanup(func() {
			_ = os.RemoveAll(testDirAbs)
		})

		subDir := filepath.Join(testDirAbs, "subdir")
		err = os.Mkdir(subDir, 0755)
		require.NoError(t, err)

		files := map[string][]byte{
			"file1.txt": []byte("content1"),
			"file2.pdf": []byte("%PDF-1.4"),
			"file3.jpg": []byte{0xFF, 0xD8, 0xFF},
		}

		for name, content := range files {
			err = os.WriteFile(filepath.Join(testDirAbs, name), content, 0644)
			require.NoError(t, err)
		}

		err = os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644)
		require.NoError(t, err)

		entries, err := storage.List(context.Background(), testDir)
		require.NoError(t, err)
		assert.Len(t, entries, 4)

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
	})

	t.Run("list empty directory", func(t *testing.T) {
		t.Parallel()
		dir := "empty"

		entries, err := storage.List(context.Background(), dir)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrDirectoryNotFound))
		assert.Len(t, entries, 0)
	})

	t.Run("list file as directory", func(t *testing.T) {
		t.Parallel()
		testFile := "test-file.txt"
		filePath := filepath.Join(tempDir, testFile)
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)

		// Cleanup file
		t.Cleanup(func() {
			_ = os.Remove(filePath)
		})

		entries, err := storage.List(context.Background(), testFile)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrNotDirectory))
		assert.Len(t, entries, 0)
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		t.Parallel()
		entries, err := storage.List(context.Background(), "../../../etc")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidPath))
		assert.Len(t, entries, 0)
	})
}

func TestLocalStorage_URL(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			storage, err := file.NewLocalStorage(t.TempDir(), tt.baseURL)
			require.NoError(t, err)
			got := storage.URL(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLocalStorage_WithTimeout(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	t.Run("upload times out", func(t *testing.T) {
		t.Parallel()
		// Create storage with very short timeout
		storage, err := file.NewLocalStorage(tempDir, "/files/", file.WithLocalUploadTimeout(1*time.Nanosecond))
		require.NoError(t, err)

		// Create a large file that will take time to copy
		content := make([]byte, 10*1024*1024) // 10MB
		fh := createFileHeader("large.bin", content)

		ctx := context.Background()
		_, err = storage.Save(ctx, fh, "timeout-test.bin")

		// Should timeout
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
	})

	t.Run("upload completes within timeout", func(t *testing.T) {
		t.Parallel()
		// Create storage with reasonable timeout
		storage, err := file.NewLocalStorage(tempDir, "/files/", file.WithLocalUploadTimeout(5*time.Second))
		require.NoError(t, err)

		content := []byte("small file")
		fh := createFileHeader("small.txt", content)

		ctx := context.Background()
		f, err := storage.Save(ctx, fh, "success-test.txt")

		// Should succeed
		require.NoError(t, err)
		assert.Equal(t, "small.txt", f.Filename)
		assert.Equal(t, int64(len(content)), f.Size)
	})
}

func TestLocalStorage_Integration(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	storage, err := file.NewLocalStorage(tempDir, "/files/")
	require.NoError(t, err)
	ctx := context.Background()

	content := []byte("integration test content")
	fh := createFileHeader("integration.txt", content)

	savePath := "integration/test.txt"
	file, err := storage.Save(ctx, fh, savePath)
	require.NoError(t, err)
	require.NotNil(t, file)

	exists := storage.Exists(ctx, savePath)
	assert.True(t, exists)

	entries, err := storage.List(ctx, "integration")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "test.txt", entries[0].Name)
	assert.Equal(t, int64(len(content)), entries[0].Size)
	assert.False(t, entries[0].IsDir)

	url := storage.URL(file.RelativePath)
	assert.Equal(t, "/files/"+file.RelativePath, url)

	err = storage.Delete(ctx, savePath)
	require.NoError(t, err)

	exists = storage.Exists(ctx, savePath)
	assert.False(t, exists)

	err = storage.DeleteDir(ctx, "integration")
	require.NoError(t, err)

	_, err = storage.List(ctx, "integration")
	assert.Error(t, err)
}
