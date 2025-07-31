# file

Secure file upload and storage abstraction with support for local filesystem and S3-compatible backends.

## Features

- Unified Storage interface for local filesystem and S3
- Built-in security features (path traversal protection, MIME validation)
- File type detection (images, videos, audio, PDFs)
- Content-based validation to prevent spoofing attacks

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/file"
```

## Usage

```go
package main

import (
    "context"
    "net/http"
    
    "github.com/dmitrymomot/saaskit/pkg/file"
)

func handleUpload(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form (32MB limit)
    r.ParseMultipartForm(32 << 20)
    fh, _, err := r.FormFile("avatar")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer fh.Close()
    
    // Validate file
    if err := file.ValidateSize(fh, 5<<20); err != nil { // 5MB limit
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    if !file.IsImage(fh) {
        http.Error(w, "only images allowed", http.StatusBadRequest)
        return
    }
    
    // Create storage
    storage, err := file.NewLocalStorage("./uploads", "/files/")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Save file
    ctx := context.Background()
    fileInfo, err := storage.Save(ctx, fh, "avatars/user123.jpg")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return public URL
    w.Write([]byte(storage.URL(fileInfo.RelativePath)))
}
```

## Common Operations

### Local Storage

```go
// Create local storage with options
storage, err := file.NewLocalStorage(
    "./uploads",     // Base directory
    "/files/",       // URL prefix
    file.WithLocalUploadTimeout(30*time.Second),
)

// List files
entries, err := storage.List(ctx, "avatars/")

// Check existence
exists := storage.Exists(ctx, "avatars/user123.jpg")

// Delete file
err = storage.Delete(ctx, "avatars/user123.jpg")

// Delete directory
err = storage.DeleteDir(ctx, "avatars/")
```

### S3 Storage

```go
// Create S3 storage
storage, err := file.NewS3Storage(ctx, file.S3Config{
    Bucket:      "my-bucket",
    Region:      "us-east-1", 
    AccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"),
    SecretKey:   os.Getenv("AWS_SECRET_ACCESS_KEY"),
    BaseURL:     "https://cdn.example.com", // Optional CDN URL
})

// Same interface as local storage
fileInfo, err := storage.Save(ctx, fh, "uploads/doc.pdf")
url := storage.URL(fileInfo.RelativePath)
```

### File Validation

```go
// Validate MIME types
err := file.ValidateMIMEType(fh, "image/jpeg", "image/png", "application/pdf")

// Type checking helpers
if file.IsImage(fh) { /* ... */ }
if file.IsVideo(fh) { /* ... */ }
if file.IsAudio(fh) { /* ... */ }
if file.IsPDF(fh) { /* ... */ }

// Get file hash
hash, err := file.Hash(fh, nil) // Uses SHA256 by default

// Sanitize filename (prevents path traversal)
safe := file.SanitizeFilename("../../../etc/passwd") // Returns "passwd"
```

## Error Handling

```go
// Package errors:
var (
    ErrNilFileHeader      = errors.New("file header is nil")
    ErrInvalidPath        = errors.New("invalid path")
    ErrFileNotFound       = errors.New("file not found")
    ErrDirectoryNotFound  = errors.New("directory not found")
    ErrFileTooLarge       = errors.New("file size exceeds maximum allowed size")
    ErrMIMETypeNotAllowed = errors.New("MIME type is not allowed")
)

// Usage:
if errors.Is(err, file.ErrFileNotFound) {
    // Handle missing file
}

if errors.Is(err, file.ErrFileTooLarge) {
    // Handle oversized file
}
```

## API Documentation

```bash
# Full API documentation
go doc github.com/dmitrymomot/saaskit/pkg/file

# Specific function or type
go doc github.com/dmitrymomot/saaskit/pkg/file.Storage
```

## Notes

- Path traversal attacks are prevented automatically in both storage backends
- MIME type detection reads file content, not just extensions (prevents spoofing)
- S3Storage supports any S3-compatible service (MinIO, DigitalOcean Spaces, etc.)
- Large file uploads should use context with timeout to prevent resource exhaustion
