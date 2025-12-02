package authhmac

import (
	"fmt"
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

// ValidateHMAC validates HMAC authentication and returns the UserContext if valid.
// This is a pure validation function that does NOT write to the response.
// It can be used by other middleware (like dualauth) to validate HMAC without side effects.
func ValidateHMAC(c *fiber.Ctx, payloadSecret string) (types.UserContext, error) {
	var userCtx types.UserContext

	// Get required headers
	auth := c.Get(types.HeaderHMACAuthenticate)
	uid := c.Get(types.HeaderUID)
	timestamp := c.Get(types.HeaderTimestamp)

	// CRITICAL: Enforce all required headers
	if len(auth) == 0 {
		return userCtx, fmt.Errorf("HMAC signature not provided")
	}

	if uid == "" {
		return userCtx, fmt.Errorf("uid header is required for HMAC authentication")
	}

	if timestamp == "" {
		return userCtx, fmt.Errorf("X-Timestamp header is required for HMAC authentication")
	}

	// Extract request details for canonical signing
	method := c.Method()
	path := c.Path()
	query := string(c.Context().URI().QueryString())
	body := c.Body()

	// Validate HMAC with canonical signing
	// Use the validation function from config.go (same package)
	if err := validateHMACSignature(method, path, query, body, auth, payloadSecret, uid, timestamp); err != nil {
		return userCtx, fmt.Errorf("HMAC validation failed: %w", err)
	}

	// Parse and validate uid
	userUUID, userUuidErr := uuid.FromString(uid)
	if userUuidErr != nil {
		return userCtx, fmt.Errorf("invalid uid format: %w", userUuidErr)
	}

	// Parse timestamp for context
	var createdDate int64 = 0
	if timestampInt, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		createdDate = timestampInt
	}

	// Return user context with validated uid from signed message
	userCtx = types.UserContext{
		UserID:      userUUID,
		Username:    c.Get("username"),
		DisplayName: c.Get("displayName"),
		SocialName:  c.Get("socialName"),
		SystemRole:  c.Get("systemRole"),
		CreatedDate: createdDate,
	}

	return userCtx, nil
}
