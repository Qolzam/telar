package comments

import (
	"github.com/gofiber/fiber/v2"
	dualauth "github.com/qolzam/telar/apps/api/internal/middleware/dualauth"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/comments/handlers"
)

// createDualAuthMiddleware creates dual authentication middleware using the shared helper
// This ensures consistency across all microservices following g-sol23.md specifications
func createDualAuthMiddleware(cfg *RouterConfig) fiber.Handler {
	return dualauth.CreateDualAuthMiddleware(dualauth.Config{
		PayloadSecret: cfg.PayloadSecret,
		PublicKey:     cfg.PublicKey,
	})
}


// CommentsHandlers holds all the handlers this router needs.
type CommentsHandlers struct {
	CommentHandler *handlers.CommentHandler
}

// RouterConfig holds the configuration needed for the router's middleware.
type RouterConfig struct {
	PayloadSecret string
	PublicKey     string
}

// RegisterRoutes is the single entry point for setting up comments routes.
// It implements dual authentication for all routes as per comments.yaml API specification.
func RegisterRoutes(app *fiber.App, handlers *CommentsHandlers, cfg *platformconfig.Config) {
	// Build RouterConfig from platform config
	routerConfig := &RouterConfig{
		PayloadSecret: cfg.HMAC.Secret,
		PublicKey:     cfg.JWT.PublicKey,
	}

	// Create dual auth middleware for all routes (JWT + Cookie + HMAC fallback)
	dualAuthMiddleware := createDualAuthMiddleware(routerConfig)

	group := app.Group("/comments")

	// --- User-Facing Routes: Use DUAL AUTH middleware (JWT + Cookie + HMAC fallback) ---
	// All routes support both HMACAuth and JWTAuth as per comments.yaml API specification
	group.Post("/", dualAuthMiddleware, handlers.CommentHandler.CreateComment)
	group.Put("/", dualAuthMiddleware, handlers.CommentHandler.UpdateComment)
	group.Get("/", dualAuthMiddleware, handlers.CommentHandler.GetCommentsByPost)
	group.Get("/:commentId/replies", dualAuthMiddleware, handlers.CommentHandler.GetReplies)
	group.Put("/score", dualAuthMiddleware, handlers.CommentHandler.IncrementScore)
	group.Put("/profile", dualAuthMiddleware, handlers.CommentHandler.UpdateCommentProfile)
	group.Get("/:commentId", dualAuthMiddleware, handlers.CommentHandler.GetComment)
	group.Delete("/id/:commentId/post/:postId", dualAuthMiddleware, handlers.CommentHandler.DeleteComment)
	group.Delete("/post/:postId", dualAuthMiddleware, handlers.CommentHandler.DeleteCommentsByPost)
}
