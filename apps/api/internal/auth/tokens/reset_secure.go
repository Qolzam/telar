package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// ResetTokenConfig holds configuration for reset token generation
type ResetTokenConfig struct {
	TokenLength int           // Length in bytes (default: 32)
	TTL         time.Duration // Time to live (default: 1 hour)
}

// ResetTokenData holds the generated reset token information
type ResetTokenData struct {
	VerificationId uuid.UUID
	HashedToken    string
	PlaintextToken string // Return for email sending
	UserEmail      string
	ExpiresAt      int64
	Used           bool
}

// GenerateSecureResetToken generates a high-entropy opaque reset token
func GenerateSecureResetToken(userEmail string, config ResetTokenConfig) (*ResetTokenData, error) {
	// Set defaults
	if config.TokenLength == 0 {
		config.TokenLength = 32 // 32 bytes = 256 bits of entropy
	}
	if config.TTL == 0 {
		config.TTL = time.Hour // 1 hour default
	}

	// 1. Generate high-entropy random token
	tokenBytes := make([]byte, config.TokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	// 2. Encode as URL-safe base64
	plaintextToken := base64.URLEncoding.EncodeToString(tokenBytes)

	// 3. Hash the token for storage (SHA256)
	hash := sha256.Sum256([]byte(plaintextToken))
	hashedToken := fmt.Sprintf("%x", hash)

	// 4. Generate verification ID
	verifyId := uuid.Must(uuid.NewV4())

	// 5. Calculate expiry
	expiresAt := time.Now().Add(config.TTL).Unix()

	return &ResetTokenData{
		VerificationId: verifyId,
		HashedToken:    hashedToken,
		PlaintextToken: plaintextToken, // Return for email sending
		UserEmail:      userEmail,
		ExpiresAt:      expiresAt,
		Used:           false,
	}, nil
}

// ValidateResetToken validates a plaintext reset token against stored hash
func ValidateResetToken(plaintextToken, storedHash string) bool {
	// Hash the provided token
	hash := sha256.Sum256([]byte(plaintextToken))
	computedHash := fmt.Sprintf("%x", hash)
	
	// Constant-time comparison
	return computedHash == storedHash
}
