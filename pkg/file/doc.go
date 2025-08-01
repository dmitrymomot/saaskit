// Package file provides a comprehensive file management system with support for local and S3 storage backends.
//
// The package offers a unified Storage interface for file operations across different backends,
// along with utilities for file validation, content detection, and security checks. It's designed
// to handle common file upload scenarios in web applications with built-in protection against
// common security vulnerabilities like path traversal and MIME type spoofing.
//
// # Architecture
//
// The package is built around the Storage interface which provides a consistent API for:
//   - Saving files with automatic path sanitization
//   - Deleting files and directories
//   - Checking file existence
//   - Listing directory contents
//   - Generating public URLs
//
// Two implementations are provided:
//   - LocalStorage: For filesystem-based storage
//   - S3Storage: For AWS S3 and S3-compatible services (MinIO, Wasabi, etc.)
//
// # Usage
//
// Basic file upload handling with validation:
//
//	import "github.com/dmitrymomot/saaskit/pkg/file"
//
//	// Create storage backend
//	storage := file.NewLocalStorage("/uploads", "https://example.com/files")
//
//	// In HTTP handler
//	fh := r.MultipartForm.File["avatar"][0]
//
//	// Validate file
//	if err := file.ValidateSize(fh, 5<<20); err != nil { // 5MB limit
//		return err
//	}
//
//	if err := file.ValidateMIMEType(fh, "image/jpeg", "image/png"); err != nil {
//		return err
//	}
//
//	// Save file
//	fileInfo, err := storage.Save(ctx, fh, "avatars/user123.jpg")
//	if err != nil {
//		return err
//	}
//
//	// Get public URL
//	url := storage.URL(fileInfo.RelativePath)
//
// Using S3 storage:
//
//	storage, err := file.NewS3Storage(ctx, file.S3Config{
//		Bucket:      "my-bucket",
//		Region:      "us-east-1",
//		AccessKeyID: "key",
//		SecretKey:   "secret",
//	})
//	if err != nil {
//		return err
//	}
//
//	// Same Storage interface methods work with S3
//	fileInfo, err := storage.Save(ctx, fh, "uploads/document.pdf")
//
// # File Validation
//
// The package provides several validation utilities:
//
//	// Check file type
//	if file.IsImage(fh) {
//		// Process image
//	}
//
//	// Validate size (prevents DoS from large uploads)
//	err := file.ValidateSize(fh, 10<<20) // 10MB max
//
//	// Validate MIME type (uses content detection, not extension)
//	err := file.ValidateMIMEType(fh, "application/pdf", "application/msword")
//
//	// Get file hash for deduplication
//	hash, err := file.Hash(fh, sha256.New())
//
// # Security Considerations
//
// The package implements several security measures:
//   - Path sanitization prevents directory traversal attacks
//   - MIME type detection uses file content, not extension
//   - Automatic filename sanitization removes dangerous characters
//   - Size validation prevents resource exhaustion
//   - Support for separate storage and public URL paths
//
// # Configuration
//
// LocalStorage configuration:
//   - BaseDir: Root directory for file storage
//   - BaseURL: Public URL prefix for generating file URLs
//   - DirPerm: Directory creation permissions (default: 0755)
//
// S3Storage configuration:
//   - Standard AWS credentials (key, secret, region)
//   - Custom endpoints for S3-compatible services
//   - Path-style URLs for MinIO compatibility
//   - Custom CDN base URLs
//   - Upload timeouts for large files
//
// # Error Handling
//
// The package defines specific errors for different failure scenarios:
//
//	fileInfo, err := storage.Save(ctx, fh, "test.jpg")
//	if errors.Is(err, file.ErrFileTooLarge) {
//		// File exceeds size limit
//	} else if errors.Is(err, file.ErrMIMETypeNotAllowed) {
//		// Invalid file type
//	} else if errors.Is(err, file.ErrInvalidPath) {
//		// Path contains dangerous characters
//	}
//
// S3-specific errors are mapped to generic file errors for consistency:
//   - NoSuchBucket -> ErrDirectoryNotFound
//   - NoSuchKey -> ErrFileNotFound
//   - AccessDenied -> ErrAccessDenied
//
// # Performance Considerations
//
// - File content is streamed during uploads to minimize memory usage
// - S3Storage supports configurable timeouts for large file uploads
// - Directory listings use pagination to handle large directories
// - MIME type detection reads only first 512 bytes
// - Hash calculation streams file content without loading into memory
//
// # Examples
//
// See the package examples and README.md for detailed usage patterns including
// S3-compatible services, CDN integration, and batch operations.
package file
