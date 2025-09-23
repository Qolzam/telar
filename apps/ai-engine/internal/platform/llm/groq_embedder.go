package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// GroqEmbedder adapts the Groq API to our EmbeddingClient interface.
type GroqEmbedder struct {
	apiKey string
	httpClient *http.Client
}

var _ EmbeddingClient = (*GroqEmbedder)(nil)

// NewGroqEmbedder creates a new embedder using the Groq API.
func NewGroqEmbedder(apiKey string) (*GroqEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Groq API key is required")
	}
	
	return &GroqEmbedder{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Groq embedding request/response structures
type groqEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type groqEmbeddingResponse struct {
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

// GenerateEmbeddings returns an error as Groq does not currently support embeddings.
func (e *GroqEmbedder) GenerateEmbeddings(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embeddings are currently not supported by Groq. Please use Ollama or OpenAI for embeddings, or use a hybrid configuration with Groq for completions only")
}

// Health returns an error as Groq does not currently support embeddings.
func (e *GroqEmbedder) Health(ctx context.Context) error {
	return fmt.Errorf("embeddings are currently not supported by Groq. Please use Ollama or OpenAI for embeddings, or use a hybrid configuration with Groq for completions only")
}
