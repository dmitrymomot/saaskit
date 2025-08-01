// Package binder provides type-safe HTTP request data binding for the saaskit framework.
//
// The binder package offers a comprehensive set of utilities for binding HTTP request data
// to Go structs with built-in security features and type safety. It supports JSON, form data,
// query parameters, path parameters, and file uploads.
//
// # Key Features
//
//   - Type-safe binding of HTTP request data to structs
//   - Support for multiple data sources: JSON, forms, query strings, and path parameters
//   - Unified form and file handling with secure filename sanitization
//   - Configurable memory limits for multipart forms (default 10MB)
//   - Support for optional fields using pointers
//   - Direct access to standard Go multipart.FileHeader for file uploads
//
// # Basic Usage
//
//	// Define a request struct with binding tags
//	type CreateUserRequest struct {
//	    Name     string   `json:"name" form:"name"`
//	    Email    string   `json:"email" form:"email"`
//	    Age      int      `json:"age" form:"age"`
//	    Tags     []string `json:"tags" form:"tags"`
//	    Optional *string  `json:"optional,omitempty"`
//	}
//
//	// Use with saaskit handlers
//	handler := saaskit.HandlerFunc[saaskit.Context, CreateUserRequest](
//	    func(ctx saaskit.Context, req CreateUserRequest) saaskit.Response {
//	        // req is automatically populated from request data
//	        return saaskit.JSONResponse(user)
//	    },
//	)
//
//	// Apply binders
//	http.HandleFunc("/users", saaskit.Wrap(handler,
//	    saaskit.WithBinder(binder.JSON()),  // For JSON requests
//	))
//
// # Available Binders
//
// The package provides the following binder functions:
//
//   - JSON(): Binds JSON request bodies to structs
//   - Form(): Binds form data and file uploads from multipart/form-data or urlencoded requests
//   - Query(): Binds URL query parameters to structs
//   - Path(extractor): Binds URL path parameters using a custom extractor function
//
// # File Uploads
//
// File uploads are handled through the Form() binder using the `file:` struct tag:
//
//	type UploadRequest struct {
//	    Title    string                  `form:"title"`
//	    Document *multipart.FileHeader   `file:"document"`   // Single file
//	    Images   []*multipart.FileHeader `file:"images"`     // Multiple files
//	}
//
// # Error Handling
//
// The package defines several error variables for common binding failures:
//
//   - ErrUnsupportedMediaType: Content type doesn't match expected type
//   - ErrFailedToParseJSON: Failed to parse JSON request body
//   - ErrFailedToParseForm: Failed to parse form data
//   - ErrFailedToParseQuery: Failed to parse query parameters
//   - ErrFailedToParsePath: Failed to parse path parameters
//   - ErrMissingContentType: Missing Content-Type header
//
// All binding errors are automatically handled by the saaskit framework and
// return appropriate HTTP error responses to clients.
package binder
