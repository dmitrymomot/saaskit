// Package file provides utilities for working with file uploads and storage.
//
// The package includes:
//   - Helper functions for file validation and analysis
//   - A Storage interface for abstracting file storage backends
//   - LocalStorage implementation for filesystem storage
//   - S3Storage implementation for AWS S3 and compatible services
//
// Example usage with LocalStorage:
//
//	import "github.com/dmitrymomot/saaskit/file"
//
//	// Create local storage
//	storage := file.NewLocalStorage("/files/")
//
//	// In HTTP handler
//	fh := r.MultipartForm.File["avatar"][0]
//
//	// Validate file
//	if err := file.ValidateSize(fh, 5<<20); err != nil { // 5MB limit
//	    return err
//	}
//
//	if !file.IsImage(fh) {
//	    return errors.New("only images allowed")
//	}
//
//	// Save file
//	fileInfo, err := storage.Save(ctx, fh, "uploads/avatar.jpg")
//	if err != nil {
//	    return err
//	}
//
//	// Get public URL
//	url := storage.URL(fileInfo.RelativePath)
//
// Example usage with S3Storage:
//
//	// Create S3 storage
//	storage, err := file.NewS3Storage(ctx, file.S3Config{
//	    Bucket:      "my-bucket",
//	    Region:      "us-east-1",
//	    AccessKeyID: "key",
//	    SecretKey:   "secret",
//	})
//	if err != nil {
//	    return err
//	}
//
//	// Use the same Storage interface methods
//	fileInfo, err := storage.Save(ctx, fh, "uploads/avatar.jpg")
package file
