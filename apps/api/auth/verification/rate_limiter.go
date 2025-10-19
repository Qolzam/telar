package verification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/models"
)

// RateLimiter provides rate limiting functionality for verification attempts
type RateLimiter struct {
	mu                   sync.RWMutex
	ipAttempts           map[string][]time.Time // IP -> attempt timestamps
	verifyAttempts       map[uuid.UUID]int      // Verification ID -> attempt count
	maxAttemptsPerIP     int                    // Max attempts per IP per window
	maxAttemptsPerVerify int                    // Max attempts per verification ID
	timeWindow           time.Duration          // Time window for rate limiting
	cleanupInterval      time.Duration          // How often to clean up old attempts
	lastCleanup          time.Time              // Last cleanup time
}

// NewRateLimiter creates a new rate limiter with specified limits
func NewRateLimiter(maxAttemptsPerIP, maxAttemptsPerVerify int, timeWindow time.Duration) *RateLimiter {
	return &RateLimiter{
		ipAttempts:           make(map[string][]time.Time),
		verifyAttempts:       make(map[uuid.UUID]int),
		maxAttemptsPerIP:     maxAttemptsPerIP,
		maxAttemptsPerVerify: maxAttemptsPerVerify,
		timeWindow:           timeWindow,
		cleanupInterval:      timeWindow / 2, // Cleanup twice per window
		lastCleanup:          time.Now(),
	}
}

// DefaultRateLimiter creates a rate limiter with secure default settings
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(
		10,             // Max 10 attempts per IP per 15 minutes
		5,              // Max 5 attempts per verification ID
		15*time.Minute, // 15-minute time window
	)
}

// IsAllowed checks if an attempt is allowed for the given IP and verification ID
func (rl *RateLimiter) IsAllowed(remoteIP string, verificationId uuid.UUID) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Cleanup old entries periodically
	if time.Since(rl.lastCleanup) > rl.cleanupInterval {
		rl.cleanup()
	}

	// Check IP-based rate limit
	if err := rl.checkIPLimit(remoteIP); err != nil {
		return err
	}

	// Check verification-specific limit
	if err := rl.checkVerificationLimit(verificationId); err != nil {
		return err
	}

	return nil
}

// RecordAttempt records a verification attempt
func (rl *RateLimiter) RecordAttempt(remoteIP string, verificationId uuid.UUID) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Record IP attempt
	rl.ipAttempts[remoteIP] = append(rl.ipAttempts[remoteIP], now)

	// Record verification attempt
	rl.verifyAttempts[verificationId]++
}

// checkIPLimit checks if IP has exceeded rate limit
func (rl *RateLimiter) checkIPLimit(remoteIP string) error {
	attempts := rl.ipAttempts[remoteIP]
	if len(attempts) == 0 {
		return nil
	}

	// Count attempts within time window
	cutoff := time.Now().Add(-rl.timeWindow)
	validAttempts := 0

	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			validAttempts++
		}
	}

	if validAttempts >= rl.maxAttemptsPerIP {
		return fmt.Errorf("rate limit exceeded: too many attempts from IP %s (%d/%d in %v)",
			remoteIP, validAttempts, rl.maxAttemptsPerIP, rl.timeWindow)
	}

	return nil
}

// checkVerificationLimit checks if verification ID has exceeded attempt limit
func (rl *RateLimiter) checkVerificationLimit(verificationId uuid.UUID) error {
	attempts := rl.verifyAttempts[verificationId]

	if attempts >= rl.maxAttemptsPerVerify {
		return fmt.Errorf("verification attempt limit exceeded: %d/%d attempts for verification %s",
			attempts, rl.maxAttemptsPerVerify, verificationId.String())
	}

	return nil
}

// cleanup removes old attempt records
func (rl *RateLimiter) cleanup() {
	cutoff := time.Now().Add(-rl.timeWindow)

	// Clean IP attempts
	for ip, attempts := range rl.ipAttempts {
		validAttempts := make([]time.Time, 0, len(attempts))
		for _, attempt := range attempts {
			if attempt.After(cutoff) {
				validAttempts = append(validAttempts, attempt)
			}
		}

		if len(validAttempts) == 0 {
			delete(rl.ipAttempts, ip)
		} else {
			rl.ipAttempts[ip] = validAttempts
		}
	}

	rl.lastCleanup = time.Now()
}

// Reset clears rate limiting data for a specific verification ID
func (rl *RateLimiter) Reset(verificationId uuid.UUID) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.verifyAttempts, verificationId)
}

// GetStats returns current rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"tracked_ips":             len(rl.ipAttempts),
		"tracked_verifications":   len(rl.verifyAttempts),
		"max_attempts_per_ip":     rl.maxAttemptsPerIP,
		"max_attempts_per_verify": rl.maxAttemptsPerVerify,
		"time_window_minutes":     rl.timeWindow.Minutes(),
		"last_cleanup":            rl.lastCleanup,
	}
}

// SessionTimeoutManager manages verification session timeouts
type SessionTimeoutManager struct {
	defaultTimeout time.Duration
	maxTimeout     time.Duration
}

// NewSessionTimeoutManager creates a new session timeout manager
func NewSessionTimeoutManager(defaultTimeout, maxTimeout time.Duration) *SessionTimeoutManager {
	return &SessionTimeoutManager{
		defaultTimeout: defaultTimeout,
		maxTimeout:     maxTimeout,
	}
}

// DefaultSessionTimeoutManager creates a timeout manager with secure defaults
func DefaultSessionTimeoutManager() *SessionTimeoutManager {
	return NewSessionTimeoutManager(
		15*time.Minute, // Default 15-minute timeout
		60*time.Minute, // Maximum 1-hour timeout
	)
}

// GetVerificationTimeout returns the timeout for a verification session
func (stm *SessionTimeoutManager) GetVerificationTimeout() time.Duration {
	return stm.defaultTimeout
}

// IsExpired checks if a verification has expired
func (stm *SessionTimeoutManager) IsExpired(createdAt, expiresAt int64) bool {
	now := time.Now().Unix()

	// Check explicit expiration
	if expiresAt > 0 && now > expiresAt {
		return true
	}

	// Check default timeout from creation
	if now > createdAt+int64(stm.defaultTimeout.Seconds()) {
		return true
	}

	return false
}

// ValidateTimeout ensures timeout is within acceptable limits
func (stm *SessionTimeoutManager) ValidateTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return stm.defaultTimeout
	}

	if timeout > stm.maxTimeout {
		return stm.maxTimeout
	}

	return timeout
}

// SecurityValidationResult represents the result of security validations
type SecurityValidationResult struct {
	Allowed      bool
	Reason       string
	RetryAfter   time.Duration // How long to wait before retry
	BlockedUntil time.Time     // When the block expires
}

// SecurityValidator combines rate limiting and session management
type SecurityValidator struct {
	rateLimiter *RateLimiter
	timeoutMgr  *SessionTimeoutManager
	service     *Service // For database operations
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(service *Service) *SecurityValidator {
	return &SecurityValidator{
		rateLimiter: DefaultRateLimiter(),
		timeoutMgr:  DefaultSessionTimeoutManager(),
		service:     service,
	}
}

// ValidateVerificationAttempt performs comprehensive security validation
func (sv *SecurityValidator) ValidateVerificationAttempt(ctx context.Context, params VerifySignupParams, verification *models.UserVerification) *SecurityValidationResult {
	// 1. Check rate limiting
	if err := sv.rateLimiter.IsAllowed(params.RemoteIpAddress, params.VerificationId); err != nil {
		return &SecurityValidationResult{
			Allowed:    false,
			Reason:     fmt.Sprintf("Rate limit exceeded: %v", err),
			RetryAfter: 15 * time.Minute,
		}
	}

	// 2. Check session timeout
	if sv.timeoutMgr.IsExpired(verification.CreatedDate, verification.ExpiresAt) {
		return &SecurityValidationResult{
			Allowed: false,
			Reason:  "Verification session expired",
		}
	}

	// 3. Check if verification is already used
	if verification.Used {
		return &SecurityValidationResult{
			Allowed: false,
			Reason:  "Verification already used",
		}
	}

	// 4. Check IP address consistency
	if verification.RemoteIpAddress != params.RemoteIpAddress {
		return &SecurityValidationResult{
			Allowed: false,
			Reason:  "IP address mismatch - potential security threat",
		}
	}

	// 5. Check attempt counter from database
	if verification.Counter >= 5 { // Max 5 attempts per verification
		return &SecurityValidationResult{
			Allowed: false,
			Reason:  "Maximum verification attempts exceeded",
		}
	}

	return &SecurityValidationResult{Allowed: true}
}

// RecordFailedAttempt records a failed verification attempt
func (sv *SecurityValidator) RecordFailedAttempt(ctx context.Context, params VerifySignupParams) {
	// Record in rate limiter
	sv.rateLimiter.RecordAttempt(params.RemoteIpAddress, params.VerificationId)

	// Increment database counter
	sv.service.incrementVerificationAttempts(ctx, params.VerificationId)
}

// ResetVerificationLimits clears rate limiting for a successful verification
func (sv *SecurityValidator) ResetVerificationLimits(verificationId uuid.UUID) {
	sv.rateLimiter.Reset(verificationId)
}
