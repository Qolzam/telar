package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/comments"
	"github.com/qolzam/telar/apps/api/comments/handlers"
	commentsServices "github.com/qolzam/telar/apps/api/comments/services"
	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
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

	commentsService := commentsServices.NewCommentService(baseService, cfg)

	// Create database indexes on startup
	log.Println("🔧 Creating database indexes for Comments service...")
	indexCtx, indexCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := commentsService.CreateIndexes(indexCtx); err != nil {
		indexCancel()
		log.Printf("⚠️  Warning: Failed to create indexes (may already exist): %v", err)
	} else {
		indexCancel()
		log.Println("✅ Comments database indexes created successfully")
	}

	commentsHandler := handlers.NewCommentHandler(commentsService, cfg.JWT, cfg.HMAC)

	commentsHandlers := &comments.CommentsHandlers{
		CommentHandler: commentsHandler,
	}

	comments.RegisterRoutes(app, commentsHandlers, cfg)

	log.Printf("Starting Comments Service on port 8083")
	log.Fatal(app.Listen(":8083"))
}

