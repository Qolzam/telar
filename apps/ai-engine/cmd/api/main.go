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

	// Initialize dependencies
	log.Printf("Initializing LLM client with provider: %s", cfg.LLM.Provider)
	
	var llmClient llm.Client

	switch cfg.LLM.Provider {
	case "groq":
		llmClient, err = llm.NewGroqClient(llm.GroqConfig{
			APIKey:          cfg.LLM.GroqAPIKey,
			EmbeddingModel:  cfg.LLM.GroqEmbeddingModel,
			CompletionModel: cfg.LLM.GroqModel,
		})
		if err != nil {
			log.Fatalf("Failed to create Groq client: %v", err)
		}
		log.Println("Using Groq LLM provider")
	case "ollama":
		llmClient = llm.NewOllamaClient(llm.OllamaConfig{
			BaseURL:         cfg.LLM.OllamaBaseURL,
			EmbeddingModel:  cfg.LLM.EmbeddingModel,
			CompletionModel: cfg.LLM.CompletionModel,
		})
		log.Println("Using Ollama LLM provider")
	default:
		log.Fatalf("Invalid LLM_PROVIDER specified: %s (supported: ollama, groq)", cfg.LLM.Provider)
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

	log.Printf("Initializing knowledge service with embedding model: %s", cfg.LLM.EmbeddingModel)
	knowledgeService := knowledge.NewService(llmClient, weaviateClient, knowledge.Config{
		EmbeddingModel: cfg.LLM.EmbeddingModel,
	})

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
		log.Printf("LLM Provider: %s", cfg.LLM.Provider)
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
