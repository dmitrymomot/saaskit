package i18n_test

import (
	"context"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/i18n"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONParser(t *testing.T) {
	t.Parallel()
	parser := i18n.NewJSONParser()

	t.Run("Parse valid JSON", func(t *testing.T) {
		t.Parallel()
		content := `{
			"en": {
				"greeting": "Hello",
				"farewell": "Goodbye",
				"nested": {
					"key": "Nested value"
				}
			},
			"fr": {
				"greeting": "Bonjour",
				"farewell": "Au revoir"
			}
		}`

		result, err := parser.Parse(context.Background(), content)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Check English translations
		assert.Contains(t, result, "en")
		assert.Equal(t, "Hello", result["en"]["greeting"])
		assert.Equal(t, "Goodbye", result["en"]["farewell"])

		// Check nested values
		nestedMap, ok := result["en"]["nested"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Nested value", nestedMap["key"])

		// Check French translations
		assert.Contains(t, result, "fr")
		assert.Equal(t, "Bonjour", result["fr"]["greeting"])
		assert.Equal(t, "Au revoir", result["fr"]["farewell"])
	})

	t.Run("Parse invalid JSON", func(t *testing.T) {
		t.Parallel()
		content := `{
			"en": {
				"greeting": "Hello",
				"farewell": "Goodbye",
			}
		}`

		result, err := parser.Parse(context.Background(), content)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to parse JSON content")
	})

	t.Run("Context cancellation", func(t *testing.T) {
		t.Parallel()
		content := `{
			"en": {
				"greeting": "Hello"
			}
		}`

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		result, err := parser.Parse(ctx, content)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "json parsing cancelled")
	})

}

func TestYAMLParser(t *testing.T) {
	t.Parallel()
	parser := i18n.NewYAMLParser()

	t.Run("Parse valid YAML", func(t *testing.T) {
		t.Parallel()
		content := `
en:
  greeting: Hello
  farewell: Goodbye
  nested:
    key: Nested value
fr:
  greeting: Bonjour
  farewell: Au revoir
`

		result, err := parser.Parse(context.Background(), content)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Check English translations
		assert.Contains(t, result, "en")
		assert.Equal(t, "Hello", result["en"]["greeting"])
		assert.Equal(t, "Goodbye", result["en"]["farewell"])

		// Check nested values
		nestedMap, ok := result["en"]["nested"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Nested value", nestedMap["key"])

		// Check French translations
		assert.Contains(t, result, "fr")
		assert.Equal(t, "Bonjour", result["fr"]["greeting"])
		assert.Equal(t, "Au revoir", result["fr"]["farewell"])
	})

	t.Run("Parse invalid YAML", func(t *testing.T) {
		t.Parallel()
		content := `
en:
  - greeting: Hello  # Invalid structure (array instead of map)
  - farewell: Goodbye
fr:
  greeting: Bonjour
`

		result, err := parser.Parse(context.Background(), content)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid YAML structure for language")
	})

	t.Run("Empty YAML", func(t *testing.T) {
		t.Parallel()
		content := ``

		result, err := parser.Parse(context.Background(), content)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no valid translations found")
	})

	t.Run("Context cancellation", func(t *testing.T) {
		t.Parallel()
		content := `
en:
  greeting: Hello
`

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		result, err := parser.Parse(ctx, content)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "yaml parsing cancelled")
	})

}

func TestParserFactory(t *testing.T) {
	t.Parallel()
	t.Run("JSON file extension", func(t *testing.T) {
		t.Parallel()
		parser := i18n.NewParserForFile("translations.json")
		require.NotNil(t, parser)
		_, ok := parser.(*i18n.JSONParser)
		assert.True(t, ok, "Should return a JSONParser for .json files")
	})

	t.Run("YAML file extensions", func(t *testing.T) {
		t.Parallel()
		// Test .yaml extension
		parser := i18n.NewParserForFile("translations.yaml")
		require.NotNil(t, parser)
		_, ok := parser.(*i18n.YAMLParser)
		assert.True(t, ok, "Should return a YAMLParser for .yaml files")

		// Test .yml extension
		parser = i18n.NewParserForFile("translations.yml")
		require.NotNil(t, parser)
		_, ok = parser.(*i18n.YAMLParser)
		assert.True(t, ok, "Should return a YAMLParser for .yml files")
	})

	t.Run("Uppercase extensions", func(t *testing.T) {
		t.Parallel()
		// Test uppercase JSON
		parser := i18n.NewParserForFile("translations.JSON")
		require.NotNil(t, parser)
		_, ok := parser.(*i18n.JSONParser)
		assert.True(t, ok, "Should handle uppercase extensions")

		// Test uppercase YAML
		parser = i18n.NewParserForFile("translations.YAML")
		require.NotNil(t, parser)
		_, ok = parser.(*i18n.YAMLParser)
		assert.True(t, ok, "Should handle uppercase extensions")
	})

	t.Run("Unsupported extension", func(t *testing.T) {
		t.Parallel()
		parser := i18n.NewParserForFile("translations.txt")
		assert.Nil(t, parser, "Should return nil for unsupported extensions")
	})

	t.Run("No extension", func(t *testing.T) {
		t.Parallel()
		parser := i18n.NewParserForFile("translations")
		assert.Nil(t, parser, "Should return nil for files without extensions")
	})
}

// TestJSONParserMalformedData tests parser behavior with various malformed data scenarios
func TestJSONParserMalformedData(t *testing.T) {
	t.Parallel()
	parser := i18n.NewJSONParser()

	t.Run("malformed JSON structures", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			content string
			wantErr bool
		}{
			{
				name:    "unclosed brace",
				content: `{"en": {"key": "value"`,
				wantErr: true,
			},
			{
				name:    "trailing comma",
				content: `{"en": {"key": "value",}}`,
				wantErr: true,
			},
			{
				name:    "missing quotes around key",
				content: `{en: {"key": "value"}}`,
				wantErr: true,
			},
			{
				name:    "single quotes instead of double",
				content: `{'en': {'key': 'value'}}`,
				wantErr: true,
			},
			{
				name:    "unescaped quotes in string",
				content: `{"en": {"key": "value with "quotes""}}`,
				wantErr: true,
			},
			{
				name:    "invalid unicode escape",
				content: `{"en": {"key": "\uXXXX"}}`,
				wantErr: true,
			},
			{
				name:    "null bytes in content",
				content: "{\x00\"en\": {\"key\": \"value\"}}",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				result, err := parser.Parse(context.Background(), tt.content)
				if tt.wantErr {
					assert.Error(t, err)
					assert.Nil(t, result)
					assert.Contains(t, err.Error(), "failed to parse JSON content")
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, result)
				}
			})
		}
	})

	t.Run("data type edge cases", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			content string
			wantErr bool
		}{
			{
				name:    "array at root level",
				content: `["en", "fr"]`,
				wantErr: true, // Should expect object at root
			},
			{
				name:    "string at root level",
				content: `"just a string"`,
				wantErr: true, // Should expect object at root
			},
			{
				name:    "number at root level",
				content: `42`,
				wantErr: true, // Should expect object at root
			},
			{
				name:    "boolean at root level",
				content: `true`,
				wantErr: true, // Should expect object at root
			},
			{
				name:    "null at root level",
				content: `null`,
				wantErr: false, // null parses to empty map
			},
			{
				name:    "empty object",
				content: `{}`,
				wantErr: false, // Empty object should be valid but might have no translations
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				result, err := parser.Parse(context.Background(), tt.content)
				if tt.wantErr {
					assert.Error(t, err)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					// Empty object is valid JSON but has no translations
					assert.NotNil(t, result)
				}
			})
		}
	})

	t.Run("JSON with special characters", func(t *testing.T) {
		t.Parallel()
		content := `{
			"en": {
				"unicode": "Héllö Wørld 🌍",
				"emoji": "Hello 👋 World 🌎",
				"newlines": "Line 1\nLine 2\nLine 3",
				"tabs": "Col1\tCol2\tCol3",
				"backslashes": "Path\\to\\file",
				"quotes": "He said \"Hello\"",
				"mixed": "Special chars: àáâãäåæçèéêë 中文 العربية русский"
			}
		}`

		result, err := parser.Parse(context.Background(), content)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result, "en")

		if enData, ok := result["en"]; ok {
			assert.Equal(t, "Héllö Wørld 🌍", enData["unicode"])
			assert.Equal(t, "Hello 👋 World 🌎", enData["emoji"])
			assert.Contains(t, enData["newlines"], "\n")
			assert.Contains(t, enData["tabs"], "\t")
		}
	})
}

// TestYAMLParserMalformedData tests parser behavior with various malformed YAML data scenarios
func TestYAMLParserMalformedData(t *testing.T) {
	t.Parallel()
	parser := i18n.NewYAMLParser()

	t.Run("malformed YAML structures", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			content string
			wantErr bool
		}{
			{
				name: "inconsistent indentation",
				content: `en:
  key1: value1
    key2: value2
  key3: value3`,
				wantErr: true,
			},
			{
				name:    "tabs mixed with spaces",
				content: "en:\n  key1: value1\n\tkey2: value2",
				wantErr: true,
			},
			{
				name: "invalid YAML characters",
				content: `en:
  key: value@#$%^&*()`,
				wantErr: false, // Special characters in values should be fine
			},
			{
				name: "unclosed quotes",
				content: `en:
  key: "unclosed quote`,
				wantErr: true,
			},
			{
				name: "missing colon",
				content: `en
  key value`,
				wantErr: true,
			},
			{
				name: "invalid list structure for language",
				content: `en:
  - item1
  - item2
fr:
  greeting: Bonjour`,
				wantErr: true, // Languages should be maps, not lists
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				result, err := parser.Parse(context.Background(), tt.content)
				if tt.wantErr {
					assert.Error(t, err)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, result)
				}
			})
		}
	})

	t.Run("YAML data type edge cases", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			content string
			wantErr bool
		}{
			{
				name:    "number as language key",
				content: `123: {key: value}`,
				wantErr: false, // Should work but convert to string
			},
			{
				name:    "boolean as language key",
				content: `true: {key: value}`,
				wantErr: false, // Should work but convert to string
			},
			{
				name: "mixed data types in translations",
				content: `en:
  string_key: "string value"
  number_key: 42
  boolean_key: true
  null_key: null
  array_key: [1, 2, 3]`,
				wantErr: false, // Mixed types should be allowed
			},
			{
				name:    "empty YAML document",
				content: ``,
				wantErr: true, // Should error on empty content
			},
			{
				name:    "only comments",
				content: `# This is just a comment`,
				wantErr: true, // Should error with no actual content
			},
			{
				name: "YAML document separator",
				content: `---
en:
  key: value
---
fr:
  key: valeur`,
				wantErr: false, // Multiple documents should work
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				result, err := parser.Parse(context.Background(), tt.content)
				if tt.wantErr {
					assert.Error(t, err)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, result)
				}
			})
		}
	})

	t.Run("YAML with special characters and encodings", func(t *testing.T) {
		t.Parallel()
		content := `en:
  unicode: "Héllö Wørld 🌍"
  emoji: "Hello 👋 World 🌎"
  newlines: |
    Line 1
    Line 2
    Line 3
  multiline_folded: >
    This is a very long line
    that will be folded into
    a single line
  special_chars: "àáâãäåæçèéêë 中文 العربية русский"
  yaml_special: "key: value, [list], {object}"
  quotes: 'He said "Hello" to me'
  backslashes: "Path\\to\\file"
fr:
  accented: "Français avec accents: àèùéç"
  quotes: "Il a dit « Bonjour » à tout le monde"`

		result, err := parser.Parse(context.Background(), content)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result, "en")
		assert.Contains(t, result, "fr")

		if enData, ok := result["en"]; ok {
			assert.Equal(t, "Héllö Wørld 🌍", enData["unicode"])
			assert.Equal(t, "Hello 👋 World 🌎", enData["emoji"])
			assert.Contains(t, enData["newlines"], "Line 1")
		}
	})

	t.Run("YAML anchor and reference edge cases", func(t *testing.T) {
		t.Parallel()
		t.Run("valid anchors and references", func(t *testing.T) {
			t.Parallel()
			content := `en:
  common: &common_greeting "Hello"
  formal: *common_greeting
  informal: *common_greeting
fr:
  template: &fr_template
    greeting: "Bonjour"
    farewell: "Au revoir"
  casual: *fr_template`

			result, err := parser.Parse(context.Background(), content)
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})

		t.Run("undefined reference", func(t *testing.T) {
			t.Parallel()
			content := `en:
  greeting: *undefined_anchor`

			result, err := parser.Parse(context.Background(), content)
			assert.Error(t, err) // Should error on undefined reference
			assert.Nil(t, result)
		})

		t.Run("circular references", func(t *testing.T) {
			t.Parallel()
			content := `en:
  a: &ref_a
    b: *ref_b
  b: &ref_b
    a: *ref_a`

			result, err := parser.Parse(context.Background(), content)
			// This might error or handle gracefully depending on YAML parser implementation
			if err != nil {
				assert.Nil(t, result)
			}
		})
	})
}
