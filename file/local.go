package file

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage implements Storage interface for local filesystem.
type LocalStorage struct {
	baseURL string
}

// NewLocalStorage creates a new local filesystem storage.
// baseURL is used for generating public URLs (e.g., "/files").
func NewLocalStorage(baseURL string) *LocalStorage {
	if baseURL != "" && !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &LocalStorage{
		baseURL: baseURL,
	}
}

// Save stores a file to the local filesystem.
func (s *LocalStorage) Save(ctx context.Context, fh *multipart.FileHeader, path string) (*File, error) {
	if fh == nil {
		return nil, ErrNilFileHeader
	}

	filename := SanitizeFilename(fh.Filename)

	path = filepath.Clean(path)
	if strings.HasPrefix(path, "..") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPath, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToGetAbsolutePath, err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateDirectory, err)
	}

	src, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToOpenFile, err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateFile, err)
	}
	defer func() { _ = dst.Close() }()

	written, err := io.Copy(dst, src)
	if err != nil {
		_ = os.Remove(absPath)
		return nil, fmt.Errorf("%w: %v", ErrFailedToWriteFile, err)
	}

	mimeType, err := GetMIMEType(fh)
	if err != nil {
		mimeType = "application/octet-stream"
	}

	return &File{
		Filename:     filename,
		Size:         written,
		MIMEType:     mimeType,
		Extension:    GetExtension(fh),
		AbsolutePath: absPath,
		RelativePath: path,
	}, nil
}

// Delete removes a single file.
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, "..") {
		return fmt.Errorf("%w: %s", ErrInvalidPath, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToGetAbsolutePath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return fmt.Errorf("%w: %v", ErrFailedToStatPath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%w, use DeleteDir instead: %s", ErrIsDirectory, path)
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDeleteFile, err)
	}

	return nil
}

// DeleteDir recursively removes a directory and all its contents.
func (s *LocalStorage) DeleteDir(ctx context.Context, path string) error {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, "..") {
		return fmt.Errorf("%w: %s", ErrInvalidPath, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToGetAbsolutePath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrDirectoryNotFound, path)
		}
		return fmt.Errorf("%w: %v", ErrFailedToStatPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotDirectory, path)
	}

	if err := os.RemoveAll(absPath); err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDeleteDirectory, err)
	}

	return nil
}

// Exists checks if a file or directory exists.
func (s *LocalStorage) Exists(ctx context.Context, path string) bool {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, "..") {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(absPath)
	return err == nil
}

// List returns all entries in a directory (non-recursive).
func (s *LocalStorage) List(ctx context.Context, dir string) ([]Entry, error) {
	dir = filepath.Clean(dir)
	if strings.HasPrefix(dir, "..") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPath, dir)
	}

	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToGetAbsolutePath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrDirectoryNotFound, dir)
		}
		return nil, fmt.Errorf("%w: %v", ErrFailedToStatPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, dir)
	}

	dirEntries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToReadDirectory, err)
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		entryPath := filepath.Join(dir, dirEntry.Name())
		info, err := dirEntry.Info()
		if err != nil {
			continue
		}

		entry := Entry{
			Name:  dirEntry.Name(),
			Path:  entryPath,
			IsDir: dirEntry.IsDir(),
			Size:  0,
		}

		if !dirEntry.IsDir() {
			entry.Size = info.Size()
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// URL returns the public URL for a file.
func (s *LocalStorage) URL(path string) string {
	path = filepath.Clean(path)

	if filepath.IsAbs(path) {
		return path
	}

	path = strings.TrimPrefix(path, "/")

	return s.baseURL + path
}
