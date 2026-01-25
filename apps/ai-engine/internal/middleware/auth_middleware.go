package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/ai-engine/internal/repository"
	"github.com/qolzam/telar/apps/ai-engine/internal/types"
)

func APIKeyAuth(appRepo repository.AppRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "missing_authorization",
				"message": "Authorization header with Bearer token is required",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "invalid_format",
				"message": "Authorization header must be in format: Bearer <token>",
			})
		}

		apiKey := parts[1]

		// Validate via Repository (database lookup)
		app, err := appRepo.FindAndValidateKey(c.Context(), apiKey)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "invalid_api_key",
				"message": "The provided API key is invalid or expired",
			})
		}

		// Inject Tenant/App Context for downstream handlers
		c.Locals(types.CtxTenantID, app.TenantID)
		c.Locals(types.CtxAppID, app.ID)

		return c.Next()
	}
}
