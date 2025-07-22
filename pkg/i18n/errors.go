package i18n

import "errors"

// Predefined package errors
var (
	// JSON parsing errors
	ErrFailedToMarshalJSON  = errors.New("failed to marshal translations to JSON")
	ErrJSONParsingCancelled = errors.New("json parsing cancelled")
	ErrFailedToParseJSON    = errors.New("failed to parse JSON content")

	// YAML parsing errors
	ErrYAMLParsingCancelled = errors.New("yaml parsing cancelled")
	ErrFailedToParseYAML    = errors.New("failed to parse YAML content")

	// File loading errors
	ErrLoadingFileCancelled = errors.New("loading translation file cancelled")
	ErrFailedToReadFile     = errors.New("failed to read translation file")
	ErrFailedToParseFile    = errors.New("failed to parse translation file")

	// Directory operations errors
	ErrFailedToAccessDirectory          = errors.New("failed to access directory")
	ErrLoadingDirectoryCancelled        = errors.New("loading from directory cancelled")
	ErrFailedToReadDirectory            = errors.New("failed to read directory")
	ErrContextCancelledDuringProcessing = errors.New("context canceled while processing directory")

	// Embedded file system errors
	ErrLoadingTranslationsCancelled  = errors.New("loading translations canceled before starting")
	ErrFailedToReadEmbeddedDirectory = errors.New("failed to read embedded directory")
	ErrLoadingEmbeddedFileCancelled  = errors.New("loading embedded translation file cancelled")
	ErrFailedToReadEmbeddedFile      = errors.New("failed to read embedded translation file")
	ErrFailedToParseEmbeddedFile     = errors.New("failed to parse embedded translation file")
)
