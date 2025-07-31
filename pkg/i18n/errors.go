package i18n

import "errors"

// Package errors use descriptive messages for debugging while avoiding implementation details.
// Context cancellation errors are separated to allow proper error handling in timeouts.
var (
	// JSON operations
	ErrFailedToMarshalJSON  = errors.New("failed to marshal translations to JSON")
	ErrJSONParsingCancelled = errors.New("json parsing cancelled")
	ErrFailedToParseJSON    = errors.New("failed to parse JSON content")

	// YAML operations
	ErrYAMLParsingCancelled = errors.New("yaml parsing cancelled")
	ErrFailedToParseYAML    = errors.New("failed to parse YAML content")

	// File operations
	ErrLoadingFileCancelled = errors.New("loading translation file cancelled")
	ErrFailedToReadFile     = errors.New("failed to read translation file")
	ErrFailedToParseFile    = errors.New("failed to parse translation file")

	// Directory operations
	ErrFailedToAccessDirectory          = errors.New("failed to access directory")
	ErrLoadingDirectoryCancelled        = errors.New("loading from directory cancelled")
	ErrFailedToReadDirectory            = errors.New("failed to read directory")
	ErrContextCancelledDuringProcessing = errors.New("context canceled while processing directory")

	// Embedded filesystem operations
	ErrLoadingTranslationsCancelled  = errors.New("loading translations canceled before starting")
	ErrFailedToReadEmbeddedDirectory = errors.New("failed to read embedded directory")
	ErrLoadingEmbeddedFileCancelled  = errors.New("loading embedded translation file cancelled")
	ErrFailedToReadEmbeddedFile      = errors.New("failed to read embedded translation file")
	ErrFailedToParseEmbeddedFile     = errors.New("failed to parse embedded translation file")
)
