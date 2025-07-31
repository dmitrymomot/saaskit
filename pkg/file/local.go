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
// All operations are confined to baseDir to prevent path traversal attacks.
// Safe for concurrent use with proper file locking by the OS.
type LocalStorage struct {
	baseDir       string        // Absolute path - all files stored within this directory
	baseURL       string        // URL prefix for serving files (e.g., "/files/")
	uploadTimeout time.Duration // Optional timeout to prevent hanging uploads
}

// LocalOption defines a function that configures LocalStorage.
type LocalOption func(*LocalStorage)

// WithLocalUploadTimeout sets the timeout for upload operations.
// Prevents hanging uploads from consuming resources indefinitely.
// If not set, relies on context deadline from caller.
func WithLocalUploadTimeout(timeout time.Duration) LocalOption {
	return func(s *LocalStorage) {
		s.uploadTimeout = timeout
	}
}

// NewLocalStorage creates a new local filesystem storage.
// baseDir is resolved to absolute path and created if it doesn't exist.
// baseURL is used for generating public URLs (e.g., "/files/").
// All file operations are confined to baseDir to prevent path traversal attacks.
func NewLocalStorage(baseDir, baseURL string, opts ...LocalOption) (*LocalStorage, error) {
	if baseDir == "" {
		return nil, ErrInvalidConfig
	}

	// Must resolve to absolute path for security - prevents relative path confusion
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to resolve base directory: %v", ErrFailedToGetAbsolutePath, err)
	}

	// Create directory with restrictive permissions (755 = rwxr-xr-x)
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

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Save stores a file to the local filesystem.
// Uses buffered I/O with context cancellation support to handle large files efficiently
// while allowing early termination. Cleans up partial files on errors.
func (s *LocalStorage) Save(ctx context.Context, fh *multipart.FileHeader, path string) (*File, error) {
	if s.uploadTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.uploadTimeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if fh == nil {
		return nil, ErrNilFileHeader
	}

	filename := SanitizeFilename(fh.Filename)

	// Handle both directory paths and full file paths
	dir := filepath.Dir(path)
	baseFilename := filepath.Base(path)
	if baseFilename == "." || baseFilename == "" {
		path = filepath.Join(dir, filename)
	}

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

	// Create with restrictive permissions (644 = rw-r--r--)
	dst, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateFile, err)
	}
	defer func() { _ = dst.Close() }()

	// Manual buffered copy with context checking - allows cancellation during large uploads
	written := int64(0)
	buf := make([]byte, 32*1024) // 32KB balances memory usage and syscall overhead
	for {
		select {
		case <-ctx.Done():
			_ = dst.Close()
			_ = os.Remove(absPath) // Clean up partial file
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
		mimeType = "application/octet-stream" // Safe fallback for unknown types
	}

	relPath, err := filepath.Rel(s.baseDir, absPath)
	if err != nil {
		relPath = path // Fallback to original path
	}

	return &File{
		Filename:     filename,
		Size:         written, // Actual bytes written, not FileHeader.Size
		MIMEType:     mimeType,
		Extension:    GetExtension(fh),
		AbsolutePath: absPath,
		RelativePath: relPath,
	}, nil
}

// Delete removes a single file.
// Verifies the target is a file, not a directory, to prevent accidental data loss.
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

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

	// Safety check - prevent accidental directory deletion
	if info.IsDir() {
		return fmt.Errorf("%w: %s, use DeleteDir instead", ErrIsDirectory, path)
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDeleteFile, err)
	}

	return nil
}

// DeleteDir recursively removes a directory and all its contents.
// Verifies the target is a directory to prevent accidental file deletion.
func (s *LocalStorage) DeleteDir(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

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

	// Safety check - ensure we're deleting a directory
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotDirectory, path)
	}

	if err := os.RemoveAll(absPath); err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDeleteDirectory, err)
	}

	return nil
}

// Exists checks if a file or directory exists.
// Returns false for invalid paths or on context cancellation.
func (s *LocalStorage) Exists(ctx context.Context, path string) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}

	absPath, err := s.resolvePath(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(absPath)
	return err == nil
}

// List returns all entries in a directory (non-recursive).
// Checks context cancellation periodically during iteration to handle large directories.
func (s *LocalStorage) List(ctx context.Context, dir string) ([]Entry, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

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
		// Allow cancellation during large directory listings
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		entryAbsPath := filepath.Join(absPath, dirEntry.Name())
		entryRelPath, err := filepath.Rel(s.baseDir, entryAbsPath)
		if err != nil {
			entryRelPath = filepath.Join(dir, dirEntry.Name())
		}

		info, err := dirEntry.Info()
		if err != nil {
			continue // Skip entries we can't read
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
// Critical security function that prevents path traversal attacks by ensuring
// all resolved paths stay within baseDir bounds using string prefix checking.
func (s *LocalStorage) resolvePath(path string) (string, error) {
	path = filepath.Clean(path)
	absPath := filepath.Join(s.baseDir, path)

	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedToGetAbsolutePath, err)
	}

	// Security check: ensure path stays within baseDir (prevents ../ attacks)
	if !strings.HasPrefix(absPath, s.baseDir+string(filepath.Separator)) && absPath != s.baseDir {
		return "", fmt.Errorf("%w: %s", ErrInvalidPath, path)
	}

	return absPath, nil
}
