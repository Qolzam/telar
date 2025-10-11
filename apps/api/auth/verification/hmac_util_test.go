package verification

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestHMACUtil_VerificationHMAC(t *testing.T) {
	hmacUtil := NewHMACUtil("test-secret")

	// Test HMAC generation and validation
	data := VerificationHMACData{
		VerificationId:  uuid.Must(uuid.NewV4()),
		Code:            "123456",
		RemoteIpAddress: "192.168.1.1",
		Timestamp:       time.Now().Unix(),
		UserId:          uuid.Must(uuid.NewV4()),
	}

	// Generate HMAC
	signature := hmacUtil.GenerateVerificationHMAC(data)
	if signature == "" {
		t.Error("Expected non-empty HMAC signature")
	}

	// Validate HMAC
	err := hmacUtil.ValidateVerificationHMAC(data, signature)
	if err != nil {
		t.Errorf("HMAC validation failed: %v", err)
	}

	// Test with invalid signature
	err = hmacUtil.ValidateVerificationHMAC(data, "invalid-signature")
	if err == nil {
		t.Error("Expected HMAC validation to fail with invalid signature")
	}

	// Test with expired timestamp
	oldData := data
	oldData.Timestamp = time.Now().Unix() - 400 // 400 seconds ago (> 300 second limit)
	oldSignature := hmacUtil.GenerateVerificationHMAC(oldData)
	err = hmacUtil.ValidateVerificationHMAC(oldData, oldSignature)
	if err == nil {
		t.Error("Expected HMAC validation to fail with expired timestamp")
	}
}

func TestHMACUtil_SecureVerificationToken(t *testing.T) {
	hmacUtil := NewHMACUtil("test-secret")

	verificationId := uuid.Must(uuid.NewV4())
	userId := uuid.Must(uuid.NewV4())
	remoteIP := "192.168.1.1"

	// Generate secure token
	token := hmacUtil.GenerateSecureVerificationToken(verificationId, userId, remoteIP)
	if token == nil {
		t.Fatal("Expected non-nil secure verification token")
	}

	if token.VerificationId != verificationId {
		t.Error("Verification ID mismatch in token")
	}

	if token.Signature == "" {
		t.Error("Expected non-empty signature in token")
	}

	// Validate token
	err := hmacUtil.ValidateSecureVerificationToken(token, userId, remoteIP)
	if err != nil {
		t.Errorf("Token validation failed: %v", err)
	}

	// Test with wrong IP
	err = hmacUtil.ValidateSecureVerificationToken(token, userId, "192.168.1.2")
	if err == nil {
		t.Error("Expected token validation to fail with wrong IP")
	}

	// Test with wrong user ID
	wrongUserId := uuid.Must(uuid.NewV4())
	err = hmacUtil.ValidateSecureVerificationToken(token, wrongUserId, remoteIP)
	if err == nil {
		t.Error("Expected token validation to fail with wrong user ID")
	}
}

func TestParseTimestampFromHeader(t *testing.T) {
	// Valid timestamp
	timestamp, err := ParseTimestampFromHeader("1640995200")
	if err != nil {
		t.Errorf("Failed to parse valid timestamp: %v", err)
	}
	if timestamp != 1640995200 {
		t.Errorf("Expected 1640995200, got %d", timestamp)
	}

	// Empty header
	_, err = ParseTimestampFromHeader("")
	if err == nil {
		t.Error("Expected error for empty timestamp header")
	}

	// Invalid format
	_, err = ParseTimestampFromHeader("invalid")
	if err == nil {
		t.Error("Expected error for invalid timestamp format")
	}

	// Timestamp with whitespace
	timestamp, err = ParseTimestampFromHeader("  1640995200  ")
	if err != nil {
		t.Errorf("Failed to parse timestamp with whitespace: %v", err)
	}
	if timestamp != 1640995200 {
		t.Errorf("Expected 1640995200, got %d", timestamp)
	}
}
