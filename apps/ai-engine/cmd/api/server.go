package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/ai-engine/internal/api"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
)

// startServer starts the HTTP server and returns the app instance
func startServer(services *Services, cfg *config.Config) *fiber.App {
	app := api.Router(services.KnowledgeService, services.GeneratorService, services.ModPipeline, cfg, services.AppRepo)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting %s %s on %s", serviceName, serviceVersion, addr)
	log.Printf("Architecture: Domain-Driven (Knowledge: %s, Generator: %s, Moderation: %s)",
		cfg.LLM.KnowledgeEmbeddingProvider, cfg.LLM.GeneratorProvider, cfg.LLM.ModerationFallbackProvider)
	log.Printf("Weaviate URL: %s", cfg.Weaviate.URL)

	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	return app
}

// waitForShutdown waits for shutdown signals and performs graceful shutdown
func waitForShutdown(app *fiber.App, cancel context.CancelFunc) {
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
