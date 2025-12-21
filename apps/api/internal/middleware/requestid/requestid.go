package requestid

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
)

const (
	// HeaderRequestID is the HTTP header name for request ID
	HeaderRequestID = "X-Request-ID"
	// ContextKeyRequestID is the key used to store request ID in Fiber context
	ContextKeyRequestID = "request_id"
)

// New creates a middleware that generates or uses an existing X-Request-ID header
func New() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if request ID already exists in header
		requestID := c.Get(HeaderRequestID)
		
		// If not present, generate a new UUID
		if requestID == "" {
			id, err := uuid.NewV4()
			if err != nil {
				// Fallback: generate another UUID (should never fail)
				id, _ = uuid.NewV4()
			}
			requestID = id.String()
		}
		
		// Store in context for use by handlers and logger
		c.Locals(ContextKeyRequestID, requestID)
		
		// Set response header so client can track the request
		c.Set(HeaderRequestID, requestID)
		
		return c.Next()
	}
}

// GetRequestID retrieves the request ID from Fiber context
func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals(ContextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

