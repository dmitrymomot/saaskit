# Package binder

Type-safe HTTP request data binding for the saaskit framework.

## Overview

The binder package provides a comprehensive set of utilities for binding HTTP request data to Go structs. It handles JSON, form data, query parameters, path parameters, and file uploads with built-in security features and type safety.

## Internal Usage

This package is internal to the project and provides request data binding functionality for the saaskit framework's handler system.

## Features

- Type-safe binding of HTTP request data to structs
- Support for JSON, form data, query parameters, and path parameters
- Secure file upload handling with path traversal protection
- Configurable memory limits for multipart forms
- Automatic content type detection for uploaded files
- Support for optional fields using pointers and slices for multi-value parameters

## Usage

### Basic Example

```go
import "github.com/dmitrymomot/saaskit/binder"

// Define request struct with tags
type CreateUserRequest struct {
    Name     string   `json:"name" form:"name"`
    Email    string   `json:"email" form:"email"`
    Age      int      `json:"age" form:"age"`
    Tags     []string `json:"tags" form:"tags"`
    Optional *string  `json:"optional,omitempty"`
}

// Use with saaskit handlers
handler := saaskit.HandlerFunc[saaskit.Context, CreateUserRequest](
    func(ctx saaskit.Context, req CreateUserRequest) saaskit.Response {
        // req is automatically populated from request data
        return saaskit.JSONResponse(user)
    },
)

// Apply binders
http.HandleFunc("/users", saaskit.Wrap(handler,
    saaskit.WithBinder(binder.BindJSON()),  // For JSON requests
))
```

### Additional Usage Scenarios

```go
// Combining multiple binders for flexible input
type ProfileRequest struct {
    UserID   string      `path:"id"`        // From URL path
    Username string      `path:"username"`  // From URL path
    Name     string      `form:"name"`      // From form data
    Expand   bool        `query:"expand"`   // From query string
    Avatar   FileUpload  `file:"avatar"`    // File upload
}

// Route with multiple data sources
r.Get("/users/{id}/profile/{username}", saaskit.Wrap(handler,
    saaskit.WithBinders(
        binder.Path(chi.URLParam),    // Path parameters
        binder.BindQuery(),           // Query parameters
        binder.BindForm(),            // Form fields
        binder.File(),                // File uploads
    ),
))

// File upload with validation
handler := saaskit.HandlerFunc[saaskit.Context, UploadRequest](
    func(ctx saaskit.Context, req UploadRequest) saaskit.Response {
        if req.Avatar.Size > 0 {
            // Validate file type by content
            detectedType := req.Avatar.DetectContentType()
            if !strings.HasPrefix(detectedType, "image/") {
                return saaskit.Error(http.StatusBadRequest, "Avatar must be an image")
            }
        }
        return saaskit.JSONResponse(result)
    },
)
```

### Error Handling

The framework handles binding errors automatically
Common errors include:

- ErrUnsupportedMediaType: Wrong content type
- ErrInvalidJSON: Malformed JSON data
- ErrInvalidForm: Invalid form data
- ErrInvalidQuery: Invalid query parameters
- ErrInvalidPath: Invalid path parameters
- ErrMissingContentType: Missing content type header

## Best Practices

### Integration Guidelines

- Use appropriate binders based on content type
- Combine multiple binders when data comes from different sources
- Always validate file uploads using DetectContentType() for security
- Set appropriate memory limits for large file uploads

### Project-Specific Considerations

- File uploads are sanitized automatically to prevent path traversal
- Memory limits default to 10MB but can be customized per endpoint
- All binders are designed to work seamlessly with the saaskit handler system
- Struct tag names follow standard conventions (json, form, query, path, file)

## API Reference

### Configuration Variables

```go
const DefaultMaxMemory = 10 << 20 // 10 MB default limit for multipart forms
```

### Types

```go
type FileUpload struct {
    Filename string                    // Original filename (sanitized)
    Size     int64                    // File size in bytes
    Header   textproto.MIMEHeader     // MIME headers
    Content  []byte                   // File content
}

type FileHeader struct {
    Filename string
    Size     int64
    Header   textproto.MIMEHeader
}
```

### Functions

```go
func BindJSON() func(r *http.Request, v any) error
func BindQuery() func(r *http.Request, v any) error
func BindForm() func(r *http.Request, v any) error
func Path(extractor func(r *http.Request, fieldName string) string) func(r *http.Request, v any) error
func File() func(r *http.Request, v any) error

// Standalone file handling functions
func GetFile(r *http.Request, field string) (*FileUpload, error)
func GetFiles(r *http.Request, field string) ([]*FileUpload, error)
func GetAllFiles(r *http.Request) (map[string][]*FileUpload, error)
func GetFileWithLimit(r *http.Request, field string, maxMemory int64) (*FileUpload, error)
func StreamFile(r *http.Request, field string, handler func(io.Reader, *FileHeader) error) error
```

### Methods

```go
func (f *FileUpload) ContentType() string         // Returns MIME type from headers
func (f *FileUpload) DetectContentType() string   // Detects MIME type from content
```

### Error Types

```go
var ErrUnsupportedMediaType = errors.New("unsupported media type")
var ErrInvalidJSON          = errors.New("invalid JSON")
var ErrInvalidForm          = errors.New("invalid form data")
var ErrInvalidQuery         = errors.New("invalid query parameter")
var ErrInvalidPath          = errors.New("invalid path parameter")
var ErrMissingContentType   = errors.New("missing content type")
```
