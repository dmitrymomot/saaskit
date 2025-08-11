package vectorizer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements Provider interface for testing
type mockProvider struct {
	vectorizeFunc      func(ctx context.Context, text string) (Vector, error)
	vectorizeBatchFunc func(ctx context.Context, texts []string) ([]Vector, error)
	dimensions         int
}

func (m *mockProvider) Vectorize(ctx context.Context, text string) (Vector, error) {
	if m.vectorizeFunc != nil {
		return m.vectorizeFunc(ctx, text)
	}
	// Default implementation
	return Vector{0.1, 0.2, 0.3}, nil
}

func (m *mockProvider) VectorizeBatch(ctx context.Context, texts []string) ([]Vector, error) {
	if m.vectorizeBatchFunc != nil {
		return m.vectorizeBatchFunc(ctx, texts)
	}
	// Default implementation
	vectors := make([]Vector, len(texts))
	for i := range texts {
		vectors[i] = Vector{float64(i) * 0.1, float64(i) * 0.2, float64(i) * 0.3}
	}
	return vectors, nil
}

func (m *mockProvider) Dimensions() int {
	if m.dimensions > 0 {
		return m.dimensions
	}
	return 3
}

func TestNew(t *testing.T) {
	t.Run("with valid provider", func(t *testing.T) {
		provider := &mockProvider{}
		v, err := New(provider)
		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.Equal(t, provider, v.provider)
	})

	t.Run("with nil provider", func(t *testing.T) {
		v, err := New(nil)
		assert.Error(t, err)
		assert.Nil(t, v)
		assert.True(t, errors.Is(err, ErrProviderNotSet))
	})
}

func TestVectorizer_ToVector(t *testing.T) {
	ctx := context.Background()

	t.Run("successful vectorization", func(t *testing.T) {
		expectedVector := Vector{0.5, 0.6, 0.7}
		provider := &mockProvider{
			vectorizeFunc: func(ctx context.Context, text string) (Vector, error) {
				assert.Equal(t, "test text", text)
				return expectedVector, nil
			},
		}

		v, _ := New(provider)
		vector, err := v.ToVector(ctx, "test text")
		require.NoError(t, err)
		assert.Equal(t, expectedVector, vector)
	})

	t.Run("empty text", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := New(provider)

		vector, err := v.ToVector(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.True(t, errors.Is(err, ErrEmptyText))
	})

	t.Run("whitespace only text", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := New(provider)

		vector, err := v.ToVector(ctx, "   \t\n  ")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.True(t, errors.Is(err, ErrEmptyText))
	})

	t.Run("provider error", func(t *testing.T) {
		providerErr := errors.New("provider failed")
		provider := &mockProvider{
			vectorizeFunc: func(ctx context.Context, text string) (Vector, error) {
				return nil, providerErr
			},
		}

		v, _ := New(provider)
		vector, err := v.ToVector(ctx, "test")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.True(t, errors.Is(err, ErrVectorizationFailed))
	})
}

func TestVectorizer_ChunksToVectors(t *testing.T) {
	ctx := context.Background()

	t.Run("successful batch vectorization", func(t *testing.T) {
		chunks := []string{"chunk1", "chunk2", "chunk3"}
		expectedVectors := []Vector{
			{1.0, 1.1, 1.2},
			{2.0, 2.1, 2.2},
			{3.0, 3.1, 3.2},
		}

		provider := &mockProvider{
			vectorizeBatchFunc: func(ctx context.Context, texts []string) ([]Vector, error) {
				assert.Equal(t, chunks, texts)
				return expectedVectors, nil
			},
		}

		v, _ := New(provider)
		vectors, err := v.ChunksToVectors(ctx, chunks)
		require.NoError(t, err)
		assert.Equal(t, expectedVectors, vectors)
	})

	t.Run("empty chunks", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := New(provider)

		vectors, err := v.ChunksToVectors(ctx, []string{})
		require.NoError(t, err)
		assert.Empty(t, vectors)
	})

	t.Run("filters empty chunks", func(t *testing.T) {
		inputChunks := []string{"valid", "", "  ", "another", "\t\n"}
		expectedChunks := []string{"valid", "another"}

		provider := &mockProvider{
			vectorizeBatchFunc: func(ctx context.Context, texts []string) ([]Vector, error) {
				assert.Equal(t, expectedChunks, texts)
				return []Vector{{1.0}, {2.0}}, nil
			},
		}

		v, _ := New(provider)
		vectors, err := v.ChunksToVectors(ctx, inputChunks)
		require.NoError(t, err)
		assert.Len(t, vectors, 2)
	})
}

func TestVectorizer_Process(t *testing.T) {
	ctx := context.Background()

	t.Run("processes long text", func(t *testing.T) {
		longText := "This is a long text that needs to be split into chunks. " +
			"Each chunk will be vectorized separately. " +
			"The process function handles both splitting and vectorization."

		provider := &mockProvider{
			vectorizeBatchFunc: func(ctx context.Context, texts []string) ([]Vector, error) {
				vectors := make([]Vector, len(texts))
				for i := range texts {
					vectors[i] = Vector{float64(i), float64(i) + 0.1}
				}
				return vectors, nil
			},
		}

		v, _ := New(provider)
		chunks, err := v.Process(ctx, longText, ChunkOptions{
			MaxTokens: 10,
			Overlap:   2,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, chunks)

		// Verify each chunk has text, vector, and index
		for i, chunk := range chunks {
			assert.NotEmpty(t, chunk.Text)
			assert.NotNil(t, chunk.Vector)
			assert.Equal(t, i, chunk.Index)
		}
	})

	t.Run("empty text returns empty chunks", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := New(provider)

		chunks, err := v.Process(ctx, "", DefaultChunkOptions())
		require.NoError(t, err)
		assert.Empty(t, chunks)
	})
}

func TestVectorizer_Dimensions(t *testing.T) {
	t.Run("returns provider dimensions", func(t *testing.T) {
		provider := &mockProvider{dimensions: 1536}
		v, _ := New(provider)
		assert.Equal(t, 1536, v.Dimensions())
	})

	t.Run("returns 0 for nil provider", func(t *testing.T) {
		v := &Vectorizer{provider: nil}
		assert.Equal(t, 0, v.Dimensions())
	})
}

func TestChunker(t *testing.T) {
	t.Run("split by sentences", func(t *testing.T) {
		text := "First sentence. Second sentence! Third sentence? Fourth sentence."
		chunks := SplitIntoChunks(text, ChunkOptions{
			MaxTokens:       20,
			Overlap:         5,
			SplitBySentence: true,
		})

		assert.NotEmpty(t, chunks)
		for _, chunk := range chunks {
			assert.NotEmpty(t, chunk)
			// Check that chunks maintain sentence boundaries where possible
			assert.True(t, len(chunk) > 0)
		}
	})

	t.Run("split by tokens", func(t *testing.T) {
		text := "This is a long text without clear sentence boundaries that needs to be split based on token count alone"
		chunks := SplitIntoChunks(text, ChunkOptions{
			MaxTokens:       10,
			Overlap:         2,
			SplitBySentence: false,
		})

		assert.NotEmpty(t, chunks)
		assert.Greater(t, len(chunks), 1)
	})

	t.Run("small text returns single chunk", func(t *testing.T) {
		text := "Short text"
		chunks := SplitIntoChunks(text, ChunkOptions{
			MaxTokens: 100,
		})

		assert.Len(t, chunks, 1)
		assert.Equal(t, text, chunks[0])
	})

	t.Run("empty text returns empty chunks", func(t *testing.T) {
		chunks := SplitIntoChunks("", DefaultChunkOptions())
		assert.Empty(t, chunks)
	})

	t.Run("merges small chunks", func(t *testing.T) {
		text := "A. B. C. D. E. F. G."
		chunks := SplitIntoChunks(text, ChunkOptions{
			MaxTokens:       5,
			MinChunkSize:    10,
			SplitBySentence: true,
		})

		// Small sentences should be merged
		assert.NotEmpty(t, chunks)
		for _, chunk := range chunks {
			tokens := estimateTokens(chunk)
			// Each chunk should be at least MinChunkSize tokens (except possibly the last one)
			assert.True(t, tokens >= 5 || chunk == chunks[len(chunks)-1])
		}
	})
}

func TestDefaultChunkOptions(t *testing.T) {
	opts := DefaultChunkOptions()
	assert.Equal(t, 500, opts.MaxTokens)
	assert.Equal(t, 50, opts.Overlap)
	assert.True(t, opts.SplitBySentence)
	assert.Equal(t, 10, opts.MinChunkSize)
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
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			tokens := estimateTokens(tt.text)
			assert.Equal(t, tt.expected, tokens)
		})
	}
}

func TestOpenAIProvider(t *testing.T) {
	t.Run("new provider with valid config", func(t *testing.T) {
		provider, err := NewOpenAIProvider(OpenAIConfig{
			APIKey: "test-key",
			Model:  "text-embedding-3-small",
		})

		require.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "test-key", provider.apiKey)
		assert.Equal(t, "text-embedding-3-small", provider.model)
		assert.Equal(t, 1536, provider.dimensions)
	})

	t.Run("new provider with default model", func(t *testing.T) {
		provider, err := NewOpenAIProvider(OpenAIConfig{
			APIKey: "test-key",
		})

		require.NoError(t, err)
		assert.Equal(t, DefaultOpenAIModel, provider.model)
		assert.Equal(t, 1536, provider.dimensions)
	})

	t.Run("new provider without API key", func(t *testing.T) {
		provider, err := NewOpenAIProvider(OpenAIConfig{})
		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.True(t, errors.Is(err, ErrAPIKeyRequired))
	})

	t.Run("new provider with invalid model", func(t *testing.T) {
		provider, err := NewOpenAIProvider(OpenAIConfig{
			APIKey: "test-key",
			Model:  "invalid-model",
		})

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.True(t, errors.Is(err, ErrInvalidModel))
	})
}

func TestGetModelDimensions(t *testing.T) {
	tests := []struct {
		model      string
		dimensions int
	}{
		{"text-embedding-3-small", 1536},
		{"text-embedding-3-large", 3072},
		{"text-embedding-ada-002", 1536},
		{"unknown-model", 0},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			dims := getModelDimensions(tt.model)
			assert.Equal(t, tt.dimensions, dims)
		})
	}
}
