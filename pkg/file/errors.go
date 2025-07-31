package file

import "errors"

var (
	// Security and validation errors
	ErrNilFileHeader = errors.New("file header is nil")
	ErrInvalidPath   = errors.New("invalid path") // Prevents path traversal attacks

	// File system errors
	ErrFileNotFound      = errors.New("file not found")
	ErrDirectoryNotFound = errors.New("directory not found")
	ErrNotDirectory      = errors.New("path is not a directory")
	ErrIsDirectory       = errors.New("path is a directory")

	// File validation errors
	ErrFileTooLarge       = errors.New("file size exceeds maximum allowed size")
	ErrMIMETypeNotAllowed = errors.New("MIME type is not allowed")

	// I/O operation errors - wrapped with context for debugging
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

	// S3-specific errors for proper error classification
	ErrBucketNotFound     = errors.New("bucket not found")
	ErrAccessDenied       = errors.New("access denied")
	ErrRequestTimeout     = errors.New("request timed out")
	ErrServiceUnavailable = errors.New("service temporarily unavailable") // Used for throttling and retries
	ErrInvalidObjectState = errors.New("invalid object state")

	// Context and cancellation errors
	ErrOperationTimeout  = errors.New("operation timed out")
	ErrOperationCanceled = errors.New("operation canceled")

	// Configuration errors
	ErrPaginatorNil       = errors.New("paginator factory returned nil") // Testing support
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrFailedToLoadConfig = errors.New("failed to load AWS config")
)
