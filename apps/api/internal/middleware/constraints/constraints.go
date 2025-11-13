package constraints

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
)

// RequireUUID is a Fiber middleware that ensures a path parameter is a valid UUID.
// If the parameter is not a valid UUID, it returns 404 Not Found (route doesn't match).
// This effectively acts as a route constraint by preventing the handler from being called.
// 
// IMPORTANT: This middleware should only be applied to routes with UUID parameters.
// Static routes like /cursor must be registered BEFORE parameterized routes like /:postId
// to ensure correct route matching precedence.
func RequireUUID(param string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		paramValue := c.Params(param)
		if paramValue == "" {
			// No parameter value, continue (might be optional)
			return c.Next()
		}
		if _, err := uuid.FromString(paramValue); err != nil {
			// Not a UUID, return 404 to indicate this route doesn't match
			// CRITICAL: Use SendStatus to ensure execution stops and response is sent
			// This prevents the handler from executing when UUID is invalid
			return c.SendStatus(fiber.StatusNotFound)
		}
		// It is a UUID, continue to next handler
		return c.Next()
	}
}

