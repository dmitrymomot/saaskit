// Package vectorizer provides text-to-vector conversion functionality with support
// for chunking long texts and multiple embedding providers. It enables semantic
// search, similarity matching, and other vector-based operations on text data.
//
// The package follows a provider pattern, allowing different vectorization backends
// while maintaining a consistent API. It includes intelligent text chunking that
// respects sentence boundaries and provides overlap for context continuity.
//
// # Architecture
//
// The package is organized into three main components:
//   - Provider Interface – abstracts embedding generation (OpenAI, etc.)
//   - Vectorizer – high-level API for text processing
//   - Chunker – intelligent text splitting with configurable options
//
// Core types:
//   - Vector            – slice of float64 representing an embedding
//   - Chunk             – text segment with its vector and position
//   - Provider          – interface for vectorization backends
//   - ChunkOptions      – configuration for text splitting
//
// # Usage
//
// Basic vectorization:
//
//	provider, err := vectorizer.NewOpenAIProvider(vectorizer.OpenAIConfig{
//	    APIKey: os.Getenv("OPENAI_API_KEY"),
//	})
//	v, err := vectorizer.New(provider)
//
//	// Single text to vector
//	vector, err := v.ToVector(ctx, "Hello, world!")
//
//	// Process long document
//	chunks, err := v.Process(ctx, longText, vectorizer.DefaultChunkOptions())
//
// Custom chunking:
//
//	options := vectorizer.ChunkOptions{
//	    MaxTokens:       300,  // Smaller chunks
//	    Overlap:         30,   // Less overlap
//	    SplitBySentence: true, // Maintain sentence boundaries
//	    MinChunkSize:    20,   // Avoid tiny chunks
//	}
//	chunks, err := v.Process(ctx, document, options)
//
// # Provider Implementation
//
// The package includes an OpenAI provider by default, supporting models like
// text-embedding-3-small (1536 dimensions) and text-embedding-3-large (3072 dimensions).
// Custom providers can be implemented by satisfying the Provider interface.
//
// # Chunking Strategy
//
// The chunker uses a multi-level approach:
//  1. Attempts to split at sentence boundaries for semantic coherence
//  2. Falls back to word-based splitting for very long sentences
//  3. Provides configurable overlap to maintain context between chunks
//  4. Merges chunks that are too small to avoid fragmentation
//
// Token estimation uses the approximation: 1 word ≈ 1.3 tokens (for English text).
//
// # Performance Considerations
//
// - Batch processing: Use ChunksToVectors for multiple texts (reduces API calls)
// - Chunk size: Balance between context (larger) and precision (smaller)
// - Overlap: Higher overlap maintains context but increases processing
// - Provider timeout: Configure HTTP client timeout for large batches
//
// # Error Handling
//
// The package provides domain-specific errors that can be wrapped with additional
// context. All errors follow the standard errors.Is/As patterns for inspection.
//
// Common errors:
//   - ErrProviderNotSet       – vectorizer created without provider
//   - ErrEmptyText            – attempting to vectorize empty text
//   - ErrVectorizationFailed  – provider failed to generate embedding
//   - ErrRateLimitExceeded    – API rate limit hit (provider-specific)
//   - ErrContextLengthExceeded – text too long for model
//
// # Examples
//
// See the accompanying README.md for complete examples including semantic search,
// document processing, and custom provider implementation.
package vectorizer
