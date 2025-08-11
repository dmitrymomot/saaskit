package vectorizer

import (
	"strings"
	"unicode"
)

// ChunkOptions configures how text is split into chunks.
// All parameters are optional with sensible defaults.
type ChunkOptions struct {
	// MaxTokens is the approximate maximum number of tokens per chunk.
	// Default: 500. Note: this uses a simple word-based approximation (1 word ≈ 1.3 tokens).
	MaxTokens int

	// Overlap is the number of tokens to overlap between chunks for context continuity.
	// Default: 50. Set to 0 to disable overlap.
	Overlap int

	// SplitBySentence attempts to split at sentence boundaries when possible.
	// Default: true. This helps maintain semantic coherence.
	SplitBySentence bool

	// MinChunkSize is the minimum number of tokens for a chunk to be included.
	// Default: 10. Chunks smaller than this are merged with adjacent chunks.
	MinChunkSize int
}

// DefaultChunkOptions returns sensible defaults for text chunking.
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{
		MaxTokens:       500,
		Overlap:         50,
		SplitBySentence: true,
		MinChunkSize:    10,
	}
}

// SplitIntoChunks splits text into smaller chunks based on the provided options.
// It attempts to maintain semantic boundaries (sentences) while respecting size limits.
func SplitIntoChunks(text string, options ChunkOptions) []string {
	// Apply defaults for zero values
	if options.MaxTokens <= 0 {
		options.MaxTokens = 500
	}
	if options.Overlap < 0 {
		options.Overlap = 50
	}
	if options.MinChunkSize <= 0 {
		options.MinChunkSize = 10
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	// If text is small enough, return as single chunk
	tokenCount := estimateTokens(text)
	if tokenCount <= options.MaxTokens {
		return []string{text}
	}

	var chunks []string

	if options.SplitBySentence {
		chunks = splitBySentences(text, options)
	} else {
		chunks = splitByTokens(text, options)
	}

	// Merge small chunks
	chunks = mergeSmallChunks(chunks, options.MinChunkSize)

	return chunks
}

// estimateTokens provides a rough estimate of token count.
// Uses the approximation that 1 word ≈ 1.3 tokens for English text.
func estimateTokens(text string) int {
	words := strings.Fields(text)
	return int(float64(len(words)) * 1.3)
}

// splitBySentences splits text at sentence boundaries while respecting token limits.
func splitBySentences(text string, options ChunkOptions) []string {
	sentences := splitIntoSentences(text)
	if len(sentences) == 0 {
		return []string{}
	}

	var chunks []string
	var currentChunk []string
	currentTokens := 0

	for _, sentence := range sentences {
		sentenceTokens := estimateTokens(sentence)

		// If single sentence exceeds max tokens, split it by words
		if sentenceTokens > options.MaxTokens {
			// Flush current chunk if not empty
			if len(currentChunk) > 0 {
				chunks = append(chunks, strings.Join(currentChunk, " "))
				currentChunk = []string{}
				currentTokens = 0
			}

			// Split long sentence by words
			wordChunks := splitByTokens(sentence, options)
			chunks = append(chunks, wordChunks...)
			continue
		}

		// Check if adding this sentence would exceed the limit
		if currentTokens+sentenceTokens > options.MaxTokens && len(currentChunk) > 0 {
			// Save current chunk
			chunks = append(chunks, strings.Join(currentChunk, " "))

			// Start new chunk with overlap if enabled
			if options.Overlap > 0 && len(currentChunk) > 0 {
				// Calculate overlap from the end of current chunk
				overlapText := getOverlapText(currentChunk, options.Overlap)
				currentChunk = []string{overlapText}
				currentTokens = estimateTokens(overlapText)
			} else {
				currentChunk = []string{}
				currentTokens = 0
			}
		}

		currentChunk = append(currentChunk, sentence)
		currentTokens += sentenceTokens
	}

	// Add remaining chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, " "))
	}

	return chunks
}

// splitByTokens splits text by approximate token count without considering sentences.
func splitByTokens(text string, options ChunkOptions) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	// Calculate words per chunk (approximation)
	wordsPerChunk := int(float64(options.MaxTokens) / 1.3)
	overlapWords := int(float64(options.Overlap) / 1.3)

	var chunks []string

	for i := 0; i < len(words); {
		end := min(i+wordsPerChunk, len(words))
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)

		// Move forward with overlap
		if overlapWords > 0 && end < len(words) {
			i = end - overlapWords
		} else {
			i = end
		}
	}

	return chunks
}

// splitIntoSentences splits text into sentences using common punctuation.
func splitIntoSentences(text string) []string {
	// Simple sentence splitting by common terminators
	// This is a basic implementation - could be enhanced with better NLP
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)

		// Check for sentence endings
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to check if this is really a sentence end
			if i+1 < len(runes) {
				next := runes[i+1]
				// Check if next character suggests continuation (e.g., lowercase letter after period might be abbreviation)
				if unicode.IsSpace(next) || unicode.IsUpper(next) {
					sentence := strings.TrimSpace(current.String())
					if sentence != "" {
						sentences = append(sentences, sentence)
						current.Reset()
					}
				}
			} else {
				// End of text
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
			}
		}
	}

	// Add any remaining text
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// getOverlapText extracts overlap text from the end of a chunk.
func getOverlapText(sentences []string, overlapTokens int) string {
	if len(sentences) == 0 {
		return ""
	}

	// Work backwards to get approximately overlapTokens worth of text
	var overlapSentences []string
	currentTokens := 0

	for i := len(sentences) - 1; i >= 0; i-- {
		sentenceTokens := estimateTokens(sentences[i])
		if currentTokens+sentenceTokens > overlapTokens && len(overlapSentences) > 0 {
			break
		}
		overlapSentences = append([]string{sentences[i]}, overlapSentences...)
		currentTokens += sentenceTokens
	}

	return strings.Join(overlapSentences, " ")
}

// mergeSmallChunks combines chunks that are too small with adjacent chunks.
func mergeSmallChunks(chunks []string, minSize int) []string {
	if len(chunks) <= 1 {
		return chunks
	}

	var merged []string
	var buffer strings.Builder
	bufferTokens := 0

	for _, chunk := range chunks {
		chunkTokens := estimateTokens(chunk)

		if bufferTokens == 0 {
			// Start new buffer
			buffer.WriteString(chunk)
			bufferTokens = chunkTokens
		} else if bufferTokens < minSize {
			// Current buffer is too small, add this chunk
			buffer.WriteString(" ")
			buffer.WriteString(chunk)
			bufferTokens += chunkTokens
		} else {
			// Buffer is large enough, save it
			merged = append(merged, buffer.String())
			buffer.Reset()
			buffer.WriteString(chunk)
			bufferTokens = chunkTokens
		}
	}

	// Add remaining buffer
	if bufferTokens > 0 {
		merged = append(merged, buffer.String())
	}

	return merged
}
