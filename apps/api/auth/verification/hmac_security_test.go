package verification

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// TestHMACSecurityHardening validates Phase 1.2 security enhancements
func TestHMACSecurityHardening(t *testing.T) {
	t.Run("CRITICAL_SecretInjection", func(t *testing.T) {
		// Test that service properly accepts injected secrets
		base := &platform.BaseService{}

		// Test with various secret strengths
		testCases := []struct {
			name     string
			secret   string
			expected bool
		}{
			{"Strong secret", "very-strong-secret-at-least-32-chars-long-for-testing", true},
			{"Weak but valid", "weak", true}, // Should work but warn in production
			{"Empty secret", "", false},      // Should NOT work (secure by default)
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := &ServiceConfig{
					JWTConfig: platformconfig.JWTConfig{
						PublicKey:  "test-public-key",
						PrivateKey: "test-private-key",
					},
					HMACConfig: platformconfig.HMACConfig{
						Secret: tc.secret,
					},
					AppConfig: platformconfig.AppConfig{
						OrgName:   "TestOrg",
						WebDomain: "http://localhost:3000",
					},
				}

				service := NewService(base, config)
				if service == nil {
					t.Fatal("Service creation should never fail with any secret")
				}

				if tc.expected {
					if service.hmacUtil == nil {
						t.Error("HMAC util should be initialized with valid secret")
					}
				} else {
					if service.hmacUtil != nil {
						t.Error("HMAC util should NOT be initialized with empty secret (security)")
					}
				}
			})
		}
	})

	t.Run("CRITICAL_NoHardcodedSecrets", func(t *testing.T) {
		// Ensure we never fall back to old hardcoded secrets
		base := &platform.BaseService{}

		config := &ServiceConfig{
			JWTConfig: platformconfig.JWTConfig{
				PublicKey:  "test-public-key",
				PrivateKey: "test-private-key",
			},
			HMACConfig: platformconfig.HMACConfig{
				Secret: "", // Empty secret test - should NOT initialize HMAC (secure by default)
			},
			AppConfig: platformconfig.AppConfig{
				OrgName:   "TestOrg",
				WebDomain: "http://localhost:3000",
			},
		}

		service := NewService(base, config)

		// Production logic: HMAC util should NOT be initialized with empty secret (security)
		if service.hmacUtil != nil {
			t.Error("HMAC util should NOT be initialized with empty secret (security best practice)")
		} else {
			// Service correctly avoids initializing HMAC with empty secret
			t.Log("Service correctly handles empty secret by not initializing HMAC util")
		}
	})
	t.Run("MandatoryHMACValidation", func(t *testing.T) {
		// Test that HMAC validation is now mandatory
		base := &platform.BaseService{}
		config := &ServiceConfig{
			JWTConfig: platformconfig.JWTConfig{
				PublicKey:  "test-public-key",
				PrivateKey: "test-private-key",
			},
			HMACConfig: platformconfig.HMACConfig{
				Secret: "test-secret-for-mandatory-hmac-validation",
			},
			AppConfig: platformconfig.AppConfig{
				OrgName:   "TestOrg",
				WebDomain: "http://localhost:3000",
			},
		}
		service := NewService(base, config)

		params := VerifySignupParams{
			VerificationId: uuid.Must(uuid.NewV4()),
			Code:           "123456",
			UserId:         uuid.Must(uuid.NewV4()).String(), // Provide required UserId
			// Missing HMAC signature and timestamp should cause validation to fail
		}

		err := service.validateHMACSignature(context.Background(), &params)
		if err == nil {
			t.Fatal("CRITICAL: HMAC validation should be mandatory")
		}

		// Accept appropriate HMAC validation errors
		acceptableErrors := []string{
			"HMAC signature cannot be empty",
			"HMAC signature is required",
			"invalid HMAC signature",
		}
		errorFound := false
		for _, expectedErr := range acceptableErrors {
			if strings.Contains(err.Error(), expectedErr) {
				errorFound = true
				break
			}
		}
		if !errorFound {
			t.Errorf("Expected HMAC validation error, got: %v", err)
		}
	})

	t.Run("TimestampValidation", func(t *testing.T) {
		// Test timestamp validation for replay attack prevention
		base := &platform.BaseService{}
		config := &ServiceConfig{
			JWTConfig: platformconfig.JWTConfig{
				PublicKey:  "test-public-key",
				PrivateKey: "test-private-key",
			},
			HMACConfig: platformconfig.HMACConfig{
				Secret: "test-secret-for-timestamp-validation",
			},
			AppConfig: platformconfig.AppConfig{
				OrgName:   "TestOrg",
				WebDomain: "http://localhost:3000",
			},
		}
		service := NewService(base, config)

		// Test with old timestamp (should fail)
		oldTimestamp := time.Now().Unix() - 600 // 10 minutes ago
		params := VerifySignupParams{
			VerificationId: uuid.Must(uuid.NewV4()),
			Code:           "123456",
			UserId:         uuid.Must(uuid.NewV4()).String(), // Provide required UserId
			HMACSignature:  "dummy-signature",
			Timestamp:      oldTimestamp,
		}

		err := service.validateHMACSignature(context.Background(), &params)
		if err == nil {
			t.Fatal("Old timestamp should be rejected (replay attack prevention)")
		}

		// Accept various validation errors (timestamp or HMAC format)
		if !strings.Contains(err.Error(), "expired") &&
			!strings.Contains(err.Error(), "invalid HMAC") &&
			!strings.Contains(err.Error(), "signature format") {
			t.Errorf("Expected timestamp or HMAC validation error, got: %v", err)
		}

		// Test with future timestamp (should fail)
		futureTimestamp := time.Now().Unix() + 120 // 2 minutes in future
		params.Timestamp = futureTimestamp

		err = service.validateHMACSignature(context.Background(), &params)
		if err == nil {
			t.Fatal("Future timestamp should be rejected")
		}

		if !strings.Contains(err.Error(), "future") &&
			!strings.Contains(err.Error(), "invalid HMAC") &&
			!strings.Contains(err.Error(), "signature format") {
			t.Errorf("Expected future timestamp or HMAC validation error, got: %v", err)
		}
	})

	t.Run("HMACUtilSecurityValidation", func(t *testing.T) {
		// Test HMAC utility security enhancements
		secret := "test-secret-for-hmac-util-security"
		util := NewHMACUtil(secret)

		// Test hex format validation
		invalidHex := "not-hex-format-signature"
		hmacData := VerificationHMACData{
			VerificationId: uuid.Must(uuid.NewV4()),
			Code:           "123456",
			Timestamp:      time.Now().Unix(),
		}

		err := util.ValidateVerificationHMAC(hmacData, invalidHex)
		if err == nil {
			t.Fatal("Invalid hex format should be rejected")
		}

		// Test empty signature
		err = util.ValidateVerificationHMAC(hmacData, "")
		if err == nil {
			t.Fatal("Empty signature should be rejected")
		}
	})
}
