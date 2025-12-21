package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	bookmarksRepository "github.com/qolzam/telar/apps/api/bookmarks/repository"
	commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/posts"
	"github.com/qolzam/telar/apps/api/posts/handlers"
	postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
	postsServices "github.com/qolzam/telar/apps/api/posts/services"
	votesRepository "github.com/qolzam/telar/apps/api/votes/repository"
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

	// Create repositories
	postRepo := postsRepository.NewPostgresRepository(pgClient)
	voteRepo := votesRepository.NewPostgresVoteRepository(pgClient)
	commentRepo := commentRepository.NewPostgresCommentRepository(pgClient)
	bookmarkRepo := bookmarksRepository.NewPostgresRepository(pgClient)

	// Create post service with repository
	postsService := postsServices.NewPostService(postRepo, voteRepo, bookmarkRepo, cfg, nil, commentRepo)

	postsHandler := handlers.NewPostHandler(postsService, cfg.JWT, cfg.HMAC)

	postsHandlers := &posts.PostsHandlers{
		PostHandler: postsHandler,
	}

	posts.RegisterRoutes(app, postsHandlers, cfg)

	log.Printf("Starting Posts Service on port 8082")
	log.Fatal(app.Listen(":8082"))
}
