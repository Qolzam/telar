package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds application configuration loaded from environment variables
type Config struct {
	Server   ServerConfig   `json:"server"`
	LLM      LLMConfig      `json:"llm"`
	Weaviate WeaviateConfig `json:"weaviate"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Port         string        `json:"port"`
	Host         string        `json:"host"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

// LLMConfig contains language model provider settings
type LLMConfig struct {
	Provider           string `json:"provider"`           
	CompletionProvider string `json:"completion_provider"`
	OpenAIAPIKey       string `json:"openai_api_key,omitempty"`
	GroqAPIKey         string `json:"groq_api_key,omitempty"`
	GroqModel          string `json:"groq_model,omitempty"`
	OllamaBaseURL      string `json:"ollama_base_url,omitempty"`
	EmbeddingModel     string `json:"embedding_model,omitempty"`
	CompletionModel    string `json:"completion_model,omitempty"`
}

// WeaviateConfig contains vector database settings
type WeaviateConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key,omitempty"`
}

// Load reads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("READ_TIMEOUT", "30s")
	viper.SetDefault("WRITE_TIMEOUT", "30s")
	viper.SetDefault("LLM_PROVIDER", "ollama")
	viper.SetDefault("COMPLETION_PROVIDER", "ollama")
	viper.SetDefault("OLLAMA_BASE_URL", "http://localhost:11434")
	viper.SetDefault("EMBEDDING_MODEL", "nomic-embed-text")
	viper.SetDefault("COMPLETION_MODEL", "llama3:8b")
	viper.SetDefault("GROQ_MODEL", "llama3-8b-8192")
	viper.SetDefault("WEAVIATE_URL", "http://localhost:8080")

	viper.AutomaticEnv()

	config := &Config{
		Server: ServerConfig{
			Port:         viper.GetString("PORT"),
			Host:         viper.GetString("HOST"),
			ReadTimeout:  viper.GetDuration("READ_TIMEOUT"),
			WriteTimeout: viper.GetDuration("WRITE_TIMEOUT"),
		},
		LLM: LLMConfig{
			Provider:           viper.GetString("LLM_PROVIDER"),
			CompletionProvider: viper.GetString("COMPLETION_PROVIDER"),
			OpenAIAPIKey:       viper.GetString("OPENAI_API_KEY"),
			GroqAPIKey:         viper.GetString("GROQ_API_KEY"),
			GroqModel:          viper.GetString("GROQ_MODEL"),
			OllamaBaseURL:      viper.GetString("OLLAMA_BASE_URL"),
			EmbeddingModel:     viper.GetString("EMBEDDING_MODEL"),
			CompletionModel:    viper.GetString("COMPLETION_MODEL"),
		},
		Weaviate: WeaviateConfig{
			URL:    viper.GetString("WEAVIATE_URL"),
			APIKey: viper.GetString("WEAVIATE_API_KEY"),
		},
	}
	
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// validate ensures required configuration values are present
func (c *Config) validate() error {
	if c.LLM.OllamaBaseURL == "" {
		return fmt.Errorf("OLLAMA_BASE_URL is required (Ollama is always used for embeddings)")
	}
	
	completionProvider := c.LLM.CompletionProvider
	if completionProvider == "" {
		// fall back to legacy provider field for backward compatibility
		completionProvider = c.LLM.Provider
	}
	
	switch completionProvider {
	case "openai":
		if c.LLM.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when using OpenAI completion provider")
		}
	case "groq":
		if c.LLM.GroqAPIKey == "" {
			return fmt.Errorf("GROQ_API_KEY is required when using Groq completion provider")
		}
	case "ollama":
	default:
		return fmt.Errorf("unsupported completion provider: %s (supported: ollama, groq, openai)", completionProvider)
	}
	
	if c.Weaviate.URL == "" {
		return fmt.Errorf("WEAVIATE_URL is required")
	}
	
	return nil
}