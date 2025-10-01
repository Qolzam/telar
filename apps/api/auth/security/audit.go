package security

import (
	"encoding/json"
	"time"

	log "github.com/qolzam/telar/apps/api/internal/pkg/log"
)

// SecurityEvent represents a security-related event for audit logging
// Following AUTH_SECURITY_REFACTORING_PLAN.md Task 4.3 specification
type SecurityEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"eventType"`
	UserID    string    `json:"userId,omitempty"`
	IPAddress string    `json:"ipAddress"`
	UserAgent string    `json:"userAgent"`
	Success   bool      `json:"success"`
	ErrorCode string    `json:"errorCode,omitempty"`
	Details   string    `json:"details,omitempty"`
}

// LogSecurityEvent logs a security event to the audit system
// Following AUTH_SECURITY_REFACTORING_PLAN.md Task 4.3 specification
func LogSecurityEvent(event SecurityEvent) {
	// Ensure timestamp is set if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Serialize event to JSON for structured logging
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Error("Failed to serialize security event: %v", err)
		return
	}

	// Log using the existing log infrastructure with structured format
	// Following the pattern used in validation/password_validation.go
	log.Info("[AUDIT] auth_security_event: %s", string(eventJSON))
}

// Predefined event types for consistency
const (
	EventTypeLoginAttempt        = "login_attempt"
	EventTypeLoginSuccess        = "login_success"
	EventTypeLoginFailure        = "login_failure"
	EventTypeSignupAttempt       = "signup_attempt"
	EventTypeSignupSuccess       = "signup_success"
	EventTypeSignupFailure       = "signup_failure"
	EventTypeVerificationAttempt = "verification_attempt"
	EventTypeVerificationSuccess = "verification_success"
	EventTypeVerificationFailure = "verification_failure"
	EventTypePasswordReset       = "password_reset"
	EventTypePasswordChange      = "password_change"
	EventTypeHMACValidation      = "hmac_validation"
	EventTypeRateLimit           = "rate_limit_triggered"
	EventTypeSecurityViolation   = "security_violation"
	EventTypePrivilegeEscalation = "privilege_escalation"
)

// Helper functions for common security events

// LogLoginAttempt logs a login attempt
func LogLoginAttempt(userID, ipAddress, userAgent string, success bool, errorCode string) {
	LogSecurityEvent(SecurityEvent{
		EventType: EventTypeLoginAttempt,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
		ErrorCode: errorCode,
	})
}

// LogVerificationAttempt logs a verification attempt
func LogVerificationAttempt(userID, ipAddress, userAgent string, success bool, details string) {
	eventType := EventTypeVerificationSuccess
	if !success {
		eventType = EventTypeVerificationFailure
	}

	LogSecurityEvent(SecurityEvent{
		EventType: eventType,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
		Details:   details,
	})
}

// LogHMACValidation logs HMAC validation attempts
func LogHMACValidation(ipAddress, userAgent string, success bool, errorCode string) {
	LogSecurityEvent(SecurityEvent{
		EventType: EventTypeHMACValidation,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
		ErrorCode: errorCode,
	})
}

// LogHMACValidationDetailed logs detailed HMAC validation attempts with additional context
func LogHMACValidationDetailed(userId, ipAddress, userAgent, eventType, details string, metadata map[string]interface{}) {
	// Determine success based on event type
	success := eventType == "hmac_validation_success"

	// Extract error code from metadata or use event type
	errorCode := ""
	if !success {
		if err, exists := metadata["error"]; exists {
			if errStr, ok := err.(string); ok {
				errorCode = errStr
			}
		}
		if errorCode == "" {
			errorCode = eventType
		}
	}

	LogSecurityEvent(SecurityEvent{
		EventType: EventTypeHMACValidation,
		UserID:    userId,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
		ErrorCode: errorCode,
		Details:   details,
	})
}

// LogRateLimit logs rate limiting events
func LogRateLimit(ipAddress, userAgent, endpoint string) {
	LogSecurityEvent(SecurityEvent{
		EventType: EventTypeRateLimit,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   false,
		Details:   "Rate limit exceeded for endpoint: " + endpoint,
	})
}

// LogSecurityViolation logs security violations
func LogSecurityViolation(userID, ipAddress, userAgent, violationType, details string) {
	LogSecurityEvent(SecurityEvent{
		EventType: EventTypeSecurityViolation,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   false,
		ErrorCode: violationType,
		Details:   details,
	})
}
