package vectorizer

import (
	"strings"
	"unicode"
)

// SimpleChunker implements basic text chunking with sentence awareness.
// It attempts to split at sentence boundaries when possible while respecting
// token limits, and provides overlap between chunks for context continuity.
type SimpleChunker struct {
	// splitBySentence controls whether to attempt sentence boundary splitting.
	// When false, splits purely by token count.
	splitBySentence bool
}

// NewSimpleChunker creates a chunker that splits text intelligently.
// By default, it attempts to maintain sentence boundaries.
func NewSimpleChunker() *SimpleChunker {
	return &SimpleChunker{
		splitBySentence: true,
	}
}

// NewSimpleChunkerWithOptions creates a chunker with specific behavior.
func NewSimpleChunkerWithOptions(splitBySentence bool) *SimpleChunker {
	return &SimpleChunker{
		splitBySentence: splitBySentence,
	}
}

// Split divides text into chunks according to the options.
// It respects sentence boundaries when possible and provides overlap between chunks.
func (c *SimpleChunker) Split(text string, options ChunkOptions) []string {
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

	// Check for custom split by sentence option
	splitBySentence := c.splitBySentence
	if val, ok := options.Custom["splitBySentence"].(bool); ok {
		splitBySentence = val
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	// If text is small enough, return as single chunk
	tokenCount := EstimateTokens(text)
	if tokenCount <= options.MaxTokens {
		return []string{text}
	}

	var chunks []string

	if splitBySentence {
		chunks = c.splitBySentences(text, options)
	} else {
		chunks = c.splitByTokens(text, options)
	}

	// Merge small chunks
	chunks = c.mergeSmallChunks(chunks, options.MinChunkSize)

	return chunks
}

// splitBySentences splits text at sentence boundaries while respecting token limits.
func (c *SimpleChunker) splitBySentences(text string, options ChunkOptions) []string {
	sentences := c.splitIntoSentences(text)
	if len(sentences) == 0 {
		return []string{}
	}

	var chunks []string
	var currentChunk []string
	currentTokens := 0

	for _, sentence := range sentences {
		sentenceTokens := EstimateTokens(sentence)

		// If single sentence exceeds max tokens, split it by words
		if sentenceTokens > options.MaxTokens {
			// Flush current chunk if not empty
			if len(currentChunk) > 0 {
				chunks = append(chunks, strings.Join(currentChunk, " "))
				currentChunk = []string{}
				currentTokens = 0
			}

			// Split long sentence by words
			wordChunks := c.splitByTokens(sentence, options)
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
				overlapText := c.getOverlapText(currentChunk, options.Overlap)
				currentChunk = []string{overlapText}
				currentTokens = EstimateTokens(overlapText)
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
func (c *SimpleChunker) splitByTokens(text string, options ChunkOptions) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	// Use 1.3 tokens per word based on average English word length (4-5 chars + space)
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
			if i <= 0 {
				i = 1 // Prevent infinite loop while maintaining overlap
			}
		} else {
			i = end
		}
	}

	return chunks
}

// splitIntoSentences splits text into sentences using common punctuation.
func (c *SimpleChunker) splitIntoSentences(text string) []string {
	// Basic sentence splitting using punctuation - kept simple for extensibility.
	// Users can implement more sophisticated NLP-based chunkers if needed.
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
				// Space or uppercase after punctuation indicates sentence boundary (avoids splitting on abbreviations)
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
					current.Reset()
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
func (c *SimpleChunker) getOverlapText(sentences []string, overlapTokens int) string {
	if len(sentences) == 0 {
		return ""
	}

	// Work backwards to get approximately overlapTokens worth of text
	// First pass: count sentences needed
	sentenceCount := 0
	currentTokens := 0

	for i := len(sentences) - 1; i >= 0; i-- {
		sentenceTokens := EstimateTokens(sentences[i])
		if currentTokens+sentenceTokens > overlapTokens && sentenceCount > 0 {
			break
		}
		sentenceCount++
		currentTokens += sentenceTokens
	}

	// Second pass: build slice with correct capacity to avoid reallocations
	startIdx := len(sentences) - sentenceCount
	overlapSentences := make([]string, sentenceCount)
	copy(overlapSentences, sentences[startIdx:])

	return strings.Join(overlapSentences, " ")
}

// mergeSmallChunks combines chunks that are too small with adjacent chunks.
func (c *SimpleChunker) mergeSmallChunks(chunks []string, minSize int) []string {
	if len(chunks) <= 1 {
		return chunks
	}

	var merged []string
	var buffer strings.Builder
	bufferTokens := 0

	for _, chunk := range chunks {
		chunkTokens := EstimateTokens(chunk)

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
