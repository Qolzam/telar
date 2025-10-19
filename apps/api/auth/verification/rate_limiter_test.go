package verification

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/models"
)

func TestRateLimiter_IPRateLimit(t *testing.T) {
	rl := NewRateLimiter(3, 5, 5*time.Minute) // 3 attempts per IP, 5 per verification, 5-minute window

	ip := "192.168.1.100"
	verifyId := uuid.Must(uuid.NewV4())

	// First 3 attempts should be allowed
	for i := 0; i < 3; i++ {
		err := rl.IsAllowed(ip, verifyId)
		if err != nil {
			t.Errorf("Attempt %d should be allowed: %v", i+1, err)
		}
		rl.RecordAttempt(ip, verifyId)
	}

	// 4th attempt should be blocked
	err := rl.IsAllowed(ip, verifyId)
	if err == nil {
		t.Error("4th attempt should be blocked due to IP rate limit")
	}

	// Different IP should still be allowed
	differentIP := "192.168.1.101"
	err = rl.IsAllowed(differentIP, verifyId)
	if err != nil {
		t.Errorf("Different IP should be allowed: %v", err)
	}
}

func TestRateLimiter_VerificationLimit(t *testing.T) {
	rl := NewRateLimiter(10, 2, 5*time.Minute) // 10 per IP, 2 per verification, 5-minute window

	ip := "192.168.1.100"
	verifyId := uuid.Must(uuid.NewV4())

	// First 2 attempts should be allowed
	for i := 0; i < 2; i++ {
		err := rl.IsAllowed(ip, verifyId)
		if err != nil {
			t.Errorf("Attempt %d should be allowed: %v", i+1, err)
		}
		rl.RecordAttempt(ip, verifyId)
	}

	// 3rd attempt should be blocked
	err := rl.IsAllowed(ip, verifyId)
	if err == nil {
		t.Error("3rd attempt should be blocked due to verification limit")
	}

	// Different verification should still be allowed
	differentVerifyId := uuid.Must(uuid.NewV4())
	err = rl.IsAllowed(ip, differentVerifyId)
	if err != nil {
		t.Errorf("Different verification should be allowed: %v", err)
	}
}

func TestRateLimiter_TimeWindow(t *testing.T) {
	rl := NewRateLimiter(2, 5, 100*time.Millisecond) // Short window for testing

	ip := "192.168.1.100"
	verifyId := uuid.Must(uuid.NewV4())

	// Use up the limit
	for i := 0; i < 2; i++ {
		rl.RecordAttempt(ip, verifyId)
	}

	// Should be blocked immediately
	err := rl.IsAllowed(ip, verifyId)
	if err == nil {
		t.Error("Should be blocked due to rate limit")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	err = rl.IsAllowed(ip, verifyId)
	if err != nil {
		t.Errorf("Should be allowed after time window: %v", err)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(10, 2, 5*time.Minute)

	ip := "192.168.1.100"
	verifyId := uuid.Must(uuid.NewV4())

	// Use up verification limit
	for i := 0; i < 2; i++ {
		rl.RecordAttempt(ip, verifyId)
	}

	// Should be blocked
	err := rl.IsAllowed(ip, verifyId)
	if err == nil {
		t.Error("Should be blocked due to verification limit")
	}

	// Reset the verification
	rl.Reset(verifyId)

	// Should be allowed again
	err = rl.IsAllowed(ip, verifyId)
	if err != nil {
		t.Errorf("Should be allowed after reset: %v", err)
	}
}

func TestSessionTimeoutManager_Expiration(t *testing.T) {
	stm := NewSessionTimeoutManager(100*time.Millisecond, 1*time.Second)

	now := time.Now().Unix()

	// Test not expired
	if stm.IsExpired(now, now+10) {
		t.Error("Should not be expired")
	}

	// Test expired by explicit expiration
	if !stm.IsExpired(now, now-10) {
		t.Error("Should be expired due to explicit expiration")
	}

	// Test expired by default timeout
	oldTime := now - 1 // 1 second ago, beyond 100ms timeout
	if !stm.IsExpired(oldTime, 0) {
		t.Error("Should be expired due to default timeout")
	}
}

func TestSessionTimeoutManager_ValidateTimeout(t *testing.T) {
	stm := NewSessionTimeoutManager(5*time.Minute, 30*time.Minute)

	// Test zero timeout
	timeout := stm.ValidateTimeout(0)
	if timeout != 5*time.Minute {
		t.Errorf("Expected default timeout, got %v", timeout)
	}

	// Test negative timeout
	timeout = stm.ValidateTimeout(-1 * time.Minute)
	if timeout != 5*time.Minute {
		t.Errorf("Expected default timeout for negative value, got %v", timeout)
	}

	// Test timeout over maximum
	timeout = stm.ValidateTimeout(60 * time.Minute)
	if timeout != 30*time.Minute {
		t.Errorf("Expected max timeout, got %v", timeout)
	}

	// Test valid timeout
	timeout = stm.ValidateTimeout(10 * time.Minute)
	if timeout != 10*time.Minute {
		t.Errorf("Expected 10 minutes, got %v", timeout)
	}
}

func TestSecurityValidator_ValidateVerificationAttempt(t *testing.T) {
	// Mock service (nil for testing, would need proper mock in real scenario)
	sv := &SecurityValidator{
		rateLimiter: NewRateLimiter(5, 3, 15*time.Minute),
		timeoutMgr:  NewSessionTimeoutManager(15*time.Minute, 60*time.Minute),
		service:     nil, // Would need proper mock
	}

	verifyId := uuid.Must(uuid.NewV4())
	userId := uuid.Must(uuid.NewV4())
	ip := "192.168.1.100"
	now := time.Now().Unix()

	params := VerifySignupParams{
		VerificationId:  verifyId,
		Code:            "123456",
		RemoteIpAddress: ip,
		ResponseType:    "spa",
	}

	verification := &models.UserVerification{
		ObjectId:        verifyId,
		UserId:          userId,
		Code:            "123456",
		RemoteIpAddress: ip,
		CreatedDate:     now,
		ExpiresAt:       now + 900, // 15 minutes
		Used:            false,
		Counter:         0,
	}

	// Should be allowed initially
	result := sv.ValidateVerificationAttempt(context.Background(), params, verification)
	if !result.Allowed {
		t.Errorf("Should be allowed initially: %s", result.Reason)
	}

	// Test with used verification
	verification.Used = true
	result = sv.ValidateVerificationAttempt(context.Background(), params, verification)
	if result.Allowed {
		t.Error("Should not be allowed with used verification")
	}
	verification.Used = false

	// Test with expired verification
	verification.ExpiresAt = now - 100 // Expired
	result = sv.ValidateVerificationAttempt(context.Background(), params, verification)
	if result.Allowed {
		t.Error("Should not be allowed with expired verification")
	}
	verification.ExpiresAt = now + 900 // Reset

	// Test with IP mismatch
	verification.RemoteIpAddress = "192.168.1.999"
	result = sv.ValidateVerificationAttempt(context.Background(), params, verification)
	if result.Allowed {
		t.Error("Should not be allowed with IP mismatch")
	}
	verification.RemoteIpAddress = ip // Reset

	// Test with too many attempts
	verification.Counter = 10
	result = sv.ValidateVerificationAttempt(context.Background(), params, verification)
	if result.Allowed {
		t.Error("Should not be allowed with too many attempts")
	}
}
