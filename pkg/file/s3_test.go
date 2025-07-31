package file_test

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/file"
)

// MockS3Client is a mock implementation of the S3Client interface
type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func (m *MockS3Client) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectsOutput), args.Error(1)
}

// MockS3ListObjectsV2Paginator is a mock implementation of the S3ListObjectsV2Paginator interface
type MockS3ListObjectsV2Paginator struct {
	mock.Mock
}

func (m *MockS3ListObjectsV2Paginator) HasMorePages() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockS3ListObjectsV2Paginator) NextPage(ctx context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func TestNewS3Storage(t *testing.T) {
	t.Parallel()
	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket:      "test-bucket",
			Region:      "us-east-1",
			AccessKeyID: "test-key",
			SecretKey:   "test-secret",
		}

		storage, err := file.NewS3Storage(context.Background(), config)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})

	t.Run("with custom endpoint", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket:         "test-bucket",
			Region:         "us-east-1",
			Endpoint:       "http://localhost:9000",
			ForcePathStyle: true,
		}

		storage, err := file.NewS3Storage(context.Background(), config)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})

	t.Run("with custom base URL", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket:  "test-bucket",
			Region:  "us-east-1",
			BaseURL: "https://cdn.example.com/",
		}

		storage, err := file.NewS3Storage(context.Background(), config)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})

	t.Run("missing bucket", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Region: "us-east-1",
		}

		storage, err := file.NewS3Storage(context.Background(), config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidConfig))
		assert.Nil(t, storage)
	})

	t.Run("missing region", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket: "test-bucket",
		}

		storage, err := file.NewS3Storage(context.Background(), config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidConfig))
		assert.Nil(t, storage)
	})

	t.Run("with mock client", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}

		mockClient := new(MockS3Client)
		opts := []file.S3Option{
			file.WithS3Client(mockClient),
		}

		storage, err := file.NewS3Storage(context.Background(), config, opts...)
		require.NoError(t, err)
		require.NotNil(t, storage)

		mockClient.AssertExpectations(t)
	})
}

func TestS3Storage_Save(t *testing.T) {
	t.Parallel()
	t.Run("successful save", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		// Set up expectation
		mockClient.On("PutObject",
			mock.Anything, // context
			mock.MatchedBy(func(params *s3.PutObjectInput) bool {
				return params.Bucket != nil && *params.Bucket == "test-bucket" &&
					params.Key != nil && *params.Key == "uploads/test.txt" &&
					params.Body != nil &&
					params.ContentType != nil && *params.ContentType == "text/plain; charset=utf-8"
			}),
			mock.Anything, // optFns
		).Return(&s3.PutObjectOutput{}, nil)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		content := []byte("test content")
		fh := createFileHeader("test.txt", content)

		result, err := storage.Save(context.Background(), fh, "uploads/test.txt")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test.txt", result.Filename)
		assert.Equal(t, int64(len(content)), result.Size)
		assert.Equal(t, ".txt", result.Extension)
		assert.Equal(t, "uploads/test.txt", result.RelativePath)
		assert.Empty(t, result.AbsolutePath)

		mockClient.AssertExpectations(t)
	})

	t.Run("path traversal attempt", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)
		// No expectations needed - the path validation should fail before S3 is called

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		content := []byte("malicious")
		fh := createFileHeader("test.txt", content)

		result, err := storage.Save(context.Background(), fh, "../../../etc/passwd")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidPath))
		assert.Nil(t, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("nil file header", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)
		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		var fh *multipart.FileHeader
		result, err := storage.Save(context.Background(), fh, "test.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrNilFileHeader))
		assert.Nil(t, result)
	})

	t.Run("S3 error", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)
		mockClient.On("PutObject", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("S3 error"))

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		content := []byte("test content")
		fh := createFileHeader("test.txt", content)

		result, err := storage.Save(context.Background(), fh, "uploads/test.txt")
		assert.Error(t, err)
		assert.Error(t, err)
		// Check that it's a wrapped error from classifyS3Error
		assert.Nil(t, result)

		mockClient.AssertExpectations(t)
	})
}

func TestS3Storage_Delete(t *testing.T) {
	t.Parallel()
	t.Run("successful delete", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		// Set up expectations
		mockClient.On("HeadObject",
			mock.Anything, // context
			mock.MatchedBy(func(params *s3.HeadObjectInput) bool {
				return params.Bucket != nil && *params.Bucket == "test-bucket" &&
					params.Key != nil && *params.Key == "uploads/test.txt"
			}),
			mock.Anything, // optFns
		).Return(&s3.HeadObjectOutput{}, nil)

		mockClient.On("DeleteObject",
			mock.Anything, // context
			mock.MatchedBy(func(params *s3.DeleteObjectInput) bool {
				return params.Bucket != nil && *params.Bucket == "test-bucket" &&
					params.Key != nil && *params.Key == "uploads/test.txt"
			}),
			mock.Anything, // optFns
		).Return(&s3.DeleteObjectOutput{}, nil)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		err = storage.Delete(context.Background(), "uploads/test.txt")
		assert.NoError(t, err)

		mockClient.AssertExpectations(t)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("HeadObject",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(nil, &types.NoSuchKey{
			Message: aws.String("The specified key does not exist"),
		})

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		err = storage.Delete(context.Background(), "uploads/notfound.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrFileNotFound))

		mockClient.AssertExpectations(t)
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		err = storage.Delete(context.Background(), "../../../etc/passwd")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidPath))
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("HeadObject",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(&s3.HeadObjectOutput{}, nil)

		mockClient.On("DeleteObject",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(nil, errors.New("delete failed"))

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		err = storage.Delete(context.Background(), "uploads/test.txt")
		assert.Error(t, err)
		assert.Error(t, err)
		// Check that it's a wrapped error from classifyS3Error

		mockClient.AssertExpectations(t)
	})
}

func TestS3Storage_DeleteDir(t *testing.T) {
	t.Parallel()
	t.Run("successful delete", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("DeleteObjects",
			mock.Anything, // context
			mock.MatchedBy(func(params *s3.DeleteObjectsInput) bool {
				return params.Bucket != nil && *params.Bucket == "test-bucket" &&
					params.Delete != nil && len(params.Delete.Objects) == 2
			}),
			mock.Anything, // optFns
		).Return(&s3.DeleteObjectsOutput{}, nil)

		paginator := new(MockS3ListObjectsV2Paginator)
		paginator.On("HasMorePages").Return(true).Once()
		paginator.On("NextPage", mock.Anything, mock.Anything).Return(&s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("uploads/file1.txt"), Size: aws.Int64(100)},
				{Key: aws.String("uploads/file2.txt"), Size: aws.Int64(200)},
			},
		}, nil).Once()
		paginator.On("HasMorePages").Return(false).Once()

		paginatorFactory := func(client file.S3Client, params *s3.ListObjectsV2Input) file.S3ListObjectsV2Paginator {
			return paginator
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient), file.WithPaginatorFactory(paginatorFactory))
		require.NoError(t, err)

		err = storage.DeleteDir(context.Background(), "uploads")
		assert.NoError(t, err)

		mockClient.AssertExpectations(t)
		paginator.AssertExpectations(t)
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)
		// No DeleteObjects call expected for empty directory

		paginator := new(MockS3ListObjectsV2Paginator)
		paginator.On("HasMorePages").Return(true).Once()
		paginator.On("NextPage", mock.Anything, mock.Anything).Return(&s3.ListObjectsV2Output{
			Contents: []types.Object{},
		}, nil).Once()
		paginator.On("HasMorePages").Return(false).Once()

		paginatorFactory := func(client file.S3Client, params *s3.ListObjectsV2Input) file.S3ListObjectsV2Paginator {
			return paginator
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient), file.WithPaginatorFactory(paginatorFactory))
		require.NoError(t, err)

		err = storage.DeleteDir(context.Background(), "empty")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrDirectoryNotFound))

		mockClient.AssertExpectations(t)
		paginator.AssertExpectations(t)
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		err = storage.DeleteDir(context.Background(), "../../../etc")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidPath))
	})

	t.Run("list error", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)
		// No DeleteObjects call expected when list fails

		paginator := new(MockS3ListObjectsV2Paginator)
		paginator.On("HasMorePages").Return(true).Once()
		paginator.On("NextPage", mock.Anything, mock.Anything).Return(nil, errors.New("list failed"))

		paginatorFactory := func(client file.S3Client, params *s3.ListObjectsV2Input) file.S3ListObjectsV2Paginator {
			return paginator
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient), file.WithPaginatorFactory(paginatorFactory))
		require.NoError(t, err)

		err = storage.DeleteDir(context.Background(), "uploads")
		assert.Error(t, err)
		assert.Error(t, err)
		// Check that it's a wrapped error from classifyS3Error

		mockClient.AssertExpectations(t)
		paginator.AssertExpectations(t)
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("DeleteObjects",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(nil, errors.New("delete failed"))

		paginator := new(MockS3ListObjectsV2Paginator)
		paginator.On("HasMorePages").Return(true).Once()
		paginator.On("NextPage", mock.Anything, mock.Anything).Return(&s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("uploads/file1.txt")},
			},
		}, nil).Once()
		paginator.On("HasMorePages").Return(false).Once()

		paginatorFactory := func(client file.S3Client, params *s3.ListObjectsV2Input) file.S3ListObjectsV2Paginator {
			return paginator
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient), file.WithPaginatorFactory(paginatorFactory))
		require.NoError(t, err)

		err = storage.DeleteDir(context.Background(), "uploads")
		assert.Error(t, err)
		assert.Error(t, err)
		// Check that it's a wrapped error from classifyS3Error

		mockClient.AssertExpectations(t)
		paginator.AssertExpectations(t)
	})

	t.Run("paginated delete", func(t *testing.T) {
		t.Parallel()
		objects := make([]types.Object, 1500)
		for i := range 1500 {
			objects[i] = types.Object{Key: aws.String(fmt.Sprintf("large-dir/file%d.txt", i))}
		}

		mockClient := new(MockS3Client)
		// Expect two calls to DeleteObjects
		mockClient.On("DeleteObjects",
			mock.Anything, // context
			mock.MatchedBy(func(params *s3.DeleteObjectsInput) bool {
				// First call should have 1000 objects, second call should have 500
				return len(params.Delete.Objects) == 1000 || len(params.Delete.Objects) == 500
			}),
			mock.Anything, // optFns
		).Return(&s3.DeleteObjectsOutput{}, nil).Times(2)

		paginator := new(MockS3ListObjectsV2Paginator)
		// Set up paginator expectations
		paginator.On("HasMorePages").Return(true).Once()
		paginator.On("NextPage", mock.Anything, mock.Anything).Return(&s3.ListObjectsV2Output{
			Contents: objects[:1000],
		}, nil).Once()
		paginator.On("HasMorePages").Return(true).Once()
		paginator.On("NextPage", mock.Anything, mock.Anything).Return(&s3.ListObjectsV2Output{
			Contents: objects[1000:],
		}, nil).Once()
		paginator.On("HasMorePages").Return(false).Once()

		paginatorFactory := func(client file.S3Client, params *s3.ListObjectsV2Input) file.S3ListObjectsV2Paginator {
			return paginator
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient), file.WithPaginatorFactory(paginatorFactory))
		require.NoError(t, err)

		err = storage.DeleteDir(context.Background(), "large-dir")
		assert.NoError(t, err)

		mockClient.AssertExpectations(t)
		paginator.AssertExpectations(t)
	})
}

func TestS3Storage_Exists(t *testing.T) {
	t.Parallel()
	t.Run("file exists", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("HeadObject",
			mock.Anything, // context
			mock.MatchedBy(func(params *s3.HeadObjectInput) bool {
				return params.Bucket != nil && *params.Bucket == "test-bucket" &&
					params.Key != nil && *params.Key == "uploads/test.txt"
			}),
			mock.Anything, // optFns
		).Return(&s3.HeadObjectOutput{}, nil)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		exists := storage.Exists(context.Background(), "uploads/test.txt")
		assert.True(t, exists)

		mockClient.AssertExpectations(t)
	})

	t.Run("file does not exist", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("HeadObject",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(nil, errors.New("not found"))

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		exists := storage.Exists(context.Background(), "uploads/notfound.txt")
		assert.False(t, exists)

		mockClient.AssertExpectations(t)
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		exists := storage.Exists(context.Background(), "../../../etc/passwd")
		assert.False(t, exists)
	})
}

func TestS3Storage_List(t *testing.T) {
	t.Parallel()
	t.Run("list files and directories", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("ListObjectsV2",
			mock.Anything, // context
			mock.MatchedBy(func(params *s3.ListObjectsV2Input) bool {
				return params.Bucket != nil && *params.Bucket == "test-bucket" &&
					params.Prefix != nil && *params.Prefix == "uploads/" &&
					params.Delimiter != nil && *params.Delimiter == "/"
			}),
			mock.Anything, // optFns
		).Return(&s3.ListObjectsV2Output{
			CommonPrefixes: []types.CommonPrefix{
				{Prefix: aws.String("uploads/images/")},
				{Prefix: aws.String("uploads/docs/")},
			},
			Contents: []types.Object{
				{Key: aws.String("uploads/file1.txt"), Size: aws.Int64(100)},
				{Key: aws.String("uploads/file2.pdf"), Size: aws.Int64(200)},
				{Key: aws.String("uploads/")},
			},
		}, nil)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		entries, err := storage.List(context.Background(), "uploads")
		require.NoError(t, err)
		assert.Len(t, entries, 4)

		dirCount := 0
		fileCount := 0
		for _, entry := range entries {
			if entry.IsDir {
				dirCount++
				assert.Contains(t, []string{"images", "docs"}, entry.Name)
				assert.Equal(t, int64(0), entry.Size)
			} else {
				fileCount++
				assert.Contains(t, []string{"file1.txt", "file2.pdf"}, entry.Name)
				assert.Greater(t, entry.Size, int64(0))
			}
		}
		assert.Equal(t, 2, dirCount)
		assert.Equal(t, 2, fileCount)

		mockClient.AssertExpectations(t)
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("ListObjectsV2",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(&s3.ListObjectsV2Output{}, nil)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		entries, err := storage.List(context.Background(), "empty")
		require.NoError(t, err)
		assert.Len(t, entries, 0)

		mockClient.AssertExpectations(t)
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		entries, err := storage.List(context.Background(), "../../../etc")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrInvalidPath))
		assert.Len(t, entries, 0)
	})

	t.Run("list error", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("ListObjectsV2",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(nil, errors.New("list failed"))

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		entries, err := storage.List(context.Background(), "uploads")
		assert.Error(t, err)
		assert.Error(t, err)
		// Check that it's a wrapped error from classifyS3Error
		assert.Len(t, entries, 0)
	})

	t.Run("root directory", func(t *testing.T) {
		t.Parallel()
		mockClient := new(MockS3Client)

		mockClient.On("ListObjectsV2",
			mock.Anything, // context
			mock.MatchedBy(func(params *s3.ListObjectsV2Input) bool {
				return params.Prefix != nil && *params.Prefix == ""
			}),
			mock.Anything, // optFns
		).Return(&s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("file.txt"), Size: aws.Int64(50)},
			},
		}, nil)

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		entries, err := storage.List(context.Background(), "")
		require.NoError(t, err)
		assert.Len(t, entries, 1)

		mockClient.AssertExpectations(t)
	})
}

func TestS3Storage_URL(t *testing.T) {
	t.Parallel()
	t.Run("default AWS URL", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket: "my-bucket",
			Region: "us-east-1",
		}

		storage, err := file.NewS3Storage(context.Background(), config, file.WithS3Client(new(MockS3Client)))
		require.NoError(t, err)

		url := storage.URL("uploads/image.jpg")
		assert.Equal(t, "https://my-bucket.s3.us-east-1.amazonaws.com/uploads/image.jpg", url)
	})

	t.Run("custom endpoint", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket:   "my-bucket",
			Region:   "us-east-1",
			Endpoint: "http://localhost:9000",
		}

		storage, err := file.NewS3Storage(context.Background(), config, file.WithS3Client(new(MockS3Client)))
		require.NoError(t, err)

		url := storage.URL("uploads/image.jpg")
		assert.Equal(t, "http://localhost:9000/my-bucket/uploads/image.jpg", url)
	})

	t.Run("custom base URL", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket:  "my-bucket",
			Region:  "us-east-1",
			BaseURL: "https://cdn.example.com",
		}

		storage, err := file.NewS3Storage(context.Background(), config, file.WithS3Client(new(MockS3Client)))
		require.NoError(t, err)

		url := storage.URL("uploads/image.jpg")
		assert.Equal(t, "https://cdn.example.com/uploads/image.jpg", url)
	})

	t.Run("path with leading slash", func(t *testing.T) {
		t.Parallel()
		config := file.S3Config{
			Bucket: "my-bucket",
			Region: "us-east-1",
		}

		storage, err := file.NewS3Storage(context.Background(), config, file.WithS3Client(new(MockS3Client)))
		require.NoError(t, err)

		url := storage.URL("/uploads/image.jpg")
		assert.Equal(t, "https://my-bucket.s3.us-east-1.amazonaws.com/uploads/image.jpg", url)
	})
}

func TestS3Storage_ErrorClassification(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("NoSuchKey error", func(t *testing.T) {
		t.Parallel()
		client := new(MockS3Client)

		client.On("HeadObject",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(nil, &types.NoSuchKey{
			Message: aws.String("The specified key does not exist"),
		})

		storage, err := file.NewS3Storage(ctx, file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(client))
		require.NoError(t, err)

		err = storage.Delete(ctx, "nonexistent.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrFileNotFound))
		assert.True(t, errors.Is(err, file.ErrFileNotFound))

		client.AssertExpectations(t)
	})

	t.Run("AccessDenied error", func(t *testing.T) {
		t.Parallel()
		client := new(MockS3Client)

		client.On("PutObject",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(nil, &smithy.GenericAPIError{
			Code:    "AccessDenied",
			Message: "Access Denied",
		})

		storage, err := file.NewS3Storage(ctx, file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(client))
		require.NoError(t, err)

		fh := createFileHeader("test.txt", []byte("content"))
		_, err = storage.Save(ctx, fh, "test.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrAccessDenied))

		client.AssertExpectations(t)
	})

	t.Run("Context timeout", func(t *testing.T) {
		t.Parallel()
		client := new(MockS3Client)

		client.On("PutObject",
			mock.Anything, // context
			mock.Anything, // params
			mock.Anything, // optFns
		).Return(nil, context.DeadlineExceeded).Run(func(args mock.Arguments) {
			// Simulate slow operation
			time.Sleep(100 * time.Millisecond)
		})

		storage, err := file.NewS3Storage(ctx, file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(client), file.WithS3UploadTimeout(10*time.Millisecond))
		require.NoError(t, err)

		fh := createFileHeader("test.txt", []byte("content"))
		_, err = storage.Save(ctx, fh, "test.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrOperationTimeout))

		client.AssertExpectations(t)
	})
}

func TestS3Storage_Integration(t *testing.T) {
	t.Parallel()
	operations := []string{}

	mockClient := new(MockS3Client)

	// Set up expectations with operation tracking
	mockClient.On("PutObject",
		mock.Anything, // context
		mock.MatchedBy(func(params *s3.PutObjectInput) bool {
			return params.Bucket != nil && *params.Bucket == "test-bucket" &&
				params.Key != nil && *params.Key == "integration/test.txt"
		}),
		mock.Anything, // optFns
	).Return(&s3.PutObjectOutput{}, nil).Run(func(args mock.Arguments) {
		operations = append(operations, "put")
	})

	mockClient.On("HeadObject",
		mock.Anything, // context
		mock.MatchedBy(func(params *s3.HeadObjectInput) bool {
			return params.Bucket != nil && *params.Bucket == "test-bucket" &&
				params.Key != nil && *params.Key == "integration/test.txt"
		}),
		mock.Anything, // optFns
	).Return(&s3.HeadObjectOutput{}, nil).Run(func(args mock.Arguments) {
		operations = append(operations, "head")
	})

	mockClient.On("ListObjectsV2",
		mock.Anything, // context
		mock.MatchedBy(func(params *s3.ListObjectsV2Input) bool {
			return params.Bucket != nil && *params.Bucket == "test-bucket" &&
				params.Prefix != nil && *params.Prefix == "integration/"
		}),
		mock.Anything, // optFns
	).Return(&s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("integration/test.txt"), Size: aws.Int64(12)},
		},
	}, nil).Run(func(args mock.Arguments) {
		operations = append(operations, "list")
	})

	mockClient.On("DeleteObject",
		mock.Anything, // context
		mock.MatchedBy(func(params *s3.DeleteObjectInput) bool {
			return params.Bucket != nil && *params.Bucket == "test-bucket" &&
				params.Key != nil && *params.Key == "integration/test.txt"
		}),
		mock.Anything, // optFns
	).Return(&s3.DeleteObjectOutput{}, nil).Run(func(args mock.Arguments) {
		operations = append(operations, "delete")
	})

	storage, err := file.NewS3Storage(context.Background(), file.S3Config{
		Bucket: "test-bucket",
		Region: "us-east-1",
	}, file.WithS3Client(mockClient))
	require.NoError(t, err)

	ctx := context.Background()

	// Save a file
	content := []byte("test content")
	fh := createFileHeader("test.txt", content)
	result, err := storage.Save(ctx, fh, "integration/test.txt")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check it exists
	exists := storage.Exists(ctx, "integration/test.txt")
	assert.True(t, exists)

	// List the directory
	entries, err := storage.List(ctx, "integration")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "test.txt", entries[0].Name)

	// Get URL
	url := storage.URL("integration/test.txt")
	assert.NotEmpty(t, url)

	// Delete the file
	err = storage.Delete(ctx, "integration/test.txt")
	require.NoError(t, err)

	assert.Equal(t, []string{"put", "head", "list", "head", "delete"}, operations)

	mockClient.AssertExpectations(t)
}
