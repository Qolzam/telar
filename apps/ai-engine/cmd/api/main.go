// AI Engine - Community Knowledge Engine for Telar Platform
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/qolzam/telar/apps/ai-engine/internal/analyzer"
	"github.com/qolzam/telar/apps/ai-engine/internal/api"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
	"github.com/qolzam/telar/apps/ai-engine/internal/generator"
	"github.com/qolzam/telar/apps/ai-engine/internal/knowledge"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/llm"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/weaviate"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	serviceName    = "ai-engine"
	serviceVersion = "v1.0.0"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found or failed to load: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("✅ Configuration loaded and validated successfully.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("=== Initializing Fully Configurable LLM Architecture ===")
	
	var embeddingClient llm.EmbeddingClient

	embeddingProvider := cfg.LLM.EmbeddingProvider
	if embeddingProvider == "" {
		embeddingProvider = "ollama" 
	}

	log.Printf("Initializing embedding client with provider: %s", embeddingProvider)
	switch embeddingProvider {
	case "openai":
		apiKey := cfg.LLM.OpenAIAPIKey
		var err error
		embeddingClient, err = llm.NewOpenAIEmbedder(apiKey)
		if err != nil {
			log.Fatalf("Failed to create OpenAI embedding client: %v", err)
		}
		log.Printf("✓ Embedding provider: OpenAI")
	case "groq":
		apiKey := cfg.LLM.GroqAPIKey
		var err error
		embeddingClient, err = llm.NewGroqEmbedder(apiKey)
		if err != nil {
			log.Fatalf("Failed to create Groq embedding client: %v", err)
		}
		log.Printf("✓ Embedding provider: Groq")
	case "openrouter":
		apiKey := cfg.LLM.OpenAIAPIKey
		model := cfg.LLM.OpenAIModel
		var err error
		embeddingClient, err = llm.NewOpenRouterEmbedderWithModel(apiKey, model)
		if err != nil {
			log.Fatalf("Failed to create OpenRouter embedding client: %v", err)
		}
		log.Printf("✓ Embedding provider: OpenRouter (model: %s)", model)
	case "ollama":
		embeddingClient = llm.NewOllamaClient(llm.OllamaConfig{
			BaseURL:         cfg.LLM.OllamaBaseURL,
			EmbeddingModel:  cfg.LLM.EmbeddingModel,
			CompletionModel: cfg.LLM.CompletionModel,
		})
		log.Printf("✓ Embedding provider: Ollama (model: %s)", cfg.LLM.EmbeddingModel)
	default:
		log.Fatalf("Invalid EMBEDDING_PROVIDER specified: %s (supported: ollama, openai, groq, openrouter)", embeddingProvider)
	}

	var completionClient llms.Model
	completionProvider := cfg.LLM.CompletionProvider
	if completionProvider == "" {
		// fall back to legacy provider field for backward compatibility
		completionProvider = cfg.LLM.Provider
	}

	log.Printf("Initializing completion client with provider: %s", completionProvider)
	switch completionProvider {
	case "openai":
		apiKey := cfg.LLM.OpenAIAPIKey
		baseURL := cfg.LLM.OpenAIBaseURL
		model := cfg.LLM.OpenAIModel
		
		llmClient, err := openai.New(
			openai.WithToken(apiKey),
			openai.WithBaseURL(baseURL),
			openai.WithModel(model),
		)
		if err != nil {
			log.Fatalf("Failed to create OpenAI client: %v", err)
		}
		completionClient = llmClient
		log.Printf("✓ Completion provider: OpenAI (base: %s, model: %s)", baseURL, model)
	case "openrouter":
		apiKey := cfg.LLM.OpenAIAPIKey
		baseURL := "https://openrouter.ai/api/v1"
		model := cfg.LLM.OpenAIModel
		
		if cfg.LLM.OpenAIBaseURL != "https://api.openai.com/v1" {
			baseURL = cfg.LLM.OpenAIBaseURL
		}
		
		llmClient, err := openai.New(
			openai.WithToken(apiKey),
			openai.WithBaseURL(baseURL),
			openai.WithModel(model),
		)
		if err != nil {
			log.Fatalf("Failed to create OpenRouter client: %v", err)
		}
		completionClient = llmClient
		log.Printf("✓ Completion provider: OpenRouter (base: %s, model: %s)", baseURL, model)
	case "groq":
		groqClient, err := llm.NewGroqClient(llm.GroqConfig{
			APIKey:          cfg.LLM.GroqAPIKey,
			CompletionModel: cfg.LLM.GroqModel,
		})
		if err != nil {
			log.Fatalf("Failed to create Groq completion client: %v", err)
		}
		completionClient = llm.NewGroqLangChainAdapter(groqClient)
		log.Printf("✓ Completion provider: Groq (model: %s)", cfg.LLM.GroqModel)
	case "ollama":
		ollamaClient := llm.NewOllamaClient(llm.OllamaConfig{
			BaseURL:         cfg.LLM.OllamaBaseURL,
			EmbeddingModel:  cfg.LLM.EmbeddingModel,
			CompletionModel: cfg.LLM.CompletionModel,
		})
		completionClient = llm.NewOllamaLangChainAdapter(ollamaClient)
		log.Printf("✓ Completion provider: Ollama (base: %s, model: %s)", cfg.LLM.OllamaBaseURL, cfg.LLM.CompletionModel)
	default:
		log.Fatalf("Invalid COMPLETION_PROVIDER specified: %s (supported: openai, openrouter, ollama, groq)", completionProvider)
	}

	log.Printf("Initializing Weaviate client at: %s", cfg.Weaviate.URL)
	weaviateClient, err := weaviate.NewClient(weaviate.Config{
		URL:    cfg.Weaviate.URL,
		APIKey: cfg.Weaviate.APIKey,
	})
	if err != nil {
		log.Fatalf("Failed to create Weaviate client: %v", err)
	}

	if err := weaviateClient.EnsureSchema(ctx); err != nil {
		log.Printf("Warning: Failed to ensure Weaviate schema: %v", err)
	}

	log.Printf("Initializing knowledge service...")
	knowledgeService := knowledge.NewService(embeddingClient, completionClient, weaviateClient, knowledge.Config{
		EmbeddingModel: cfg.LLM.EmbeddingModel,
	})
	log.Println("✓ Knowledge service initialized")

	log.Printf("Initializing generator service...")
	generatorService := generator.NewService(completionClient, cfg.LLM.MaxConcurrent)
	log.Printf("✓ Generator service initialized (max concurrent: %d)", cfg.LLM.MaxConcurrent)

	log.Printf("Initializing analyzer service...")
	analyzerService := analyzer.NewService(completionClient)
	log.Printf("✓ Analyzer service initialized")

	log.Println("Performing health checks...")
	if err := knowledgeService.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Health check failed: %v", err)
		log.Println("Continuing startup, but some features may not work properly")
	} else {
		log.Println("All health checks passed")
	}

	app := api.Router(knowledgeService, generatorService, analyzerService, cfg)

	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Starting %s %s on %s", serviceName, serviceVersion, addr)
		log.Printf("Architecture: Fully Configurable (Embedding: %s, Completion: %s)", embeddingProvider, completionProvider)
		log.Printf("Weaviate URL: %s", cfg.Weaviate.URL)
		
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	ctx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
