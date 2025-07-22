// Package file provides utilities for working with file uploads and storage.
//
// The package includes:
//   - Helper functions for file validation and analysis
//   - A Storage interface for abstracting file storage backends
//   - LocalStorage implementation for filesystem storage
//   - Support for future storage backends (S3, etc.)
//
// Example usage:
//
//	import "github.com/dmitrymomot/saaskit/file"
//
//	// Create storage
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
package file
