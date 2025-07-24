// Package qrcode provides simple helpers for generating QR code images either
// as raw PNG bytes or as a data-URI string that can be embedded directly into
// HTML pages.
//
// The package is a thin wrapper around github.com/skip2/go-qrcode that adds
// sensible defaults, input validation, and convenient helpers for web
// applications.
//
// # Architecture
//
// The core of the package lives in the Generate and GenerateBase64Image
// functions. Both functions delegate QR-code generation to the upstream
// library and then post-process the result:
//
//   • Generate validates the input and returns a PNG image in a byte slice.
//   • GenerateBase64Image builds upon Generate and returns a data-URI
//     (base64-encoded PNG) which can be used inside an <img> tag.
//
// Errors that can be returned are declared as package-level variables so they
// can be compared with errors.Is.
//
// # Usage
//
//	import "github.com/dmitrymomot/saaskit/pkg/qrcode"
//
//	// Create PNG bytes
//	img, err := qrcode.Generate("https://example.com", 256)
//	if err != nil {
//		// handle error
//	}
//
//	// Create base64 data URI
//	dataURI, err := qrcode.GenerateBase64Image("https://example.com", 256)
//	if err != nil {
//		// handle error
//	}
//
// # Error Handling
//
// The functions return well-defined sentinel errors:
//
//   • ErrEmptyContent             – the content argument was empty.
//   • ErrorFailedToGenerateQRCode – the underlying library could not
//     generate the QR code.
//
// Wrap your error handling with errors.Is for robust comparisons.
//
// See the package tests for more usage examples.
package qrcode
