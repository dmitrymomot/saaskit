package randomname

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"
)

// builderPool is used to reduce allocations when building names.
var builderPool = &sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

// Generate creates a random name based on the provided options.
// If options is nil, uses default pattern (adjective-noun).
// The function always returns a valid name and never returns an error.
func Generate(opts *Options) string {
	options := opts.merge(defaultOptions())

	// Validate pattern has at least one word type
	if len(options.Pattern) == 0 {
		options.Pattern = defaultOptions().Pattern
	}

	// Get a string builder from the pool
	builder := builderPool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		builderPool.Put(builder)
	}()

	const maxRetries = 100
	for range maxRetries {
		builder.Reset()

		// Build the name based on pattern
		validWords := 0
		for _, wordType := range options.Pattern {
			words := getWords(wordType, options.Words)
			if len(words) == 0 {
				continue // Skip if no words available for this type
			}

			if validWords > 0 {
				builder.WriteString(options.Separator)
			}

			// Select a random word
			index := secureRandInt(len(words))
			builder.WriteString(words[index])
			validWords++
		}

		// If no valid words found in pattern, fall back to default
		if validWords == 0 {
			return Generate(&Options{
				Pattern:   defaultOptions().Pattern,
				Separator: options.Separator,
				Suffix:    options.Suffix,
				Validator: options.Validator,
			})
		}

		// Add suffix if requested
		if options.Suffix != NoSuffix {
			if builder.Len() > 0 {
				builder.WriteString(options.Separator)
			}
			builder.WriteString(generateSuffix(options.Suffix))
		}

		name := builder.String()

		// Validate if callback provided
		if options.Validator == nil || options.Validator(name) {
			return name
		}
	}

	// If validation failed after max retries, return the last generated name
	return builder.String()
}

// Simple generates a name with pattern: adjective-noun
func Simple() string {
	return Generate(nil)
}

// Colorful generates a name with pattern: color-noun
func Colorful() string {
	return Generate(&Options{
		Pattern: []WordType{Color, Noun},
	})
}

// Descriptive generates a name with pattern: adjective-color-noun
func Descriptive() string {
	return Generate(&Options{
		Pattern: []WordType{Adjective, Color, Noun},
	})
}

// WithSuffix generates a name with pattern: adjective-noun-hex6
func WithSuffix() string {
	return Generate(&Options{
		Suffix: Hex6,
	})
}

// Sized generates a name with pattern: size-noun
func Sized() string {
	return Generate(&Options{
		Pattern: []WordType{Size, Noun},
	})
}

// Complex generates a name with pattern: size-adjective-noun
func Complex() string {
	return Generate(&Options{
		Pattern: []WordType{Size, Adjective, Noun},
	})
}

// Full generates a name with pattern: size-adjective-color-noun
func Full() string {
	return Generate(&Options{
		Pattern: []WordType{Size, Adjective, Color, Noun},
	})
}

// secureRandInt returns a cryptographically secure random integer in range [0, max).
func secureRandInt(max int) int {
	if max <= 0 {
		return 0
	}

	// To avoid modulo bias, we need to generate within a range that's
	// a multiple of max
	nBig := uint32(max)
	maxValid := (^uint32(0) / nBig) * nBig

	// For typical word list sizes (< 1000), we'll need very few retries
	// The probability of needing a retry is (2^32 - maxValid) / 2^32
	// For max=1000, this is ~0.0023% chance per iteration
	const maxRetries = 10

	for range maxRetries {
		var n uint32
		if err := binary.Read(rand.Reader, binary.LittleEndian, &n); err != nil {
			// If crypto/rand fails, don't give up - use time-based seed
			// This can happen in some CI environments
			n = uint32(time.Now().UnixNano())
			return int(n % nBig)
		}

		// Reject values that would cause modulo bias
		if n < maxValid {
			return int(n % nBig)
		}
		// Try again if we got a biased value
	}

	// After max retries, fall back to simple modulo (tiny bias acceptable)
	var n uint32
	if err := binary.Read(rand.Reader, binary.LittleEndian, &n); err != nil {
		// Use time-based fallback instead of returning 0
		n = uint32(time.Now().UnixNano())
	}
	return int(n % nBig)
}

// generateSuffix creates a suffix based on the specified type.
func generateSuffix(suffixType SuffixType) string {
	switch suffixType {
	case Hex6:
		// Generate 3 random bytes = 6 hex characters
		bytes := make([]byte, 3)
		if _, err := rand.Read(bytes); err != nil {
			// Fallback to zeros on error
			return "000000"
		}
		return fmt.Sprintf("%x", bytes)

	case Hex8:
		// Generate 4 random bytes = 8 hex characters
		bytes := make([]byte, 4)
		if _, err := rand.Read(bytes); err != nil {
			// Fallback to zeros on error
			return "00000000"
		}
		return fmt.Sprintf("%x", bytes)

	case Numeric4:
		// Generate a 4-digit number (1000-9999)
		n := secureRandInt(9000) + 1000
		return fmt.Sprintf("%04d", n)

	default:
		return ""
	}
}
