package randomname_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/randomname"
)

func TestDefaultOptions(t *testing.T) {
	opts := randomname.Generate(nil)
	assert.NotEmpty(t, opts)
	// Should generate with default pattern
	assert.Regexp(t, `^[a-z]+-[a-z]+$`, opts)
}

func TestOptionsMerge(t *testing.T) {
	tests := []struct {
		name     string
		opts     *randomname.Options
		expected string
	}{
		{
			name:     "nil options uses defaults",
			opts:     nil,
			expected: `^[a-z]+-[a-z]+$`,
		},
		{
			name: "custom separator",
			opts: &randomname.Options{
				Separator: "_",
			},
			expected: `^[a-z]+_[a-z]+$`,
		},
		{
			name: "custom pattern",
			opts: &randomname.Options{
				Pattern: []randomname.WordType{randomname.Color, randomname.Noun},
			},
			expected: `^[a-z]+-[a-z]+$`,
		},
		{
			name: "empty pattern falls back to default",
			opts: &randomname.Options{
				Pattern: []randomname.WordType{},
			},
			expected: `^[a-z]+-[a-z]+$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := randomname.Generate(tt.opts)
			assert.Regexp(t, tt.expected, name)
		})
	}
}

func TestSuffixTypes(t *testing.T) {
	tests := []struct {
		name    string
		suffix  randomname.SuffixType
		pattern string
	}{
		{
			name:    "no suffix",
			suffix:  randomname.NoSuffix,
			pattern: `^[a-z]+-[a-z]+$`,
		},
		{
			name:    "hex6 suffix",
			suffix:  randomname.Hex6,
			pattern: `^[a-z]+-[a-z]+-[0-9a-f]{6}$`,
		},
		{
			name:    "hex8 suffix",
			suffix:  randomname.Hex8,
			pattern: `^[a-z]+-[a-z]+-[0-9a-f]{8}$`,
		},
		{
			name:    "numeric4 suffix",
			suffix:  randomname.Numeric4,
			pattern: `^[a-z]+-[a-z]+-\d{4}$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := randomname.Generate(&randomname.Options{
				Suffix: tt.suffix,
			})
			assert.Regexp(t, tt.pattern, name)
		})
	}
}

func TestWordTypes(t *testing.T) {
	// Test that all word types have words defined
	wordTypes := []randomname.WordType{
		randomname.Adjective,
		randomname.Noun,
		randomname.Color,
		randomname.Size,
		randomname.Origin,
		randomname.Action,
	}

	for _, wordType := range wordTypes {
		t.Run("word type availability", func(t *testing.T) {
			// Generate with single word type
			name := randomname.Generate(&randomname.Options{
				Pattern: []randomname.WordType{wordType},
			})
			assert.NotEmpty(t, name)
			assert.Regexp(t, `^[a-z]+$`, name)
		})
	}
}

func TestValidator(t *testing.T) {
	t.Run("accept first attempt", func(t *testing.T) {
		attempts := 0
		name := randomname.Generate(&randomname.Options{
			Validator: func(s string) bool {
				attempts++
				return true
			},
		})
		assert.NotEmpty(t, name)
		assert.Equal(t, 1, attempts)
	})

	t.Run("reject first few attempts", func(t *testing.T) {
		attempts := 0
		name := randomname.Generate(&randomname.Options{
			Validator: func(s string) bool {
				attempts++
				return attempts >= 3
			},
		})
		assert.NotEmpty(t, name)
		assert.Equal(t, 3, attempts)
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		attempts := 0
		name := randomname.Generate(&randomname.Options{
			Validator: func(s string) bool {
				attempts++
				return false // Always reject
			},
		})
		assert.NotEmpty(t, name)
		assert.Equal(t, 100, attempts) // Should try exactly 100 times
	})
}
