package file

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
)

// File represents stored file metadata.
type File struct {
	Filename     string
	Size         int64
	MIMEType     string
	Extension    string
	AbsolutePath string
	RelativePath string
}

// Entry represents a file or directory entry.
type Entry struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

// Storage interface for different backends.
type Storage interface {
	// Save stores a file and returns metadata.
	Save(ctx context.Context, fh *multipart.FileHeader, path string) (*File, error)
	// Delete removes a single file.
	Delete(ctx context.Context, path string) error
	// DeleteDir recursively removes a directory and all its contents.
	DeleteDir(ctx context.Context, path string) error
	// Exists checks if a file or directory exists.
	Exists(ctx context.Context, path string) bool
	// List returns all entries in a directory (non-recursive).
	List(ctx context.Context, dir string) ([]Entry, error)
	// URL returns the public URL for a file.
	URL(path string) string
}

var (
	imageMIMETypes = map[string]bool{
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/png":     true,
		"image/gif":     true,
		"image/webp":    true,
		"image/svg+xml": true,
		"image/bmp":     true,
		"image/tiff":    true,
		"image/heic":    true,
		"image/heif":    true,
		"image/avif":    true,
		"image/jxl":     true,
	}

	videoMIMETypes = map[string]bool{
		"video/mp4":        true,
		"video/mpeg":       true,
		"video/ogg":        true,
		"video/webm":       true,
		"video/quicktime":  true,
		"video/x-msvideo":  true,
		"video/x-flv":      true,
		"video/3gpp":       true,
		"video/x-matroska": true,
		"video/av1":        true,
	}

	audioMIMETypes = map[string]bool{
		"audio/mpeg":   true,
		"audio/ogg":    true,
		"audio/wav":    true,
		"audio/wave":   true,
		"audio/webm":   true,
		"audio/aac":    true,
		"audio/mp4":    true,
		"audio/x-m4a":  true,
		"audio/m4a":    true,
		"audio/opus":   true,
		"audio/flac":   true,
		"audio/x-flac": true,
		"audio/3gpp":   true,
		"audio/3gpp2":  true,
	}
)

// IsImage checks if the file is an image based on MIME type.
// Falls back to extension check if MIME type detection fails to handle cases
// where http.DetectContentType can't read the file or returns generic types.
// This dual validation prevents bypass attacks using renamed extensions.
//
// Example:
//
//	if file.IsImage(fh) {
//	    // Process image file
//	}
func IsImage(fh *multipart.FileHeader) bool {
	if fh == nil {
		return false
	}

	mimeType, err := GetMIMEType(fh)
	if err == nil && mimeType != "" {
		return imageMIMETypes[mimeType]
	}

	// Fallback to extension check for files where MIME detection fails
	ext := strings.ToLower(GetExtension(fh))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".tif", ".heic", ".heif", ".avif", ".jxl":
		return true
	default:
		return false
	}
}

// IsVideo checks if the file is a video based on MIME type.
// Uses the same dual validation approach as IsImage for security.
func IsVideo(fh *multipart.FileHeader) bool {
	if fh == nil {
		return false
	}

	mimeType, err := GetMIMEType(fh)
	if err == nil && mimeType != "" {
		return videoMIMETypes[mimeType]
	}

	ext := strings.ToLower(GetExtension(fh))
	switch ext {
	case ".mp4", ".mpeg", ".mpg", ".ogg", ".webm", ".mov", ".avi", ".flv", ".3gp", ".mkv", ".av1":
		return true
	default:
		return false
	}
}

// IsAudio checks if the file is an audio file based on MIME type.
// Uses the same dual validation approach as IsImage for security.
func IsAudio(fh *multipart.FileHeader) bool {
	if fh == nil {
		return false
	}

	mimeType, err := GetMIMEType(fh)
	if err == nil && mimeType != "" {
		return audioMIMETypes[mimeType]
	}

	ext := strings.ToLower(GetExtension(fh))
	switch ext {
	case ".mp3", ".ogg", ".wav", ".webm", ".aac", ".mp4", ".m4a", ".opus", ".flac", ".3gp", ".3g2":
		return true
	default:
		return false
	}
}

// IsPDF checks if the file is a PDF.
func IsPDF(fh *multipart.FileHeader) bool {
	if fh == nil {
		return false
	}

	mimeType, err := GetMIMEType(fh)
	if err == nil && mimeType == "application/pdf" {
		return true
	}

	return strings.ToLower(GetExtension(fh)) == ".pdf"
}

// GetExtension returns the file extension including the dot.
//
// Example:
//
//	ext := file.GetExtension(fh) // ".jpg"
func GetExtension(fh *multipart.FileHeader) string {
	if fh == nil {
		return ""
	}
	return filepath.Ext(fh.Filename)
}

// GetMIMEType detects the MIME type by reading the file content.
// Uses http.DetectContentType which reads the first 512 bytes to identify file types
// based on magic bytes rather than trusting file extensions (prevents spoofing).
// Resets file position to allow subsequent reads of the same file.
func GetMIMEType(fh *multipart.FileHeader) (string, error) {
	if fh == nil {
		return "", ErrNilFileHeader
	}

	file, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedToOpenFile, err)
	}
	defer func() { _ = file.Close() }()

	// 512 bytes is the maximum http.DetectContentType reads
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("%w: %v", ErrFailedToReadFile, err)
	}

	// Reset file position for subsequent operations
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	return http.DetectContentType(buffer[:n]), nil
}

// ValidateSize checks if the file size is within the allowed limit.
// Note: For streamed uploads, FileHeader.Size may be 0. Storage implementations
// should perform actual size validation during upload to prevent DoS attacks
// from oversized files that bypass this check.
//
// Example:
//
//	if err := file.ValidateSize(fh, 5<<20); err != nil { // 5MB limit
//	    return err
//	}
func ValidateSize(fh *multipart.FileHeader, maxBytes int64) error {
	if fh == nil {
		return ErrNilFileHeader
	}
	if fh.Size > maxBytes {
		return fmt.Errorf("file size %d bytes exceeds %d bytes limit: %w", fh.Size, maxBytes, ErrFileTooLarge)
	}
	return nil
}

// ValidateMIMEType checks if the file's MIME type is in the allowed list.
// Uses actual content detection to prevent MIME type spoofing attacks.
// Pass no types to allow all MIME types (useful for generic file storage).
//
// Example:
//
//	if err := file.ValidateMIMEType(fh, "image/jpeg", "image/png"); err != nil {
//	    return err
//	}
func ValidateMIMEType(fh *multipart.FileHeader, allowedTypes ...string) error {
	if fh == nil {
		return ErrNilFileHeader
	}
	if len(allowedTypes) == 0 {
		return nil
	}

	mimeType, err := GetMIMEType(fh)
	if err != nil {
		return err
	}

	if slices.Contains(allowedTypes, mimeType) {
		return nil
	}

	return fmt.Errorf("MIME type %s not in allowed types %v: %w", mimeType, allowedTypes, ErrMIMETypeNotAllowed)
}

// ReadAll reads the entire file content into memory.
// Use with caution for large files as it can cause memory exhaustion.
// Consider streaming approaches for files larger than available memory.
func ReadAll(fh *multipart.FileHeader) ([]byte, error) {
	if fh == nil {
		return nil, ErrNilFileHeader
	}

	file, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToOpenFile, err)
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToReadFile, err)
	}

	return data, nil
}

// Hash calculates the hash of the file content.
// Defaults to SHA256 for security and compatibility with most systems.
// Used for deduplication, integrity verification, and content-based addressing.
//
// Example:
//
//	hashStr, err := file.Hash(fh, sha256.New())
func Hash(fh *multipart.FileHeader, h hash.Hash) (string, error) {
	if fh == nil {
		return "", ErrNilFileHeader
	}
	if h == nil {
		h = sha256.New() // SHA256 provides good security/performance balance
	}

	file, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedToOpenFile, err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedToHashFile, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// SanitizeFilename removes any path components and dangerous characters from a filename
// to prevent path traversal attacks and other security issues.
// Returns "unnamed" for empty or special directory references.
//
// Example:
//
//	safe := file.SanitizeFilename("../../../etc/passwd") // Returns "passwd"
//	safe = file.SanitizeFilename("C:\\Windows\\file.txt") // Returns "file.txt"
func SanitizeFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, "\x00", "")

	if filename == "." || filename == ".." || filename == "" || filename == "/" {
		filename = "unnamed"
	}

	return filename
}
