package authhmac

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// New creates a new middleware handler
func New(config Config) fiber.Handler {
	// Set default config
	cfg := configDefault(config)

	// Return new handler
	return func(c *fiber.Ctx) error {
		// Don't execute middleware if Next returns true
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		// Get required headers
		auth := c.Get(types.HeaderHMACAuthenticate)
		uid := c.Get(types.HeaderUID)
		timestamp := c.Get(types.HeaderTimestamp)

		// CRITICAL: Enforce all required headers
		if len(auth) == 0 {
			log.Error("Unauthorized! HMAC signature not provided!")
			return cfg.Unauthorized(c)
		}

		if uid == "" {
			log.Error("Unauthorized! uid header is required for HMAC authentication!")
			return cfg.Unauthorized(c)
		}

		if timestamp == "" {
			log.Error("Unauthorized! X-Timestamp header is required for HMAC authentication!")
			return cfg.Unauthorized(c)
		}

		// Extract request details for canonical signing
		method := c.Method()
		path := c.Path()
		query := string(c.Context().URI().QueryString()) // Get raw query string
		body := c.Body()


		// Validate HMAC with canonical signing
		if err := cfg.Authorizer(method, path, query, body, auth, uid, timestamp); err != nil {
			log.Error("HMAC validation failed: %v", err)
			return cfg.Unauthorized(c)
		}

		// Parse and validate uid
		userUUID, userUuidErr := uuid.FromString(uid)
		if userUuidErr != nil {
			log.Error("Invalid uid format: %v", userUuidErr)
			return cfg.Unauthorized(c)
		}

		// Parse timestamp for context
		var createdDate int64 = 0
		if timestampInt, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
			createdDate = timestampInt
		}

		// Set user context with validated uid from signed message
		c.Locals(cfg.UserCtxName, types.UserContext{
			UserID:      userUUID,
			Username:    c.Get("username"),
			DisplayName: c.Get("displayName"),
			SocialName:  c.Get("socialName"),
			SystemRole:  c.Get("systemRole"),
			CreatedDate: createdDate,
		})

		return c.Next()
	}
}
