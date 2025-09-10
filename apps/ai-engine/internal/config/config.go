package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	LLM      LLMConfig      `json:"llm"`
	Weaviate WeaviateConfig `json:"weaviate"`
}

type ServerConfig struct {
	Port         string        `json:"port"`
	Host         string        `json:"host"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
}

type LLMConfig struct {
	Provider        string `json:"provider"`
	OpenAIAPIKey    string `json:"openai_api_key,omitempty"`
	GroqAPIKey      string `json:"groq_api_key,omitempty"`
	OllamaBaseURL   string `json:"ollama_base_url,omitempty"`
	EmbeddingModel  string `json:"embedding_model,omitempty"`
	CompletionModel string `json:"completion_model,omitempty"`
}

type WeaviateConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key,omitempty"`
}

// Load reads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Host:         getEnv("HOST", "0.0.0.0"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
		},
		LLM: LLMConfig{
			Provider:        getEnv("LLM_PROVIDER", "ollama"),
			OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
			GroqAPIKey:      getEnv("GROQ_API_KEY", ""),
			OllamaBaseURL:   getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			EmbeddingModel:  getEnv("EMBEDDING_MODEL", "nomic-embed-text"),
			CompletionModel: getEnv("COMPLETION_MODEL", "llama3"),
		},
		Weaviate: WeaviateConfig{
			URL:    getEnv("WEAVIATE_URL", "http://localhost:8080"),
			APIKey: getEnv("WEAVIATE_API_KEY", ""),
		},
	}
	
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

func (c *Config) validate() error {
	switch c.LLM.Provider {
	case "openai":
		if c.LLM.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when using OpenAI provider")
		}
	case "groq":
		if c.LLM.GroqAPIKey == "" {
			return fmt.Errorf("GROQ_API_KEY is required when using Groq provider")
		}
	case "ollama":
		if c.LLM.OllamaBaseURL == "" {
			return fmt.Errorf("OLLAMA_BASE_URL is required when using Ollama provider")
		}
	default:
		return fmt.Errorf("unsupported LLM provider: %s (supported: ollama, groq, openai)", c.LLM.Provider)
	}
	
	if c.Weaviate.URL == "" {
		return fmt.Errorf("WEAVIATE_URL is required")
	}
	
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}