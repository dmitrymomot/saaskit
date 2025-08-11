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

// mockChunker implements Chunker interface for testing
type mockChunker struct {
	splitFunc func(text string, options ChunkOptions) []string
}

func (m *mockChunker) Split(text string, options ChunkOptions) []string {
	if m.splitFunc != nil {
		return m.splitFunc(text, options)
	}
	return []string{text} // Default: return as single chunk
}

func TestNew(t *testing.T) {
	t.Run("with valid provider and chunker", func(t *testing.T) {
		provider := &mockProvider{}
		chunker := NewSimpleChunker()
		v, err := New(provider, chunker)
		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.Equal(t, provider, v.provider)
		assert.Equal(t, chunker, v.chunker)
	})

	t.Run("with nil provider", func(t *testing.T) {
		chunker := NewSimpleChunker()
		v, err := New(nil, chunker)
		assert.Error(t, err)
		assert.Nil(t, v)
		assert.True(t, errors.Is(err, ErrProviderNotSet))
	})

	t.Run("with nil chunker", func(t *testing.T) {
		provider := &mockProvider{}
		v, err := New(provider, nil)
		assert.Error(t, err)
		assert.Nil(t, v)
		assert.Contains(t, err.Error(), "chunker cannot be nil")
	})
}

func TestNewWithDefaults(t *testing.T) {
	t.Run("creates vectorizer with default chunker", func(t *testing.T) {
		provider := &mockProvider{}
		v, err := NewWithDefaults(provider)
		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.NotNil(t, v.chunker)
		assert.IsType(t, &SimpleChunker{}, v.chunker)
	})

	t.Run("with nil provider", func(t *testing.T) {
		v, err := NewWithDefaults(nil)
		assert.Error(t, err)
		assert.Nil(t, v)
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

		v, _ := NewWithDefaults(provider)
		vector, err := v.ToVector(ctx, "test text")
		require.NoError(t, err)
		assert.Equal(t, expectedVector, vector)
	})

	t.Run("empty text", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := NewWithDefaults(provider)

		vector, err := v.ToVector(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.True(t, errors.Is(err, ErrEmptyText))
	})

	t.Run("whitespace only text", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := NewWithDefaults(provider)

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

		v, _ := NewWithDefaults(provider)
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

		v, _ := NewWithDefaults(provider)
		vectors, err := v.ChunksToVectors(ctx, chunks)
		require.NoError(t, err)
		assert.Equal(t, expectedVectors, vectors)
	})

	t.Run("empty chunks", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := NewWithDefaults(provider)

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

		v, _ := NewWithDefaults(provider)
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

		v, _ := NewWithDefaults(provider)
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
		v, _ := NewWithDefaults(provider)

		chunks, err := v.Process(ctx, "", DefaultChunkOptions())
		require.NoError(t, err)
		assert.Empty(t, chunks)
	})
}

func TestVectorizer_Dimensions(t *testing.T) {
	t.Run("returns provider dimensions", func(t *testing.T) {
		provider := &mockProvider{dimensions: 1536}
		v, _ := NewWithDefaults(provider)
		assert.Equal(t, 1536, v.Dimensions())
	})

	t.Run("returns 0 for nil provider", func(t *testing.T) {
		v := &Vectorizer{provider: nil, chunker: NewSimpleChunker()}
		assert.Equal(t, 0, v.Dimensions())
	})
}

func TestVectorizer_SetChunker(t *testing.T) {
	t.Run("sets new chunker", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := NewWithDefaults(provider)

		newChunker := NewSimpleChunkerWithOptions(false)
		err := v.SetChunker(newChunker)

		assert.NoError(t, err)
		assert.Equal(t, newChunker, v.chunker)
	})

	t.Run("rejects nil chunker", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := NewWithDefaults(provider)

		err := v.SetChunker(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "chunker cannot be nil")
	})
}

func TestVectorizer_Chunk(t *testing.T) {
	t.Run("chunks text without vectorizing", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := NewWithDefaults(provider)

		text := "This is a test. Another sentence. And one more."
		chunks := v.Chunk(text, ChunkOptions{
			MaxTokens: 10,
		})

		assert.NotEmpty(t, chunks)
	})

	t.Run("returns empty for nil chunker", func(t *testing.T) {
		v := &Vectorizer{provider: &mockProvider{}, chunker: nil}
		chunks := v.Chunk("test", DefaultChunkOptions())
		assert.Empty(t, chunks)
	})
}

func TestVectorizer_ProcessWithChunker(t *testing.T) {
	ctx := context.Background()

	t.Run("processes with custom chunker", func(t *testing.T) {
		provider := &mockProvider{
			vectorizeBatchFunc: func(ctx context.Context, texts []string) ([]Vector, error) {
				vectors := make([]Vector, len(texts))
				for i := range texts {
					vectors[i] = Vector{float64(i)}
				}
				return vectors, nil
			},
		}

		v, _ := NewWithDefaults(provider)
		customChunker := NewSimpleChunkerWithOptions(false)

		text := "Test text for custom chunker"
		chunks, err := v.ProcessWithChunker(ctx, text, customChunker, DefaultChunkOptions())

		assert.NoError(t, err)
		assert.NotEmpty(t, chunks)
	})

	t.Run("rejects nil chunker", func(t *testing.T) {
		provider := &mockProvider{}
		v, _ := NewWithDefaults(provider)

		_, err := v.ProcessWithChunker(ctx, "test", nil, DefaultChunkOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "chunker cannot be nil")
	})
}

func TestVectorizer_WithCustomChunker(t *testing.T) {
	ctx := context.Background()

	t.Run("uses custom chunker", func(t *testing.T) {
		customChunks := []string{"chunk1", "chunk2", "chunk3"}
		customChunker := &mockChunker{
			splitFunc: func(text string, options ChunkOptions) []string {
				return customChunks
			},
		}

		provider := &mockProvider{
			vectorizeBatchFunc: func(ctx context.Context, texts []string) ([]Vector, error) {
				assert.Equal(t, customChunks, texts)
				vectors := make([]Vector, len(texts))
				for i := range texts {
					vectors[i] = Vector{float64(i)}
				}
				return vectors, nil
			},
		}

		v, err := New(provider, customChunker)
		require.NoError(t, err)

		chunks, err := v.Process(ctx, "any text", DefaultChunkOptions())
		require.NoError(t, err)
		assert.Len(t, chunks, 3)
		assert.Equal(t, "chunk1", chunks[0].Text)
		assert.Equal(t, "chunk2", chunks[1].Text)
		assert.Equal(t, "chunk3", chunks[2].Text)
	})
}

func TestDefaultChunkOptions(t *testing.T) {
	opts := DefaultChunkOptions()
	assert.Equal(t, 500, opts.MaxTokens)
	assert.Equal(t, 50, opts.Overlap)
	assert.Equal(t, 10, opts.MinChunkSize)
	assert.NotNil(t, opts.Custom)
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
