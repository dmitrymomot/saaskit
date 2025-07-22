package qrcode

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	skipqrcode "github.com/skip2/go-qrcode"
)

// Error variables for QR code generation
var (
	// ErrEmptyContent is returned when content string is empty or only whitespace
	ErrEmptyContent = errors.New("content cannot be empty")
	// ErrorFailedToGenerateQRCode is returned when the QR code generation fails.
	ErrorFailedToGenerateQRCode = errors.New("failed to generate QR code")
)

// defaultSize is the size in pixels used when no size is specified
const defaultSize = 256

// Generate creates a QR code image in PNG format with the given content.
// Returns the image as a byte slice or an error if generation fails.
func Generate(content string, size int) ([]byte, error) {
	if strings.TrimSpace(content) == "" {
		return nil, ErrEmptyContent
	}
	if size <= 0 {
		size = defaultSize
	}
	png, err := skipqrcode.Encode(content, skipqrcode.Medium, size)
	if err != nil {
		return nil, errors.Join(ErrorFailedToGenerateQRCode, err)
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
