package i18n

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLParser implements the Parser interface for YAML files
type YAMLParser struct{}

// NewYAMLParser creates a new YAMLParser instance
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// Parse parses YAML content and returns a map of translations
func (p *YAMLParser) Parse(ctx context.Context, content string) (map[string]map[string]any, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(ErrYAMLParsingCancelled, err)
	}

	// Parse YAML content
	var data map[string]any
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return nil, errors.Join(ErrFailedToParseYAML, err)
	}

	// Convert to expected format
	result := make(map[string]map[string]any)
	for lang, val := range data {
		// Handle nested maps
		if transMap, ok := val.(map[string]any); ok {
			result[lang] = transMap
		} else {
			// If it's not a map, create a map with a single entry
			// This handles simple key-value pairs at the root level
			return nil, fmt.Errorf("invalid YAML structure for language '%s': expected map, got %T", lang, val)
		}
	}

	// Validate the result
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid translations found in YAML content")
	}

	return result, nil
}

// SupportsFileExtension checks if the parser supports the given file extension
func (p *YAMLParser) SupportsFileExtension(ext string) bool {
	ext = strings.TrimPrefix(ext, ".")
	return strings.EqualFold(ext, "yaml") || strings.EqualFold(ext, "yml")
}
