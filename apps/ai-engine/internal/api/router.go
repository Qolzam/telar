package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
	"github.com/qolzam/telar/apps/ai-engine/internal/generator"
	"github.com/qolzam/telar/apps/ai-engine/internal/knowledge"
	"github.com/qolzam/telar/apps/ai-engine/internal/middleware"
	"github.com/qolzam/telar/apps/ai-engine/internal/moderation"
	"github.com/qolzam/telar/apps/ai-engine/internal/repository"
)

// Router creates and configures the Fiber application with middleware and routes
func Router(knowledgeService *knowledge.Service, generatorService *generator.Service, modPipeline *moderation.Pipeline, config *config.Config, appRepo repository.AppRepository) *fiber.App {
	handler := NewHandler(knowledgeService, generatorService, modPipeline, config)

	app := fiber.New(fiber.Config{
		AppName: "AI Engine v1.0.0",
	})

	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization",
	}))

	// API routes
	app.Get("/health", handler.Health)
	app.Get("/status", handler.GetStatus)

	// Serve static files from public directory
	app.Static("/", "./public", fiber.Static{
		Index: "index.html",
	})

	v1 := app.Group("/api/v1")
	v1.Post("/ingest", handler.Ingest)
	v1.Post("/query", handler.Query)
	v1.Post("/generate/conversation-starters", handler.GenerateConversationStarters)
	v1.Get("/concurrent-status", handler.GetConcurrentStatus)
	v1.Get("/model-config", handler.GetModelConfig)

	apiKeyAuth := middleware.APIKeyAuth(appRepo)
	v1.Post("/analyze/content", apiKeyAuth, handler.AnalyzeContent)

	return app
}
