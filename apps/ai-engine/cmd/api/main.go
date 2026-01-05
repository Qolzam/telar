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
	"github.com/qolzam/telar/apps/ai-engine/internal/moderation"
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

	// Create separate completion clients for generation and classification
	completionProvider := cfg.LLM.CompletionProvider
	if completionProvider == "" {
		// fall back to legacy provider field for backward compatibility
		completionProvider = cfg.LLM.Provider
	}

	log.Printf("Initializing completion clients with provider: %s (multi-model strategy)", completionProvider)

	var generationClient llms.Model
	var classificationClient llms.Model

	switch completionProvider {
	case "openai":
		apiKey := cfg.LLM.OpenAIAPIKey
		baseURL := cfg.LLM.OpenAIBaseURL
		genModel := cfg.LLM.OpenAIGenerationModel
		classModel := cfg.LLM.OpenAIClassificationModel

		genLLM, err := openai.New(
			openai.WithToken(apiKey),
			openai.WithBaseURL(baseURL),
			openai.WithModel(genModel),
		)
		if err != nil {
			log.Fatalf("Failed to create OpenAI generation client: %v", err)
		}
		generationClient = genLLM

		classLLM, err := openai.New(
			openai.WithToken(apiKey),
			openai.WithBaseURL(baseURL),
			openai.WithModel(classModel),
		)
		if err != nil {
			log.Fatalf("Failed to create OpenAI classification client: %v", err)
		}
		classificationClient = classLLM

		log.Printf("✓ Completion provider: OpenAI")
		log.Printf("  Generation model: %s", genModel)
		log.Printf("  Classification model: %s", classModel)
	case "openrouter":
		apiKey := cfg.LLM.OpenAIAPIKey
		baseURL := "https://openrouter.ai/api/v1"
		genModel := cfg.LLM.OpenAIGenerationModel
		classModel := cfg.LLM.OpenAIClassificationModel

		if cfg.LLM.OpenAIBaseURL != "https://api.openai.com/v1" {
			baseURL = cfg.LLM.OpenAIBaseURL
		}

		genLLM, err := openai.New(
			openai.WithToken(apiKey),
			openai.WithBaseURL(baseURL),
			openai.WithModel(genModel),
		)
		if err != nil {
			log.Fatalf("Failed to create OpenRouter generation client: %v", err)
		}
		generationClient = genLLM

		classLLM, err := openai.New(
			openai.WithToken(apiKey),
			openai.WithBaseURL(baseURL),
			openai.WithModel(classModel),
		)
		if err != nil {
			log.Fatalf("Failed to create OpenRouter classification client: %v", err)
		}
		classificationClient = classLLM

		log.Printf("✓ Completion provider: OpenRouter")
		log.Printf("  Generation model: %s", genModel)
		log.Printf("  Classification model: %s", classModel)
	case "groq":
		groqClient, err := llm.NewGroqClient(llm.GroqConfig{
			APIKey:              cfg.LLM.GroqAPIKey,
			CompletionModel:     cfg.LLM.GroqModel, // Legacy support
			GenerationModel:     cfg.LLM.GroqGenerationModel,
			ClassificationModel: cfg.LLM.GroqClassificationModel,
		})
		if err != nil {
			log.Fatalf("Failed to create Groq completion client: %v", err)
		}
		generationClient = llm.NewGroqLangChainAdapter(groqClient, llm.ModelTypeGeneration)
		classificationClient = llm.NewGroqLangChainAdapter(groqClient, llm.ModelTypeClassification)

		log.Printf("✓ Completion provider: Groq")
		log.Printf("  Generation model: %s", cfg.LLM.GroqGenerationModel)
		log.Printf("  Classification model: %s", cfg.LLM.GroqClassificationModel)
	case "ollama":
		ollamaClient := llm.NewOllamaClient(llm.OllamaConfig{
			BaseURL:             cfg.LLM.OllamaBaseURL,
			EmbeddingModel:      cfg.LLM.EmbeddingModel,
			CompletionModel:     cfg.LLM.CompletionModel, // Legacy support
			GenerationModel:     cfg.LLM.OllamaGenerationModel,
			ClassificationModel: cfg.LLM.OllamaClassificationModel,
		})
		generationClient = llm.NewOllamaLangChainAdapter(ollamaClient, llm.ModelTypeGeneration)
		classificationClient = llm.NewOllamaLangChainAdapter(ollamaClient, llm.ModelTypeClassification)

		log.Printf("✓ Completion provider: Ollama")
		log.Printf("  Generation model: %s", cfg.LLM.OllamaGenerationModel)
		log.Printf("  Classification model: %s", cfg.LLM.OllamaClassificationModel)
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
	knowledgeService := knowledge.NewService(embeddingClient, generationClient, weaviateClient, knowledge.Config{
		EmbeddingModel: cfg.LLM.EmbeddingModel,
	})
	log.Println("✓ Knowledge service initialized (using generation model)")

	log.Printf("Initializing generator service...")
	generatorService := generator.NewService(generationClient, cfg.LLM.MaxConcurrent)
	log.Printf("✓ Generator service initialized (using generation model, max concurrent: %d)", cfg.LLM.MaxConcurrent)

	log.Printf("Initializing analyzer service...")
	// Configure moderation thresholds from config
	moderationThresholds := analyzer.ModerationThresholds{
		Toxicity:       cfg.LLM.ModerationToxicityThreshold,
		Spam:           cfg.LLM.ModerationSpamThreshold,
		Sexual:         cfg.LLM.ModerationSexualThreshold,
		Violence:       cfg.LLM.ModerationViolenceThreshold,
		Misinformation: cfg.LLM.ModerationMisinformationThreshold,
	}
	analyzerService := analyzer.NewService(classificationClient, moderationThresholds)
	log.Printf("✓ Analyzer service initialized (using classification model)")
	log.Printf("  Moderation thresholds: Toxicity=%.2f, Spam=%.2f, Sexual=%.2f, Violence=%.2f, Misinformation=%.2f",
		moderationThresholds.Toxicity, moderationThresholds.Spam, moderationThresholds.Sexual,
		moderationThresholds.Violence, moderationThresholds.Misinformation)

	// Determine classification model name for logging
	var classificationModelName string
	switch completionProvider {
	case "openai", "openrouter":
		classificationModelName = cfg.LLM.OpenAIClassificationModel
	case "groq":
		classificationModelName = cfg.LLM.GroqClassificationModel
	case "ollama":
		classificationModelName = cfg.LLM.OllamaClassificationModel
	default:
		classificationModelName = "unknown"
	}

	log.Printf("Initializing tiered moderation pipeline...")
	// Initialize L1 Cache
	modCache := moderation.NewInMemoryCache()
	log.Printf("✓ L1 Cache initialized (InMemoryCache)")

	// Initialize L2 Heuristics
	keywordFilter := moderation.NewKeywordFilter()
	log.Printf("✓ L2 Keyword Filter initialized")

	// Initialize L3 LLM Analyzer (adapter)
	llmAnalyzer := moderation.NewLLMAnalyzerAdapter(analyzerService, classificationModelName)
	log.Printf("✓ L3 LLM Analyzer initialized (model: %s)", classificationModelName)

	// Build Pipeline
	modPipeline := moderation.NewPipeline(modCache, llmAnalyzer, keywordFilter)
	log.Printf("✓ Tiered Moderation Pipeline initialized (L1 Cache → L2 Heuristics → L3 LLM)")

	log.Println("Performing health checks...")
	if err := knowledgeService.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Health check failed: %v", err)
		log.Println("Continuing startup, but some features may not work properly")
	} else {
		log.Println("All health checks passed")
	}

	app := api.Router(knowledgeService, generatorService, modPipeline, cfg)

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
