package randomname_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmitrymomot/saaskit/pkg/randomname"
)

func TestWordLists(t *testing.T) {
	// Test expected word counts from the refactoring plan
	tests := []struct {
		wordType      randomname.WordType
		expectedCount int
		examples      []string
	}{
		{
			wordType:      randomname.Adjective,
			expectedCount: 142,
			examples:      []string{"brave", "mighty", "swift", "clever", "elegant"},
		},
		{
			wordType:      randomname.Noun,
			expectedCount: 160,
			examples:      []string{"tiger", "eagle", "dolphin", "fox", "wolf"},
		},
		{
			wordType:      randomname.Color,
			expectedCount: 40,
			examples:      []string{"red", "blue", "crimson", "azure", "quantum"},
		},
		{
			wordType:      randomname.Size,
			expectedCount: 25,
			examples:      []string{"tiny", "small", "huge", "massive", "nano"},
		},
		{
			wordType:      randomname.Origin,
			expectedCount: 30,
			examples:      []string{"arctic", "tropical", "lunar", "cosmic", "urban"},
		},
		{
			wordType:      randomname.Action,
			expectedCount: 40,
			examples:      []string{"flying", "running", "dancing", "blazing", "soaring"},
		},
	}

	for _, tt := range tests {
		t.Run("word type validation", func(t *testing.T) {
			// Generate many names with single word type to collect unique words
			seen := make(map[string]bool)
			for i := 0; i < 500; i++ {
				name := randomname.Generate(&randomname.Options{
					Pattern: []randomname.WordType{tt.wordType},
				})
				seen[name] = true
			}

			// Check that we're seeing a reasonable variety
			assert.Greater(t, len(seen), tt.expectedCount/3, "Should see at least 1/3 of available words")

			// Verify examples are being used
			foundExample := false
			for word := range seen {
				for _, example := range tt.examples {
					if word == example {
						foundExample = true
						break
					}
				}
				if foundExample {
					break
				}
			}
			assert.True(t, foundExample, "Should find at least one example word")
		})
	}
}

func TestCustomWordsMerging(t *testing.T) {
	t.Run("custom words are used alongside defaults", func(t *testing.T) {
		customAdj := []string{"testadjone", "testadjtwo"}
		customNoun := []string{"testnounone", "testnountwo"}

		seenCustomAdj := false
		seenDefaultAdj := false
		seenCustomNoun := false
		seenDefaultNoun := false

		// Generate enough names to likely see both custom and default words
		for i := 0; i < 500; i++ {
			name := randomname.Generate(&randomname.Options{
				Words: map[randomname.WordType][]string{
					randomname.Adjective: customAdj,
					randomname.Noun:      customNoun,
				},
			})

			parts := strings.Split(name, "-")
			adj := parts[0]
			noun := parts[1]

			// Check adjectives
			if adj == "testadjone" || adj == "testadjtwo" {
				seenCustomAdj = true
			} else {
				seenDefaultAdj = true
			}

			// Check nouns
			if noun == "testnounone" || noun == "testnountwo" {
				seenCustomNoun = true
			} else {
				seenDefaultNoun = true
			}

			// Break early if we've seen all types
			if seenCustomAdj && seenDefaultAdj && seenCustomNoun && seenDefaultNoun {
				break
			}
		}

		assert.True(t, seenCustomAdj, "Should see custom adjectives")
		assert.True(t, seenDefaultAdj, "Should see default adjectives")
		assert.True(t, seenCustomNoun, "Should see custom nouns")
		assert.True(t, seenDefaultNoun, "Should see default nouns")
	})

	t.Run("custom words are merged with defaults", func(t *testing.T) {
		customWords := map[randomname.WordType][]string{
			randomname.Adjective: {"customadj", "testadj", "myadj"},
			randomname.Noun:      {"customnoun", "testnoun", "mynoun"},
		}

		// Custom words should be used along with defaults
		seenCustomAdj := false
		seenCustomNoun := false
		for i := 0; i < 1000; i++ {
			name := randomname.Generate(&randomname.Options{
				Words: customWords,
			})
			parts := strings.Split(name, "-")
			if len(parts) >= 2 {
				// Check if we see any custom adjectives
				if parts[0] == "customadj" || parts[0] == "testadj" || parts[0] == "myadj" {
					seenCustomAdj = true
				}
				// Check if we see any custom nouns
				if parts[1] == "customnoun" || parts[1] == "testnoun" || parts[1] == "mynoun" {
					seenCustomNoun = true
				}
			}
			if seenCustomAdj && seenCustomNoun {
				break
			}
		}
		assert.True(t, seenCustomAdj, "Should see custom adjectives in generation")
		assert.True(t, seenCustomNoun, "Should see custom nouns in generation")
	})

	t.Run("partial custom words", func(t *testing.T) {
		// Only provide custom colors, use default adjectives and nouns
		customWords := map[randomname.WordType][]string{
			randomname.Color: {"testcolorone", "testcolortwo"},
		}

		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			name := randomname.Generate(&randomname.Options{
				Pattern: []randomname.WordType{randomname.Adjective, randomname.Color, randomname.Noun},
				Words:   customWords,
			})
			parts := strings.Split(name, "-")
			if len(parts) >= 3 {
				seen[parts[1]] = true // Color is the middle word
			}
		}

		// Should see our custom colors
		hasCustomColor := false
		for color := range seen {
			if color == "testcolorone" || color == "testcolortwo" {
				hasCustomColor = true
				break
			}
		}
		assert.True(t, hasCustomColor, "Should use custom colors")
	})
}

func TestWordTypesCoverage(t *testing.T) {
	// Test that all word types can be used in various combinations
	patterns := [][]randomname.WordType{
		{randomname.Adjective},
		{randomname.Noun},
		{randomname.Color},
		{randomname.Size},
		{randomname.Origin},
		{randomname.Action},
		{randomname.Size, randomname.Color},
		{randomname.Action, randomname.Noun},
		{randomname.Origin, randomname.Adjective, randomname.Noun},
		{randomname.Size, randomname.Action, randomname.Color, randomname.Noun},
	}

	for _, pattern := range patterns {
		t.Run("pattern generation", func(t *testing.T) {
			name := randomname.Generate(&randomname.Options{
				Pattern: pattern,
			})
			parts := strings.Split(name, "-")
			assert.Len(t, parts, len(pattern))
			assert.NotEmpty(t, name)
		})
	}
}
