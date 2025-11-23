package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/internal/types"
)

type Config struct {
	UserCtxName string
	// Optional override to check custom permission instead of strict role
	HasAccess func(u types.UserContext) bool
}

func New(config Config) fiber.Handler {
	userKey := config.UserCtxName
	if userKey == "" {
		userKey = types.UserCtxName
	}
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals(userKey).(types.UserContext)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "missing user context",
			})
		}
		// Custom access hook if provided
		if config.HasAccess != nil {
			if !config.HasAccess(user) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"code":    "FORBIDDEN",
					"message": "admin access required",
				})
			}
			return c.Next()
		}
		// Default: require system role 'admin'
		if user.SystemRole != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code":    "FORBIDDEN",
				"message": "admin access required",
			})
		}
		return c.Next()
	}
}






