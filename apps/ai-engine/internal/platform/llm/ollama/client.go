package ollama

import (
	"time"

	"github.com/qolzam/telar/apps/ai-engine/internal/platform/llm"
)

// Config holds Ollama client configuration
type Config struct {
	BaseURL         string
	EmbeddingModel  string
	CompletionModel string
	Timeout         time.Duration
}

// NewClient creates an Ollama LLM client that implements the llm.Client interface
func NewClient(config Config) (llm.Client, error) {
	ollamaConfig := llm.OllamaConfig{
		BaseURL:         config.BaseURL,
		EmbeddingModel:  config.EmbeddingModel,
		CompletionModel: config.CompletionModel,
		Timeout:         config.Timeout,
	}

	client := llm.NewOllamaClient(ollamaConfig)
	return client, nil
}
