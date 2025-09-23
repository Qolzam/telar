package llm

import (
	"context"
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"
	"time"
)

// OpenAIEmbedder adapts the OpenAI API to our EmbeddingClient interface.
type OpenAIEmbedder struct {
	apiKey string
	httpClient *http.Client
}

var _ EmbeddingClient = (*OpenAIEmbedder)(nil)

// NewOpenAIEmbedder creates a new embedder using the OpenAI API.
func NewOpenAIEmbedder(apiKey string) (*OpenAIEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	
	return &OpenAIEmbedder{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// OpenAI embedding request/response structures
type openaiEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openaiEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GenerateEmbeddings uses the OpenAI API to create embeddings for a single text.
func (e *OpenAIEmbedder) GenerateEmbeddings(ctx context.Context, text string) ([]float32, error) {
	reqBody := openaiEmbeddingRequest{
		Input: []string{text},
		Model: "text-embedding-3-small", // Default model
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}
	
	var embeddingResp openaiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if embeddingResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", embeddingResp.Error.Message)
	}
	
	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no embeddings")
	}
	
	return embeddingResp.Data[0].Embedding, nil
}

// Health checks if the OpenAI embedding service is accessible
func (e *OpenAIEmbedder) Health(ctx context.Context) error {
	// Test with a simple embedding request
	_, err := e.GenerateEmbeddings(ctx, "health check")
	return err
}
