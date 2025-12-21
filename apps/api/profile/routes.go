package profile

import (
	"github.com/gofiber/fiber/v2"
	authhmac "github.com/qolzam/telar/apps/api/internal/middleware/authhmac"
	dualauth "github.com/qolzam/telar/apps/api/internal/middleware/dualauth"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

func createDualAuthMiddleware(cfg *RouterConfig) fiber.Handler {
	return dualauth.CreateDualAuthMiddleware(dualauth.Config{
		PayloadSecret: cfg.PayloadSecret,
		PublicKey:     cfg.PublicKey,
	})
}

type ProfileHandlers struct {
	ProfileHandler *ProfileHandler
}

type RouterConfig struct {
	PayloadSecret string
	PublicKey     string
}

func RegisterRoutes(app *fiber.App, handlers *ProfileHandlers, cfg *platformconfig.Config) {
	group := app.Group("/profile")

	routerConfig := &RouterConfig{
		PayloadSecret: cfg.HMAC.Secret,
		PublicKey:     cfg.JWT.PublicKey,
	}

	hmacMiddleware := authhmac.New(authhmac.Config{
		PayloadSecret: routerConfig.PayloadSecret,
	})

	dualAuthMiddleware := createDualAuthMiddleware(routerConfig)

	// Public search endpoint for autocomplete
	group.Get("/search", handlers.ProfileHandler.SearchProfiles)

	// User-facing routes with JWT/Cookie auth
	group.Get("/my", dualAuthMiddleware, handlers.ProfileHandler.ReadMyProfile)
	group.Get("/", dualAuthMiddleware, handlers.ProfileHandler.QueryUserProfile)
	group.Get("/id/:userId", dualAuthMiddleware, handlers.ProfileHandler.ReadProfile)
	group.Get("/social/:name", dualAuthMiddleware, handlers.ProfileHandler.GetBySocialName)
	group.Post("/ids", dualAuthMiddleware, handlers.ProfileHandler.GetProfileByIds)
	group.Put("/", dualAuthMiddleware, handlers.ProfileHandler.UpdateProfile)

	// Service-to-service routes with HMAC auth
	group.Post("/index", hmacMiddleware, handlers.ProfileHandler.InitProfileIndex)
	group.Put("/last-seen", hmacMiddleware, handlers.ProfileHandler.UpdateLastSeen)
	group.Get("/dto/id/:userId", hmacMiddleware, handlers.ProfileHandler.ReadDtoProfile)
	group.Post("/dto", hmacMiddleware, handlers.ProfileHandler.CreateDtoProfile)
	group.Post("/dispatch", hmacMiddleware, handlers.ProfileHandler.DispatchProfiles)
	group.Post("/dto/ids", hmacMiddleware, handlers.ProfileHandler.GetProfileByIds)
	group.Put("/follow/inc/:inc/:userId", hmacMiddleware, handlers.ProfileHandler.IncreaseFollowCount)
	group.Put("/follower/inc/:inc/:userId", hmacMiddleware, handlers.ProfileHandler.IncreaseFollowerCount)
}
