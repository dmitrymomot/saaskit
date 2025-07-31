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
		size := 0 // Should fall back to 256px default

		result, err := qrcode.Generate(content, size)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Critical: verify fallback to 256px when invalid size provided
		img, err := png.Decode(bytes.NewReader(result))
		require.NoError(t, err)

		defaultSize := 256
		assert.Equal(t, defaultSize, img.Bounds().Dx())
		assert.Equal(t, defaultSize, img.Bounds().Dy())
	})

	t.Run("uses default size when size is negative", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := -10 // Should fall back to 256px default

		result, err := qrcode.Generate(content, size)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Critical: verify fallback to 256px when invalid size provided
		img, err := png.Decode(bytes.NewReader(result))
		require.NoError(t, err)

		defaultSize := 256
		assert.Equal(t, defaultSize, img.Bounds().Dx())
		assert.Equal(t, defaultSize, img.Bounds().Dy())
	})

	t.Run("generates QR code with custom size", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 400

		result, err := qrcode.Generate(content, size)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify custom sizing works for different display contexts
		img, err := png.Decode(bytes.NewReader(result))
		require.NoError(t, err)
		assert.Equal(t, size, img.Bounds().Dx())
		assert.Equal(t, size, img.Bounds().Dy())
	})
}

func TestGenerateBase64Image(t *testing.T) {
	t.Parallel()
	t.Run("returns error when content is empty", func(t *testing.T) {
		t.Parallel()
		content := ""
		size := 256

		result, err := qrcode.GenerateBase64Image(content, size)

		require.Error(t, err)
		require.Empty(t, result)
		assert.True(t, errors.Is(err, qrcode.ErrEmptyContent))
	})

	t.Run("returns error when content is whitespace only", func(t *testing.T) {
		t.Parallel()
		content := "   \t\n"
		size := 256

		result, err := qrcode.GenerateBase64Image(content, size)

		require.Error(t, err)
		require.Empty(t, result)
		assert.True(t, errors.Is(err, qrcode.ErrEmptyContent))
	})

	t.Run("generates base64 data URI with valid content and size", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 256

		result, err := qrcode.GenerateBase64Image(content, size)

		require.NoError(t, err)
		require.NotEmpty(t, result)

		// Verify proper data URI format for direct embedding in HTML img tags
		expectedPrefix := "data:image/png;base64,"
		assert.True(t, strings.HasPrefix(result, expectedPrefix))

		base64Content := strings.TrimPrefix(result, expectedPrefix)
		assert.NotEmpty(t, base64Content)
	})

	t.Run("uses default size when size is zero", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 0 // Should fall back to 256px default

		result, err := qrcode.GenerateBase64Image(content, size)

		require.NoError(t, err)
		require.NotEmpty(t, result)

		expectedPrefix := "data:image/png;base64,"
		assert.True(t, strings.HasPrefix(result, expectedPrefix))
	})

	t.Run("uses default size when size is negative", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := -10 // Should fall back to 256px default

		result, err := qrcode.GenerateBase64Image(content, size)

		require.NoError(t, err)
		require.NotEmpty(t, result)

		expectedPrefix := "data:image/png;base64,"
		assert.True(t, strings.HasPrefix(result, expectedPrefix))
	})

	t.Run("can decode base64 content to valid PNG", func(t *testing.T) {
		t.Parallel()
		content := "https://example.com"
		size := 256

		result, err := qrcode.GenerateBase64Image(content, size)
		require.NoError(t, err)

		// Verify round-trip encoding/decoding integrity for web applications
		expectedPrefix := "data:image/png;base64,"
		require.True(t, strings.HasPrefix(result, expectedPrefix))

		base64Content := strings.TrimPrefix(result, expectedPrefix)
		require.NotEmpty(t, base64Content)

		decodedBytes, err := base64.StdEncoding.DecodeString(base64Content)
		require.NoError(t, err)
		require.NotEmpty(t, decodedBytes)

		img, err := png.Decode(bytes.NewReader(decodedBytes))
		require.NoError(t, err)
		assert.Equal(t, size, img.Bounds().Dx())
		assert.Equal(t, size, img.Bounds().Dy())
	})
}
