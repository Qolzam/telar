package main

import (
	"fmt"

	"github.com/qolzam/telar/apps/ai-engine/internal/config"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/llm"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// createEmbeddingClient creates an embedding client based on provider
// Returns: llm.EmbeddingClient
// Supported providers: "ollama", "openai"
// Note: Groq and OpenRouter do NOT support embeddings
func createEmbeddingClient(provider, model string, cfg config.LLMConfig) (llm.EmbeddingClient, error) {
	switch provider {
	case "ollama":
		return llm.NewOllamaClient(llm.OllamaConfig{
			BaseURL:        cfg.OllamaBaseURL,
			EmbeddingModel: model,
		}), nil
	case "openai":
		return llm.NewOpenAIEmbedder(cfg.OpenAIAPIKey)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s (supported: ollama, openai)", provider)
	}
}

// createCompletionClient creates a completion client (llms.Model) based on provider
// Returns: llms.Model (LangChain interface)
// Supported providers: "ollama", "groq", "openai", "openrouter"
// maxTokens: Maximum tokens to generate (0 = unlimited). Only used for Ollama provider.
func createCompletionClient(provider, model string, cfg config.LLMConfig, maxTokens int) (llms.Model, error) {
	switch provider {
	case "ollama":
		ollamaClient := llm.NewOllamaClient(llm.OllamaConfig{
			BaseURL:             cfg.OllamaBaseURL,
			GenerationModel:     model,
			ClassificationModel: model,
			MaxTokens:           maxTokens,
		})
		return llm.NewOllamaLangChainAdapter(ollamaClient, llm.ModelTypeGeneration), nil
	case "groq":
		groqClient, err := llm.NewGroqClient(llm.GroqConfig{
			APIKey:              cfg.GroqAPIKey,
			GenerationModel:     model,
			ClassificationModel: model,
		})
		if err != nil {
			return nil, err
		}
		return llm.NewGroqLangChainAdapter(groqClient, llm.ModelTypeGeneration), nil
	case "openai":
		llmClient, err := openai.New(
			openai.WithToken(cfg.OpenAIAPIKey),
			openai.WithBaseURL(cfg.OpenAIBaseURL),
			openai.WithModel(model),
		)
		return llmClient, err
	case "openrouter":
		baseURL := "https://openrouter.ai/api/v1"
		if cfg.OpenAIBaseURL != "https://api.openai.com/v1" {
			baseURL = cfg.OpenAIBaseURL
		}
		llmClient, err := openai.New(
			openai.WithToken(cfg.OpenAIAPIKey),
			openai.WithBaseURL(baseURL),
			openai.WithModel(model),
		)
		return llmClient, err
	default:
		return nil, fmt.Errorf("unsupported completion provider: %s", provider)
	}
}
