package vectorizer

// Chunker defines the interface for text splitting strategies.
// Implementations can provide different approaches like sentence-based,
// markdown-aware, or document-specific chunking.
type Chunker interface {
	// Split divides text into chunks according to the implementation's strategy.
	// Returns a slice of text chunks that can be vectorized independently.
	Split(text string, options ChunkOptions) []string
}

// ChunkOptions configures how text is split into chunks.
// The Custom field allows chunker-specific configuration.
type ChunkOptions struct {
	// MaxTokens is the approximate maximum number of tokens per chunk.
	// Default varies by chunker implementation.
	MaxTokens int

	// Overlap is the number of tokens to overlap between chunks for context continuity.
	// Set to 0 to disable overlap.
	Overlap int

	// MinChunkSize is the minimum number of tokens for a chunk to be included.
	// Chunks smaller than this are merged with adjacent chunks.
	MinChunkSize int

	// Custom allows chunker-specific options.
	// For example, SimpleChunker might use Custom["splitBySentence"] = true.
	Custom map[string]any
}

// DefaultChunkOptions returns sensible defaults for text chunking.
// These work well with most chunker implementations.
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{
		MaxTokens:    500,
		Overlap:      50,
		MinChunkSize: 10,
		Custom:       make(map[string]any),
	}
}

// EstimateTokens provides a rough estimate of token count for a text.
// Uses the approximation that 1 word ≈ 1.3 tokens for English text.
// This is a utility function available to all chunker implementations.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Simple word counting
	wordCount := 0
	inWord := false

	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if inWord {
				wordCount++
				inWord = false
			}
		} else {
			inWord = true
		}
	}

	if inWord {
		wordCount++
	}

	// Approximation: 1 word ≈ 1.3 tokens
	return int(float64(wordCount) * 1.3)
}
