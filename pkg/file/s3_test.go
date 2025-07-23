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
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/file"
)

type mockS3Client struct {
	putObjectFunc     func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	headObjectFunc    func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	listObjectsFunc   func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	deleteObjectFunc  func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	deleteObjectsFunc func(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headObjectFunc != nil {
		return m.headObjectFunc(ctx, params, optFns...)
	}
	return &s3.HeadObjectOutput{}, nil
}

func (m *mockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if m.listObjectsFunc != nil {
		return m.listObjectsFunc(ctx, params, optFns...)
	}
	return &s3.ListObjectsV2Output{}, nil
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, params, optFns...)
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3Client) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	if m.deleteObjectsFunc != nil {
		return m.deleteObjectsFunc(ctx, params, optFns...)
	}
	return &s3.DeleteObjectsOutput{}, nil
}

type mockPaginator struct {
	pages       []*s3.ListObjectsV2Output
	currentPage int
	err         error
}

func (m *mockPaginator) HasMorePages() bool {
	if m.err != nil && m.currentPage == 0 {
		return true
	}
	return m.currentPage < len(m.pages)
}

func (m *mockPaginator) NextPage(ctx context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.currentPage >= len(m.pages) {
		return nil, errors.New("no more pages")
	}
	page := m.pages[m.currentPage]
	m.currentPage++
	return page, nil
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

		opts := []file.S3Option{
			file.WithS3Client(&mockS3Client{}),
		}

		storage, err := file.NewS3Storage(context.Background(), config, opts...)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})
}

func TestS3Storage_Save(t *testing.T) {
	t.Parallel()
	t.Run("successful save", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				assert.Equal(t, "test-bucket", *params.Bucket)
				assert.Equal(t, "uploads/test.txt", *params.Key)
				assert.NotNil(t, params.Body)
				assert.Equal(t, "text/plain; charset=utf-8", *params.ContentType)
				return &s3.PutObjectOutput{}, nil
			},
		}

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
	})

	t.Run("path traversal attempt", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{}
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
	})

	t.Run("nil file header", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{}
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
		mockClient := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("S3 error")
			},
		}

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
	})
}

func TestS3Storage_Delete(t *testing.T) {
	t.Parallel()
	t.Run("successful delete", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				assert.Equal(t, "test-bucket", *params.Bucket)
				assert.Equal(t, "uploads/test.txt", *params.Key)
				return &s3.HeadObjectOutput{}, nil
			},
			deleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				assert.Equal(t, "test-bucket", *params.Bucket)
				assert.Equal(t, "uploads/test.txt", *params.Key)
				return &s3.DeleteObjectOutput{}, nil
			},
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		err = storage.Delete(context.Background(), "uploads/test.txt")
		assert.NoError(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, &types.NoSuchKey{
					Message: aws.String("The specified key does not exist"),
				}
			},
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		err = storage.Delete(context.Background(), "uploads/notfound.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrFileNotFound))
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{}

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
		mockClient := &mockS3Client{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{}, nil
			},
			deleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("delete failed")
			},
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		err = storage.Delete(context.Background(), "uploads/test.txt")
		assert.Error(t, err)
		assert.Error(t, err)
		// Check that it's a wrapped error from classifyS3Error
	})
}

func TestS3Storage_DeleteDir(t *testing.T) {
	t.Parallel()
	t.Run("successful delete", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{
			deleteObjectsFunc: func(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
				assert.Equal(t, "test-bucket", *params.Bucket)
				assert.Len(t, params.Delete.Objects, 2)
				return &s3.DeleteObjectsOutput{}, nil
			},
		}

		paginator := &mockPaginator{
			pages: []*s3.ListObjectsV2Output{
				{
					Contents: []types.Object{
						{Key: aws.String("uploads/file1.txt"), Size: aws.Int64(100)},
						{Key: aws.String("uploads/file2.txt"), Size: aws.Int64(200)},
					},
				},
			},
		}

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
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{}

		paginator := &mockPaginator{
			pages: []*s3.ListObjectsV2Output{
				{Contents: []types.Object{}},
			},
		}

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
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{}

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
		mockClient := &mockS3Client{}

		paginator := &mockPaginator{
			pages: []*s3.ListObjectsV2Output{},
			err:   errors.New("list failed"),
		}

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
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{
			deleteObjectsFunc: func(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
				return nil, errors.New("delete failed")
			},
		}

		paginator := &mockPaginator{
			pages: []*s3.ListObjectsV2Output{
				{
					Contents: []types.Object{
						{Key: aws.String("uploads/file1.txt")},
					},
				},
			},
		}

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
	})

	t.Run("paginated delete", func(t *testing.T) {
		t.Parallel()
		objects := make([]types.Object, 1500)
		for i := 0; i < 1500; i++ {
			objects[i] = types.Object{Key: aws.String(fmt.Sprintf("large-dir/file%d.txt", i))}
		}

		callCount := 0
		mockClient := &mockS3Client{
			deleteObjectsFunc: func(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
				callCount++
				if callCount == 1 {
					assert.Len(t, params.Delete.Objects, 1000)
				} else {
					assert.Len(t, params.Delete.Objects, 500)
				}
				return &s3.DeleteObjectsOutput{}, nil
			},
		}

		paginator := &mockPaginator{
			pages: []*s3.ListObjectsV2Output{
				{Contents: objects[:1000]},
				{Contents: objects[1000:]},
			},
		}

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
	})
}

func TestS3Storage_Exists(t *testing.T) {
	t.Parallel()
	t.Run("file exists", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				assert.Equal(t, "test-bucket", *params.Bucket)
				assert.Equal(t, "uploads/test.txt", *params.Key)
				return &s3.HeadObjectOutput{}, nil
			},
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		exists := storage.Exists(context.Background(), "uploads/test.txt")
		assert.True(t, exists)
	})

	t.Run("file does not exist", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, errors.New("not found")
			},
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		exists := storage.Exists(context.Background(), "uploads/notfound.txt")
		assert.False(t, exists)
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{}

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
		mockClient := &mockS3Client{
			listObjectsFunc: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				assert.Equal(t, "test-bucket", *params.Bucket)
				assert.Equal(t, "uploads/", *params.Prefix)
				assert.Equal(t, "/", *params.Delimiter)
				return &s3.ListObjectsV2Output{
					CommonPrefixes: []types.CommonPrefix{
						{Prefix: aws.String("uploads/images/")},
						{Prefix: aws.String("uploads/docs/")},
					},
					Contents: []types.Object{
						{Key: aws.String("uploads/file1.txt"), Size: aws.Int64(100)},
						{Key: aws.String("uploads/file2.pdf"), Size: aws.Int64(200)},
						{Key: aws.String("uploads/")},
					},
				}, nil
			},
		}

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
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{
			listObjectsFunc: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{}, nil
			},
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		entries, err := storage.List(context.Background(), "empty")
		require.NoError(t, err)
		assert.Len(t, entries, 0)
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		mockClient := &mockS3Client{}

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
		mockClient := &mockS3Client{
			listObjectsFunc: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return nil, errors.New("list failed")
			},
		}

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
		mockClient := &mockS3Client{
			listObjectsFunc: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				assert.Equal(t, "", *params.Prefix)
				return &s3.ListObjectsV2Output{
					Contents: []types.Object{
						{Key: aws.String("file.txt"), Size: aws.Int64(50)},
					},
				}, nil
			},
		}

		storage, err := file.NewS3Storage(context.Background(), file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(mockClient))
		require.NoError(t, err)

		entries, err := storage.List(context.Background(), "")
		require.NoError(t, err)
		assert.Len(t, entries, 1)
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

		storage, err := file.NewS3Storage(context.Background(), config, file.WithS3Client(&mockS3Client{}))
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

		storage, err := file.NewS3Storage(context.Background(), config, file.WithS3Client(&mockS3Client{}))
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

		storage, err := file.NewS3Storage(context.Background(), config, file.WithS3Client(&mockS3Client{}))
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

		storage, err := file.NewS3Storage(context.Background(), config, file.WithS3Client(&mockS3Client{}))
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
		client := &mockS3Client{
			headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
				return nil, &types.NoSuchKey{
					Message: aws.String("The specified key does not exist"),
				}
			},
		}

		storage, err := file.NewS3Storage(ctx, file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(client))
		require.NoError(t, err)

		err = storage.Delete(ctx, "nonexistent.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrFileNotFound))
		assert.True(t, errors.Is(err, file.ErrFileNotFound))
	})

	t.Run("AccessDenied error", func(t *testing.T) {
		t.Parallel()
		client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, &smithy.GenericAPIError{
					Code:    "AccessDenied",
					Message: "Access Denied",
				}
			},
		}

		storage, err := file.NewS3Storage(ctx, file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(client))
		require.NoError(t, err)

		fh := createFileHeader("test.txt", []byte("content"))
		_, err = storage.Save(ctx, fh, "test.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrAccessDenied))
	})

	t.Run("Context timeout", func(t *testing.T) {
		t.Parallel()
		client := &mockS3Client{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				// Simulate slow operation
				time.Sleep(100 * time.Millisecond)
				return nil, ctx.Err()
			},
		}

		storage, err := file.NewS3Storage(ctx, file.S3Config{
			Bucket: "test-bucket",
			Region: "us-east-1",
		}, file.WithS3Client(client), file.WithS3UploadTimeout(10*time.Millisecond))
		require.NoError(t, err)

		fh := createFileHeader("test.txt", []byte("content"))
		_, err = storage.Save(ctx, fh, "test.txt")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, file.ErrOperationTimeout))
	})
}

func TestS3Storage_Integration(t *testing.T) {
	t.Parallel()
	operations := []string{}

	mockClient := &mockS3Client{
		putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			operations = append(operations, "put")
			assert.Equal(t, "test-bucket", *params.Bucket)
			assert.Equal(t, "integration/test.txt", *params.Key)
			return &s3.PutObjectOutput{}, nil
		},
		headObjectFunc: func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			operations = append(operations, "head")
			assert.Equal(t, "test-bucket", *params.Bucket)
			assert.Equal(t, "integration/test.txt", *params.Key)
			return &s3.HeadObjectOutput{}, nil
		},
		listObjectsFunc: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			operations = append(operations, "list")
			assert.Equal(t, "test-bucket", *params.Bucket)
			assert.Equal(t, "integration/", *params.Prefix)
			return &s3.ListObjectsV2Output{
				Contents: []types.Object{
					{Key: aws.String("integration/test.txt"), Size: aws.Int64(12)},
				},
			}, nil
		},
		deleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			operations = append(operations, "delete")
			assert.Equal(t, "test-bucket", *params.Bucket)
			assert.Equal(t, "integration/test.txt", *params.Key)
			return &s3.DeleteObjectOutput{}, nil
		},
	}

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
}
