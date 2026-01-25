package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/qolzam/telar/apps/ai-engine/internal/analyzer"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
	"github.com/qolzam/telar/apps/ai-engine/internal/generator"
	"github.com/qolzam/telar/apps/ai-engine/internal/knowledge"
	"github.com/qolzam/telar/apps/ai-engine/internal/moderation"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/weaviate"
	"github.com/qolzam/telar/apps/ai-engine/internal/prompt"
	"github.com/qolzam/telar/apps/ai-engine/internal/repository"
	ort "github.com/yalue/onnxruntime_go"
)

// Services holds all initialized services
type Services struct {
	KnowledgeService *knowledge.Service
	GeneratorService *generator.Service
	ModPipeline      *moderation.Pipeline
	WeaviateClient   *weaviate.Client
	TenantRepo       *repository.PostgresTenantRepository
	AppRepo          *repository.PostgresAppRepository
}

// initializeDatabase initializes PostgreSQL database connection
func initializeDatabase(ctx context.Context, cfg *config.Config) (*sqlx.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Postgres.Host,
		cfg.Database.Postgres.Port,
		cfg.Database.Postgres.Username,
		cfg.Database.Postgres.Password,
		cfg.Database.Postgres.Database,
		cfg.Database.Postgres.SSLMode,
	)

	db, err := sqlx.ConnectContext(ctx, "postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Configure connection pool
	if cfg.Database.Postgres.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.Database.Postgres.MaxOpenConns)
	}
	if cfg.Database.Postgres.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.Database.Postgres.MaxIdleConns)
	}
	if cfg.Database.Postgres.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.Database.Postgres.ConnMaxLifetime)
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return db, nil
}

// initializeONNX initializes ONNX Runtime if configured
func initializeONNX(globalONNXLibPath string) {
	if globalONNXLibPath != "" {
		ort.SetSharedLibraryPath(globalONNXLibPath)
		if err := ort.InitializeEnvironment(); err != nil {
			log.Printf("WARNING: Failed to initialize ONNX Runtime: %v", err)
		} else {
			log.Printf("✓ ONNX Runtime initialized (lib: %s)", globalONNXLibPath)
		}
	}
}

// initializeWeaviate initializes and configures Weaviate client
func initializeWeaviate(ctx context.Context, cfg *config.Config) (*weaviate.Client, error) {
	log.Printf("Initializing Weaviate client at: %s", cfg.Weaviate.URL)
	client, err := weaviate.NewClient(weaviate.Config{
		URL:    cfg.Weaviate.URL,
		APIKey: cfg.Weaviate.APIKey,
	})
	if err != nil {
		return nil, err
	}

	if err := client.EnsureSchema(ctx); err != nil {
		log.Printf("Warning: Failed to ensure Weaviate schema: %v", err)
	}

	return client, nil
}

// initializeKnowledgeService initializes the knowledge service with all dependencies
func initializeKnowledgeService(cfg *config.Config, weaviateClient *weaviate.Client) (*knowledge.Service, error) {
	embeddingClient, err := createEmbeddingClient(
		cfg.LLM.KnowledgeEmbeddingProvider,
		cfg.LLM.KnowledgeEmbeddingModel,
		cfg.LLM,
	)
	if err != nil {
		return nil, err
	}
	log.Printf("✓ Knowledge embedding client initialized (provider: %s, model: %s)",
		cfg.LLM.KnowledgeEmbeddingProvider, cfg.LLM.KnowledgeEmbeddingModel)

	ragGenerationClient, err := createCompletionClient(
		cfg.LLM.GeneratorProvider,
		cfg.LLM.GeneratorModel,
		cfg.LLM,
		cfg.LLM.RAGMaxResponseTokens,
	)
	if err != nil {
		return nil, err
	}

	service := knowledge.NewService(
		embeddingClient,
		ragGenerationClient,
		weaviateClient,
		knowledge.Config{
			EmbeddingModel:    cfg.LLM.KnowledgeEmbeddingModel,
			ContextMaxChars:   cfg.LLM.RAGContextMaxChars,
			MaxResponseTokens: cfg.LLM.RAGMaxResponseTokens,
			TopK:              cfg.LLM.RAGTopK,
			ChunkSize:         cfg.LLM.RAGChunkSize,
			ChunkOverlap:      cfg.LLM.RAGChunkOverlap,
		},
	)
	log.Printf("✓ Knowledge service initialized (context_max_chars=%d, max_response_tokens=%d, top_k=%d, chunk_size=%d, chunk_overlap=%d)",
		cfg.LLM.RAGContextMaxChars, cfg.LLM.RAGMaxResponseTokens, cfg.LLM.RAGTopK, cfg.LLM.RAGChunkSize, cfg.LLM.RAGChunkOverlap)

	return service, nil
}

// initializeGeneratorService initializes the generator service
func initializeGeneratorService(cfg *config.Config) (*generator.Service, error) {
	generationClient, err := createCompletionClient(
		cfg.LLM.GeneratorProvider,
		cfg.LLM.GeneratorModel,
		cfg.LLM,
		0,
	)
	if err != nil {
		return nil, err
	}
	log.Printf("✓ Generator client initialized (provider: %s, model: %s)",
		cfg.LLM.GeneratorProvider, cfg.LLM.GeneratorModel)

	service := generator.NewService(
		generationClient,
		cfg.LLM.MaxConcurrent,
	)
	log.Printf("✓ Generator service initialized (max concurrent: %d)", cfg.LLM.MaxConcurrent)

	return service, nil
}

// initializeModerationPipeline initializes the moderation pipeline with all layers
func initializeModerationPipeline(cfg *config.Config) (*moderation.Pipeline, error) {
	// Initialize Prompt Registry
	promptRegistry, err := prompt.NewRegistry(cfg.LLM.PromptsPath)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to load prompt registry from %s: %v. Using fallback prompts.", cfg.LLM.PromptsPath, err)
		promptRegistry = nil
	} else {
		log.Printf("✓ Prompt Registry initialized (path: %s)", cfg.LLM.PromptsPath)
	}

	modCache := moderation.NewInMemoryCache()
	keywordFilter := moderation.NewKeywordFilter()

	var modLayers []moderation.ContentModerator
	modLayers = append(modLayers, keywordFilter)

	hasONNXModel := cfg.LLM.GlobalONNXLibPath != ""
	if hasONNXModel {
		toxicityThreshold := cfg.LLM.ModerationONNXToxicityThreshold
		if toxicityThreshold == 0 {
			toxicityThreshold = 0.90
		}

		spamThreshold := cfg.LLM.ModerationONNXSpamThreshold
		if spamThreshold == 0 {
			spamThreshold = 0.80
		}

		toxicityModelPath := cfg.LLM.ModerationONNXToxicityModelPath
		if toxicityModelPath != "" {
			// Detect if this is the multi-label toxic-bert model
			isMultiLabel := len(cfg.LLM.ModerationONNXThresholds) > 0

			var toxicMod moderation.ContentModerator
			var err error

			if isMultiLabel {
				// Initialize multi-label toxic-bert model with forensic decision matrix
				toxicMod, err = moderation.NewONNXModerator(toxicityModelPath, moderation.ONNXConfig{
					ModelPath:  toxicityModelPath,
					Name:       "toxic-bert",
					Thresholds: cfg.LLM.ModerationONNXThresholds,
				})
				if err == nil {
					modLayers = append(modLayers, toxicMod)
					log.Printf("✓ L3a ONNX Toxic-BERT Multi-Label Moderator initialized (model: %s)", toxicityModelPath)
					log.Printf("  Thresholds: toxic=%.2f, threat=%.2f, insult=%.2f, identity_hate=%.2f, severe_toxic=%.2f, obscene=%.2f",
						cfg.LLM.ModerationONNXThresholds["toxic"],
						cfg.LLM.ModerationONNXThresholds["threat"],
						cfg.LLM.ModerationONNXThresholds["insult"],
						cfg.LLM.ModerationONNXThresholds["identity_hate"],
						cfg.LLM.ModerationONNXThresholds["severe_toxic"],
						cfg.LLM.ModerationONNXThresholds["obscene"])
				} else {
					log.Printf("⚠️ Skipping L3a ONNX Toxic-BERT: %v", err)
				}
			} else {
				// Initialize binary toxicity model (legacy)
				toxicMod, err = moderation.NewONNXModerator(toxicityModelPath, moderation.ONNXConfig{
					ModelPath:     toxicityModelPath,
					Name:          "toxicity",
					Threshold:     toxicityThreshold,
					BadClassIndex: 1,
				})
				if err == nil {
					modLayers = append(modLayers, toxicMod)
					log.Printf("✓ L3a ONNX Toxicity Moderator initialized (model: %s, threshold: %.2f)", toxicityModelPath, toxicityThreshold)
				} else {
					log.Printf("⚠️ Skipping L3a ONNX Toxicity: %v", err)
				}
			}
		}

		if cfg.LLM.ModerationONNXSpamModelPath != "" {
			spamMod, err := moderation.NewONNXModerator(cfg.LLM.ModerationONNXSpamModelPath, moderation.ONNXConfig{
				ModelPath:     cfg.LLM.ModerationONNXSpamModelPath,
				Name:          "spam",
				Threshold:     spamThreshold,
				BadClassIndex: 1,
			})
			if err == nil {
				modLayers = append(modLayers, spamMod)
				log.Printf("✓ L3b ONNX Spam Moderator initialized (model: %s, threshold: %.2f)", cfg.LLM.ModerationONNXSpamModelPath, spamThreshold)
			} else {
				log.Printf("⚠️ Skipping L3b ONNX Spam: %v", err)
			}
		}
	}

	modFallbackClient, err := createCompletionClient(
		cfg.LLM.ModerationFallbackProvider,
		cfg.LLM.ModerationFallbackModel,
		cfg.LLM,
		0,
	)
	if err != nil {
		return nil, err
	}

	analyzerService := analyzer.NewService(
		modFallbackClient,
		promptRegistry,
		cfg.LLM.ModerationFallbackModel,
		analyzer.ModerationThresholds{
			Toxicity:       cfg.LLM.ModerationToxicityThreshold,
			Spam:           cfg.LLM.ModerationSpamThreshold,
			Sexual:         cfg.LLM.ModerationSexualThreshold,
			Violence:       cfg.LLM.ModerationViolenceThreshold,
			Misinformation: cfg.LLM.ModerationMisinformationThreshold,
		},
	)
	// Set variant override for A/B testing if configured
	if cfg.LLM.PromptVariantOverride != "" {
		analyzerService.SetVariantOverride(cfg.LLM.PromptVariantOverride)
		log.Printf("✓ Prompt variant override set: %s", cfg.LLM.PromptVariantOverride)
	}

	l4Mod := moderation.NewLLMAnalyzerAdapter(analyzerService, cfg.LLM.ModerationFallbackModel)
	modLayers = append(modLayers, l4Mod)
	log.Printf("✓ L4 LLM Analyzer initialized (provider: %s, model: %s)",
		cfg.LLM.ModerationFallbackProvider, cfg.LLM.ModerationFallbackModel)

	pipeline := moderation.NewPipeline(modCache, modLayers...)
	log.Printf("✓ Moderation Pipeline initialized (%d layers)", len(modLayers)+1)

	return pipeline, nil
}

// initializeServices initializes all services in the correct order
func initializeServices(ctx context.Context, cfg *config.Config) (*Services, error) {
	log.Println("=== Initializing Domain-Driven LLM Architecture ===")

	// Initialize database connection
	db, err := initializeDatabase(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	log.Printf("✓ PostgreSQL database connected (host: %s, db: %s)",
		cfg.Database.Postgres.Host, cfg.Database.Postgres.Database)

	// Initialize repositories
	tenantRepo := repository.NewPostgresTenantRepository(db)
	appRepo := repository.NewPostgresAppRepository(db)
	log.Printf("✓ Repositories initialized")

	initializeONNX(cfg.LLM.GlobalONNXLibPath)

	weaviateClient, err := initializeWeaviate(ctx, cfg)
	if err != nil {
		return nil, err
	}

	knowledgeService, err := initializeKnowledgeService(cfg, weaviateClient)
	if err != nil {
		return nil, err
	}

	generatorService, err := initializeGeneratorService(cfg)
	if err != nil {
		return nil, err
	}

	modPipeline, err := initializeModerationPipeline(cfg)
	if err != nil {
		return nil, err
	}

	return &Services{
		KnowledgeService: knowledgeService,
		GeneratorService: generatorService,
		ModPipeline:      modPipeline,
		WeaviateClient:   weaviateClient,
		TenantRepo:       tenantRepo,
		AppRepo:          appRepo,
	}, nil
}

// performHealthChecks runs health checks on initialized services
func performHealthChecks(ctx context.Context, services *Services) {
	log.Println("Performing health checks...")
	if err := services.KnowledgeService.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Health check failed: %v", err)
		log.Println("Continuing startup, but some features may not work properly")
	} else {
		log.Println("All health checks passed")
	}
}
