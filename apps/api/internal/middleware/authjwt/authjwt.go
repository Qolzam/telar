package authjwt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/cache"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// Config defines the config for the JWT middleware.
type Config struct {
	// The EC public key for validating ES256 tokens.
	PublicKey string
	// The claim key where the UserContext is stored.
	ClaimKey string
	// The context key to store the UserContext.
	UserCtxName string
	// JWKS URL for key fetching (optional, fallback to PublicKey)
	JWKSUrl string
	// Expected Key ID (optional)
	KeyID string
	// Optional cache service for session allowlisting
	CacheService *cache.GenericCacheService
}

// New creates a new middleware handler.
func New(cfg Config) fiber.Handler {
	// Parse the key once on startup.
	ecPublicKey, err := jwt.ParseECPublicKeyFromPEM([]byte(cfg.PublicKey))
	if err != nil {
		panic(fmt.Sprintf("failed to parse EC public key: %v", err))
	}

	// Use only the provided cache instance; do NOT auto-create one here
	var sessionCache *cache.GenericCacheService
	if cfg.CacheService != nil && cfg.CacheService.IsEnabled() {
		sessionCache = cfg.CacheService
	}

	return func(c *fiber.Ctx) error {
		var tokenString string

		// 1. Try Authorization header first (for mobile/API clients)
		authHeader := c.Get(types.HeaderAuthorization)
		if authHeader != "" && strings.HasPrefix(authHeader, types.BearerPrefix) {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 {
				tokenString = parts[1]
			}
		}

		// 2. Fall back to session cookie (for web browsers)
		if tokenString == "" {
			tokenString = c.Cookies("session")
		}

		// 3. If no token found in either place, return error
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Missing or invalid JWT",
			})
		}

		// 4. Continue with existing JWT validation
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// CRITICAL: Enforce the expected signing algorithm.
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return ecPublicKey, nil
		})

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Invalid token",
				"details": err.Error(),
			})
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Check if token is expired
			if exp, ok := claims["exp"].(float64); ok {
				if int64(exp) < time.Now().Unix() {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"code":    "UNAUTHORIZED",
						"message": "Token has expired",
					})
				}
			}

			// Extract the claim data
			claimData, claimOk := claims[cfg.ClaimKey].(map[string]interface{})
			if !claimOk {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Invalid token claim format",
				})
			}

			// Optional session allowlist check via cache
			if sessionCache != nil {
				jtiStr, _ := claims["jti"].(string)
				if jtiStr == "" {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"code":    "UNAUTHORIZED",
						"message": "Missing session ID",
					})
				}
				uidStr, _ := claimData[types.HeaderUID].(string)
				if uidStr == "" {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"code":    "UNAUTHORIZED",
						"message": "Missing user ID",
					})
				}
				key := sessionCache.GenerateHashKey("sessions", map[string]interface{}{"uid": uidStr})
				isMember, err := sessionCache.SetIsMember(context.Background(), key, jtiStr)
				if err != nil {
					// Fail-closed: deny access on cache check error
					log.Warn("CRITICAL: Redis session check failed for user %s: %v", uidStr, err)
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"code":    "UNAUTHORIZED",
						"message": "Session validation failed. Please log in again.",
					})
				}
				if !isMember {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"code":    "UNAUTHORIZED",
						"message": "Session has been invalidated.",
					})
				}
			}

			// Map claim data to UserContext
			userCtx, err := mapToUserContext(claimData)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Invalid user context in token",
					"details": err.Error(),
				})
			}

			c.Locals(cfg.UserCtxName, userCtx)
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": "Invalid token",
		})
	}
}

// mapToUserContext converts claim data to UserContext
func mapToUserContext(claimData map[string]interface{}) (types.UserContext, error) {
	var userCtx types.UserContext

	// Extract user ID
	if userIDStr, ok := claimData[types.HeaderUID].(string); ok {
		userID, err := uuid.FromString(userIDStr)
		if err != nil {
			return userCtx, fmt.Errorf("invalid user ID: %v", err)
		}
		userCtx.UserID = userID
	} else {
		return userCtx, errors.New("missing or invalid uid in claim")
	}

	// Extract username/email
	if username, ok := claimData["username"].(string); ok {
		userCtx.Username = username
	}

	// Extract display name
	if displayName, ok := claimData["displayName"].(string); ok {
		userCtx.DisplayName = displayName
	}

	// Extract avatar
	if avatar, ok := claimData["avatar"].(string); ok {
		userCtx.Avatar = avatar
	}

	// Extract system role
	if systemRole, ok := claimData["role"].(string); ok {
		userCtx.SystemRole = systemRole
	}

	// Extract social name
	if socialName, ok := claimData["socialName"].(string); ok {
		userCtx.SocialName = socialName
	}

	// Extract banner
	if banner, ok := claimData["banner"].(string); ok {
		userCtx.Banner = banner
	}

	// Extract tag line
	if tagLine, ok := claimData["tagLine"].(string); ok {
		userCtx.TagLine = tagLine
	}

	// Extract created date
	if createdDate, ok := claimData["createdDate"].(float64); ok {
		userCtx.CreatedDate = int64(createdDate)
	}

	return userCtx, nil
}
