package verification

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// HMACUtil provides HMAC utilities for secure verification flow
type HMACUtil struct {
	secret string
}

// NewHMACUtil creates a new HMAC utility with the given secret
// Phase 1.2: Following dependency injection pattern - accepts any secret (caller's responsibility)
func NewHMACUtil(secret string) *HMACUtil {
	// With dependency injection, we accept whatever secret is provided
	// It's the caller's responsibility to provide a secure secret
	// This allows for testing with any secret while maintaining production security

	return &HMACUtil{secret: secret}
}

// VerificationHMACData represents data used for HMAC verification
type VerificationHMACData struct {
	VerificationId  uuid.UUID
	Code            string
	RemoteIpAddress string
	Timestamp       int64
	UserId          uuid.UUID
}

// GenerateVerificationHMAC generates an HMAC signature for verification data
// Phase 1.2: Enhanced security - prevents tampering with verification parameters
func (h *HMACUtil) GenerateVerificationHMAC(data VerificationHMACData) string {
	// SECURITY: Create canonical string for HMAC with ordered, unambiguous format
	// Format: verificationId|code|remoteIpAddress|timestamp|userId
	// Using pipe separator to prevent collision attacks
	canonicalString := fmt.Sprintf("%s|%s|%s|%d|%s",
		data.VerificationId.String(),
		data.Code,
		data.RemoteIpAddress,
		data.Timestamp,
		data.UserId.String(),
	)

	// SECURITY: Use SHA-256 HMAC (industry standard, FIPS approved)
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write([]byte(canonicalString))

	// Return hex-encoded signature for safe transport in HTTP headers/JSON
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidateVerificationHMAC validates an HMAC signature for verification data
// Phase 1.2: Enhanced timing-attack protection and security validation
func (h *HMACUtil) ValidateVerificationHMAC(data VerificationHMACData, signature string) error {
	// Input validation first (before any crypto operations)
	if signature == "" {
		return fmt.Errorf("HMAC signature cannot be empty")
	}

	// Validate hex encoding (prevent invalid input attacks)
	if _, err := hex.DecodeString(signature); err != nil {
		return fmt.Errorf("invalid HMAC signature format: must be hex-encoded")
	}

	expectedSignature := h.GenerateVerificationHMAC(data)

	// CRITICAL: Use constant-time comparison to prevent timing attacks
	// This prevents attackers from determining correct signatures through timing analysis
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		return fmt.Errorf("HMAC signature validation failed")
	}

	// Validate timestamp (prevent replay attacks) - ENHANCED security checks
	currentTime := time.Now().Unix()
	maxAge := int64(300) // 5 minutes - industry standard for HMAC freshness

	if currentTime-data.Timestamp > maxAge {
		return fmt.Errorf("HMAC signature expired (age: %d seconds, max: %d)", currentTime-data.Timestamp, maxAge)
	}

	if data.Timestamp > currentTime+60 { // Allow 1 minute clock skew for distributed systems
		return fmt.Errorf("HMAC signature from future (clock skew: %d seconds)", data.Timestamp-currentTime)
	}

	return nil
}

// SecureVerificationToken represents a secure verification token with HMAC protection
type SecureVerificationToken struct {
	VerificationId uuid.UUID `json:"verificationId"`
	Signature      string    `json:"signature"`
	Timestamp      int64     `json:"timestamp"`
	ExpiresAt      int64     `json:"expiresAt"`
}

// GenerateSecureVerificationToken creates a secure verification token with HMAC protection
func (h *HMACUtil) GenerateSecureVerificationToken(verificationId, userId uuid.UUID, remoteIP string) *SecureVerificationToken {
	timestamp := time.Now().Unix()
	expiresAt := timestamp + 900 // 15 minutes

	hmacData := VerificationHMACData{
		VerificationId:  verificationId,
		Code:            "", // Code is not included in token for security
		RemoteIpAddress: remoteIP,
		Timestamp:       timestamp,
		UserId:          userId,
	}

	signature := h.GenerateVerificationHMAC(hmacData)

	return &SecureVerificationToken{
		VerificationId: verificationId,
		Signature:      signature,
		Timestamp:      timestamp,
		ExpiresAt:      expiresAt,
	}
}

// ValidateSecureVerificationToken validates a secure verification token
func (h *HMACUtil) ValidateSecureVerificationToken(token *SecureVerificationToken, userId uuid.UUID, remoteIP string) error {
	// Check expiration
	if time.Now().Unix() > token.ExpiresAt {
		return fmt.Errorf("verification token expired")
	}

	hmacData := VerificationHMACData{
		VerificationId:  token.VerificationId,
		Code:            "", // Code is not included in token
		RemoteIpAddress: remoteIP,
		Timestamp:       token.Timestamp,
		UserId:          userId,
	}

	return h.ValidateVerificationHMAC(hmacData, token.Signature)
}

// ParseTimestampFromHeader extracts timestamp from HTTP header
func ParseTimestampFromHeader(header string) (int64, error) {
	if header == "" {
		return 0, fmt.Errorf("timestamp header is empty")
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(header), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid timestamp format: %w", err)
	}

	return timestamp, nil
}
