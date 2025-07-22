package file

import "errors"

var (
	// ErrNilFileHeader is returned when a nil file header is provided
	ErrNilFileHeader = errors.New("file header is nil")

	// ErrInvalidPath is returned when the path contains invalid characters or traversal attempts
	ErrInvalidPath = errors.New("invalid path")

	// ErrFileNotFound is returned when a file does not exist
	ErrFileNotFound = errors.New("file not found")

	// ErrDirectoryNotFound is returned when a directory does not exist
	ErrDirectoryNotFound = errors.New("directory not found")

	// ErrNotDirectory is returned when a path is expected to be a directory but isn't
	ErrNotDirectory = errors.New("path is not a directory")

	// ErrIsDirectory is returned when a path is expected to be a file but is a directory
	ErrIsDirectory = errors.New("path is a directory")

	// ErrFileTooLarge is returned when a file exceeds the maximum allowed size
	ErrFileTooLarge = errors.New("file size exceeds maximum allowed size")

	// ErrMIMETypeNotAllowed is returned when a file's MIME type is not in the allowed list
	ErrMIMETypeNotAllowed = errors.New("MIME type is not allowed")

	// ErrFailedToOpenFile is returned when a file cannot be opened
	ErrFailedToOpenFile = errors.New("failed to open file")

	// ErrFailedToReadFile is returned when a file cannot be read
	ErrFailedToReadFile = errors.New("failed to read file")

	// ErrFailedToWriteFile is returned when a file cannot be written
	ErrFailedToWriteFile = errors.New("failed to write file")

	// ErrFailedToCreateFile is returned when a file cannot be created
	ErrFailedToCreateFile = errors.New("failed to create file")

	// ErrFailedToDeleteFile is returned when a file cannot be deleted
	ErrFailedToDeleteFile = errors.New("failed to delete file")

	// ErrFailedToCreateDirectory is returned when a directory cannot be created
	ErrFailedToCreateDirectory = errors.New("failed to create directory")

	// ErrFailedToDeleteDirectory is returned when a directory cannot be deleted
	ErrFailedToDeleteDirectory = errors.New("failed to delete directory")

	// ErrFailedToReadDirectory is returned when a directory cannot be read
	ErrFailedToReadDirectory = errors.New("failed to read directory")

	// ErrFailedToStatPath is returned when file/directory info cannot be obtained
	ErrFailedToStatPath = errors.New("failed to stat path")

	// ErrFailedToGetAbsolutePath is returned when absolute path cannot be determined
	ErrFailedToGetAbsolutePath = errors.New("failed to get absolute path")

	// ErrFailedToDetectMIMEType is returned when MIME type detection fails
	ErrFailedToDetectMIMEType = errors.New("failed to detect MIME type")

	// ErrFailedToHashFile is returned when file hashing fails
	ErrFailedToHashFile = errors.New("failed to hash file")

	// ErrBucketNotFound is returned when S3 bucket does not exist
	ErrBucketNotFound = errors.New("bucket not found")

	// ErrAccessDenied is returned when access to a resource is denied
	ErrAccessDenied = errors.New("access denied")

	// ErrRequestTimeout is returned when a request times out
	ErrRequestTimeout = errors.New("request timed out")

	// ErrServiceUnavailable is returned when the service is temporarily unavailable
	ErrServiceUnavailable = errors.New("service temporarily unavailable")

	// ErrInvalidObjectState is returned when object is in an invalid state
	ErrInvalidObjectState = errors.New("invalid object state")

	// ErrOperationTimeout is returned when an operation times out
	ErrOperationTimeout = errors.New("operation timed out")

	// ErrOperationCanceled is returned when an operation is canceled
	ErrOperationCanceled = errors.New("operation canceled")

	// ErrPaginatorNil is returned when paginator factory returns nil
	ErrPaginatorNil = errors.New("paginator factory returned nil")

	// ErrInvalidConfig is returned when configuration is invalid
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrFailedToLoadConfig is returned when AWS config cannot be loaded
	ErrFailedToLoadConfig = errors.New("failed to load AWS config")
)
