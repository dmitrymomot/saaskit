package i18n

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

// TranslationAdapter interface defines how translations are loaded
type TranslationAdapter interface {
	Load(ctx context.Context) (map[string]map[string]any, error)
}

// MapAdapter is a simple adapter that uses an in-memory map as the translation source
type MapAdapter struct {
	Data map[string]map[string]any
}

// Load implements the TranslationAdapter interface
func (a *MapAdapter) Load(_ context.Context) (map[string]map[string]any, error) {
	if a.Data == nil {
		return make(map[string]map[string]any), nil
	}
	return a.Data, nil
}

// FileAdapter is a simple adapter that uses a file as the translation source
// It implements the TranslationAdapter interface
type FileAdapter struct {
	parser Parser
	path   string
}

// NewFileAdapter creates a new FileAdapter instance
// Returns nil if parser is nil or path is empty
func NewFileAdapter(parser Parser, path string) *FileAdapter {
	// Validate inputs
	if parser == nil {
		return nil
	}
	if path == "" {
		return nil
	}
	return &FileAdapter{parser: parser, path: path}
}

// Load implements the TranslationAdapter interface
func (a *FileAdapter) Load(ctx context.Context) (map[string]map[string]any, error) {
	// Validate adapter state
	if a.parser == nil {
		return nil, fmt.Errorf("parser is nil")
	}
	if a.path == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	// Use context for cancellation
	// Create a channel for done signal
	done := make(chan struct{})
	var content []byte
	var readErr error

	// Start file reading in a goroutine
	go func() {
		content, readErr = os.ReadFile(a.path)
		close(done)
	}()

	// Wait for either context cancellation or file reading completion
	select {
	case <-ctx.Done():
		return nil, errors.Join(ErrLoadingFileCancelled, ctx.Err())
	case <-done:
		// Continue with normal processing
	}

	// Handle file reading error with context
	if readErr != nil {
		return nil, errors.Join(ErrFailedToReadFile, readErr)
	}

	// Handle empty files
	if len(content) == 0 {
		return nil, fmt.Errorf("translation file '%s' is empty", a.path)
	}

	// Parse the content with improved error handling
	translations, err := a.parser.Parse(ctx, string(content))
	if err != nil {
		return nil, errors.Join(ErrFailedToParseFile, err)
	}

	// Validate parsed content
	if translations == nil {
		return nil, fmt.Errorf("parser returned nil translations for file '%s'", a.path)
	}

	return translations, nil
}

// DirectoryAdapter is a simple adapter that uses a directory as the translation source
// It implements the TranslationAdapter interface
type DirectoryAdapter struct {
	parser Parser
	path   string
}

// NewDirectoryAdapter creates a new DirectoryAdapter instance
// Returns nil if parser is nil or path is empty
func NewDirectoryAdapter(parser Parser, path string) *DirectoryAdapter {
	// Validate inputs
	if parser == nil {
		return nil
	}
	if path == "" {
		return nil
	}
	return &DirectoryAdapter{
		parser: parser,
		path:   path,
	}
}

// Load implements the TranslationAdapter interface
func (a *DirectoryAdapter) Load(ctx context.Context) (map[string]map[string]any, error) {
	// Validate adapter state
	if a.parser == nil {
		return nil, fmt.Errorf("parser is nil")
	}
	if a.path == "" {
		return nil, fmt.Errorf("directory path is empty")
	}

	// Check if directory exists and is actually a directory
	fileInfo, err := os.Stat(a.path)
	if err != nil {
		return nil, errors.Join(ErrFailedToAccessDirectory, err)
	}
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("path '%s' is not a directory", a.path)
	}

	// Create result map to store all translations
	allTranslations := make(map[string]map[string]any)

	// Process each file in a controlled manner, respecting context cancellation
	err = a.processDirectory(ctx, allTranslations)
	if err != nil {
		return nil, err
	}

	// Check if we found any translations
	if len(allTranslations) == 0 {
		return nil, fmt.Errorf("no valid translation files found in directory '%s'", a.path)
	}

	return allTranslations, nil
}

// processDirectory processes all files in the directory
func (a *DirectoryAdapter) processDirectory(ctx context.Context, allTranslations map[string]map[string]any) error {
	// Create a channel to signal directory reading is done
	done := make(chan struct{})
	var entries []os.DirEntry
	var readErr error

	// Start reading directory in a goroutine to respect context cancellation
	go func() {
		entries, readErr = os.ReadDir(a.path)
		close(done)
	}()

	// Wait for either context cancellation or reading completion
	select {
	case <-ctx.Done():
		return errors.Join(ErrLoadingDirectoryCancelled, ctx.Err())
	case <-done:
		// Continue with processing
	}

	// Handle directory reading error
	if readErr != nil {
		return errors.Join(ErrFailedToReadDirectory, readErr)
	}

	// Process each file
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Get file extension
		ext := filepath.Ext(entry.Name())
		// Remove the leading dot if present
		if len(ext) > 0 && ext[0] == '.' {
			ext = ext[1:]
		}

		// Check if parser supports this file extension
		if !a.parser.SupportsFileExtension(ext) {
			continue
		}

		// Check context before processing each file
		if ctx.Err() != nil {
			return errors.Join(ErrContextCancelledDuringProcessing, ctx.Err())
		}

		// Process the file
		filePath := filepath.Join(a.path, entry.Name())
		err := a.processFile(ctx, filePath, allTranslations)
		if err != nil {
			// Log the error but continue with other files
			// This makes the adapter more resilient to single file failures
			fmt.Printf("Warning: failed to process file '%s': %v\n", filePath, err)
			continue
		}
	}

	return nil
}

// processFile reads and parses a single file, merging its translations into the result map
func (a *DirectoryAdapter) processFile(ctx context.Context, filePath string, allTranslations map[string]map[string]any) error {
	// Create a channel for done signal
	done := make(chan struct{})
	var content []byte
	var readErr error

	// Start file reading in a goroutine
	go func() {
		content, readErr = os.ReadFile(filePath)
		close(done)
	}()

	// Wait for either context cancellation or file reading completion
	select {
	case <-ctx.Done():
		return errors.Join(ErrLoadingFileCancelled, ctx.Err())
	case <-done:
		// Continue with normal processing
	}

	// Handle file reading error
	if readErr != nil {
		return errors.Join(ErrFailedToReadFile, readErr)
	}

	// Skip empty files
	if len(content) == 0 {
		return fmt.Errorf("translation file '%s' is empty", filePath)
	}

	// Parse the content
	fileTranslations, err := a.parser.Parse(ctx, string(content))
	if err != nil {
		return errors.Join(ErrFailedToParseFile, err)
	}

	// Skip if parser returned nil
	if fileTranslations == nil {
		return fmt.Errorf("parser returned nil translations for file '%s'", filePath)
	}

	// Merge translations from this file into the overall result
	for lang, translations := range fileTranslations {
		if allTranslations[lang] == nil {
			allTranslations[lang] = make(map[string]any)
		}
		// Merge translations for this language
		maps.Copy(allTranslations[lang], translations)
	}

	return nil
}

// EmbeddedFsAdapter is an adapter that uses Go's embed.FS as the translation source
// It implements the TranslationAdapter interface
type EmbeddedFsAdapter struct {
	parser Parser
	fs     embed.FS
	dir    string // Directory in the embedded filesystem
}

// NewEmbeddedFsAdapter creates a new EmbeddedFsAdapter instance
// Returns nil if parser is nil, fs is nil, or dir is empty
func NewEmbeddedFsAdapter(parser Parser, fs embed.FS, dir string) *EmbeddedFsAdapter {
	if parser == nil {
		return nil
	}

	if dir == "" {
		return nil
	}

	return &EmbeddedFsAdapter{
		parser: parser,
		fs:     fs,
		dir:    dir,
	}
}

// Load implements the TranslationAdapter interface
func (a *EmbeddedFsAdapter) Load(ctx context.Context) (map[string]map[string]any, error) {
	// Check for context cancellation before starting
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(ErrLoadingTranslationsCancelled, err)
	}

	// Read directory entries
	entries, err := a.fs.ReadDir(a.dir)
	if err != nil {
		return nil, errors.Join(ErrFailedToReadEmbeddedDirectory, err)
	}

	// Check if there are any files to process
	if len(entries) == 0 {
		return nil, fmt.Errorf("no files found in embedded directory '%s'", a.dir)
	}

	// Create the result map to hold all translations
	allTranslations := make(map[string]map[string]any)

	// Flag to check if we processed at least one valid file
	validFileProcessed := false

	// Process each file in the directory
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Check if this file's extension is supported by the parser
		filename := entry.Name()
		ext := filepath.Ext(filename)
		if ext == "" || !a.parser.SupportsFileExtension(ext[1:]) {
			continue
		}

		// Construct full path to the file in the embedded filesystem
		filePath := filepath.Join(a.dir, filename)

		// Process the file and merge translations
		if err := a.processFile(ctx, filePath, allTranslations); err != nil {
			// Log the error but continue processing other files
			fmt.Printf("Warning: failed to process file '%s': %v\n", filePath, err)
			continue
		}

		validFileProcessed = true
	}

	// If no valid files were processed, return an error
	if !validFileProcessed {
		return nil, fmt.Errorf("no valid translation files found in embedded directory '%s'", a.dir)
	}

	return allTranslations, nil
}

// processFile reads and parses a single file from the embedded filesystem,
// merging its translations into the result map
func (a *EmbeddedFsAdapter) processFile(ctx context.Context, filePath string, allTranslations map[string]map[string]any) error {
	// Create channels for done signal and results
	done := make(chan struct{})
	var content []byte
	var readErr error

	// Start file reading in a goroutine
	go func() {
		content, readErr = a.fs.ReadFile(filePath)
		close(done)
	}()

	// Wait for either context cancellation or file reading completion
	select {
	case <-ctx.Done():
		return errors.Join(ErrLoadingEmbeddedFileCancelled, ctx.Err())
	case <-done:
		// Continue with normal processing
	}

	// Handle file reading error
	if readErr != nil {
		return errors.Join(ErrFailedToReadEmbeddedFile, readErr)
	}

	// Skip empty files
	if len(content) == 0 {
		return fmt.Errorf("embedded translation file '%s' is empty", filePath)
	}

	// Parse the file content
	fileTranslations, err := a.parser.Parse(ctx, string(content))
	if err != nil {
		return errors.Join(ErrFailedToParseEmbeddedFile, err)
	}

	// Check if parsing returned nil translations
	if fileTranslations == nil {
		return fmt.Errorf("parser returned nil translations for embedded file '%s'", filePath)
	}

	// Merge translations from this file into the overall result
	for lang, translations := range fileTranslations {
		if allTranslations[lang] == nil {
			allTranslations[lang] = make(map[string]any)
		}
		// Merge translations for this language
		maps.Copy(allTranslations[lang], translations)
	}

	return nil
}
