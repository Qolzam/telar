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
	// Separate middleware instances
	jwtMiddleware := authjwt.New(authjwt.Config{
		PublicKey:   cfg.PublicKey,
		ClaimKey:    "claim",
		UserCtxName: types.UserCtxName,
	})

	hmacMiddleware := authhmac.New(authhmac.Config{
		PayloadSecret: cfg.PayloadSecret,
	})

	// Create dual auth middleware (JWT/Cookie + HMAC)
	return func(c *fiber.Ctx) error {
		authHeader := c.Get(types.HeaderAuthorization)
		hmacHeader := c.Get(types.HeaderHMACAuthenticate)
		sessionCookie := c.Cookies("session")
		
		// Try JWT middleware first (Authorization header OR session cookie)
		if authHeader != "" && strings.HasPrefix(authHeader, types.BearerPrefix) {
			err := jwtMiddleware(c)
			if err == nil {
				return c.Next()
			}
		}

		// Try JWT middleware with session cookie
		if sessionCookie != "" {
			err := jwtMiddleware(c)
			if err == nil {
				return c.Next()
			}
		}

		// Try HMAC middleware as final fallback
		if hmacHeader != "" {
			return hmacMiddleware(c)
		}

		// No valid authentication found
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": "Missing or invalid authentication credentials",
		})
	}
}
