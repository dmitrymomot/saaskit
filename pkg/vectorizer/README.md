# Vectorizer Package

Text-to-vector conversion with intelligent chunking and multiple provider support.

## Features

- 🚀 **Simple API** - Convert text to vectors with minimal configuration
- 📚 **Smart Chunking** - Automatically split long texts while preserving context
- 🔌 **Provider Pattern** - Support for multiple embedding providers (OpenAI included)
- ⚡ **Batch Processing** - Efficient vectorization of multiple texts
- 🎯 **Type Safe** - Full type safety with generics where applicable
- 🧩 **Zero Dependencies** - Core package has minimal external dependencies

## Installation

```go
import "github.com/dmitrymomot/saaskit/pkg/vectorizer"
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
    
    // Create vectorizer
    v, err := vectorizer.New(provider)
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
    return err
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
    APIKey: apiKey,
    Model:  "text-embedding-3-large", // 3072 dimensions
})

// Check model dimensions
v, _ := vectorizer.New(provider)
fmt.Printf("Vector dimensions: %d\n", v.Dimensions()) // Output: 3072
```

### Manual Text Chunking

```go
// Split text without vectorization
chunks := vectorizer.SplitIntoChunks(text, vectorizer.ChunkOptions{
    MaxTokens:       500,
    Overlap:         50,
    SplitBySentence: true,
})

// Vectorize chunks later
vectors, err := v.ChunksToVectors(ctx, chunks)
```

## Semantic Search Example

```go
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

## Custom Provider Implementation

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

// Use custom provider
v, err := vectorizer.New(&CustomProvider{})
```

## Configuration

### OpenAI Provider Options

```go
provider, err := vectorizer.NewOpenAIProvider(vectorizer.OpenAIConfig{
    APIKey: "your-api-key",
    Model:  "text-embedding-3-small", // or text-embedding-3-large
    HTTPClient: &http.Client{
        Timeout: 60 * time.Second, // Custom timeout for large batches
    },
})
```

### Available Models

| Model | Dimensions | Description |
|-------|------------|-------------|
| text-embedding-3-small | 1536 | Fast, cost-effective, good quality |
| text-embedding-3-large | 3072 | Higher quality, more dimensions |
| text-embedding-ada-002 | 1536 | Legacy model, still supported |

## Error Handling

```go
vector, err := v.ToVector(ctx, text)
if err != nil {
    switch {
    case errors.Is(err, vectorizer.ErrEmptyText):
        // Handle empty input
    case errors.Is(err, vectorizer.ErrRateLimitExceeded):
        // Implement backoff/retry
    case errors.Is(err, vectorizer.ErrContextLengthExceeded):
        // Text too long, use chunking
    default:
        // Generic error
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