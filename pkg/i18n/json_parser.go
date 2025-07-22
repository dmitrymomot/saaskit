package i18n

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// JSONParser implements the Parser interface for JSON files
type JSONParser struct{}

// NewJSONParser creates a new JSONParser instance
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse parses JSON content and returns a map of translations
func (p *JSONParser) Parse(ctx context.Context, content string) (map[string]map[string]any, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(ErrJSONParsingCancelled, err)
	}

	// Parse JSON content
	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, errors.Join(ErrFailedToParseJSON, err)
	}

	// Convert to expected format
	result := make(map[string]map[string]any)
	for lang, translations := range data {
		if transMap, ok := translations.(map[string]any); ok {
			result[lang] = transMap
		}
	}

	return result, nil
}

// SupportsFileExtension checks if the parser supports the given file extension
func (p *JSONParser) SupportsFileExtension(ext string) bool {
	ext = strings.TrimPrefix(ext, ".")
	return strings.EqualFold(ext, "json")
}
