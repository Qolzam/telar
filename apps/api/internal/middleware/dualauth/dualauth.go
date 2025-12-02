package dualauth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	authjwt "github.com/qolzam/telar/apps/api/internal/middleware/authjwt"
	authhmac "github.com/qolzam/telar/apps/api/internal/middleware/authhmac"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// Config holds the configuration needed for dual authentication middleware
type Config struct {
	PayloadSecret string // HMAC secret for S2S authentication
	PublicKey     string // ECDSA public key for JWT validation
}

// CreateDualAuthMiddleware creates dual authentication middleware for JWT + HMAC
// This middleware tries JWT first, then falls back to HMAC authentication
//
// Authentication Flow:
// 1. JWT Authentication (Authorization: Bearer) - for user-facing requests
// 2. HMAC Authentication (X-Telar-Signature) - for S2S communication
//
// Usage:
//   dualAuthMiddleware := dualauth.CreateDualAuthMiddleware(dualauth.Config{
//       PayloadSecret: cfg.PayloadSecret,
//       PublicKey:     cfg.PublicKey,
//   })
//
//   // Apply to user-facing routes that also support S2S
//   group.Post("/", dualAuthMiddleware, handlers.CreateHandler)
func CreateDualAuthMiddleware(cfg Config) fiber.Handler {
	// Create dual auth middleware (JWT/Cookie + HMAC)
	// IMPORTANT: We use validation helpers instead of calling middleware directly
	// to avoid double execution (c.Next() called twice) and response corruption
	return func(c *fiber.Ctx) error {
		authHeader := c.Get(types.HeaderAuthorization)
		hmacHeader := c.Get(types.HeaderHMACAuthenticate)
		// Per blueprint: Cookie name must be "access_token" (strictly enforced)
		accessTokenCookie := c.Cookies("access_token")

		// Strategy A: JWT Authentication (Header or Cookie)
		var tokenString string
		if authHeader != "" && strings.HasPrefix(authHeader, types.BearerPrefix) {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 {
				tokenString = parts[1]
			}
		} else if accessTokenCookie != "" {
			tokenString = accessTokenCookie
		}

		if tokenString != "" {
			// Use validation helper (does NOT write response or call c.Next())
			userCtx, err := authjwt.ValidateToken(tokenString, cfg.PublicKey, "claim", nil)
			if err == nil {
				// Set user context and proceed (call Next ONLY once)
				c.Locals(types.UserCtxName, userCtx)
				return c.Next()
			}
			// If JWT validation fails, do NOT fall through to HMAC
			// An invalid token should fail immediately to prevent confusion
		}

		// Strategy B: HMAC Authentication (S2S)
		if hmacHeader != "" {
			// Use validation helper (does NOT write response or call c.Next())
			userCtx, err := authhmac.ValidateHMAC(c, cfg.PayloadSecret)
			if err == nil {
				// Set user context and proceed (call Next ONLY once)
				c.Locals(types.UserCtxName, userCtx)
				return c.Next()
			}
		}

		// No valid authentication found
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": "Missing or invalid authentication credentials",
		})
	}
}
