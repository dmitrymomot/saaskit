package file

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage implements Storage interface for local filesystem.
// It is safe for concurrent use.
type LocalStorage struct {
	baseDir       string        // Base directory for all file operations
	baseURL       string        // Base URL for generating public URLs
	uploadTimeout time.Duration // Timeout for upload operations
}

// LocalOption defines a function that configures LocalStorage.
type LocalOption func(*LocalStorage)

// WithLocalUploadTimeout sets the timeout for upload operations.
// If not set, no timeout is applied (context deadline from caller is used).
func WithLocalUploadTimeout(timeout time.Duration) LocalOption {
	return func(s *LocalStorage) {
		s.uploadTimeout = timeout
	}
}

// NewLocalStorage creates a new local filesystem storage.
// baseDir is the root directory where all files will be stored.
// baseURL is used for generating public URLs (e.g., "/files").
// All file operations are confined to baseDir to prevent path traversal attacks.
func NewLocalStorage(baseDir, baseURL string, opts ...LocalOption) (*LocalStorage, error) {
	if baseDir == "" {
		return nil, ErrInvalidConfig
	}

	// Resolve base directory to absolute path
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to resolve base directory: %v", ErrFailedToGetAbsolutePath, err)
	}

	// Ensure base directory exists
	if err := os.MkdirAll(absBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateDirectory, err)
	}

	if baseURL != "" && !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	s := &LocalStorage{
		baseDir: absBaseDir,
		baseURL: baseURL,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Save stores a file to the local filesystem.
func (s *LocalStorage) Save(ctx context.Context, fh *multipart.FileHeader, path string) (*File, error) {
	// Apply upload timeout if configured
	if s.uploadTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.uploadTimeout)
		defer cancel()
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if fh == nil {
		return nil, ErrNilFileHeader
	}

	filename := SanitizeFilename(fh.Filename)

	// Use the filename from the path if it's provided, otherwise use the sanitized filename
	dir := filepath.Dir(path)
	baseFilename := filepath.Base(path)
	if baseFilename == "." || baseFilename == "" {
		// If no filename in path, use sanitized filename from upload
		path = filepath.Join(dir, filename)
	}

	// Validate and resolve the path within base directory
	absPath, err := s.resolvePath(path)
	if err != nil {
		return nil, err
	}

	fileDir := filepath.Dir(absPath)
	if err = os.MkdirAll(fileDir, 0755); err != nil {
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

	// Use io.CopyN with context checking for cancellation support
	written := int64(0)
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			_ = dst.Close()
			_ = os.Remove(absPath)
			return nil, ctx.Err()
		default:
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			nw, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				_ = dst.Close()
				_ = os.Remove(absPath)
				return nil, fmt.Errorf("%w: %v", ErrFailedToWriteFile, writeErr)
			}
			written += int64(nw)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = dst.Close()
			_ = os.Remove(absPath)
			return nil, fmt.Errorf("%w: %v", ErrFailedToReadFile, readErr)
		}
	}

	mimeType, err := GetMIMEType(fh)
	if err != nil {
		mimeType = "application/octet-stream"
	}

	// Calculate relative path from base directory
	relPath, err := filepath.Rel(s.baseDir, absPath)
	if err != nil {
		relPath = path
	}

	return &File{
		Filename:     filename,
		Size:         written,
		MIMEType:     mimeType,
		Extension:    GetExtension(fh),
		AbsolutePath: absPath,
		RelativePath: relPath,
	}, nil
}

// Delete removes a single file.
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate and resolve the path within base directory
	absPath, err := s.resolvePath(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return fmt.Errorf("%w: %v", ErrFailedToStatPath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%w: %s, use DeleteDir instead", ErrIsDirectory, path)
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDeleteFile, err)
	}

	return nil
}

// DeleteDir recursively removes a directory and all its contents.
func (s *LocalStorage) DeleteDir(ctx context.Context, path string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate and resolve the path within base directory
	absPath, err := s.resolvePath(path)
	if err != nil {
		return err
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
	// Check context cancellation
	select {
	case <-ctx.Done():
		return false
	default:
	}

	// Validate and resolve the path within base directory
	absPath, err := s.resolvePath(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(absPath)
	return err == nil
}

// List returns all entries in a directory (non-recursive).
func (s *LocalStorage) List(ctx context.Context, dir string) ([]Entry, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate and resolve the path within base directory
	absPath, err := s.resolvePath(dir)
	if err != nil {
		return nil, err
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
		// Check context cancellation periodically
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calculate relative path from base directory
		entryAbsPath := filepath.Join(absPath, dirEntry.Name())
		entryRelPath, err := filepath.Rel(s.baseDir, entryAbsPath)
		if err != nil {
			entryRelPath = filepath.Join(dir, dirEntry.Name())
		}

		info, err := dirEntry.Info()
		if err != nil {
			continue
		}

		entry := Entry{
			Name:  dirEntry.Name(),
			Path:  entryRelPath,
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

	// Convert backslashes to forward slashes for URLs
	path = filepath.ToSlash(path)

	if strings.HasPrefix(path, "/") {
		return path
	}

	return s.baseURL + path
}

// resolvePath validates and resolves a path within the base directory.
// It ensures the resolved path is within baseDir to prevent path traversal attacks.
func (s *LocalStorage) resolvePath(path string) (string, error) {
	// Clean the path
	path = filepath.Clean(path)

	// Join with base directory
	absPath := filepath.Join(s.baseDir, path)

	// Resolve to absolute path
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedToGetAbsolutePath, err)
	}

	// Ensure the resolved path is within base directory
	if !strings.HasPrefix(absPath, s.baseDir+string(filepath.Separator)) && absPath != s.baseDir {
		return "", fmt.Errorf("%w: %s", ErrInvalidPath, path)
	}

	return absPath, nil
}
