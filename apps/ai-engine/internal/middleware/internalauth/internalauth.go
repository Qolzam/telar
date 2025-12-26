package internalauth

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"
)

const HeaderInternalAPIKey = "X-Internal-API-Key"

// New creates middleware that validates the internal API key for inter-service communication
func New(apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If no API key is configured, skip authentication (for development)
		if apiKey == "" {
			return c.Next()
		}

		providedKey := c.Get(HeaderInternalAPIKey)
		if providedKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Missing X-Internal-API-Key header",
			})
		}

		// Use constant-time comparison to prevent timing attacks
		// Even for internal services, cryptographic best practices should be followed
		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(apiKey)) != 1 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "Forbidden",
				"message": "Invalid API key",
			})
		}

		return c.Next()
	}
}


