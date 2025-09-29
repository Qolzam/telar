// Package ratelimit provides rate limiting middleware for authentication endpoints
// Following the AUTH_SECURITY_REFACTORING_PLAN.md Phase 2.1 implementation
package ratelimit

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
)

// EndpointLimits defines rate limiting configuration for specific endpoints
type EndpointLimits struct {
	// Login attempts: 5 per 15 minutes per IP
	LoginMaxRequests    int
	LoginWindowDuration time.Duration

	// Password reset: 3 per hour per IP
	PasswordResetMaxRequests    int
	PasswordResetWindowDuration time.Duration

	// Signup: 10 per hour per IP
	SignupMaxRequests    int
	SignupWindowDuration time.Duration

	// Verification: 10 attempts per verification ID
	VerificationMaxRequests    int
	VerificationWindowDuration time.Duration
}

// DefaultEndpointLimits returns the secure default rate limits per the security plan
func DefaultEndpointLimits() EndpointLimits {
	return EndpointLimits{
		// Login rate limits
		LoginMaxRequests:    5,
		LoginWindowDuration: 15 * time.Minute,

		// Password reset rate limits
		PasswordResetMaxRequests:    3,
		PasswordResetWindowDuration: 1 * time.Hour,

		// Signup rate limits
		SignupMaxRequests:    10,
		SignupWindowDuration: 1 * time.Hour,

		// Verification rate limits
		VerificationMaxRequests:    10,
		VerificationWindowDuration: 15 * time.Minute,
	}
}

// EndpointType represents different authentication endpoints for rate limiting
type EndpointType int

const (
	EndpointLogin EndpointType = iota
	EndpointPasswordReset
	EndpointSignup
	EndpointVerification
)

// Config holds the configuration for rate limiting middleware
type Config struct {
	// Endpoint type to determine which limits to apply
	EndpointType EndpointType

	// Custom limits (optional - uses defaults if not provided)
	Limits *EndpointLimits

	// Next defines a function to skip this middleware when returned true
	Next func(c *fiber.Ctx) bool

	// Custom key generator (optional - uses default IP-based if not provided)
	KeyGenerator func(c *fiber.Ctx) string

	// LimitReached defines the response when rate limit is exceeded
	LimitReached func(c *fiber.Ctx) error
}

// configDefault sets default configuration values
func configDefault(config Config) Config {
	// Set default limits if not provided
	if config.Limits == nil {
		limits := DefaultEndpointLimits()
		config.Limits = &limits
	}

	// Set default key generator (rate limit by IP + endpoint path)
	if config.KeyGenerator == nil {
		config.KeyGenerator = func(c *fiber.Ctx) string {
			return c.IP() + ":" + c.Path()
		}
	}

	// Set default limit reached handler
	if config.LimitReached == nil {
		config.LimitReached = func(c *fiber.Ctx) error {
			endpointName := getEndpointName(config.EndpointType)
			windowDuration := getWindowDuration(config.EndpointType, config.Limits)

			log.Warn("[RateLimit] Rate limit exceeded for %s from IP: %s", endpointName, c.IP())

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":     "Rate limit exceeded",
				"code":      "RATE_LIMIT_EXCEEDED",
				"message":   fmt.Sprintf("Too many %s attempts. Please try again later.", endpointName),
				"retryAfter": int(windowDuration.Seconds()),
			})
		}
	}

	return config
}

// getEndpointName returns human-readable endpoint name for logging
func getEndpointName(endpointType EndpointType) string {
	switch endpointType {
	case EndpointLogin:
		return "login"
	case EndpointPasswordReset:
		return "password reset"
	case EndpointSignup:
		return "signup"
	case EndpointVerification:
		return "verification"
	default:
		return "unknown"
	}
}

// getMaxRequests returns the max requests for the endpoint type
func getMaxRequests(endpointType EndpointType, limits *EndpointLimits) int {
	switch endpointType {
	case EndpointLogin:
		return limits.LoginMaxRequests
	case EndpointPasswordReset:
		return limits.PasswordResetMaxRequests
	case EndpointSignup:
		return limits.SignupMaxRequests
	case EndpointVerification:
		return limits.VerificationMaxRequests
	default:
		return 5 // Conservative default
	}
}

// getWindowDuration returns the window duration for the endpoint type
func getWindowDuration(endpointType EndpointType, limits *EndpointLimits) time.Duration {
	switch endpointType {
	case EndpointLogin:
		return limits.LoginWindowDuration
	case EndpointPasswordReset:
		return limits.PasswordResetWindowDuration
	case EndpointSignup:
		return limits.SignupWindowDuration
	case EndpointVerification:
		return limits.VerificationWindowDuration
	default:
		return 15 * time.Minute // Conservative default
	}
}

// New creates a new rate limiting middleware handler
func New(config Config) fiber.Handler {
	// Apply default configuration
	cfg := configDefault(config)

	// Get limits for this endpoint type
	maxRequests := getMaxRequests(cfg.EndpointType, cfg.Limits)
	windowDuration := getWindowDuration(cfg.EndpointType, cfg.Limits)

	// Create limiter configuration
	limiterConfig := limiter.Config{
		Max:          maxRequests,
		Expiration:   windowDuration,
		KeyGenerator: cfg.KeyGenerator,
		LimitReached: cfg.LimitReached,
		Next:         cfg.Next,
	}

	// Create and return the limiter middleware
	return limiter.New(limiterConfig)
}

// NewLoginLimiter creates a rate limiter specifically for login endpoints
func NewLoginLimiter(customLimits *EndpointLimits) fiber.Handler {
	return New(Config{
		EndpointType: EndpointLogin,
		Limits:       customLimits,
	})
}

// NewPasswordResetLimiter creates a rate limiter specifically for password reset endpoints
func NewPasswordResetLimiter(customLimits *EndpointLimits) fiber.Handler {
	return New(Config{
		EndpointType: EndpointPasswordReset,
		Limits:       customLimits,
	})
}

// NewSignupLimiter creates a rate limiter specifically for signup endpoints
func NewSignupLimiter(customLimits *EndpointLimits) fiber.Handler {
	return New(Config{
		EndpointType: EndpointSignup,
		Limits:       customLimits,
	})
}

// NewVerificationLimiter creates a rate limiter specifically for verification endpoints
func NewVerificationLimiter(customLimits *EndpointLimits) fiber.Handler {
	return New(Config{
		EndpointType: EndpointVerification,
		Limits:       customLimits,
	})
}

// NewVerificationByIDLimiter creates a rate limiter for verification attempts by verification ID
// This provides additional protection against brute force attacks on specific verification codes
func NewVerificationByIDLimiter(customLimits *EndpointLimits) fiber.Handler {
	return New(Config{
		EndpointType: EndpointVerification,
		Limits:       customLimits,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Extract verification ID from request for targeted rate limiting
			var verifyID string
			
			// Try to get verification ID from different sources
			if c.Method() == "POST" {
				// Parse request body to get verification ID
				var requestBody map[string]interface{}
				if err := c.BodyParser(&requestBody); err == nil {
					if id, ok := requestBody["verificationId"].(string); ok {
						verifyID = id
					}
				}
			}
			
			// Fallback to IP-based if no verification ID found
			if verifyID == "" {
				verifyID = c.IP()
			}
			
			return fmt.Sprintf("verify:%s", verifyID)
		},
	})
}