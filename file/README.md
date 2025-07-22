# File Package

Provides utilities for working with file uploads and storage in SaaS applications.

## Overview

The file package offers a comprehensive solution for handling file uploads, validation, and storage in Go applications. It provides a unified Storage interface with implementations for local filesystem and AWS S3, along with utilities for file validation, MIME type detection, and security.

## Internal Usage

This package is internal to the project and provides file handling capabilities for user uploads, document storage, and media management within the SaaS application.

## Features

- Unified Storage interface for abstracting storage backends
- Local filesystem storage implementation with path traversal protection
- AWS S3 and S3-compatible storage implementation (MinIO, DigitalOcean Spaces, etc.)
- File validation utilities (size, MIME type, content detection)
- Security features including path traversal protection and filename sanitization
- Support for listing directory contents and checking file existence
- Thread-safe implementations for concurrent use

## Usage

### Basic Example

```go
import "github.com/dmitrymomot/saaskit/file"

// Create local storage
storage, err := file.NewLocalStorage("/var/www/uploads", "/files/")
if err != nil {
    return err
}

// In HTTP handler
fh := r.MultipartForm.File["avatar"][0]

// Validate file
if err := file.ValidateSize(fh, 5<<20); err != nil { // 5MB limit
    return err
}

if !file.IsImage(fh) {
    return errors.New("only images allowed")
}

// Save file
fileInfo, err := storage.Save(ctx, fh, "uploads/avatar.jpg")
if err != nil {
    return err
}

// Get public URL
url := storage.URL(fileInfo.RelativePath)
// url = "/files/uploads/avatar.jpg"
```

### Additional Usage Scenarios

```go
// S3 Storage
storage, err := file.NewS3Storage(ctx, file.S3Config{
    Bucket:      "my-bucket",
    Region:      "us-east-1",
    AccessKeyID: "key",
    SecretKey:   "secret",
})
if err != nil {
    return err
}

// Check if file exists
if storage.Exists(ctx, "uploads/document.pdf") {
    // File exists
}

// List directory contents
entries, err := storage.List(ctx, "uploads")
if err != nil {
    return err
}
for _, entry := range entries {
    if entry.IsDir {
        // Handle directory: entry.Name, entry.Path
    } else {
        // Handle file: entry.Name, entry.Size, entry.Path
    }
}

// Delete file
err = storage.Delete(ctx, "uploads/old-file.txt")

// Delete entire directory
err = storage.DeleteDir(ctx, "uploads/temp")

// File hashing (requires: import "crypto/sha256")
hashStr, err := file.Hash(fh, sha256.New())
// hashStr = "2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"

// Sanitize filename for security
safe := file.SanitizeFilename("../../../etc/passwd")
// safe = "passwd"
```

### Error Handling

```go
// All storage methods return specific errors
fileInfo, err := storage.Save(ctx, fh, "uploads/file.txt")
if err != nil {
    if errors.Is(err, file.ErrNilFileHeader) {
        return fmt.Errorf("no file provided")
    }
    if errors.Is(err, file.ErrInvalidPath) {
        return fmt.Errorf("invalid file path")
    }
    if errors.Is(err, file.ErrFailedToWriteFile) {
        return fmt.Errorf("failed to save file")
    }
    return err
}

// Validation errors
err = file.ValidateSize(fh, maxSize)
if errors.Is(err, file.ErrFileTooLarge) {
    return fmt.Errorf("file too large, max size is %d bytes", maxSize)
}

err = file.ValidateMIMEType(fh, "image/jpeg", "image/png")
if errors.Is(err, file.ErrMIMETypeNotAllowed) {
    return fmt.Errorf("only JPEG and PNG images are allowed")
}
```

## Best Practices

### Integration Guidelines

- Always validate files before saving (size, MIME type, content)
- Use `SanitizeFilename` for user-provided filenames to prevent security issues
- Store files with generated paths rather than user-provided paths
- Check storage errors and handle them appropriately
- Use the same Storage interface for all backends to allow easy switching

### Project-Specific Considerations

- Configure appropriate file size limits based on your application needs
- Use S3 storage for production deployments to enable horizontal scaling
- Local storage is suitable for development and single-server deployments
- Consider using a CDN with S3 for better performance
- Implement proper access control at the application level

## API Reference

### Types

```go
// File represents stored file metadata
type File struct {
    Filename     string
    Size         int64
    MIMEType     string
    Extension    string
    AbsolutePath string // Empty for S3 storage
    RelativePath string
}

// Entry represents a file or directory entry
type Entry struct {
    Name  string
    Path  string
    IsDir bool
    Size  int64
}

// Storage interface for different backends
type Storage interface {
    Save(ctx context.Context, fh *multipart.FileHeader, path string) (*File, error)
    Delete(ctx context.Context, path string) error
    DeleteDir(ctx context.Context, path string) error
    Exists(ctx context.Context, path string) bool
    List(ctx context.Context, dir string) ([]Entry, error)
    URL(path string) string
}

// LocalStorage implements Storage interface for local filesystem
type LocalStorage struct {
    // contains filtered or unexported fields
}

// S3Storage implements Storage interface for Amazon S3 and S3-compatible services
type S3Storage struct {
    // contains filtered or unexported fields
}

// S3Config contains configuration for S3 storage
type S3Config struct {
    Bucket         string
    Region         string
    AccessKeyID    string
    SecretKey      string
    Endpoint       string // Optional: for S3-compatible services
    BaseURL        string // Public URL base for serving files
    ForcePathStyle bool   // For S3-compatible services like MinIO
}

// S3Option defines a function that configures S3Storage
type S3Option func(*s3Options)

// S3Client defines the interface for S3 operations used by S3Storage
type S3Client interface {
    PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
    HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
    ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
    DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
    DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
}

// S3ListObjectsV2Paginator defines the interface for paginated list operations
type S3ListObjectsV2Paginator interface {
    HasMorePages() bool
    NextPage(ctx context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}
```

### Functions

```go
// NewLocalStorage creates a new local filesystem storage
func NewLocalStorage(baseDir, baseURL string) (*LocalStorage, error)

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(ctx context.Context, cfg S3Config, opts ...S3Option) (*S3Storage, error)

// IsImage checks if the file is an image based on MIME type
func IsImage(fh *multipart.FileHeader) bool

// IsVideo checks if the file is a video based on MIME type
func IsVideo(fh *multipart.FileHeader) bool

// IsAudio checks if the file is an audio file based on MIME type
func IsAudio(fh *multipart.FileHeader) bool

// IsPDF checks if the file is a PDF
func IsPDF(fh *multipart.FileHeader) bool

// GetExtension returns the file extension including the dot
func GetExtension(fh *multipart.FileHeader) string

// GetMIMEType detects the MIME type by reading the file content
func GetMIMEType(fh *multipart.FileHeader) (string, error)

// ValidateSize checks if the file size is within the allowed limit
func ValidateSize(fh *multipart.FileHeader, maxBytes int64) error

// ValidateMIMEType checks if the file's MIME type is in the allowed list
func ValidateMIMEType(fh *multipart.FileHeader, allowedTypes ...string) error

// ReadAll reads the entire file content into memory
func ReadAll(fh *multipart.FileHeader) ([]byte, error)

// Hash calculates the hash of the file content
func Hash(fh *multipart.FileHeader, h hash.Hash) (string, error)

// SanitizeFilename removes any path components and dangerous characters from a filename
func SanitizeFilename(filename string) string

// WithS3Client sets a custom pre-configured S3 client
func WithS3Client(client S3Client) S3Option

// WithHTTPClient sets a custom HTTP client for S3 requests
func WithHTTPClient(client *http.Client) S3Option

// WithS3ConfigOption adds a custom AWS config option
func WithS3ConfigOption(option func(*config.LoadOptions) error) S3Option

// WithS3ClientOption adds a custom S3 client option
func WithS3ClientOption(option func(*s3.Options)) S3Option

// WithPaginatorFactory sets a custom paginator factory
func WithPaginatorFactory(factory func(client S3Client, params *s3.ListObjectsV2Input) S3ListObjectsV2Paginator) S3Option
```

### Methods

```go
// LocalStorage methods
func (s *LocalStorage) Save(ctx context.Context, fh *multipart.FileHeader, path string) (*File, error)
func (s *LocalStorage) Delete(ctx context.Context, path string) error
func (s *LocalStorage) DeleteDir(ctx context.Context, path string) error
func (s *LocalStorage) Exists(ctx context.Context, path string) bool
func (s *LocalStorage) List(ctx context.Context, dir string) ([]Entry, error)
func (s *LocalStorage) URL(path string) string

// S3Storage methods
func (s *S3Storage) Save(ctx context.Context, fh *multipart.FileHeader, path string) (*File, error)
func (s *S3Storage) Delete(ctx context.Context, path string) error
func (s *S3Storage) DeleteDir(ctx context.Context, dir string) error
func (s *S3Storage) Exists(ctx context.Context, path string) bool
func (s *S3Storage) List(ctx context.Context, dir string) ([]Entry, error)
func (s *S3Storage) URL(path string) string
```

### Error Types

```go
var (
    ErrNilFileHeader           = errors.New("file header is nil")
    ErrInvalidPath             = errors.New("invalid path")
    ErrFileNotFound            = errors.New("file not found")
    ErrDirectoryNotFound       = errors.New("directory not found")
    ErrNotDirectory            = errors.New("path is not a directory")
    ErrIsDirectory             = errors.New("path is a directory")
    ErrFileTooLarge            = errors.New("file size exceeds maximum allowed size")
    ErrMIMETypeNotAllowed      = errors.New("MIME type is not allowed")
    ErrFailedToOpenFile        = errors.New("failed to open file")
    ErrFailedToReadFile        = errors.New("failed to read file")
    ErrFailedToWriteFile       = errors.New("failed to write file")
    ErrFailedToCreateFile      = errors.New("failed to create file")
    ErrFailedToDeleteFile      = errors.New("failed to delete file")
    ErrFailedToCreateDirectory = errors.New("failed to create directory")
    ErrFailedToDeleteDirectory = errors.New("failed to delete directory")
    ErrFailedToReadDirectory   = errors.New("failed to read directory")
    ErrFailedToStatPath        = errors.New("failed to stat path")
    ErrFailedToGetAbsolutePath = errors.New("failed to get absolute path")
    ErrFailedToDetectMIMEType  = errors.New("failed to detect MIME type")
    ErrFailedToHashFile        = errors.New("failed to hash file")
)
```