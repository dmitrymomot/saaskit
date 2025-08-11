package vectorizer

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

// BenchmarkEstimateTokens tests the performance of the EstimateTokens function
func BenchmarkEstimateTokens(b *testing.B) {
	testCases := []struct {
		name string
		text string
	}{
		{
			name: "Short",
			text: "This is a short text.",
		},
		{
			name: "Medium",
			text: strings.Repeat("This is a medium length text that contains multiple sentences. ", 10),
		},
		{
			name: "Long",
			text: strings.Repeat("This is a long text with many words that will be tokenized. It contains various sentences and paragraphs. ", 100),
		},
		{
			name: "VeryLong",
			text: strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. ", 1000),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = EstimateTokens(tc.text)
			}
		})
	}
}

// BenchmarkSimpleChunker_Split tests the performance of the SimpleChunker.Split method
func BenchmarkSimpleChunker_Split(b *testing.B) {
	chunker := NewSimpleChunker()

	testCases := []struct {
		name      string
		text      string
		maxTokens int
		overlap   int
	}{
		{
			name:      "SmallChunks",
			text:      strings.Repeat("This is a test sentence. Another sentence here. And one more for good measure. ", 100),
			maxTokens: 100,
			overlap:   10,
		},
		{
			name:      "MediumChunks",
			text:      strings.Repeat("This is a test sentence. Another sentence here. And one more for good measure. ", 200),
			maxTokens: 500,
			overlap:   50,
		},
		{
			name:      "LargeChunks",
			text:      strings.Repeat("This is a test sentence. Another sentence here. And one more for good measure. ", 500),
			maxTokens: 1000,
			overlap:   100,
		},
		{
			name:      "NoOverlap",
			text:      strings.Repeat("This is a test sentence. Another sentence here. And one more for good measure. ", 100),
			maxTokens: 500,
			overlap:   0,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			options := ChunkOptions{
				MaxTokens:    tc.maxTokens,
				Overlap:      tc.overlap,
				MinChunkSize: 10,
				Custom:       make(map[string]interface{}),
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = chunker.Split(tc.text, options)
			}
		})
	}
}

// BenchmarkSimpleChunker_SplitBySentence tests sentence-aware vs non-sentence-aware splitting
func BenchmarkSimpleChunker_SplitBySentence(b *testing.B) {
	text := strings.Repeat("This is a test sentence. Another sentence here! And one more for good measure? ", 200)
	options := ChunkOptions{
		MaxTokens:    500,
		Overlap:      50,
		MinChunkSize: 10,
		Custom:       make(map[string]interface{}),
	}

	b.Run("WithSentenceSplitting", func(b *testing.B) {
		chunker := NewSimpleChunkerWithOptions(true)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = chunker.Split(text, options)
		}
	})

	b.Run("WithoutSentenceSplitting", func(b *testing.B) {
		chunker := NewSimpleChunkerWithOptions(false)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = chunker.Split(text, options)
		}
	})
}

// BenchmarkOverlapProcessing tests the performance of chunking with different overlap settings
func BenchmarkOverlapProcessing(b *testing.B) {
	chunker := NewSimpleChunker()
	text := strings.Repeat("This is a test sentence. Another sentence here. And one more for good measure. ", 200)

	testCases := []struct {
		name    string
		overlap int
	}{
		{
			name:    "NoOverlap",
			overlap: 0,
		},
		{
			name:    "SmallOverlap_10",
			overlap: 10,
		},
		{
			name:    "MediumOverlap_50",
			overlap: 50,
		},
		{
			name:    "LargeOverlap_100",
			overlap: 100,
		},
		{
			name:    "VeryLargeOverlap_200",
			overlap: 200,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			options := ChunkOptions{
				MaxTokens:    500,
				Overlap:      tc.overlap,
				MinChunkSize: 10,
				Custom:       make(map[string]interface{}),
			}

			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = chunker.Split(text, options)
			}
		})
	}
}

// MockProviderBench is a mock provider for benchmarking
type MockProviderBench struct{}

func (m *MockProviderBench) Vectorize(ctx context.Context, text string) (Vector, error) {
	// Return a dummy vector
	return Vector{0.1, 0.2, 0.3}, nil
}

func (m *MockProviderBench) VectorizeBatch(ctx context.Context, texts []string) ([]Vector, error) {
	vectors := make([]Vector, len(texts))
	for i := range texts {
		vectors[i] = Vector{float64(i) * 0.1, float64(i) * 0.2, float64(i) * 0.3}
	}
	return vectors, nil
}

func (m *MockProviderBench) Dimensions() int {
	return 3
}

// BenchmarkVectorizeBatch tests the performance of batch vectorization
func BenchmarkVectorizeBatch(b *testing.B) {
	ctx := context.Background()
	provider := &MockProviderBench{}
	v, _ := NewWithDefaults(provider)

	testCases := []struct {
		name   string
		chunks []string
	}{
		{
			name:   "Small_5",
			chunks: generateChunks(5),
		},
		{
			name:   "Medium_20",
			chunks: generateChunks(20),
		},
		{
			name:   "Large_50",
			chunks: generateChunks(50),
		},
		{
			name:   "VeryLarge_100",
			chunks: generateChunks(100),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = v.ChunksToVectors(ctx, tc.chunks)
			}
		})
	}
}

// BenchmarkProcess tests the performance of the full processing pipeline
func BenchmarkProcess(b *testing.B) {
	ctx := context.Background()
	provider := &MockProviderBench{}
	v, _ := NewWithDefaults(provider)

	testCases := []struct {
		name string
		text string
		opts ChunkOptions
	}{
		{
			name: "ShortDoc",
			text: strings.Repeat("This is a test document with multiple sentences. It contains various paragraphs. ", 50),
			opts: ChunkOptions{
				MaxTokens:    500,
				Overlap:      50,
				MinChunkSize: 10,
				Custom:       make(map[string]interface{}),
			},
		},
		{
			name: "MediumDoc",
			text: strings.Repeat("This is a test document with multiple sentences. It contains various paragraphs. ", 200),
			opts: ChunkOptions{
				MaxTokens:    500,
				Overlap:      50,
				MinChunkSize: 10,
				Custom:       make(map[string]interface{}),
			},
		},
		{
			name: "LongDoc",
			text: strings.Repeat("This is a test document with multiple sentences. It contains various paragraphs. ", 1000),
			opts: ChunkOptions{
				MaxTokens:    500,
				Overlap:      50,
				MinChunkSize: 10,
				Custom:       make(map[string]interface{}),
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = v.Process(ctx, tc.text, tc.opts)
			}
		})
	}
}

// Helper function to generate test chunks
func generateChunks(n int) []string {
	chunks := make([]string, n)
	for i := range n {
		chunks[i] = "This is test chunk number " + strconv.Itoa(i) + ". It contains some text for vectorization."
	}
	return chunks
}
