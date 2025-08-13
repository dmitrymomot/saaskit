// Package vectorizer provides text-to-vector conversion functionality with pluggable
// chunking strategies and multiple embedding provider backends. It enables semantic
// search, similarity matching, and other vector-based operations on text data.
//
// The package follows a clean separation of concerns with pluggable interfaces,
// allowing different vectorization backends and text chunking strategies while
// maintaining a consistent, type-safe API. It includes intelligent text chunking
// that respects sentence boundaries and provides overlap for context continuity.
//
// # Architecture
//
// The package is organized into four main components working together:
//
//   - Provider Interface – abstracts embedding generation (OpenAI, Cohere, custom)
//   - Chunker Interface – defines text splitting strategies (sentence-aware, markdown, custom)
//   - Vectorizer – orchestrates Provider and Chunker for high-level operations
//   - Utilities – token estimation, default configurations, and error handling
//
// Core types and interfaces:
//
//   - Vector            – []float64 representing an embedding vector
//   - Chunk             – text segment with its vector and position metadata
//   - Provider          – interface for vectorization backends
//   - Chunker           – interface for text splitting strategies
//   - ChunkOptions      – configuration for text splitting behavior
//   - OpenAIProvider    – built-in OpenAI API implementation
//   - SimpleChunker     – built-in sentence-aware chunking implementation
//
// # Basic Usage
//
// Simple vectorization with defaults:
//
//	import (
//	    "context"
//	    "os"
//	    "github.com/your-org/saaskit/pkg/vectorizer"
//	)
//
//	// Create provider and vectorizer
//	provider, err := vectorizer.NewOpenAIProvider(vectorizer.OpenAIConfig{
//	    APIKey: os.Getenv("OPENAI_API_KEY"),
//	    Model:  "text-embedding-3-small", // Optional, this is the default
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create vectorizer with default SimpleChunker
//	v, err := vectorizer.NewWithDefaults(provider)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Single text to vector
//	vector, err := v.ToVector(ctx, "Hello, world!")
//	fmt.Printf("Vector dimensions: %d\n", len(vector))
//
//	// Process long document with default chunking
//	longText := "Your long document content here..."
//	chunks, err := v.Process(ctx, longText, vectorizer.DefaultChunkOptions())
//	for i, chunk := range chunks {
//	    fmt.Printf("Chunk %d: %d tokens, %d dimensions\n",
//	        i, vectorizer.EstimateTokens(chunk.Text), len(chunk.Vector))
//	}
//
// # Custom Chunking Configuration
//
// Fine-tune chunking behavior for your specific use case:
//
//	// Custom chunking options for technical documentation
//	options := vectorizer.ChunkOptions{
//	    MaxTokens:    300,  // Smaller chunks for precise search
//	    Overlap:      50,   // Moderate overlap for context
//	    MinChunkSize: 30,   // Avoid tiny fragments
//	    Custom: map[string]any{
//	        "splitBySentence": true, // Maintain sentence boundaries
//	    },
//	}
//
//	chunks, err := v.Process(ctx, document, options)
//
//	// Or use a different chunker for one operation
//	customChunker := vectorizer.NewSimpleChunkerWithOptions(false) // Disable sentence splitting
//	chunks, err = v.ProcessWithChunker(ctx, document, customChunker, options)
//
// # Implementing Custom Providers
//
// Create your own embedding provider by implementing the Provider interface:
//
//	type MyCustomProvider struct {
//	    apiKey string
//	    client *http.Client
//	}
//
//	func (p *MyCustomProvider) Vectorize(ctx context.Context, text string) (vectorizer.Vector, error) {
//	    // Implement single text vectorization
//	    // Make API call to your embedding service
//	    return vectorizer.Vector{0.1, 0.2, 0.3}, nil
//	}
//
//	func (p *MyCustomProvider) VectorizeBatch(ctx context.Context, texts []string) ([]vectorizer.Vector, error) {
//	    // Implement batch vectorization for efficiency
//	    vectors := make([]vectorizer.Vector, len(texts))
//	    for i, text := range texts {
//	        vector, err := p.Vectorize(ctx, text)
//	        if err != nil {
//	            return nil, err
//	        }
//	        vectors[i] = vector
//	    }
//	    return vectors, nil
//	}
//
//	func (p *MyCustomProvider) Dimensions() int {
//	    return 384 // Your model's embedding dimensions
//	}
//
//	// Usage
//	customProvider := &MyCustomProvider{apiKey: "your-key"}
//	v, err := vectorizer.NewWithDefaults(customProvider)
//
// # Implementing Custom Chunkers
//
// Create specialized text splitting strategies by implementing the Chunker interface:
//
//	type MarkdownChunker struct {
//	    preserveHeaders bool
//	}
//
//	func (c *MarkdownChunker) Split(text string, options vectorizer.ChunkOptions) []string {
//	    // Custom logic for markdown-aware chunking
//	    // Split by headers, preserve code blocks, etc.
//
//	    if c.preserveHeaders {
//	        // Split at ## headers but keep header with content
//	        return c.splitByHeaders(text, options)
//	    }
//
//	    // Fallback to simple splitting
//	    return c.splitSimple(text, options)
//	}
//
//	func (c *MarkdownChunker) splitByHeaders(text string, options vectorizer.ChunkOptions) []string {
//	    // Your markdown-specific splitting logic
//	    return []string{text} // Simplified
//	}
//
//	func (c *MarkdownChunker) splitSimple(text string, options vectorizer.ChunkOptions) []string {
//	    // Use utility function for token estimation
//	    if vectorizer.EstimateTokens(text) <= options.MaxTokens {
//	        return []string{text}
//	    }
//	    // Implement your splitting logic
//	    return []string{text}
//	}
//
//	// Usage
//	markdownChunker := &MarkdownChunker{preserveHeaders: true}
//	v, err := vectorizer.New(provider, markdownChunker)
//
// # Built-in Components
//
// OpenAIProvider supports multiple models with automatic dimension detection:
//   - text-embedding-3-small (1536 dimensions) - Default, good balance of performance and cost
//   - text-embedding-3-large (3072 dimensions) - Higher quality, more expensive
//   - text-embedding-ada-002 (1536 dimensions) - Legacy model
//
// SimpleChunker provides intelligent text splitting:
//   - Sentence-aware splitting maintains semantic coherence
//   - Configurable overlap preserves context between chunks
//   - Automatic merging prevents overly small fragments
//   - Word-level fallback for sentences exceeding token limits
//
// # Advanced Usage Patterns
//
// Batch processing for efficiency:
//
//	texts := []string{"text1", "text2", "text3"}
//	vectors, err := v.ChunksToVectors(ctx, texts)
//
// Runtime chunker switching:
//
//	// Start with sentence-aware chunking
//	v, _ := vectorizer.NewWithDefaults(provider)
//
//	// Switch to token-only chunking for specific content
//	tokenChunker := vectorizer.NewSimpleChunkerWithOptions(false)
//	err := v.SetChunker(tokenChunker)
//
// Provider configuration with custom HTTP client:
//
//	client := &http.Client{
//	    Timeout: 60 * time.Second,
//	    Transport: &http.Transport{
//	        MaxIdleConns:        10,
//	        IdleConnTimeout:     90 * time.Second,
//	    },
//	}
//
//	provider, err := vectorizer.NewOpenAIProvider(vectorizer.OpenAIConfig{
//	    APIKey:     os.Getenv("OPENAI_API_KEY"),
//	    Model:      "text-embedding-3-large",
//	    HTTPClient: client,
//	})
//
// # Performance Considerations
//
// Optimize for your specific use case:
//
// - Batch processing: Use ChunksToVectors for multiple texts (reduces API calls)
// - Chunk size: Balance between context (larger chunks) and precision (smaller chunks)
// - Overlap: Higher overlap maintains context but increases processing time and cost
// - HTTP client: Configure timeout and connection pooling for large batches
// - Model selection: text-embedding-3-small for speed, text-embedding-3-large for quality
//
// Token estimation accuracy:
//   - Uses 1 word ≈ 1.3 tokens approximation for English text
//   - May vary for other languages or technical content
//   - Consider implementing custom token counting for precise use cases
//
// # Error Handling
//
// The package provides domain-specific errors that can be wrapped with additional
// context. All errors follow the standard errors.Is/As patterns for inspection:
//
//	vector, err := v.ToVector(ctx, text)
//	if err != nil {
//	    if errors.Is(err, vectorizer.ErrRateLimitExceeded) {
//	        // Implement exponential backoff
//	        time.Sleep(time.Second * 5)
//	        return v.ToVector(ctx, text)
//	    }
//	    if errors.Is(err, vectorizer.ErrContextLengthExceeded) {
//	        // Text too long, chunk it first
//	        chunks := v.Chunk(text, vectorizer.DefaultChunkOptions())
//	        return v.ChunksToVectors(ctx, chunks)
//	    }
//	    return nil, err
//	}
//
// Common error types:
//   - ErrProviderNotSet        – vectorizer created without provider
//   - ErrEmptyText             – attempting to vectorize empty text
//   - ErrVectorizationFailed   – provider failed to generate embedding
//   - ErrRateLimitExceeded     – API rate limit hit (provider-specific)
//   - ErrContextLengthExceeded – text too long for model
//   - ErrAPIKeyRequired        – missing API key in provider configuration
//   - ErrInvalidModel          – unsupported model name
//
// # Integration Examples
//
// Semantic search implementation:
//
//	// Index documents
//	documents := []string{"doc1", "doc2", "doc3"}
//	var index []vectorizer.Vector
//	for _, doc := range documents {
//	    chunks, _ := v.Process(ctx, doc, vectorizer.DefaultChunkOptions())
//	    for _, chunk := range chunks {
//	        index = append(index, chunk.Vector)
//	    }
//	}
//
//	// Search
//	query := "user's search query"
//	queryVector, _ := v.ToVector(ctx, query)
//	// Implement cosine similarity comparison with index
//
// See the accompanying README.md for complete implementation examples.
package vectorizer
