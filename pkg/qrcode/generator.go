package qrcode

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	skipqrcode "github.com/skip2/go-qrcode"
)

var (
	// ErrEmptyContent is returned when content string is empty or only whitespace
	ErrEmptyContent = errors.New("content cannot be empty")
	// ErrFailedToGenerateQRCode is returned when the QR code generation fails.
	ErrFailedToGenerateQRCode = errors.New("failed to generate QR code")
)

// defaultSize of 256px provides good balance between QR code readability
// and performance - large enough for mobile scanning, small enough for web use
const defaultSize = 256

// Generate creates a QR code for web applications where the raw PNG bytes
// are needed for custom handling or non-browser environments.
func Generate(content string, size int) ([]byte, error) {
	// Whitespace-only content creates invalid QR codes that won't scan properly
	if strings.TrimSpace(content) == "" {
		return nil, ErrEmptyContent
	}
	if size <= 0 {
		size = defaultSize
	}
	// Medium error correction balances data capacity with error recovery
	// for typical web use cases (URLs, text content)
	png, err := skipqrcode.Encode(content, skipqrcode.Medium, size)
	if err != nil {
		// errors.Join preserves the original error for debugging while providing
		// a consistent domain error for client handling
		return nil, errors.Join(ErrFailedToGenerateQRCode, err)
	}
	return png, nil
}

// GenerateBase64Image creates a base64 encoded string representation of a QR code
// image with the given content. Returns the base64 encoded string or an error if
// generation fails.
//
// Usage:
//
//	base64Image, err := GenerateBase64Image("https://dmomot.com")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// And then use the base64Image string in an HTML template like this:
//
//	<img src="{{.QrCode}}">
func GenerateBase64Image(content string, size int) (string, error) {
	png, err := Generate(content, size)
	if err != nil {
		return "", err
	}
	base64Image := base64.StdEncoding.EncodeToString(png)
	return fmt.Sprintf("data:image/png;base64,%s", base64Image), nil
}
