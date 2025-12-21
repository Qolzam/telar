package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/comments"
	"github.com/qolzam/telar/apps/api/comments/handlers"
	commentsServices "github.com/qolzam/telar/apps/api/comments/services"
	commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
	postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

func main() {
	cfg, err := platformconfig.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load platform config: %v", err)
	}

	app := fiber.New()

	// Create postgres client for repositories
	ctx := context.Background()
	pgConfig := &dbi.PostgreSQLConfig{
		Host:               cfg.Database.Postgres.Host,
		Port:               cfg.Database.Postgres.Port,
		Username:           cfg.Database.Postgres.Username,
		Password:           cfg.Database.Postgres.Password,
		Database:           cfg.Database.Postgres.Database,
		SSLMode:            cfg.Database.Postgres.SSLMode,
		MaxOpenConnections: cfg.Database.Postgres.MaxOpenConns,
		MaxIdleConnections: cfg.Database.Postgres.MaxIdleConns,
		MaxLifetime:        int(cfg.Database.Postgres.ConnMaxLifetime.Seconds()),
		ConnectTimeout:     10,
	}
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	if err != nil {
		log.Fatalf("Failed to create postgres client: %v", err)
	}

	// Initialize repositories
	commentRepo := commentRepository.NewPostgresCommentRepository(pgClient)
	postRepo := postsRepository.NewPostgresRepository(pgClient)

	// Initialize services
	commentsService := commentsServices.NewCommentService(commentRepo, postRepo, cfg, nil) // nil for postStatsUpdater for now

	commentsHandler := handlers.NewCommentHandler(commentsService, cfg.JWT, cfg.HMAC)

	commentsHandlers := &comments.CommentsHandlers{
		CommentHandler: commentsHandler,
	}

	comments.RegisterRoutes(app, commentsHandlers, cfg)

	log.Printf("Starting Comments Service on port 8083")
	log.Fatal(app.Listen(":8083"))
}

