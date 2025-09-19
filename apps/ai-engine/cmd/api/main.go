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

	"github.com/qolzam/telar/apps/ai-engine/internal/api"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
	"github.com/qolzam/telar/apps/ai-engine/internal/knowledge"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/llm"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/weaviate"
)

const (
	serviceName    = "ai-engine"
	serviceVersion = "v1.0.0"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("=== Initializing Hybrid LLM Architecture ===")
	

	log.Printf("Initializing Ollama embedding client at: %s", cfg.LLM.OllamaBaseURL)
	embeddingClient := llm.NewOllamaClient(llm.OllamaConfig{
		BaseURL:         cfg.LLM.OllamaBaseURL,
		EmbeddingModel:  cfg.LLM.EmbeddingModel,
		CompletionModel: cfg.LLM.CompletionModel,
	})
	log.Printf("✓ Embedding provider: Ollama (model: %s)", cfg.LLM.EmbeddingModel)

	var completionClient llm.CompletionClient
	completionProvider := cfg.LLM.CompletionProvider
	if completionProvider == "" {
		// fall back to legacy provider field for backward compatibility
		completionProvider = cfg.LLM.Provider
	}

	log.Printf("Initializing completion client with provider: %s", completionProvider)
	switch completionProvider {
	case "groq":
		completionClient, err = llm.NewGroqClient(llm.GroqConfig{
			APIKey:          cfg.LLM.GroqAPIKey,
			CompletionModel: cfg.LLM.GroqModel,
		})
		if err != nil {
			log.Fatalf("Failed to create Groq completion client: %v", err)
		}
		log.Printf("✓ Completion provider: Groq (model: %s)", cfg.LLM.GroqModel)
	case "ollama":
		// reuse the same Ollama client for completions
		completionClient = embeddingClient
		log.Printf("✓ Completion provider: Ollama (model: %s)", cfg.LLM.CompletionModel)
	default:
		log.Fatalf("Invalid COMPLETION_PROVIDER specified: %s (supported: ollama, groq)", completionProvider)
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

	log.Printf("Initializing hybrid knowledge service...")
	knowledgeService := knowledge.NewService(embeddingClient, completionClient, weaviateClient, knowledge.Config{
		EmbeddingModel: cfg.LLM.EmbeddingModel,
	})
	log.Println("✓ Knowledge service initialized with specialized clients")

	// Health check before startup
	log.Println("Performing health checks...")
	if err := knowledgeService.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Health check failed: %v", err)
		log.Println("Continuing startup, but some features may not work properly")
	} else {
		log.Println("All health checks passed")
	}

	app := api.Router(knowledgeService)

	// Start server asynchronously
	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Starting %s %s on %s", serviceName, serviceVersion, addr)
		log.Printf("Architecture: Hybrid (Embedding: Ollama, Completion: %s)", completionProvider)
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
