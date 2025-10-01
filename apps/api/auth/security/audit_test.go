package security

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSecurityEvent_JSONSerialization(t *testing.T) {
	event := SecurityEvent{
		Timestamp: time.Date(2025, 9, 17, 10, 0, 0, 0, time.UTC),
		EventType: EventTypeLoginAttempt,
		UserID:    "test-user-123",
		IPAddress: "192.168.1.100",
		UserAgent: "Mozilla/5.0 Test",
		Success:   true,
		ErrorCode: "",
		Details:   "Successful login",
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to serialize SecurityEvent: %v", err)
	}

	var decoded SecurityEvent
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Failed to deserialize SecurityEvent: %v", err)
	}

	if decoded.EventType != event.EventType {
		t.Errorf("Expected EventType %s, got %s", event.EventType, decoded.EventType)
	}
	if decoded.UserID != event.UserID {
		t.Errorf("Expected UserID %s, got %s", event.UserID, decoded.UserID)
	}
	if decoded.Success != event.Success {
		t.Errorf("Expected Success %v, got %v", event.Success, decoded.Success)
	}
}

func TestLogSecurityEvent_TimestampHandling(t *testing.T) {
	// Test with zero timestamp (should be set automatically)
	event := SecurityEvent{
		EventType: EventTypeLoginAttempt,
		UserID:    "test-user",
		IPAddress: "192.168.1.1",
		UserAgent: "Test Agent",
		Success:   true,
	}

	// This should not panic and should set timestamp
	LogSecurityEvent(event)

	// Test with pre-set timestamp (should be preserved)
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	eventWithTime := SecurityEvent{
		Timestamp: fixedTime,
		EventType: EventTypeLoginAttempt,
		UserID:    "test-user",
		IPAddress: "192.168.1.1",
		UserAgent: "Test Agent",
		Success:   true,
	}

	// This should not panic and should preserve timestamp
	LogSecurityEvent(eventWithTime)
}

func TestLogLoginAttempt(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		ipAddress string
		userAgent string
		success   bool
		errorCode string
	}{
		{
			name:      "Successful login",
			userID:    "user-123",
			ipAddress: "192.168.1.100",
			userAgent: "Mozilla/5.0",
			success:   true,
			errorCode: "",
		},
		{
			name:      "Failed login - invalid password",
			userID:    "user-123",
			ipAddress: "192.168.1.100",
			userAgent: "Mozilla/5.0",
			success:   false,
			errorCode: "INVALID_PASSWORD",
		},
		{
			name:      "Failed login - user not found",
			userID:    "",
			ipAddress: "192.168.1.100",
			userAgent: "Mozilla/5.0",
			success:   false,
			errorCode: "USER_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			LogLoginAttempt(tt.userID, tt.ipAddress, tt.userAgent, tt.success, tt.errorCode)
		})
	}
}

func TestLogVerificationAttempt(t *testing.T) {
	tests := []struct {
		name              string
		userID            string
		ipAddress         string
		userAgent         string
		success           bool
		details           string
		expectedEventType string
	}{
		{
			name:              "Successful verification",
			userID:            "user-123",
			ipAddress:         "192.168.1.100",
			userAgent:         "Mozilla/5.0",
			success:           true,
			details:           "Email verification successful",
			expectedEventType: EventTypeVerificationSuccess,
		},
		{
			name:              "Failed verification - invalid code",
			userID:            "user-123",
			ipAddress:         "192.168.1.100",
			userAgent:         "Mozilla/5.0",
			success:           false,
			details:           "Invalid verification code",
			expectedEventType: EventTypeVerificationFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			LogVerificationAttempt(tt.userID, tt.ipAddress, tt.userAgent, tt.success, tt.details)
		})
	}
}

func TestLogHMACValidation(t *testing.T) {
	tests := []struct {
		name      string
		ipAddress string
		userAgent string
		success   bool
		errorCode string
	}{
		{
			name:      "Valid HMAC signature",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			success:   true,
			errorCode: "",
		},
		{
			name:      "Invalid HMAC signature",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			success:   false,
			errorCode: "HMAC_INVALID_SIGNATURE",
		},
		{
			name:      "HMAC timestamp expired",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			success:   false,
			errorCode: "HMAC_TIMESTAMP_EXPIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			LogHMACValidation(tt.ipAddress, tt.userAgent, tt.success, tt.errorCode)
		})
	}
}

func TestLogRateLimit(t *testing.T) {
	LogRateLimit("192.168.1.100", "Mozilla/5.0", "/auth/login")
	// Should not panic
}

func TestLogSecurityViolation(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		ipAddress     string
		userAgent     string
		violationType string
		details       string
	}{
		{
			name:          "SQL injection attempt",
			userID:        "user-123",
			ipAddress:     "192.168.1.100",
			userAgent:     "Malicious Client",
			violationType: "SQL_INJECTION",
			details:       "Detected SQL injection in login parameters",
		},
		{
			name:          "XSS attempt",
			userID:        "",
			ipAddress:     "192.168.1.100",
			userAgent:     "Malicious Client",
			violationType: "XSS_ATTEMPT",
			details:       "Detected XSS payload in user input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			LogSecurityViolation(tt.userID, tt.ipAddress, tt.userAgent, tt.violationType, tt.details)
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify all event type constants are properly defined
	eventTypes := []string{
		EventTypeLoginAttempt,
		EventTypeLoginSuccess,
		EventTypeLoginFailure,
		EventTypeSignupAttempt,
		EventTypeSignupSuccess,
		EventTypeSignupFailure,
		EventTypeVerificationAttempt,
		EventTypeVerificationSuccess,
		EventTypeVerificationFailure,
		EventTypePasswordReset,
		EventTypePasswordChange,
		EventTypeHMACValidation,
		EventTypeRateLimit,
		EventTypeSecurityViolation,
		EventTypePrivilegeEscalation,
	}

	for _, eventType := range eventTypes {
		if eventType == "" {
			t.Errorf("Event type constant is empty")
		}
		if len(eventType) < 3 {
			t.Errorf("Event type '%s' is too short", eventType)
		}
		// Event types should be lowercase with underscores
		if strings.ToLower(eventType) != eventType {
			t.Errorf("Event type '%s' should be lowercase", eventType)
		}
		if !strings.Contains(eventType, "_") {
			t.Errorf("Event type '%s' should contain underscores", eventType)
		}
	}
}

func TestSecurityEvent_RequiredFields(t *testing.T) {
	// Test that events can be created with minimal required fields
	minimalEvent := SecurityEvent{
		EventType: EventTypeLoginAttempt,
		IPAddress: "192.168.1.1",
		UserAgent: "Test Agent",
		Success:   false,
	}

	jsonData, err := json.Marshal(minimalEvent)
	if err != nil {
		t.Fatalf("Failed to serialize minimal SecurityEvent: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Serialized event should not be empty")
	}

	// Verify JSON contains expected fields
	jsonStr := string(jsonData)
	expectedFields := []string{"eventType", "ipAddress", "userAgent", "success"}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON should contain field '%s': %s", field, jsonStr)
		}
	}
}

func TestSecurityEvent_OptionalFields(t *testing.T) {
	// Test that optional fields are properly omitted when empty
	eventWithoutOptionals := SecurityEvent{
		EventType: EventTypeLoginAttempt,
		IPAddress: "192.168.1.1",
		UserAgent: "Test Agent",
		Success:   true,
		// UserID, ErrorCode, Details are empty and should be omitted
	}

	jsonData, err := json.Marshal(eventWithoutOptionals)
	if err != nil {
		t.Fatalf("Failed to serialize SecurityEvent: %v", err)
	}

	jsonStr := string(jsonData)

	// These fields should be omitted when empty due to omitempty tag
	omittedFields := []string{"userId", "errorCode", "details"}
	for _, field := range omittedFields {
		if strings.Contains(jsonStr, field) {
			t.Errorf("JSON should not contain empty field '%s': %s", field, jsonStr)
		}
	}
}

func TestLogHMACValidationDetailed(t *testing.T) {
	tests := []struct {
		name      string
		userId    string
		ipAddress string
		userAgent string
		eventType string
		details   string
		metadata  map[string]interface{}
	}{
		{
			name:      "Successful HMAC validation with user context",
			userId:    "user-123",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			eventType: "hmac_validation_success",
			details:   "HMAC signature validation successful",
			metadata: map[string]interface{}{
				"verification_id": "12345678-1234-1234-1234-123456789012",
			},
		},
		{
			name:      "Failed HMAC validation with error details",
			userId:    "user-123",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			eventType: "hmac_validation_failure",
			details:   "HMAC signature missing",
			metadata: map[string]interface{}{
				"verification_id": "12345678-1234-1234-1234-123456789012",
				"error":           "missing_hmac_signature",
			},
		},
		{
			name:      "Timestamp validation failure",
			userId:    "user-123",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			eventType: "hmac_validation_failure",
			details:   "Timestamp expired",
			metadata: map[string]interface{}{
				"verification_id": "12345678-1234-1234-1234-123456789012",
				"timestamp":       1234567890,
				"current_time":    1234568190,
				"error":           "timestamp_expired",
			},
		},
		{
			name:      "Without user ID",
			userId:    "",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			eventType: "hmac_validation_attempt",
			details:   "HMAC validation started",
			metadata: map[string]interface{}{
				"verification_id": "12345678-1234-1234-1234-123456789012",
			},
		},
		{
			name:      "Nil metadata",
			userId:    "user-123",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			eventType: "hmac_validation_attempt",
			details:   "Starting HMAC validation",
			metadata:  nil,
		},
		{
			name:      "Empty metadata",
			userId:    "user-123",
			ipAddress: "192.168.1.100",
			userAgent: "Service Client/1.0",
			eventType: "hmac_validation_attempt",
			details:   "Starting HMAC validation",
			metadata:  map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			LogHMACValidationDetailed(tt.userId, tt.ipAddress, tt.userAgent, tt.eventType, tt.details, tt.metadata)
		})
	}
}
