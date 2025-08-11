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

	// commonAbbreviations contains known abbreviations that don't end sentences
	commonAbbreviations map[string]bool
}

// NewSimpleChunker creates a chunker that splits text intelligently.
// By default, it attempts to maintain sentence boundaries.
func NewSimpleChunker() *SimpleChunker {
	return &SimpleChunker{
		splitBySentence:     true,
		commonAbbreviations: initCommonAbbreviations(),
	}
}

// NewSimpleChunkerWithOptions creates a chunker with specific behavior.
func NewSimpleChunkerWithOptions(splitBySentence bool) *SimpleChunker {
	return &SimpleChunker{
		splitBySentence:     splitBySentence,
		commonAbbreviations: initCommonAbbreviations(),
	}
}

// initCommonAbbreviations returns a map of common abbreviations that don't typically end sentences
func initCommonAbbreviations() map[string]bool {
	return map[string]bool{
		// Titles
		"Dr": true, "Mr": true, "Mrs": true, "Ms": true, "Prof": true,
		"Rev": true, "Sr": true, "Jr": true, "Capt": true, "Col": true,
		"Gen": true, "Lt": true, "Sgt": true, "Maj": true, "Hon": true,

		// Academic degrees
		"Ph": true, "M": true, "D": true, "B": true, "A": true, "S": true,
		"Ph.D": true, "M.D": true, "B.A": true, "M.A": true, "D.D.S": true,
		"B.S": true, "M.S": true, "Ed.D": true, "J.D": true,

		// Organizations
		"Inc": true, "Corp": true, "Co": true, "Ltd": true, "LLC": true,
		"LLP": true, "L.P": true, "P.C": true, "Assoc": true,

		// Geographic
		"U.S": true, "U.K": true, "E.U": true, "U.N": true, "U.S.A": true,
		"St": true, "Ave": true, "Blvd": true, "Rd": true, "Apt": true,
		"Mt": true, "Ft": true,

		// Latin abbreviations
		"i.e": true, "e.g": true, "etc": true, "vs": true, "cf": true,
		"al": true, "et": true, "viz": true, "ca": true,

		// Months
		"Jan": true, "Feb": true, "Mar": true, "Apr": true, "Jun": true,
		"Jul": true, "Aug": true, "Sept": true, "Oct": true, "Nov": true,
		"Dec": true,

		// Other common
		"No": true, "Vol": true, "Ed": true, "Est": true, "Dept": true,
		"Gov": true, "Sen": true, "Rep": true, "Pres": true, "V.P": true,
		"a.m": true, "p.m": true, "A.M": true, "P.M": true,
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

// getWordBefore extracts the word immediately before the given position
func (c *SimpleChunker) getWordBefore(text []rune, pos int) string {
	if pos <= 0 || pos > len(text) {
		return ""
	}

	// Find the start of the word
	start := pos - 1
	for start > 0 && !unicode.IsSpace(text[start-1]) {
		start--
	}

	return string(text[start:pos])
}

// isSentenceBoundary determines if a punctuation mark at the given position ends a sentence
func (c *SimpleChunker) isSentenceBoundary(text []rune, pos int) bool {
	// Rule 1: Not a boundary if followed immediately by lowercase letter (no space)
	if pos+1 < len(text) && unicode.IsLower(text[pos+1]) {
		return false
	}

	// Get the word before the period
	word := c.getWordBefore(text, pos)
	wordWithoutPeriods := strings.ReplaceAll(word, ".", "")

	// Rule 2: Check for known abbreviations
	if c.commonAbbreviations[wordWithoutPeriods] {
		return false
	}

	// Rule 3: Single uppercase letter followed by period (initial)
	if len(wordWithoutPeriods) == 1 && unicode.IsUpper(rune(wordWithoutPeriods[0])) {
		return false
	}

	// Rule 4: Multiple periods in word (e.g., U.S.A., Ph.D.)
	if strings.Count(word, ".") > 0 {
		return false
	}

	// Rule 5: Check what comes after
	if pos+1 < len(text) {
		// Two spaces or newline after period = likely sentence boundary
		if unicode.IsSpace(text[pos+1]) {
			if pos+2 < len(text) && (text[pos+1] == '\n' || unicode.IsSpace(text[pos+2])) {
				return true
			}
			// Single space followed by uppercase = likely sentence boundary
			if pos+2 < len(text) && unicode.IsUpper(text[pos+2]) {
				// But not if it's a single letter (could be an initial)
				if pos+3 < len(text) && text[pos+3] == '.' {
					return false
				}
				return true
			}
		}

		// No space but uppercase letter (e.g., "end.Start")
		if unicode.IsUpper(text[pos+1]) {
			return true
		}
	} else {
		// End of text
		return true
	}

	// Default: conservative, don't split
	return false
}

// splitIntoSentences splits text into sentences using improved punctuation and abbreviation handling
func (c *SimpleChunker) splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)

		// Check for sentence-ending punctuation
		if r == '.' || r == '!' || r == '?' {
			if c.isSentenceBoundary(runes, i) {
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
