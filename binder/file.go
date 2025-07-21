package binder

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"reflect"
	"strings"
)

// DefaultMaxMemory is the default maximum memory used for parsing multipart forms (10MB).
const DefaultMaxMemory = 10 << 20 // 10 MB

// FileUpload represents an uploaded file with its metadata and content.
type FileUpload struct {
	// Filename is the original filename provided by the client
	Filename string

	// Size is the size of the file in bytes
	Size int64

	// Header contains the MIME header fields for this file part
	Header textproto.MIMEHeader

	// Content holds the file data in memory
	Content []byte
}

// ContentType returns the MIME type of the uploaded file.
// It first checks the Content-Type header, then falls back to
// detecting the type from the file extension.
func (f *FileUpload) ContentType() string {
	if ct := f.Header.Get("Content-Type"); ct != "" {
		mediaType, _, _ := mime.ParseMediaType(ct)
		return mediaType
	}
	return mime.TypeByExtension(filepath.Ext(f.Filename))
}

// File creates a file binder that processes fields with `file:` tags.
// It extracts uploaded files from multipart/form-data requests.
//
// Supported field types:
//   - FileUpload - single file
//   - *FileUpload - optional single file
//   - []FileUpload - multiple files
//   - []*FileUpload - multiple files with pointers
//
// Example:
//
//	type UploadRequest struct {
//		Title    string      `form:"title"`
//		Avatar   FileUpload  `file:"avatar"`
//		Gallery  []FileUpload `file:"gallery"`
//		Document *FileUpload  `file:"document"`
//	}
//
//	handler := saaskit.HandlerFunc[saaskit.Context, UploadRequest](
//		func(ctx saaskit.Context, req UploadRequest) saaskit.Response {
//			if req.Avatar.Size > 0 {
//				// Process avatar
//			}
//			return saaskit.JSONResponse(result)
//		},
//	)
//
//	http.HandleFunc("/upload", saaskit.Wrap(handler,
//		saaskit.WithBinders(
//			binder.Form(),  // handles form fields
//			binder.File(),  // handles file uploads
//		),
//	))
func File() func(r *http.Request, v any) error {
	return func(r *http.Request, v any) error {
		// Only process multipart forms
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			// No content type, skip file binding
			return nil
		}

		// Check if this is multipart form data
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			// Not multipart, skip file binding
			return nil
		}

		// Ensure form is parsed
		if r.MultipartForm == nil {
			if err := r.ParseMultipartForm(DefaultMaxMemory); err != nil {
				// If parsing fails, skip file binding
				return nil
			}
		}

		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			return fmt.Errorf("%w: target must be a non-nil pointer", ErrInvalidForm)
		}

		rv = rv.Elem()
		if rv.Kind() != reflect.Struct {
			return fmt.Errorf("%w: target must be a pointer to struct", ErrInvalidForm)
		}

		rt := rv.Type()

		// Process each struct field
		for i := 0; i < rv.NumField(); i++ {
			field := rv.Field(i)
			fieldType := rt.Field(i)

			// Skip unexported fields
			if !field.CanSet() {
				continue
			}

			// Check if this field has a file tag
			tag := fieldType.Tag.Get("file")
			if tag == "" || tag == "-" {
				continue
			}

			// Get files for this field
			fileHeaders := r.MultipartForm.File[tag]
			if len(fileHeaders) == 0 {
				continue
			}

			// Set file field
			if err := setFileField(field, fieldType.Type, fileHeaders); err != nil {
				return fmt.Errorf("%w: field %s: %v", ErrInvalidForm, fieldType.Name, err)
			}
		}

		return nil
	}
}

// setFileField sets file upload values to struct fields.
func setFileField(field reflect.Value, fieldType reflect.Type, fileHeaders []*multipart.FileHeader) error {
	// Handle pointer types
	if fieldType.Kind() == reflect.Ptr {
		if len(fileHeaders) == 0 {
			// No files, leave as nil
			return nil
		}
		if field.IsNil() {
			field.Set(reflect.New(fieldType.Elem()))
		}
		return setFileField(field.Elem(), fieldType.Elem(), fileHeaders)
	}

	// Handle slice types
	if fieldType.Kind() == reflect.Slice {
		elemType := fieldType.Elem()
		slice := reflect.MakeSlice(fieldType, len(fileHeaders), len(fileHeaders))

		for i, header := range fileHeaders {
			upload, err := readFileHeader(header)
			if err != nil {
				return err
			}

			elem := slice.Index(i)
			if elemType.Kind() == reflect.Ptr {
				elem.Set(reflect.ValueOf(upload))
			} else {
				elem.Set(reflect.ValueOf(*upload))
			}
		}

		field.Set(slice)
		return nil
	}

	// Handle single FileUpload
	if len(fileHeaders) == 0 {
		// No file provided, leave as zero value
		return nil
	}

	// Check if this is a FileUpload type
	if fieldType.Name() != "FileUpload" || fieldType.PkgPath() != "github.com/dmitrymomot/saaskit/binder" {
		return fmt.Errorf("unsupported type for file field: %s", fieldType)
	}

	// Use only the first file for non-slice fields
	upload, err := readFileHeader(fileHeaders[0])
	if err != nil {
		return err
	}

	field.Set(reflect.ValueOf(*upload))
	return nil
}

// readFileHeader reads a multipart file header into a FileUpload struct.
func readFileHeader(header *multipart.FileHeader) (*FileUpload, error) {
	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", header.Filename, err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", header.Filename, err)
	}

	return &FileUpload{
		Filename: header.Filename,
		Size:     int64(len(content)),
		Header:   header.Header,
		Content:  content,
	}, nil
}

// GetFile retrieves a single file from a multipart form request.
// If multiple files are uploaded with the same field name, only the first is returned.
// Returns nil, nil if no file is found for the given field.
//
// Example:
//
//	file, err := binder.GetFile(r, "avatar")
//	if err != nil {
//		return saaskit.Error(http.StatusBadRequest, "Invalid file upload")
//	}
//	if file != nil {
//		// Process the file
//		fmt.Printf("Uploaded: %s (%d bytes)\n", file.Filename, file.Size)
//	}
func GetFile(r *http.Request, field string) (*FileUpload, error) {
	if err := parseMultipartForm(r, DefaultMaxMemory); err != nil {
		return nil, err
	}

	file, header, err := r.FormFile(field)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file %q: %w", field, err)
	}
	defer func() { _ = file.Close() }()

	return readFileUpload(file, header)
}

// GetFiles retrieves all files uploaded with the given field name.
// Returns an empty slice if no files are found.
//
// Example:
//
//	files, err := binder.GetFiles(r, "photos")
//	if err != nil {
//		return saaskit.Error(http.StatusBadRequest, "Invalid file upload")
//	}
//	for _, file := range files {
//		fmt.Printf("Uploaded: %s (%d bytes)\n", file.Filename, file.Size)
//	}
func GetFiles(r *http.Request, field string) ([]*FileUpload, error) {
	if err := parseMultipartForm(r, DefaultMaxMemory); err != nil {
		return nil, err
	}

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return []*FileUpload{}, nil
	}

	fileHeaders := r.MultipartForm.File[field]
	if len(fileHeaders) == 0 {
		return []*FileUpload{}, nil
	}

	uploads := make([]*FileUpload, 0, len(fileHeaders))
	for _, header := range fileHeaders {
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %q: %w", header.Filename, err)
		}

		upload, err := readFileUpload(file, header)
		_ = file.Close()
		if err != nil {
			return nil, err
		}

		uploads = append(uploads, upload)
	}

	return uploads, nil
}

// GetAllFiles retrieves all uploaded files from a multipart form request,
// organized by field name.
//
// Example:
//
//	files, err := binder.GetAllFiles(r)
//	if err != nil {
//		return saaskit.Error(http.StatusBadRequest, "Invalid file upload")
//	}
//	for fieldName, uploads := range files {
//		for _, file := range uploads {
//			fmt.Printf("%s: %s (%d bytes)\n", fieldName, file.Filename, file.Size)
//		}
//	}
func GetAllFiles(r *http.Request) (map[string][]*FileUpload, error) {
	if err := parseMultipartForm(r, DefaultMaxMemory); err != nil {
		return nil, err
	}

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return make(map[string][]*FileUpload), nil
	}

	result := make(map[string][]*FileUpload)
	for field, headers := range r.MultipartForm.File {
		uploads := make([]*FileUpload, 0, len(headers))
		for _, header := range headers {
			file, err := header.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file %q: %w", header.Filename, err)
			}

			upload, err := readFileUpload(file, header)
			_ = file.Close()
			if err != nil {
				return nil, err
			}

			uploads = append(uploads, upload)
		}
		result[field] = uploads
	}

	return result, nil
}

// FileHeader contains metadata about an uploaded file.
type FileHeader struct {
	Filename string
	Size     int64
	Header   textproto.MIMEHeader
}

// StreamFile processes an uploaded file without loading it entirely into memory.
// This is useful for large files that need to be streamed directly to storage
// or processed in chunks.
//
// The handler function receives an io.Reader for the file content and
// the file header containing metadata. The file is automatically closed
// after the handler returns.
//
// Example:
//
//	err := binder.StreamFile(r, "video", func(reader io.Reader, header *binder.FileHeader) error {
//		// Stream directly to S3
//		return s3.Upload(reader, header.Filename, header.Size)
//	})
func StreamFile(r *http.Request, field string, handler func(io.Reader, *FileHeader) error) error {
	if err := parseMultipartForm(r, DefaultMaxMemory); err != nil {
		return err
	}

	file, header, err := r.FormFile(field)
	if err != nil {
		return fmt.Errorf("failed to get file %q: %w", field, err)
	}
	defer func() { _ = file.Close() }()

	fileHeader := &FileHeader{
		Filename: header.Filename,
		Size:     header.Size,
		Header:   header.Header,
	}

	return handler(file, fileHeader)
}

// GetFileWithLimit retrieves a single file with a custom memory limit.
// This is useful when you need to handle files larger than the default 10MB limit.
//
// Example:
//
//	// Allow up to 50MB
//	file, err := binder.GetFileWithLimit(r, "document", 50<<20)
func GetFileWithLimit(r *http.Request, field string, maxMemory int64) (*FileUpload, error) {
	if err := parseMultipartForm(r, maxMemory); err != nil {
		return nil, err
	}

	file, header, err := r.FormFile(field)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file %q: %w", field, err)
	}
	defer func() { _ = file.Close() }()

	return readFileUpload(file, header)
}

// parseMultipartForm ensures the multipart form is parsed with the given memory limit.
func parseMultipartForm(r *http.Request, maxMemory int64) error {
	if r.MultipartForm != nil {
		return nil
	}

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}

	return nil
}

// readFileUpload reads a file into memory and creates a FileUpload.
func readFileUpload(file multipart.File, header *multipart.FileHeader) (*FileUpload, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", header.Filename, err)
	}

	return &FileUpload{
		Filename: header.Filename,
		Size:     int64(len(content)),
		Header:   header.Header,
		Content:  content,
	}, nil
}
