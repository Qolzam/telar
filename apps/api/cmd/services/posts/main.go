package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/posts"
	"github.com/qolzam/telar/apps/api/posts/handlers"
	postsServices "github.com/qolzam/telar/apps/api/posts/services"
)

func main() {
	cfg, err := platformconfig.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load platform config: %v", err)
	}

	app := fiber.New()

	baseService, err := platform.NewBaseService(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to create base service: %v", err)
	}

	postsService := postsServices.NewPostService(baseService, cfg)

	// Create database indexes on startup
	log.Println("🔧 Creating database indexes for Posts service...")
	indexCtx, indexCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := postsService.CreateIndexes(indexCtx); err != nil {
		indexCancel()
		log.Printf("⚠️  Warning: Failed to create indexes (may already exist): %v", err)
	} else {
		indexCancel()
		log.Println("✅ Posts database indexes created successfully")
	}

	postsHandler := handlers.NewPostHandler(postsService, cfg.JWT, cfg.HMAC)

	postsHandlers := &posts.PostsHandlers{
		PostHandler: postsHandler,
	}

	posts.RegisterRoutes(app, postsHandlers, cfg)

	log.Printf("Starting Posts Service on port 8082")
	log.Fatal(app.Listen(":8082"))
}

