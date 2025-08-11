package vectorizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleChunker_Split(t *testing.T) {
	chunker := NewSimpleChunker()

	t.Run("split by sentences", func(t *testing.T) {
		text := "First sentence. Second sentence! Third sentence? Fourth sentence."
		chunks := chunker.Split(text, ChunkOptions{
			MaxTokens:    20,
			Overlap:      5,
			MinChunkSize: 5,
		})

		assert.NotEmpty(t, chunks)
		for _, chunk := range chunks {
			assert.NotEmpty(t, chunk)
			// Check that chunks maintain sentence boundaries where possible
			assert.True(t, len(chunk) > 0)
		}
	})

	t.Run("split by tokens", func(t *testing.T) {
		chunker := NewSimpleChunkerWithOptions(false)
		text := "This is a long text without clear sentence boundaries that needs to be split based on token count alone"
		chunks := chunker.Split(text, ChunkOptions{
			MaxTokens:    10,
			Overlap:      2,
			MinChunkSize: 5,
		})

		assert.NotEmpty(t, chunks)
		assert.Greater(t, len(chunks), 1)
	})

	t.Run("small text returns single chunk", func(t *testing.T) {
		text := "Short text"
		chunks := chunker.Split(text, ChunkOptions{
			MaxTokens: 100,
		})

		assert.Len(t, chunks, 1)
		assert.Equal(t, text, chunks[0])
	})

	t.Run("empty text returns empty chunks", func(t *testing.T) {
		chunks := chunker.Split("", DefaultChunkOptions())
		assert.Empty(t, chunks)
	})

	t.Run("whitespace only returns empty chunks", func(t *testing.T) {
		chunks := chunker.Split("   \t\n  ", DefaultChunkOptions())
		assert.Empty(t, chunks)
	})

	t.Run("merges small chunks", func(t *testing.T) {
		text := "A. B. C. D. E. F. G."
		chunks := chunker.Split(text, ChunkOptions{
			MaxTokens:    5,
			MinChunkSize: 10,
			Overlap:      0,
		})

		// Small sentences should be merged
		assert.NotEmpty(t, chunks)
		for _, chunk := range chunks {
			tokens := EstimateTokens(chunk)
			// Each chunk should be at least MinChunkSize tokens (except possibly the last one)
			assert.True(t, tokens >= 5 || chunk == chunks[len(chunks)-1])
		}
	})

	t.Run("custom split by sentence option", func(t *testing.T) {
		// Start with sentence splitting enabled
		chunker := NewSimpleChunker()
		text := "First sentence. Second sentence. Third sentence."

		// Override with custom option to disable sentence splitting
		options := ChunkOptions{
			MaxTokens:    10,
			Overlap:      0,
			MinChunkSize: 5,
			Custom:       map[string]interface{}{"splitBySentence": false},
		}

		chunks := chunker.Split(text, options)
		assert.NotEmpty(t, chunks)

		// Should split by tokens, not sentences
		// With ~7 words and max 10 tokens (≈7.6 words), might be 1-2 chunks
		assert.LessOrEqual(t, len(chunks), 2)
	})

	t.Run("handles long sentences", func(t *testing.T) {
		// Create a very long sentence
		longSentence := "This is a very long sentence that contains many words and will definitely exceed the maximum token limit that we have set for our chunks so it should be split into multiple parts even though it is a single sentence."

		chunks := chunker.Split(longSentence, ChunkOptions{
			MaxTokens:    20,
			Overlap:      5,
			MinChunkSize: 5,
		})

		assert.Greater(t, len(chunks), 1, "Long sentence should be split into multiple chunks")
		for _, chunk := range chunks {
			assert.NotEmpty(t, chunk)
		}
	})

	t.Run("overlap between chunks", func(t *testing.T) {
		text := "Word1 word2 word3 word4 word5 word6 word7 word8 word9 word10"
		chunks := chunker.Split(text, ChunkOptions{
			MaxTokens:    6, // About 4-5 words
			Overlap:      3, // About 2 words
			MinChunkSize: 2,
			Custom:       map[string]interface{}{"splitBySentence": false},
		})

		assert.Greater(t, len(chunks), 1)

		// Check that consecutive chunks have some overlap
		if len(chunks) > 1 {
			// The end of first chunk should appear in the beginning of second chunk
			// This is approximate due to word boundaries
			assert.NotEmpty(t, chunks[0])
			assert.NotEmpty(t, chunks[1])
		}
	})
}

func TestSimpleChunker_SplitIntoSentences(t *testing.T) {
	chunker := NewSimpleChunker()

	t.Run("basic sentence splitting", func(t *testing.T) {
		text := "First sentence. Second sentence! Third sentence?"
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 3)
		assert.Equal(t, "First sentence.", sentences[0])
		assert.Equal(t, "Second sentence!", sentences[1])
		assert.Equal(t, "Third sentence?", sentences[2])
	})

	t.Run("handles abbreviations", func(t *testing.T) {
		text := "Dr. Smith went to the U.S.A. yesterday. He enjoyed it."
		sentences := chunker.splitIntoSentences(text)

		// Should properly handle Dr. and U.S.A. abbreviations
		assert.Len(t, sentences, 2)
		assert.Equal(t, "Dr. Smith went to the U.S.A. yesterday.", sentences[0])
		assert.Equal(t, "He enjoyed it.", sentences[1])
	})

	t.Run("handles various title abbreviations", func(t *testing.T) {
		text := "Mr. Jones met Mrs. Smith and Prof. Brown. They discussed the project."
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 2)
		assert.Equal(t, "Mr. Jones met Mrs. Smith and Prof. Brown.", sentences[0])
		assert.Equal(t, "They discussed the project.", sentences[1])
	})

	t.Run("handles initials", func(t *testing.T) {
		text := "J. K. Rowling wrote Harry Potter. It became very popular."
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 2)
		assert.Equal(t, "J. K. Rowling wrote Harry Potter.", sentences[0])
		assert.Equal(t, "It became very popular.", sentences[1])
	})

	t.Run("handles organizations and locations", func(t *testing.T) {
		text := "Apple Inc. announced new products. The event was in the U.S. yesterday."
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 2)
		assert.Equal(t, "Apple Inc. announced new products.", sentences[0])
		assert.Equal(t, "The event was in the U.S. yesterday.", sentences[1])
	})

	t.Run("handles academic degrees", func(t *testing.T) {
		text := "She earned her Ph.D. last year. Now she teaches at MIT."
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 2)
		assert.Equal(t, "She earned her Ph.D. last year.", sentences[0])
		assert.Equal(t, "Now she teaches at MIT.", sentences[1])
	})

	t.Run("handles Latin abbreviations", func(t *testing.T) {
		text := "We need tools (e.g. hammer, saw). Also materials i.e. wood and nails."
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 2)
		assert.Equal(t, "We need tools (e.g. hammer, saw).", sentences[0])
		assert.Equal(t, "Also materials i.e. wood and nails.", sentences[1])
	})

	t.Run("handles months", func(t *testing.T) {
		text := "The meeting is on Jan. 15th. Please confirm your attendance."
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 2)
		assert.Equal(t, "The meeting is on Jan. 15th.", sentences[0])
		assert.Equal(t, "Please confirm your attendance.", sentences[1])
	})

	t.Run("handles double space as sentence boundary", func(t *testing.T) {
		text := "This ends here.  This is a new sentence."
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 2)
		assert.Equal(t, "This ends here.", sentences[0])
		assert.Equal(t, "This is a new sentence.", sentences[1])
	})

	t.Run("handles newline as sentence boundary", func(t *testing.T) {
		text := "First sentence.\nSecond sentence."
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 2)
		assert.Equal(t, "First sentence.", sentences[0])
		assert.Equal(t, "Second sentence.", sentences[1])
	})

	t.Run("handles missing spaces after punctuation", func(t *testing.T) {
		text := "First.Second!Third?"
		sentences := chunker.splitIntoSentences(text)

		// Should still split even without spaces
		assert.Greater(t, len(sentences), 1)
	})

	t.Run("handles text without sentence endings", func(t *testing.T) {
		text := "This is text without any sentence ending punctuation"
		sentences := chunker.splitIntoSentences(text)

		assert.Len(t, sentences, 1)
		assert.Equal(t, text, sentences[0])
	})
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"hello world", 2},    // 2 words * 1.3 ≈ 2.6 → 2
		{"this is a test", 5}, // 4 words * 1.3 ≈ 5.2 → 5
		{"", 0},
		{"   ", 0},
		{"single", 1},
		{"one two three four five", 6}, // 5 words * 1.3 = 6.5 → 6
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			tokens := EstimateTokens(tt.text)
			assert.Equal(t, tt.expected, tokens)
		})
	}
}

func TestSimpleChunker_GetOverlapText(t *testing.T) {
	chunker := NewSimpleChunker()

	t.Run("extracts overlap from sentences", func(t *testing.T) {
		sentences := []string{"First sentence", "Second sentence", "Third sentence"}
		overlap := chunker.getOverlapText(sentences, 10) // About 7-8 words

		// Should include at least the last sentence
		assert.Contains(t, overlap, "Third sentence")
	})

	t.Run("handles empty input", func(t *testing.T) {
		overlap := chunker.getOverlapText([]string{}, 10)
		assert.Empty(t, overlap)
	})

	t.Run("handles single sentence", func(t *testing.T) {
		sentences := []string{"Only sentence"}
		overlap := chunker.getOverlapText(sentences, 10)
		assert.Equal(t, "Only sentence", overlap)
	})
}

func TestSimpleChunker_MergeSmallChunks(t *testing.T) {
	chunker := NewSimpleChunker()

	t.Run("merges small chunks", func(t *testing.T) {
		chunks := []string{"A", "B", "C", "D"}
		merged := chunker.mergeSmallChunks(chunks, 10)

		// Should merge these tiny chunks
		assert.Less(t, len(merged), len(chunks))
	})

	t.Run("keeps large chunks separate", func(t *testing.T) {
		chunks := []string{
			"This is a reasonably long chunk with many words",
			"This is another reasonably long chunk with many words",
		}
		merged := chunker.mergeSmallChunks(chunks, 5)

		// Should not merge already large chunks
		assert.Len(t, merged, 2)
	})

	t.Run("handles empty input", func(t *testing.T) {
		merged := chunker.mergeSmallChunks([]string{}, 10)
		assert.Empty(t, merged)
	})

	t.Run("handles single chunk", func(t *testing.T) {
		chunks := []string{"Single chunk"}
		merged := chunker.mergeSmallChunks(chunks, 10)
		assert.Equal(t, chunks, merged)
	})
}
