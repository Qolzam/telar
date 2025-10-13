package authjwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gofrs/uuid"
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
}

// New creates a new middleware handler.
func New(cfg Config) fiber.Handler {
	// Parse the key once on startup.
	ecPublicKey, err := jwt.ParseECPublicKeyFromPEM([]byte(cfg.PublicKey))
	if err != nil {
		panic(fmt.Sprintf("failed to parse EC public key: %v", err))
	}

	return func(c *fiber.Ctx) error {
		// ONLY check Authorization header - no cookie fallback
		authHeader := c.Get(types.HeaderAuthorization)
		if authHeader == "" || !strings.HasPrefix(authHeader, types.BearerPrefix) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Missing or invalid Authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Invalid authorization header format",
			})
		}

		tokenString := parts[1]

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
