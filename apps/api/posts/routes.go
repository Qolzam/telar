package posts

import (
	"github.com/gofiber/fiber/v2"
	authhmac "github.com/qolzam/telar/apps/api/internal/middleware/authhmac"
	constraints "github.com/qolzam/telar/apps/api/internal/middleware/constraints"
	dualauth "github.com/qolzam/telar/apps/api/internal/middleware/dualauth"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/posts/handlers"
)

// createDualAuthMiddleware creates dual authentication middleware using the shared helper
// This ensures consistency across all microservices following g-sol23.md specifications
func createDualAuthMiddleware(cfg *RouterConfig) fiber.Handler {
	return dualauth.CreateDualAuthMiddleware(dualauth.Config{
		PayloadSecret: cfg.PayloadSecret,
		PublicKey:     cfg.PublicKey,
	})
}

// PostsHandlers holds all the handlers this router needs.
type PostsHandlers struct {
	PostHandler *handlers.PostHandler
}

// RouterConfig holds the configuration needed for the router's middleware.
type RouterConfig struct {
	PayloadSecret string
	PublicKey     string
}

// RegisterRoutes is the single entry point for setting up posts routes.
// It implements selective authentication: JWT for user-facing routes, HMAC for S2S routes.
func RegisterRoutes(app *fiber.App, handlers *PostsHandlers, cfg *platformconfig.Config) {
	// Build RouterConfig from platform config
	routerConfig := &RouterConfig{
		PayloadSecret: cfg.HMAC.Secret,
		PublicKey:     cfg.JWT.PublicKey,
	}

	// Middleware setup using the injected configuration
	hmacMiddleware := authhmac.New(authhmac.Config{
		PayloadSecret: routerConfig.PayloadSecret,
	})

	// Create dual auth middleware for user-facing routes during migration
	dualAuthMiddleware := createDualAuthMiddleware(routerConfig)

	group := app.Group("/posts")

	// --- Service-to-Service Routes (HMAC-Only) ---
	// These are actions on the collection, so we group them.
	s2sActions := group.Group("/actions", hmacMiddleware)
	s2sActions.Put("/score", handlers.PostHandler.IncrementScore)
	s2sActions.Put("/comment/count", handlers.PostHandler.IncrementCommentCount)

	// Public search endpoint for autocomplete
	group.Get("/search", handlers.PostHandler.SearchPosts)

	// --- User-Facing Routes (Dual Auth) ---
	userGroup := group.Group("", dualAuthMiddleware)

	// Base resource routes
	userGroup.Post("/", handlers.PostHandler.CreatePost)
	userGroup.Put("/", handlers.PostHandler.UpdatePost)
	userGroup.Put("/profile", handlers.PostHandler.UpdatePostProfile)

	// Sub-resource routes
	userGroup.Put("/comment/disable", handlers.PostHandler.DisableComment)
	userGroup.Put("/share/disable", handlers.PostHandler.DisableSharing)
	userGroup.Put("/urlkey/:postId", handlers.PostHandler.GeneratePostURLKey)

	// Base query route (backward compatibility)
	userGroup.Get("/", handlers.PostHandler.QueryPosts) // GET /posts/

	// --- Query Sub-Group for Collection-Level Queries ---
	// This completely disambiguates cursor query routes from specific resource routes.
	// Cursor-based queries go here to avoid route conflicts with /:postId
	queryGroup := userGroup.Group("/queries")
	queryGroup.Get("/cursor", handlers.PostHandler.QueryPostsWithCursor)
	queryGroup.Get("/search/cursor", handlers.PostHandler.SearchPostsWithCursor)

	// --- Parameterized Routes for Specific Resources (MUST BE LAST) ---
	// These routes operate on a single post, identified by a parameter.
	userGroup.Get("/urlkey/:urlkey", handlers.PostHandler.GetPostByURLKey)

	// The constraint is still a good practice for type safety and explicit validation.
	userGroup.Get("/cursor/info/:postId", constraints.RequireUUID("postId"), handlers.PostHandler.GetCursorInfo)
	userGroup.Get("/:postId", constraints.RequireUUID("postId"), handlers.PostHandler.GetPost)
	userGroup.Delete("/:postId", constraints.RequireUUID("postId"), handlers.PostHandler.DeletePost)
}
