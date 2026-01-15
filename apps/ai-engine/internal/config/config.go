package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds application configuration loaded from environment variables
type Config struct {
	Server         ServerConfig   `json:"server"`
	LLM            LLMConfig      `json:"llm"`
	Weaviate       WeaviateConfig `json:"weaviate"`
	InternalAPIKey string         `json:"internal_api_key,omitempty"`
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
	// Global Infrastructure
	OllamaBaseURL     string `json:"ollama_base_url,omitempty"`
	GroqAPIKey        string `json:"groq_api_key,omitempty"`
	OpenAIAPIKey      string `json:"openai_api_key,omitempty"`
	OpenAIBaseURL     string `json:"openai_base_url,omitempty"`
	GlobalONNXLibPath string `json:"global_onnx_lib_path,omitempty"`
	MaxConcurrent     int    `json:"max_concurrent,omitempty"` // Required by generator service

	// Feature: Knowledge (RAG)
	KnowledgeEmbeddingProvider string `json:"knowledge_embedding_provider,omitempty"`
	KnowledgeEmbeddingModel    string `json:"knowledge_embedding_model,omitempty"`
	RAGContextMaxChars         int    `json:"rag_context_max_chars,omitempty"`   // Max context length in characters (default: 2000)
	RAGMaxResponseTokens       int    `json:"rag_max_response_tokens,omitempty"` // Max tokens to generate (default: 300)
	RAGTopK                    int    `json:"rag_top_k,omitempty"`               // Number of documents to retrieve (default: 5)
	RAGChunkSize               int    `json:"rag_chunk_size,omitempty"`          // Chunk size in characters for document splitting (default: 4000)
	RAGChunkOverlap            int    `json:"rag_chunk_overlap,omitempty"`       // Overlap size in characters between chunks (default: 800)

	// Feature: Generator
	GeneratorProvider string `json:"generator_provider,omitempty"`
	GeneratorModel    string `json:"generator_model,omitempty"`

	// Feature: Moderation
	ModerationONNXToxicityModelPath string  `json:"moderation_onnx_toxicity_model_path,omitempty"`
	ModerationONNXSpamModelPath     string  `json:"moderation_onnx_spam_model_path,omitempty"`    // Spam ONNX model
	ModerationONNXToxicityThreshold float64 `json:"moderation_onnx_toxicity_threshold,omitempty"` // Threshold for toxicity flagging (default: 0.90)
	ModerationONNXSpamThreshold     float64 `json:"moderation_onnx_spam_threshold,omitempty"`     // Threshold for spam flagging (default: 0.80)
	ModerationFallbackProvider      string  `json:"moderation_fallback_provider,omitempty"`
	ModerationFallbackModel         string  `json:"moderation_fallback_model,omitempty"`

	// Moderation Thresholds (required by analyzer)
	ModerationToxicityThreshold       float64 `json:"moderation_toxicity_threshold,omitempty"`
	ModerationSpamThreshold           float64 `json:"moderation_spam_threshold,omitempty"`
	ModerationSexualThreshold         float64 `json:"moderation_sexual_threshold,omitempty"`
	ModerationViolenceThreshold       float64 `json:"moderation_violence_threshold,omitempty"`
	ModerationMisinformationThreshold float64 `json:"moderation_misinformation_threshold,omitempty"`
}

// WeaviateConfig contains vector database settings
type WeaviateConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key,omitempty"`
}

// Load reads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	viper.SetDefault("PORT", "9066")
	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("READ_TIMEOUT", "30s")
	viper.SetDefault("WRITE_TIMEOUT", "30s")

	// Global Infrastructure
	viper.SetDefault("OLLAMA_BASE_URL", "http://localhost:11434")
	viper.SetDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
	viper.SetDefault("GLOBAL_ONNX_LIB_PATH", "./libs/libonnxruntime.so")
	viper.SetDefault("MAX_CONCURRENT", "2")

	// Feature: Knowledge
	viper.SetDefault("KNOWLEDGE_EMBEDDING_PROVIDER", "ollama")
	viper.SetDefault("KNOWLEDGE_EMBEDDING_MODEL", "nomic-embed-text")
	viper.SetDefault("RAG_CONTEXT_MAX_CHARS", 2000)  // Limit context to 2000 chars for faster inference
	viper.SetDefault("RAG_MAX_RESPONSE_TOKENS", 300) // Limit response to 300 tokens (~1200 chars)
	viper.SetDefault("RAG_TOP_K", 5)                 // Retrieve top 5 documents
	viper.SetDefault("RAG_CHUNK_SIZE", 4000)         // Chunk size in characters (~1000 tokens)
	viper.SetDefault("RAG_CHUNK_OVERLAP", 800)       // Overlap size in characters (~200 tokens)

	// Feature: Generator
	viper.SetDefault("GENERATOR_PROVIDER", "ollama")
	viper.SetDefault("GENERATOR_MODEL", "llama3:8b")

	// Feature: Moderation
	viper.SetDefault("MODERATION_ONNX_TOXICITY_MODEL_PATH", "")
	viper.SetDefault("MODERATION_ONNX_SPAM_MODEL_PATH", "")
	viper.SetDefault("MODERATION_ONNX_TOXICITY_THRESHOLD", 0.90) // Threshold for toxicity flagging
	viper.SetDefault("MODERATION_ONNX_SPAM_THRESHOLD", 0.80)     // Threshold for spam flagging
	viper.SetDefault("MODERATION_FALLBACK_PROVIDER", "ollama")
	viper.SetDefault("MODERATION_FALLBACK_MODEL", "qwen2.5:1.5b")

	viper.SetDefault("WEAVIATE_URL", "http://localhost:9077")
	viper.SetDefault("INTERNAL_API_KEY", "")

	viper.AutomaticEnv()

	config := &Config{
		Server: ServerConfig{
			Port:         viper.GetString("PORT"),
			Host:         viper.GetString("HOST"),
			ReadTimeout:  viper.GetDuration("READ_TIMEOUT"),
			WriteTimeout: viper.GetDuration("WRITE_TIMEOUT"),
		},
		LLM: LLMConfig{
			// Global Infrastructure
			OllamaBaseURL:     viper.GetString("OLLAMA_BASE_URL"),
			GroqAPIKey:        viper.GetString("GROQ_API_KEY"),
			OpenAIAPIKey:      viper.GetString("OPENAI_API_KEY"),
			OpenAIBaseURL:     viper.GetString("OPENAI_BASE_URL"),
			GlobalONNXLibPath: viper.GetString("GLOBAL_ONNX_LIB_PATH"),
			MaxConcurrent:     viper.GetInt("MAX_CONCURRENT"),

			// Feature: Knowledge
			KnowledgeEmbeddingProvider: viper.GetString("KNOWLEDGE_EMBEDDING_PROVIDER"),
			KnowledgeEmbeddingModel:    viper.GetString("KNOWLEDGE_EMBEDDING_MODEL"),
			RAGContextMaxChars:         viper.GetInt("RAG_CONTEXT_MAX_CHARS"),
			RAGMaxResponseTokens:       viper.GetInt("RAG_MAX_RESPONSE_TOKENS"),
			RAGTopK:                    viper.GetInt("RAG_TOP_K"),
			RAGChunkSize:               viper.GetInt("RAG_CHUNK_SIZE"),
			RAGChunkOverlap:            viper.GetInt("RAG_CHUNK_OVERLAP"),

			// Feature: Generator
			GeneratorProvider: viper.GetString("GENERATOR_PROVIDER"),
			GeneratorModel:    viper.GetString("GENERATOR_MODEL"),

			// Feature: Moderation
			ModerationONNXToxicityModelPath: viper.GetString("MODERATION_ONNX_TOXICITY_MODEL_PATH"),
			ModerationONNXSpamModelPath:     viper.GetString("MODERATION_ONNX_SPAM_MODEL_PATH"),
			ModerationONNXToxicityThreshold: viper.GetFloat64("MODERATION_ONNX_TOXICITY_THRESHOLD"),
			ModerationONNXSpamThreshold:     viper.GetFloat64("MODERATION_ONNX_SPAM_THRESHOLD"),
			ModerationFallbackProvider:      viper.GetString("MODERATION_FALLBACK_PROVIDER"),
			ModerationFallbackModel:         viper.GetString("MODERATION_FALLBACK_MODEL"),

			// Moderation Thresholds
			ModerationToxicityThreshold:       viper.GetFloat64("MODERATION_TOXICITY_THRESHOLD"),
			ModerationSpamThreshold:           viper.GetFloat64("MODERATION_SPAM_THRESHOLD"),
			ModerationSexualThreshold:         viper.GetFloat64("MODERATION_SEXUAL_THRESHOLD"),
			ModerationViolenceThreshold:       viper.GetFloat64("MODERATION_VIOLENCE_THRESHOLD"),
			ModerationMisinformationThreshold: viper.GetFloat64("MODERATION_MISINFORMATION_THRESHOLD"),
		},
		Weaviate: WeaviateConfig{
			URL:    viper.GetString("WEAVIATE_URL"),
			APIKey: viper.GetString("WEAVIATE_API_KEY"),
		},
		InternalAPIKey: viper.GetString("INTERNAL_API_KEY"),
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// validate ensures required configuration values are present
func (c *Config) validate() error {
	// Validate Knowledge Embedding Provider
	knowledgeProvider := c.LLM.KnowledgeEmbeddingProvider
	if knowledgeProvider == "" {
		knowledgeProvider = "ollama" // Default fallback
	}

	switch knowledgeProvider {
	case "openai":
		if c.LLM.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when using OpenAI for knowledge embeddings")
		}
	case "ollama":
		if c.LLM.OllamaBaseURL == "" {
			return fmt.Errorf("OLLAMA_BASE_URL is required when using Ollama for knowledge embeddings")
		}
	default:
		return fmt.Errorf("unsupported knowledge embedding provider: %s (supported: ollama, openai)", knowledgeProvider)
	}

	// Validate Generator Provider
	generatorProvider := c.LLM.GeneratorProvider
	if generatorProvider == "" {
		generatorProvider = "ollama" // Default fallback
	}

	switch generatorProvider {
	case "openai":
		if c.LLM.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when using OpenAI for generator")
		}
	case "openrouter":
		if c.LLM.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when using OpenRouter for generator (OpenRouter uses OpenAI compatibility)")
		}
	case "groq":
		if c.LLM.GroqAPIKey == "" {
			return fmt.Errorf("GROQ_API_KEY is required when using Groq for generator")
		}
	case "ollama":
		if c.LLM.OllamaBaseURL == "" {
			return fmt.Errorf("OLLAMA_BASE_URL is required when using Ollama for generator")
		}
	default:
		return fmt.Errorf("unsupported generator provider: %s (supported: ollama, groq, openai, openrouter)", generatorProvider)
	}

	// Validate Moderation Fallback Provider (mandatory)
	moderationProvider := c.LLM.ModerationFallbackProvider
	if moderationProvider == "" {
		return fmt.Errorf("MODERATION_FALLBACK_PROVIDER is required (moderation fallback is mandatory)")
	}

	switch moderationProvider {
	case "openai":
		if c.LLM.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when using OpenAI for moderation fallback")
		}
	case "openrouter":
		if c.LLM.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when using OpenRouter for moderation fallback (OpenRouter uses OpenAI compatibility)")
		}
	case "groq":
		if c.LLM.GroqAPIKey == "" {
			return fmt.Errorf("GROQ_API_KEY is required when using Groq for moderation fallback")
		}
	case "ollama":
		if c.LLM.OllamaBaseURL == "" {
			return fmt.Errorf("OLLAMA_BASE_URL is required when using Ollama for moderation fallback")
		}
	default:
		return fmt.Errorf("unsupported moderation fallback provider: %s (supported: ollama, groq, openai, openrouter)", moderationProvider)
	}

	if c.Weaviate.URL == "" {
		return fmt.Errorf("WEAVIATE_URL is required")
	}

	return nil
}
