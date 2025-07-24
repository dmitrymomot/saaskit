package randomname_test

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/randomname"
)

func TestGenerate(t *testing.T) {
	t.Run("default pattern", func(t *testing.T) {
		name := randomname.Generate(nil)
		assert.Regexp(t, `^[a-z]+-[a-z]+$`, name)
		parts := strings.Split(name, "-")
		assert.Len(t, parts, 2)
	})

	t.Run("custom patterns", func(t *testing.T) {
		tests := []struct {
			name    string
			pattern []randomname.WordType
			regex   string
			parts   int
		}{
			{
				name:    "single word",
				pattern: []randomname.WordType{randomname.Noun},
				regex:   `^[a-z]+$`,
				parts:   1,
			},
			{
				name:    "color-noun",
				pattern: []randomname.WordType{randomname.Color, randomname.Noun},
				regex:   `^[a-z]+-[a-z]+$`,
				parts:   2,
			},
			{
				name:    "three words",
				pattern: []randomname.WordType{randomname.Adjective, randomname.Color, randomname.Noun},
				regex:   `^[a-z]+-[a-z]+-[a-z]+$`,
				parts:   3,
			},
			{
				name:    "four words",
				pattern: []randomname.WordType{randomname.Size, randomname.Adjective, randomname.Color, randomname.Noun},
				regex:   `^[a-z]+-[a-z]+-[a-z]+-[a-z]+$`,
				parts:   4,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				name := randomname.Generate(&randomname.Options{
					Pattern: tt.pattern,
				})
				assert.Regexp(t, tt.regex, name)
				if tt.parts > 1 {
					parts := strings.Split(name, "-")
					assert.Len(t, parts, tt.parts)
				}
			})
		}
	})

	t.Run("custom separator", func(t *testing.T) {
		separators := []string{"_", ".", " ", "--"}
		for _, sep := range separators {
			t.Run(fmt.Sprintf("separator=%q", sep), func(t *testing.T) {
				name := randomname.Generate(&randomname.Options{
					Separator: sep,
				})
				assert.Contains(t, name, sep)
			})
		}

		// Empty separator is not supported since it merges to default
		t.Run("empty separator uses default", func(t *testing.T) {
			name := randomname.Generate(&randomname.Options{
				Separator: "",
			})
			assert.Contains(t, name, "-")
		})
	})

	t.Run("with suffixes", func(t *testing.T) {
		tests := []struct {
			suffix  randomname.SuffixType
			pattern string
		}{
			{randomname.Hex6, `^[a-z]+-[a-z]+-[0-9a-f]{6}$`},
			{randomname.Hex8, `^[a-z]+-[a-z]+-[0-9a-f]{8}$`},
			{randomname.Numeric4, `^[a-z]+-[a-z]+-\d{4}$`},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("suffix=%v", tt.suffix), func(t *testing.T) {
				name := randomname.Generate(&randomname.Options{
					Suffix: tt.suffix,
				})
				assert.Regexp(t, tt.pattern, name)
			})
		}
	})

	t.Run("custom words", func(t *testing.T) {
		customWords := map[randomname.WordType][]string{
			randomname.Adjective: {"custom", "test"},
			randomname.Noun:      {"word", "name"},
		}

		// Generate many names to ensure we see custom words
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			name := randomname.Generate(&randomname.Options{
				Words: customWords,
			})
			parts := strings.Split(name, "-")
			if len(parts) >= 2 {
				seen[parts[0]] = true
				seen[parts[1]] = true
			}
		}

		// Should see at least one custom word
		hasCustom := false
		for word := range seen {
			if word == "custom" || word == "test" || word == "word" || word == "name" {
				hasCustom = true
				break
			}
		}
		assert.True(t, hasCustom, "Should use custom words")
	})

	t.Run("empty custom words still uses defaults", func(t *testing.T) {
		name := randomname.Generate(&randomname.Options{
			Words: map[randomname.WordType][]string{
				randomname.Adjective: {}, // Empty custom list
			},
		})
		assert.Regexp(t, `^[a-z]+-[a-z]+$`, name)
	})

	t.Run("validator callback", func(t *testing.T) {
		// Test that validator is called and respected
		rejected := make(map[string]bool)
		name := randomname.Generate(&randomname.Options{
			Validator: func(s string) bool {
				if len(rejected) < 3 {
					rejected[s] = true
					return false
				}
				return true
			},
		})

		assert.NotEmpty(t, name)
		assert.Len(t, rejected, 3, "Should have rejected exactly 3 names")
		assert.NotContains(t, rejected, name, "Final name should not be in rejected list")
	})
}

func TestConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		pattern  string
		minParts int
	}{
		{
			name:     "Simple",
			fn:       randomname.Simple,
			pattern:  `^[a-z]+-[a-z]+$`,
			minParts: 2,
		},
		{
			name:     "Colorful",
			fn:       randomname.Colorful,
			pattern:  `^[a-z]+-[a-z]+$`,
			minParts: 2,
		},
		{
			name:     "Descriptive",
			fn:       randomname.Descriptive,
			pattern:  `^[a-z]+-[a-z]+-[a-z]+$`,
			minParts: 3,
		},
		{
			name:     "WithSuffix",
			fn:       randomname.WithSuffix,
			pattern:  `^[a-z]+-[a-z]+-[0-9a-f]{6}$`,
			minParts: 3,
		},
		{
			name:     "Sized",
			fn:       randomname.Sized,
			pattern:  `^[a-z]+-[a-z]+$`,
			minParts: 2,
		},
		{
			name:     "Complex",
			fn:       randomname.Complex,
			pattern:  `^[a-z]+-[a-z]+-[a-z]+$`,
			minParts: 3,
		},
		{
			name:     "Full",
			fn:       randomname.Full,
			pattern:  `^[a-z]+-[a-z]+-[a-z]+-[a-z]+$`,
			minParts: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := tt.fn()
			assert.Regexp(t, tt.pattern, name)
			parts := strings.Split(name, "-")
			assert.GreaterOrEqual(t, len(parts), tt.minParts)
		})
	}
}

func TestUniqueness(t *testing.T) {
	t.Run("simple pattern", func(t *testing.T) {
		names := make(map[string]bool)
		iterations := 100

		for i := range iterations {
			name := randomname.Simple()
			if names[name] {
				// Collision is possible with simple pattern
				t.Logf("Collision detected at iteration %d: %s", i, name)
			}
			names[name] = true
		}

		// With 100 iterations on 22k combinations, collisions are unlikely but possible
		uniqueRatio := float64(len(names)) / float64(iterations)
		assert.Greater(t, uniqueRatio, 0.8, "Should have at least 80% unique names")
	})

	t.Run("with hex suffix", func(t *testing.T) {
		names := make(map[string]bool)
		iterations := 1000

		for range iterations {
			name := randomname.WithSuffix()
			require.NotContains(t, names, name, "Should not have any collisions with hex suffix")
			names[name] = true
		}
	})

	t.Run("descriptive pattern", func(t *testing.T) {
		names := make(map[string]bool)
		iterations := 500

		for range iterations {
			name := randomname.Descriptive()
			names[name] = true
		}

		// With 500 iterations on 908k combinations, collisions are very unlikely but not impossible
		// Allow for up to 1% collision rate (5 collisions out of 500)
		assert.GreaterOrEqual(t, len(names), iterations-5, "Should have minimal collisions with descriptive pattern")
		assert.LessOrEqual(t, len(names), iterations, "Should not exceed iteration count")
	})
}

func TestConcurrency(t *testing.T) {
	// Test that multiple goroutines can generate names concurrently
	workers := 10
	iterations := 100

	var wg sync.WaitGroup
	names := make(chan string, workers*iterations)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				name := randomname.Generate(&randomname.Options{
					Pattern: []randomname.WordType{randomname.Adjective, randomname.Color, randomname.Noun},
					Suffix:  randomname.Hex6,
				})
				names <- name
			}
		}()
	}

	wg.Wait()
	close(names)

	// Verify all names are valid and unique
	seen := make(map[string]bool)
	pattern := regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z]+-[0-9a-f]{6}$`)

	for name := range names {
		assert.Regexp(t, pattern, name)
		assert.NotContains(t, seen, name, "Should not have duplicates")
		seen[name] = true
	}

	assert.Equal(t, workers*iterations, len(seen))
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty pattern with suffix", func(t *testing.T) {
		// Should fall back to default pattern
		name := randomname.Generate(&randomname.Options{
			Pattern: []randomname.WordType{},
			Suffix:  randomname.Hex6,
		})
		assert.Regexp(t, `^[a-z]+-[a-z]+-[0-9a-f]{6}$`, name)
	})

	t.Run("pattern with unavailable word type", func(t *testing.T) {
		// Using an invalid WordType by casting
		name := randomname.Generate(&randomname.Options{
			Pattern: []randomname.WordType{randomname.WordType(999)},
		})
		// Should fall back to default pattern when no valid words
		assert.Regexp(t, `^[a-z]+-[a-z]+$`, name)
	})

	t.Run("very long pattern", func(t *testing.T) {
		pattern := make([]randomname.WordType, 10)
		for i := range pattern {
			pattern[i] = randomname.Adjective
		}

		name := randomname.Generate(&randomname.Options{
			Pattern: pattern,
		})

		parts := strings.Split(name, "-")
		assert.Len(t, parts, 10)
	})

	t.Run("numeric suffix range", func(t *testing.T) {
		// Test that numeric suffix is always 4 digits (1000-9999)
		for range 100 {
			name := randomname.Generate(&randomname.Options{
				Suffix: randomname.Numeric4,
			})
			parts := strings.Split(name, "-")
			suffix := parts[len(parts)-1]
			assert.Regexp(t, `^\d{4}$`, suffix)

			num := 0
			fmt.Sscanf(suffix, "%d", &num)
			assert.GreaterOrEqual(t, num, 1000)
			assert.LessOrEqual(t, num, 9999)
		}
	})
}
