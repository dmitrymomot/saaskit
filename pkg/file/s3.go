package file

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Client defines the interface for S3 operations used by S3Storage.
type S3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
}

// S3ListObjectsV2Paginator defines the interface for paginated list operations.
type S3ListObjectsV2Paginator interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// S3Storage implements Storage interface for Amazon S3 and S3-compatible services.
// It is safe for concurrent use.
type S3Storage struct {
	client           S3Client
	bucket           string
	baseURL          string
	forcePathStyle   bool
	uploadTimeout    time.Duration
	paginatorFactory func(client S3Client, params *s3.ListObjectsV2Input) S3ListObjectsV2Paginator
}

// S3Config contains configuration for S3 storage.
type S3Config struct {
	Bucket         string
	Region         string
	AccessKeyID    string
	SecretKey      string
	Endpoint       string // Optional: for S3-compatible services
	BaseURL        string // Public URL base for serving files
	ForcePathStyle bool   // For S3-compatible services like MinIO
}

// S3Option defines a function that configures S3Storage.
type S3Option func(*s3Options)

// s3Options contains additional configuration options.
type s3Options struct {
	httpClient       *http.Client
	s3Client         S3Client
	s3ConfigOptions  []func(*config.LoadOptions) error
	s3ClientOptions  []func(*s3.Options)
	paginatorFactory func(client S3Client, params *s3.ListObjectsV2Input) S3ListObjectsV2Paginator
	uploadTimeout    time.Duration
}

// WithS3Client sets a custom pre-configured S3 client.
// Useful for testing with mocks.
func WithS3Client(client S3Client) S3Option {
	return func(o *s3Options) {
		o.s3Client = client
	}
}

// WithHTTPClient sets a custom HTTP client for S3 requests.
func WithHTTPClient(client *http.Client) S3Option {
	return func(o *s3Options) {
		o.httpClient = client
	}
}

// WithS3ConfigOption adds a custom AWS config option.
func WithS3ConfigOption(option func(*config.LoadOptions) error) S3Option {
	return func(o *s3Options) {
		o.s3ConfigOptions = append(o.s3ConfigOptions, option)
	}
}

// WithS3ClientOption adds a custom S3 client option.
func WithS3ClientOption(option func(*s3.Options)) S3Option {
	return func(o *s3Options) {
		o.s3ClientOptions = append(o.s3ClientOptions, option)
	}
}

// WithPaginatorFactory sets a custom paginator factory.
// Useful for testing pagination.
func WithPaginatorFactory(factory func(client S3Client, params *s3.ListObjectsV2Input) S3ListObjectsV2Paginator) S3Option {
	return func(o *s3Options) {
		o.paginatorFactory = factory
	}
}

// WithS3UploadTimeout sets the timeout for upload operations.
// If not set, no timeout is applied (context deadline from caller is used).
func WithS3UploadTimeout(timeout time.Duration) S3Option {
	return func(o *s3Options) {
		o.uploadTimeout = timeout
	}
}

// NewS3Storage creates a new S3 storage instance.
func NewS3Storage(ctx context.Context, cfg S3Config, opts ...S3Option) (*S3Storage, error) {
	if cfg.Bucket == "" || cfg.Region == "" {
		return nil, ErrInvalidConfig
	}

	// Initialize options
	options := &s3Options{}

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// If a pre-configured S3 client is provided, use it directly
	var client S3Client
	if options.s3Client != nil {
		client = options.s3Client
	} else {
		// Configure AWS SDK options
		awsOptions := []func(*config.LoadOptions) error{
			config.WithRegion(cfg.Region),
		}

		// Add credentials if provided
		if cfg.AccessKeyID != "" && cfg.SecretKey != "" {
			awsOptions = append(awsOptions,
				config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
					cfg.AccessKeyID,
					cfg.SecretKey,
					"",
				)),
			)
		}

		// Add custom HTTP client if provided
		if options.httpClient != nil {
			awsOptions = append(awsOptions, config.WithHTTPClient(options.httpClient))
		}

		// Add any additional config options
		awsOptions = append(awsOptions, options.s3ConfigOptions...)

		// Load AWS configuration
		awsConfig, err := config.LoadDefaultConfig(ctx, awsOptions...)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrFailedToLoadConfig, err)
		}

		// Create the S3 client
		client = s3.NewFromConfig(awsConfig, func(o *s3.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = aws.String(cfg.Endpoint)
			}
			o.UsePathStyle = cfg.ForcePathStyle

			// Apply any additional S3 client options
			for _, opt := range options.s3ClientOptions {
				opt(o)
			}
		})
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		if cfg.Endpoint != "" {
			baseURL = fmt.Sprintf("%s/%s", strings.TrimSuffix(cfg.Endpoint, "/"), cfg.Bucket)
		} else {
			baseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
		}
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Set default paginator factory if not provided
	paginatorFactory := options.paginatorFactory
	if paginatorFactory == nil {
		paginatorFactory = func(c S3Client, params *s3.ListObjectsV2Input) S3ListObjectsV2Paginator {
			// Create adapter for real S3 client
			if realClient, ok := c.(*s3.Client); ok {
				return s3.NewListObjectsV2Paginator(realClient, params)
			}
			// For mock clients, they should provide their own paginator
			return nil
		}
	}

	return &S3Storage{
		client:           client,
		bucket:           cfg.Bucket,
		baseURL:          baseURL,
		forcePathStyle:   cfg.ForcePathStyle,
		uploadTimeout:    options.uploadTimeout,
		paginatorFactory: paginatorFactory,
	}, nil
}

// classifyS3Error converts S3 errors to domain-specific errors.
func classifyS3Error(err error, operation string) error {
	if err == nil {
		return nil
	}

	// Check for context errors first
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %s operation", ErrOperationTimeout, operation)
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: %s operation", ErrOperationCanceled, operation)
	}

	// Check for specific S3 error types
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return fmt.Errorf("%w: %s", ErrFileNotFound, err)
	}

	var nsb *types.NoSuchBucket
	if errors.As(err, &nsb) {
		return ErrBucketNotFound
	}

	// Check for generic API errors
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		switch code {
		case "AccessDenied":
			return fmt.Errorf("%w: %s operation", ErrAccessDenied, operation)
		case "RequestTimeout":
			return fmt.Errorf("%w: %s operation", ErrRequestTimeout, operation)
		case "SlowDown", "ServiceUnavailable":
			return fmt.Errorf("%w: %s operation", ErrServiceUnavailable, operation)
		case "InvalidObjectState":
			return fmt.Errorf("%w: %s operation", ErrInvalidObjectState, operation)
		case "NoSuchKey":
			return fmt.Errorf("%w: %s", ErrFileNotFound, err)
		case "NoSuchBucket":
			return ErrBucketNotFound
		default:
			// Include error code in message for debugging
			return fmt.Errorf("%s operation failed (code: %s): %w", operation, code, err)
		}
	}

	// Default error wrapping
	return fmt.Errorf("%s operation failed: %w", operation, err)
}

// Save stores a file to S3.
func (s *S3Storage) Save(ctx context.Context, fh *multipart.FileHeader, path string) (*File, error) {
	// Apply upload timeout if configured
	if s.uploadTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.uploadTimeout)
		defer cancel()
	}

	if fh == nil {
		return nil, ErrNilFileHeader
	}

	filename := SanitizeFilename(fh.Filename)

	path = strings.TrimPrefix(path, "/")
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPath, path)
	}

	src, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToOpenFile, err)
	}
	defer func() { _ = src.Close() }()

	mimeType, err := GetMIMEType(fh)
	if err != nil {
		mimeType = "application/octet-stream"
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(path),
		Body:        src,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, classifyS3Error(err, "upload file")
	}

	return &File{
		Filename:     filename,
		Size:         fh.Size,
		MIMEType:     mimeType,
		Extension:    GetExtension(fh),
		AbsolutePath: "", // Not applicable for S3
		RelativePath: path,
	}, nil
}

// Delete removes a single file from S3.
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	path = strings.TrimPrefix(path, "/")
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: %s", ErrInvalidPath, path)
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return classifyS3Error(err, "check file")
	}

	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return classifyS3Error(err, "delete file")
	}

	return nil
}

// DeleteDir removes all objects with the given prefix from S3.
func (s *S3Storage) DeleteDir(ctx context.Context, dir string) error {
	dir = strings.TrimPrefix(dir, "/")
	if strings.Contains(dir, "..") {
		return fmt.Errorf("%w: %s", ErrInvalidPath, dir)
	}

	if dir != "" && !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	paginator := s.paginatorFactory(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(dir),
	})
	if paginator == nil {
		return ErrPaginatorNil
	}

	var objects []types.ObjectIdentifier
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return classifyS3Error(err, "list directory")
		}

		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}
	}

	if len(objects) == 0 {
		return fmt.Errorf("%w: %s", ErrDirectoryNotFound, dir)
	}

	for i := 0; i < len(objects); i += 1000 {
		end := min(i+1000, len(objects))
		_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{
				Objects: objects[i:end],
			},
		})
		if err != nil {
			return classifyS3Error(err, "delete directory")
		}
	}

	return nil
}

// Exists checks if an object exists in S3.
func (s *S3Storage) Exists(ctx context.Context, path string) bool {
	path = strings.TrimPrefix(path, "/")
	if strings.Contains(path, "..") {
		return false
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	return err == nil
}

// List returns all entries in a directory (non-recursive).
func (s *S3Storage) List(ctx context.Context, dir string) ([]Entry, error) {
	dir = strings.TrimPrefix(dir, "/")
	if strings.Contains(dir, "..") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPath, dir)
	}

	prefix := dir
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	resp, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, classifyS3Error(err, "list directory")
	}

	var entries []Entry

	for _, commonPrefix := range resp.CommonPrefixes {
		name := strings.TrimPrefix(*commonPrefix.Prefix, prefix)
		name = strings.TrimSuffix(name, "/")
		entries = append(entries, Entry{
			Name:  name,
			Path:  *commonPrefix.Prefix,
			IsDir: true,
			Size:  0,
		})
	}

	for _, obj := range resp.Contents {
		if *obj.Key == prefix {
			continue
		}
		name := strings.TrimPrefix(*obj.Key, prefix)
		if !strings.Contains(name, "/") {
			entries = append(entries, Entry{
				Name:  name,
				Path:  *obj.Key,
				IsDir: false,
				Size:  *obj.Size,
			})
		}
	}

	return entries, nil
}

// URL returns the public URL for a file.
func (s *S3Storage) URL(path string) string {
	path = strings.TrimPrefix(path, "/")
	return s.baseURL + path
}
