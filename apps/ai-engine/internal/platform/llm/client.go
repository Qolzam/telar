package llm

import (
	"context"
)

// Client defines the interface for language model operations
type Client interface {
	GenerateEmbeddings(ctx context.Context, text string) ([]float32, error)
	GenerateCompletion(ctx context.Context, prompt string) (string, error)
}

type EmbeddingRequest struct {
	Text  string `json:"text"`
	Model string `json:"model,omitempty"`
}

type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
	Usage     *Usage    `json:"usage,omitempty"`
}

type CompletionRequest struct {
	Prompt      string  `json:"prompt"`
	Model       string  `json:"model,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
}

type CompletionResponse struct {
	Text  string `json:"text"`
	Usage *Usage `json:"usage,omitempty"`
}

// Usage tracks token consumption for billing and monitoring
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
