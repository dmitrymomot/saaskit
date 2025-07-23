package qrcode_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image/png"
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/qrcode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Parallel()
	t.Run("returns error when content is empty", func(t *testing.T) {
		t.Parallel()
		content := ""
		size := 256

		result, err := qrcode.Generate(content, size)

		require.Error(t, err, "Generate should return an error with empty content")
		require.Nil(t, result, "Generate should not return PNG data")
		assert.True(t, errors.Is(err, qrcode.ErrEmptyContent),
			"Error should be ErrEmptyContent")
	})

	t.Run("returns error when content is whitespace only", func(t *testing.T) {
		t.Parallel()
		content := "   \t\n"
		size := 256

		result, err := qrcode.Generate(content, size)

		require.Error(t, err, "Generate should return an error with whitespace-only content")
		require.Nil(t, result, "Generate should not return PNG data")
		assert.True(t, errors.Is(err, qrcode.ErrEmptyContent),
			"Error should be ErrEmptyContent")
	})

	t.Run("generates QR code with valid content and size", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 256

		result, err := qrcode.Generate(content, size)

		require.NoError(t, err, "Generate should not return an error with valid input")
		require.NotNil(t, result, "Generate should return PNG data")
		require.NotEmpty(t, result, "Generate should return non-empty PNG data")

		// Verify the result is a valid PNG
		img, err := png.Decode(bytes.NewReader(result))
		require.NoError(t, err, "Result should be a valid PNG image")

		// Verify the image dimensions
		assert.Equal(t, size, img.Bounds().Dx(), "Image width should match requested size")
		assert.Equal(t, size, img.Bounds().Dy(), "Image height should match requested size")
	})

	t.Run("uses default size when size is zero", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 0 // Should default to 256

		result, err := qrcode.Generate(content, size)

		require.NoError(t, err, "Generate should not return an error with zero size")
		require.NotNil(t, result, "Generate should return PNG data")

		// Verify the result is a valid PNG with default size
		img, err := png.Decode(bytes.NewReader(result))
		require.NoError(t, err, "Result should be a valid PNG image")

		defaultSize := 256
		assert.Equal(t, defaultSize, img.Bounds().Dx(), "Image width should be default 256px")
		assert.Equal(t, defaultSize, img.Bounds().Dy(), "Image height should be default 256px")
	})

	t.Run("uses default size when size is negative", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := -10 // Should default to 256

		result, err := qrcode.Generate(content, size)

		require.NoError(t, err, "Generate should not return an error with negative size")
		require.NotNil(t, result, "Generate should return PNG data")

		// Verify the result is a valid PNG with default size
		img, err := png.Decode(bytes.NewReader(result))
		require.NoError(t, err, "Result should be a valid PNG image")

		defaultSize := 256
		assert.Equal(t, defaultSize, img.Bounds().Dx(), "Image width should be default 256px")
		assert.Equal(t, defaultSize, img.Bounds().Dy(), "Image height should be default 256px")
	})

	t.Run("generates QR code with custom size", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 400

		result, err := qrcode.Generate(content, size)

		require.NoError(t, err, "Generate should not return an error with custom size")
		require.NotNil(t, result, "Generate should return PNG data")

		// Verify the result is a valid PNG with the specified size
		img, err := png.Decode(bytes.NewReader(result))
		require.NoError(t, err, "Result should be a valid PNG image")

		assert.Equal(t, size, img.Bounds().Dx(), "Image width should match requested size")
		assert.Equal(t, size, img.Bounds().Dy(), "Image height should match requested size")
	})
}

func TestGenerateBase64Image(t *testing.T) {
	t.Parallel()
	t.Run("returns error when content is empty", func(t *testing.T) {
		t.Parallel()
		content := ""
		size := 256

		result, err := qrcode.GenerateBase64Image(content, size)

		require.Error(t, err, "GenerateBase64Image should return an error with empty content")
		require.Empty(t, result, "GenerateBase64Image should not return data URI")
		assert.True(t, errors.Is(err, qrcode.ErrEmptyContent),
			"Error should be ErrEmptyContent")
	})

	t.Run("returns error when content is whitespace only", func(t *testing.T) {
		t.Parallel()
		content := "   \t\n"
		size := 256

		result, err := qrcode.GenerateBase64Image(content, size)

		require.Error(t, err, "GenerateBase64Image should return an error with whitespace-only content")
		require.Empty(t, result, "GenerateBase64Image should not return data URI")
		assert.True(t, errors.Is(err, qrcode.ErrEmptyContent),
			"Error should be ErrEmptyContent")
	})

	t.Run("generates base64 data URI with valid content and size", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 256

		result, err := qrcode.GenerateBase64Image(content, size)

		require.NoError(t, err, "GenerateBase64Image should not return an error with valid input")
		require.NotEmpty(t, result, "GenerateBase64Image should return non-empty data URI")

		// Verify the result has the correct data URI prefix
		expectedPrefix := "data:image/png;base64,"
		assert.True(t, strings.HasPrefix(result, expectedPrefix),
			"Result should start with the data URI prefix")

		// Verify that content after prefix is non-empty
		base64Content := strings.TrimPrefix(result, expectedPrefix)
		assert.NotEmpty(t, base64Content, "Base64 content should not be empty")
	})

	t.Run("uses default size when size is zero", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 0 // Should default to 256

		result, err := qrcode.GenerateBase64Image(content, size)

		require.NoError(t, err, "GenerateBase64Image should not return an error with zero size")
		require.NotEmpty(t, result, "GenerateBase64Image should return non-empty data URI")

		expectedPrefix := "data:image/png;base64,"
		assert.True(t, strings.HasPrefix(result, expectedPrefix),
			"Result should start with the data URI prefix")
	})

	t.Run("uses default size when size is negative", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := -10 // Should default to 256

		result, err := qrcode.GenerateBase64Image(content, size)

		require.NoError(t, err, "GenerateBase64Image should not return an error with negative size")
		require.NotEmpty(t, result, "GenerateBase64Image should return non-empty data URI")

		expectedPrefix := "data:image/png;base64,"
		assert.True(t, strings.HasPrefix(result, expectedPrefix),
			"Result should start with the data URI prefix")
	})

	t.Run("can decode base64 content to valid PNG", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 256

		result, err := qrcode.GenerateBase64Image(content, size)
		require.NoError(t, err, "GenerateBase64Image should not return an error")

		// Extract the base64 content
		expectedPrefix := "data:image/png;base64,"
		require.True(t, strings.HasPrefix(result, expectedPrefix),
			"Result should start with the data URI prefix")

		base64Content := strings.TrimPrefix(result, expectedPrefix)
		require.NotEmpty(t, base64Content, "Base64 content should not be empty")

		// Decode the base64 content to bytes
		decodedBytes, err := base64.StdEncoding.DecodeString(base64Content)
		require.NoError(t, err, "Should be able to decode base64 content")
		require.NotEmpty(t, decodedBytes, "Decoded bytes should not be empty")

		// Verify the decoded bytes are a valid PNG
		img, err := png.Decode(bytes.NewReader(decodedBytes))
		require.NoError(t, err, "Decoded content should be a valid PNG")

		// Verify the image dimensions
		assert.Equal(t, size, img.Bounds().Dx(), "Image width should match requested size")
		assert.Equal(t, size, img.Bounds().Dy(), "Image height should match requested size")
	})
}
