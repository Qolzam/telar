package bookmarks

import (
	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/bookmarks/handlers"
	dualauth "github.com/qolzam/telar/apps/api/internal/middleware/dualauth"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

type Handlers struct {
	BookmarkHandler *handlers.BookmarkHandler
}

type RouterConfig struct {
	PayloadSecret string
	PublicKey     string
}

func createDualAuthMiddleware(cfg *RouterConfig) fiber.Handler {
	return dualauth.CreateDualAuthMiddleware(dualauth.Config{
		PayloadSecret: cfg.PayloadSecret,
		PublicKey:     cfg.PublicKey,
	})
}

// RegisterRoutes wires bookmark endpoints.
func RegisterRoutes(app *fiber.App, handlers *Handlers, cfg *platformconfig.Config) {
	routerCfg := &RouterConfig{
		PayloadSecret: cfg.HMAC.Secret,
		PublicKey:     cfg.JWT.PublicKey,
	}

	dualAuthMiddleware := createDualAuthMiddleware(routerCfg)

	group := app.Group("/bookmarks")
	userGroup := group.Group("", dualAuthMiddleware)

	userGroup.Post("/:postId/toggle", handlers.BookmarkHandler.Toggle)
	userGroup.Get("/", handlers.BookmarkHandler.List)
}
