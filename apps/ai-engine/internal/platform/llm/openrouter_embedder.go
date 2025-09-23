package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// OpenRouterEmbedder adapts the OpenRouter API to our EmbeddingClient interface.
type OpenRouterEmbedder struct {
	apiKey string
	model  string
	httpClient *http.Client
}

var _ EmbeddingClient = (*OpenRouterEmbedder)(nil)

// NewOpenRouterEmbedder creates a new embedder using the OpenRouter API.
func NewOpenRouterEmbedder(apiKey string) (*OpenRouterEmbedder, error) {
	return NewOpenRouterEmbedderWithModel(apiKey, "text-embedding-3-small")
}

// NewOpenRouterEmbedderWithModel creates a new embedder with a specific model.
func NewOpenRouterEmbedderWithModel(apiKey, model string) (*OpenRouterEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required")
	}
	if model == "" {
		model = "text-embedding-3-small" 
	}
	
	return &OpenRouterEmbedder{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// OpenRouter embedding request/response structures
type openrouterEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openrouterEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Object string `json:"object"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GenerateEmbeddings returns an error as OpenRouter does not currently support embeddings.
func (e *OpenRouterEmbedder) GenerateEmbeddings(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embeddings are currently not supported by OpenRouter. Please use Ollama or OpenAI for embeddings, or use a hybrid configuration with OpenRouter for completions only")
}

// Health returns an error as OpenRouter does not currently support embeddings.
func (e *OpenRouterEmbedder) Health(ctx context.Context) error {
	return fmt.Errorf("embeddings are currently not supported by OpenRouter. Please use Ollama or OpenAI for embeddings, or use a hybrid configuration with OpenRouter for completions only")
}
