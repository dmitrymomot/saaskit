package i18n

import (
	"context"
	"strings"
)

// Parser is an interface for parsing internationalization (i18n) content from various file formats.
type Parser interface {
	// Parse processes the given content string and returns a nested map structure.
	// The outer map is typically keyed by locale identifier, while the inner map
	// contains translation keys and their corresponding values.
	Parse(ctx context.Context, content string) (map[string]map[string]any, error)

	// SupportsFileExtension checks if the parser supports a given file extension
	// The extension may or may not include a leading dot (e.g. both "json" and ".json" are valid)
	SupportsFileExtension(ext string) bool
}

// NewParserForFile returns a parser based on the file extension
func NewParserForFile(filename string) Parser {
	ext := getFileExtension(filename)

	switch strings.ToLower(ext) {
	case "json":
		return NewJSONParser()
	case "yaml", "yml":
		return NewYAMLParser()
	default:
		return nil
	}
}

// getFileExtension extracts the extension from a filename
func getFileExtension(filename string) string {
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		return filename[idx+1:]
	}
	return ""
}
