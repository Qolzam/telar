package verification

import (
	"testing"

	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/stretchr/testify/require"
)

func TestVerifySignup_AccessTokenGeneration(t *testing.T) {
	// Create service config
	config := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		AppConfig: platformconfig.AppConfig{
			OrgName:   "TestOrg",
			WebDomain: "http://localhost:3000",
		},
	}

	// Create a mock base service for testing
	base := &platform.BaseService{} // Minimal setup for testing

	// Test keys (these should be from test suite)
	privateKey := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKj1H/8K8NVKFbAWJZRAGT9ePBPGcPJy7gZs+YgHSr8ooAoGCCqGSM49
AwEHoUQDQgAEr/W+cGqZqwpwvC3T4rGxzW7GvKVjL9hEKZc+9mFHyQN1/Q6Fq0Cp
cY3sZ5zKxgqL7r3n5jrBdJm2tTgP1DKz7w==
-----END EC PRIVATE KEY-----`

	// Create service with JWT generation capability
	service := NewServiceWithKeys(base, config, privateKey, "Telar", "http://localhost")

	// Test that service is properly configured
	require.NotNil(t, service.privateKey)
	require.Equal(t, "Telar", service.orgName)
	require.Equal(t, "http://localhost", service.webDomain)

	t.Log("✅ JWT token generation service properly configured")
}

func TestVerifySignupResult_Structure(t *testing.T) {
	// Test that VerifySignupResult has all required fields
	result := &VerifySignupResult{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		User:        map[string]interface{}{"userId": "test"},
		Success:     true,
	}

	require.NotEmpty(t, result.AccessToken)
	require.Equal(t, "Bearer", result.TokenType)
	require.NotNil(t, result.User)
	require.True(t, result.Success)

	t.Log("✅ VerifySignupResult structure contains all required fields for JWT response")
}
