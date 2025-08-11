package vectorizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// Default OpenAI embedding model with good balance of performance and cost
	DefaultOpenAIModel = "text-embedding-3-small"

	// OpenAI API endpoint for embeddings
	openAIEmbeddingsURL = "https://api.openai.com/v1/embeddings"

	// Maximum texts per batch request
	maxBatchSize = 100

	// Default timeout for API requests
	defaultTimeout = 30 * time.Second
)

// OpenAIProvider implements the Provider interface using OpenAI's API.
type OpenAIProvider struct {
	apiKey     string
	model      string
	dimensions int
	client     *http.Client
}

// OpenAIConfig configures the OpenAI provider.
type OpenAIConfig struct {
	// APIKey is required for authentication with OpenAI
	APIKey string

	// Model specifies which embedding model to use
	// Default: text-embedding-3-small
	Model string

	// HTTPClient allows custom HTTP client configuration
	// Default: http.Client with 30s timeout
	HTTPClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI embedding provider.
func NewOpenAIProvider(config OpenAIConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}

	model := config.Model
	if model == "" {
		model = DefaultOpenAIModel
	}

	// Set dimensions based on model
	dimensions := getModelDimensions(model)
	if dimensions == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidModel, model)
	}

	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: defaultTimeout,
		}
	}

	return &OpenAIProvider{
		apiKey:     config.APIKey,
		model:      model,
		dimensions: dimensions,
		client:     client,
	}, nil
}

// Vectorize converts a single text into a vector embedding.
func (p *OpenAIProvider) Vectorize(ctx context.Context, text string) (Vector, error) {
	vectors, err := p.VectorizeBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, ErrVectorizationFailed
	}

	return vectors[0], nil
}

// VectorizeBatch converts multiple texts into vectors in a single request.
func (p *OpenAIProvider) VectorizeBatch(ctx context.Context, texts []string) ([]Vector, error) {
	if len(texts) == 0 {
		return []Vector{}, nil
	}

	// Split into batches if necessary
	var allVectors []Vector

	for i := 0; i < len(texts); i += maxBatchSize {
		end := min(i+maxBatchSize, len(texts))
		batch := texts[i:end]
		vectors, err := p.callAPI(ctx, batch)
		if err != nil {
			return nil, err
		}

		allVectors = append(allVectors, vectors...)
	}

	return allVectors, nil
}

// Dimensions returns the vector dimensions for the current model.
func (p *OpenAIProvider) Dimensions() int {
	return p.dimensions
}

// callAPI makes the actual API request to OpenAI.
func (p *OpenAIProvider) callAPI(ctx context.Context, texts []string) ([]Vector, error) {
	// Prepare request body
	requestBody := openAIRequest{
		Model: p.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", openAIEmbeddingsURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp openAIErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			// Check for specific error types
			if strings.Contains(errorResp.Error.Message, "rate limit") {
				return nil, fmt.Errorf("%w: %s", ErrRateLimitExceeded, errorResp.Error.Message)
			}
			if strings.Contains(errorResp.Error.Message, "context length") {
				return nil, fmt.Errorf("%w: %s", ErrContextLengthExceeded, errorResp.Error.Message)
			}
			return nil, fmt.Errorf("OpenAI API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response openAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract vectors
	vectors := make([]Vector, len(response.Data))
	for i, item := range response.Data {
		vectors[i] = Vector(item.Embedding)
	}

	return vectors, nil
}

// getModelDimensions returns the vector dimensions for a given model.
func getModelDimensions(model string) int {
	switch model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-ada-002":
		return 1536
	default:
		return 0
	}
}

// OpenAI API request/response types

type openAIRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
