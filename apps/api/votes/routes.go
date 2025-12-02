// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package votes

import (
	"github.com/gofiber/fiber/v2"
	dualauth "github.com/qolzam/telar/apps/api/internal/middleware/dualauth"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/votes/handlers"
)

// VotesHandlers holds all the handlers this router needs
type VotesHandlers struct {
	VoteHandler *handlers.VoteHandler
}

// RouterConfig holds the configuration needed for the router's middleware
type RouterConfig struct {
	PayloadSecret string
	PublicKey     string
}

// createDualAuthMiddleware creates dual authentication middleware using the shared helper
func createDualAuthMiddleware(cfg *RouterConfig) fiber.Handler {
	return dualauth.CreateDualAuthMiddleware(dualauth.Config{
		PayloadSecret: cfg.PayloadSecret,
		PublicKey:     cfg.PublicKey,
	})
}

// RegisterRoutes is the single entry point for setting up votes routes
func RegisterRoutes(app *fiber.App, handlers *VotesHandlers, cfg *platformconfig.Config) {
	// Build RouterConfig from platform config
	routerConfig := &RouterConfig{
		PayloadSecret: cfg.HMAC.Secret,
		PublicKey:     cfg.JWT.PublicKey,
	}

	// Create dual auth middleware for user-facing routes
	dualAuthMiddleware := createDualAuthMiddleware(routerConfig)

	group := app.Group("/votes")

	// --- User-Facing Routes (Dual Auth) ---
	userGroup := group.Group("", dualAuthMiddleware)

	// Vote endpoint: POST /votes
	userGroup.Post("/", handlers.VoteHandler.Vote)
}

