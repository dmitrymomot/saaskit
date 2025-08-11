# Vectorizer Package

Text-to-vector conversion with intelligent chunking and multiple provider support.

## Features

- 🚀 **Simple API** - Convert text to vectors with minimal configuration
- 📚 **Pluggable Chunkers** - Extensible chunking interface with custom strategies
- 🔌 **Provider Pattern** - Support for multiple embedding providers (OpenAI included)
- ⚡ **Batch Processing** - Efficient vectorization of multiple texts
- 🎯 **Type Safe** - Full type safety with clean interfaces
- 🧩 **Extensible** - Easy to add custom chunkers and providers

## Installation

```bash
go get github.com/dmitrymomot/saaskit/pkg/vectorizer
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/dmitrymomot/saaskit/pkg/vectorizer"
)

func main() {
    // Create OpenAI provider
    provider, err := vectorizer.NewOpenAIProvider(vectorizer.OpenAIConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        panic(err)
    }

    // Create vectorizer with default chunker
    v, err := vectorizer.NewWithDefaults(provider)
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // Simple text to vector
    vector, err := v.ToVector(ctx, "Hello, world!")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Vector dimensions: %d\n", len(vector))
}
```

## Usage Examples

### Basic Text Vectorization

```go
// Create with default chunker (SimpleChunker)
v, err := vectorizer.NewWithDefaults(provider)

// Or create with explicit chunker
chunker := vectorizer.NewSimpleChunker()
v, err := vectorizer.New(provider, chunker)

// Single text
vector, err := v.ToVector(ctx, "Convert this text to a vector")

// Multiple texts (batch processing)
texts := []string{
    "First text",
    "Second text",
    "Third text",
}
vectors, err := v.ChunksToVectors(ctx, texts)
```

### Processing Long Documents

```go
// Process a long document with automatic chunking
longText := "Your very long document text here..."

chunks, err := v.Process(ctx, longText, vectorizer.DefaultChunkOptions())
if err != nil {
    panic(err)
}

// Each chunk contains text and its vector
for i, chunk := range chunks {
    fmt.Printf("Chunk %d: %d chars, vector dim: %d\n",
        i, len(chunk.Text), len(chunk.Vector))
}
```

### Custom Chunking Options

```go
options := vectorizer.ChunkOptions{
    MaxTokens:       300,   // Smaller chunks for more precision
    Overlap:         30,    // Less overlap to reduce redundancy
    SplitBySentence: true,  // Maintain sentence boundaries
    MinChunkSize:    20,    // Avoid tiny fragments
}

chunks, err := v.Process(ctx, document, options)
```

### Using Different OpenAI Models

```go
// Use larger model for better accuracy
provider, err := vectorizer.NewOpenAIProvider(vectorizer.OpenAIConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-large", // 3072 dimensions
})
if err != nil {
    panic(err)
}

// Check model dimensions
v, err := vectorizer.NewWithDefaults(provider)
if err != nil {
    panic(err)
}
fmt.Printf("Vector dimensions: %d\n", v.Dimensions()) // Output: 3072
```

### Working with Chunkers

```go
// Use chunker directly without vectorization
chunker := vectorizer.NewSimpleChunker()
chunks := chunker.Split(text, vectorizer.ChunkOptions{
    MaxTokens:    500,
    Overlap:      50,
    MinChunkSize: 10,
})

// Or use through vectorizer
chunks = v.Chunk(text, options)

// Change chunker at runtime
newChunker := vectorizer.NewSimpleChunkerWithOptions(false) // No sentence splitting
err := v.SetChunker(newChunker)

// Process with a one-off custom chunker
chunks, err := v.ProcessWithChunker(ctx, text, customChunker, options)
```

## Semantic Search Example

```go
import (
    "context"
    "fmt"
    "math"
    "sort"

    "github.com/dmitrymomot/saaskit/pkg/vectorizer"
)

// Build a simple semantic search
type Document struct {
    ID     string
    Text   string
    Vector vectorizer.Vector
}

func buildSearchIndex(v *vectorizer.Vectorizer, texts []string) ([]Document, error) {
    ctx := context.Background()
    docs := make([]Document, len(texts))

    for i, text := range texts {
        vector, err := v.ToVector(ctx, text)
        if err != nil {
            return nil, err
        }

        docs[i] = Document{
            ID:     fmt.Sprintf("doc-%d", i),
            Text:   text,
            Vector: vector,
        }
    }

    return docs, nil
}

// Calculate cosine similarity
func cosineSimilarity(a, b vectorizer.Vector) float64 {
    if len(a) != len(b) {
        return 0
    }

    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }

    if normA == 0 || normB == 0 {
        return 0
    }

    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Search for similar documents
func search(v *vectorizer.Vectorizer, docs []Document, query string, topK int) ([]Document, error) {
    ctx := context.Background()

    // Vectorize query
    queryVector, err := v.ToVector(ctx, query)
    if err != nil {
        return nil, err
    }

    // Calculate similarities
    type result struct {
        doc  Document
        score float64
    }

    results := make([]result, len(docs))
    for i, doc := range docs {
        results[i] = result{
            doc:   doc,
            score: cosineSimilarity(queryVector, doc.Vector),
        }
    }

    // Sort by similarity
    sort.Slice(results, func(i, j int) bool {
        return results[i].score > results[j].score
    })

    // Return top K
    if topK > len(results) {
        topK = len(results)
    }

    topDocs := make([]Document, topK)
    for i := 0; i < topK; i++ {
        topDocs[i] = results[i].doc
    }

    return topDocs, nil
}
```

## Architecture

The package uses a clean separation of concerns with two main interfaces:

1. **Provider Interface** - Handles text-to-vector conversion
2. **Chunker Interface** - Handles text splitting strategies

This design allows you to mix and match different providers and chunkers:

```go
// OpenAI + Simple chunking (default)
v1, _ := vectorizer.NewWithDefaults(openAIProvider)

// OpenAI + Custom markdown chunker
v2, _ := vectorizer.New(openAIProvider, markdownChunker)

// Custom provider + Custom chunker
v3, _ := vectorizer.New(customProvider, customChunker)
```

## Custom Implementations

### Custom Chunker

```go
// Implement your own chunking strategy
type MarkdownChunker struct {
    // Your configuration
}

func (c *MarkdownChunker) Split(text string, options vectorizer.ChunkOptions) []string {
    // Your chunking logic (e.g., split by headers)
    // Access custom options: options.Custom["key"]
    return []string{/* chunks */}
}

// Use custom chunker
chunker := &MarkdownChunker{}
v, err := vectorizer.New(provider, chunker)
```

### Custom Provider

```go
// Implement your own embedding provider
type CustomProvider struct {
    // Your configuration
}

func (p *CustomProvider) Vectorize(ctx context.Context, text string) (vectorizer.Vector, error) {
    // Your vectorization logic
    return vectorizer.Vector{0.1, 0.2, 0.3}, nil
}

func (p *CustomProvider) VectorizeBatch(ctx context.Context, texts []string) ([]vectorizer.Vector, error) {
    vectors := make([]vectorizer.Vector, len(texts))
    for i, text := range texts {
        v, err := p.Vectorize(ctx, text)
        if err != nil {
            return nil, err
        }
        vectors[i] = v
    }
    return vectors, nil
}

func (p *CustomProvider) Dimensions() int {
    return 768 // Your model's dimensions
}

// Use custom provider with custom chunker
v, err := vectorizer.New(&CustomProvider{}, &MarkdownChunker{})
```

## Configuration

### Chunk Options

```go
options := vectorizer.ChunkOptions{
    MaxTokens:    500,  // Max tokens per chunk
    Overlap:      50,   // Token overlap between chunks
    MinChunkSize: 10,   // Minimum chunk size
    Custom: map[string]interface{}{
        "splitBySentence": true,  // SimpleChunker option
        // Add custom options for your chunker
    },
}
```

### OpenAI Provider Options

```go
import (
    "net/http"
    "time"
)

provider, err := vectorizer.NewOpenAIProvider(vectorizer.OpenAIConfig{
    APIKey: "your-api-key",
    Model:  "text-embedding-3-small", // or text-embedding-3-large
    HTTPClient: &http.Client{
        Timeout: 60 * time.Second, // Custom timeout for large batches
    },
})
if err != nil {
    panic(err)
}
```

### Available Models

| Model                  | Dimensions | Description                        |
| ---------------------- | ---------- | ---------------------------------- |
| text-embedding-3-small | 1536       | Fast, cost-effective, good quality |
| text-embedding-3-large | 3072       | Higher quality, more dimensions    |
| text-embedding-ada-002 | 1536       | Legacy model, still supported      |

## Error Handling

```go
import "errors"

vector, err := v.ToVector(ctx, text)
if err != nil {
    switch {
    case errors.Is(err, vectorizer.ErrEmptyText):
        // Handle empty input
        fmt.Println("Text cannot be empty")
    case errors.Is(err, vectorizer.ErrProviderNotSet):
        // Provider configuration issue
        fmt.Println("Vectorization provider not configured")
    case errors.Is(err, vectorizer.ErrRateLimitExceeded):
        // Implement backoff/retry
        fmt.Println("Rate limit exceeded, retrying later")
    case errors.Is(err, vectorizer.ErrContextLengthExceeded):
        // Text too long, use chunking
        fmt.Println("Text too long, consider using Process() method")
    case errors.Is(err, vectorizer.ErrVectorizationFailed):
        // General vectorization failure
        fmt.Println("Failed to vectorize text:", err)
    case errors.Is(err, vectorizer.ErrAPIKeyRequired):
        // Missing API key
        fmt.Println("OpenAI API key is required")
    case errors.Is(err, vectorizer.ErrInvalidModel):
        // Invalid model name
        fmt.Println("Invalid OpenAI model specified")
    default:
        // Generic error
        fmt.Println("Unexpected error:", err)
    }
}
```

## Performance Tips

1. **Batch Processing**: Always use `ChunksToVectors` for multiple texts
2. **Chunk Size**: Balance between context (300-500 tokens) and precision (100-200 tokens)
3. **Overlap**: Use 10-20% overlap for maintaining context
4. **Caching**: Consider caching vectors for frequently accessed texts
5. **Rate Limiting**: Implement exponential backoff for API rate limits

## Testing

The package includes comprehensive tests with mocked providers:

```bash
make test PKG=./pkg/vectorizer
```

## License

Part of the SaasKit framework. See root LICENSE file.
