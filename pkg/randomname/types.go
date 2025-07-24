package randomname

// WordType represents the type of word in a name pattern.
type WordType int

// Word types available for name generation.
const (
	Adjective WordType = iota
	Noun
	Color
	Size
	Origin
	Action
)

// SuffixType represents the type of suffix to append to generated names.
type SuffixType int

// Suffix types for collision avoidance.
const (
	NoSuffix SuffixType = iota
	Hex6                // 6-character hexadecimal (e.g., a3f21b)
	Hex8                // 8-character hexadecimal (e.g., a3f21b9c)
	Numeric4            // 4-digit number (e.g., 4829)
)

// Options configures name generation behavior.
type Options struct {
	// Pattern defines the word types to use in order.
	// Default: [Adjective, Noun]
	Pattern []WordType

	// Separator between words.
	// Default: "-"
	Separator string

	// Suffix type for collision avoidance.
	// Default: NoSuffix
	Suffix SuffixType

	// Words provides custom word lists for any WordType.
	// These are merged with defaults if provided.
	Words map[WordType][]string

	// Validator is called to check if a generated name is acceptable.
	// Return true to accept the name, false to generate a new one.
	// The generator will retry up to 100 times before giving up.
	Validator func(string) bool
}

// defaultOptions returns the default options for name generation.
func defaultOptions() *Options {
	return &Options{
		Pattern:   []WordType{Adjective, Noun},
		Separator: "-",
		Suffix:    NoSuffix,
	}
}

// merge combines user options with defaults.
func (o *Options) merge(defaults *Options) *Options {
	if o == nil {
		return defaults
	}

	result := *o

	if len(result.Pattern) == 0 {
		result.Pattern = defaults.Pattern
	}

	// If separator is empty string and not explicitly set, use default
	if result.Separator == "" {
		result.Separator = defaults.Separator
	}

	return &result
}
