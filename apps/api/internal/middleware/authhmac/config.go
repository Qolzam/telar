package authhmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// Config defines the config for middleware.
type Config struct {
	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool

	// Realm is a string to define realm attribute of BasicAuth.
	// the realm identifies the system to authenticate against
	// and can be used by clients to save credentials
	//
	// Optional. Default: "Restricted".
	Realm string

	// Authorizer defines a function you can pass
	// to check the credentials however you want.
	// It will be called with method, path, query, body, signature, uid, and timestamp
	// and is expected to return nil or error to indicate
	// that the credentials were approved or not.
	//
	// Optional. Default: nil.
	Authorizer func(method, path, query string, body []byte, signature, uid, timestamp string) error

	// Unauthorized defines the response body for unauthorized responses.
	// By default it will return with a 401 Unauthorized and the correct WWW-Auth header
	//
	// Optional. Default: nil
	Unauthorized fiber.Handler

	// PayloadSecret is the key to validate HMAC
	//
	// Optional. Default: "secret"
	PayloadSecret string

	// UserCtxName is the key to store the user context in Locals
	//
	// Optional. Default: "user"
	UserCtxName string
}

// ConfigDefault is the default config
var ConfigDefault = Config{
	Next:          nil,
	Authorizer:    nil,
	Unauthorized:  nil,
	PayloadSecret: "secret",
	UserCtxName:   types.UserCtxName,
}

// Helper function to set default values
func configDefault(config ...Config) Config {
	// Return default config if nothing provided
	if len(config) < 1 {
		return ConfigDefault
	}

	// Override default config
	cfg := config[0]

	// Set default values
	if cfg.Next == nil {
		cfg.Next = ConfigDefault.Next
	}
	if cfg.Authorizer == nil {
		cfg.Authorizer = func(method, path, query string, body []byte, signature, uid, timestamp string) error {
			return validateHMACSignature(method, path, query, body, signature, cfg.PayloadSecret, uid, timestamp)
		}
	}
	if cfg.Unauthorized == nil {
		cfg.Unauthorized = func(c *fiber.Ctx) error {
			c.Set(fiber.HeaderWWWAuthenticate, "HMAC realm="+cfg.Realm)
			return c.SendStatus(fiber.StatusUnauthorized)
		}
	}
	if cfg.PayloadSecret == "" {
		cfg.PayloadSecret = ConfigDefault.PayloadSecret
	}
	if cfg.UserCtxName == "" {
		cfg.UserCtxName = ConfigDefault.UserCtxName
	}
	return cfg
}

// validateHMACSignature validates HMAC signature using SHA256 canonical signing
// Canonical string format: METHOD\nPATH\nCANONICAL_QUERY\nsha256(BODY)\nUID\nTIMESTAMP
func validateHMACSignature(method, path, query string, body []byte, encodedHash, secret, uid, timestamp string) error {
	// 1. Validate required parameters
	if method == "" || path == "" || encodedHash == "" || secret == "" || uid == "" || timestamp == "" {
		return fmt.Errorf("missing required parameters for HMAC validation")
	}

	// 2. Validate timestamp format and window
	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	// 3. Check timestamp window (±5 minutes)
	now := time.Now().Unix()
	timeDiff := now - timestampInt
	if timeDiff > 300 || timeDiff < -300 { // 300 seconds = 5 minutes
		return fmt.Errorf("timestamp outside valid window (±5 minutes): %d seconds difference", timeDiff)
	}

	// 4. Validate signature format
	if !strings.HasPrefix(encodedHash, types.HMACPrefix) {
		return fmt.Errorf("invalid signature format, expected '%s' prefix", types.HMACPrefix)
	}

	signature, err := hex.DecodeString(strings.TrimPrefix(encodedHash, types.HMACPrefix))
	if err != nil {
		return fmt.Errorf("failed to decode hex signature: %w", err)
	}

	// 5. Build canonical string
	// Format: METHOD\nPATH\nCANONICAL_QUERY\nsha256(BODY)\nUID\nTIMESTAMP
	bodyHash := sha256.Sum256(body)
	canonicalString := fmt.Sprintf("%s\n%s\n%s\n%x\n%s\n%s",
		method,
		path,
		query, // Already canonical from URL parsing
		bodyHash,
		uid,
		timestamp,
	)

	// 6. Generate expected HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonicalString))
	expectedMAC := mac.Sum(nil)

	// 7. Constant-time comparison
	if !hmac.Equal(signature, expectedMAC) {
		return fmt.Errorf("HMAC signature validation failed")
	}

	return nil
}
