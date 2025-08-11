package vectorizer

import (
	"context"
	"fmt"
	"strings"
)

// Vector represents a text embedding as a slice of float64 values.
// The dimensionality depends on the model used (e.g., 1536 for text-embedding-3-small).
type Vector []float64

// Chunk represents a piece of text with its corresponding vector embedding.
// Used when processing long texts that need to be split into smaller parts.
type Chunk struct {
	Text   string `json:"text"`
	Vector Vector `json:"vector"`
	Index  int    `json:"index"` // Position in the original text
}

// Provider defines the interface for vectorization backends.
// Implementations should handle API authentication, rate limiting, and error recovery.
type Provider interface {
	// Vectorize converts a single text into a vector embedding.
	Vectorize(ctx context.Context, text string) (Vector, error)

	// VectorizeBatch converts multiple texts into vectors in a single request.
	// More efficient than calling Vectorize multiple times.
	VectorizeBatch(ctx context.Context, texts []string) ([]Vector, error)

	// Dimensions returns the vector dimensions for the current model.
	Dimensions() int
}

// Vectorizer provides high-level text vectorization operations.
// It uses a Provider for the actual embedding generation and a Chunker
// for text splitting, adding convenience methods for batch processing.
type Vectorizer struct {
	provider Provider
	chunker  Chunker
}

// New creates a new Vectorizer with the specified provider and chunker.
// Returns an error if provider or chunker is nil.
func New(provider Provider, chunker Chunker) (*Vectorizer, error) {
	if provider == nil {
		return nil, ErrProviderNotSet
	}
	if chunker == nil {
		return nil, fmt.Errorf("chunker cannot be nil")
	}
	return &Vectorizer{
		provider: provider,
		chunker:  chunker,
	}, nil
}

// NewWithDefaults creates a new Vectorizer with the specified provider
// and a default SimpleChunker for convenience.
func NewWithDefaults(provider Provider) (*Vectorizer, error) {
	return New(provider, NewSimpleChunker())
}

// ToVector converts a single text string into a vector embedding.
// Returns ErrEmptyText if the input is empty or contains only whitespace.
func (v *Vectorizer) ToVector(ctx context.Context, text string) (Vector, error) {
	if v.provider == nil {
		return nil, ErrProviderNotSet
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	vector, err := v.provider.Vectorize(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVectorizationFailed, err)
	}

	return vector, nil
}

// ChunksToVectors converts multiple text chunks into vectors.
// Uses batch processing when available for better performance.
// Empty chunks are skipped and won't appear in the result.
func (v *Vectorizer) ChunksToVectors(ctx context.Context, chunks []string) ([]Vector, error) {
	if v.provider == nil {
		return nil, ErrProviderNotSet
	}

	if len(chunks) == 0 {
		return []Vector{}, nil
	}

	// Filter out empty chunks
	nonEmptyChunks := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk)
		if trimmed != "" {
			nonEmptyChunks = append(nonEmptyChunks, trimmed)
		}
	}

	if len(nonEmptyChunks) == 0 {
		return []Vector{}, nil
	}

	vectors, err := v.provider.VectorizeBatch(ctx, nonEmptyChunks)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVectorizationFailed, err)
	}

	return vectors, nil
}

// Process splits a long text into chunks and vectorizes each chunk.
// This is the main entry point for processing documents or articles.
// Returns a slice of Chunks, each containing text and its vector.
func (v *Vectorizer) Process(ctx context.Context, text string, options ChunkOptions) ([]Chunk, error) {
	if v.provider == nil {
		return nil, ErrProviderNotSet
	}

	// Split text into chunks using the configured chunker
	textChunks := v.chunker.Split(text, options)
	if len(textChunks) == 0 {
		return []Chunk{}, nil
	}

	// Vectorize all chunks
	vectors, err := v.ChunksToVectors(ctx, textChunks)
	if err != nil {
		return nil, err
	}

	// Combine texts and vectors into Chunk structs
	chunks := make([]Chunk, len(textChunks))
	for i := range len(textChunks) {
		chunks[i] = Chunk{
			Text:   textChunks[i],
			Vector: vectors[i],
			Index:  i,
		}
	}

	return chunks, nil
}

// Dimensions returns the vector dimensions for the current provider's model.
func (v *Vectorizer) Dimensions() int {
	if v.provider == nil {
		return 0
	}
	return v.provider.Dimensions()
}

// Chunk splits text into chunks using the configured chunker.
// This is useful when you want to chunk text without vectorizing it.
func (v *Vectorizer) Chunk(text string, options ChunkOptions) []string {
	if v.chunker == nil {
		return []string{}
	}
	return v.chunker.Split(text, options)
}

// SetChunker changes the chunker used by this vectorizer.
// This allows runtime switching of chunking strategies.
func (v *Vectorizer) SetChunker(chunker Chunker) error {
	if chunker == nil {
		return fmt.Errorf("chunker cannot be nil")
	}
	v.chunker = chunker
	return nil
}

// ProcessWithChunker processes text with a specific chunker for one-off use.
// The vectorizer's default chunker remains unchanged.
func (v *Vectorizer) ProcessWithChunker(ctx context.Context, text string, chunker Chunker, options ChunkOptions) ([]Chunk, error) {
	if v.provider == nil {
		return nil, ErrProviderNotSet
	}
	if chunker == nil {
		return nil, fmt.Errorf("chunker cannot be nil")
	}

	// Split text into chunks using the provided chunker
	textChunks := chunker.Split(text, options)
	if len(textChunks) == 0 {
		return []Chunk{}, nil
	}

	// Vectorize all chunks
	vectors, err := v.ChunksToVectors(ctx, textChunks)
	if err != nil {
		return nil, err
	}

	// Combine texts and vectors into Chunk structs
	chunks := make([]Chunk, len(textChunks))
	for i := range len(textChunks) {
		chunks[i] = Chunk{
			Text:   textChunks[i],
			Vector: vectors[i],
			Index:  i,
		}
	}

	return chunks, nil
}
