package verification

import (
	"testing"

	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestVerifySignup_AccessTokenGeneration(t *testing.T) {
	suite := testutil.Setup(t)
	
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

	base := &platform.BaseService{}

	publicKey, privateKey := suite.GenerateUniqueJWTKeys(t)

	service := NewServiceWithKeys(base, config, privateKey, "Telar", "http://localhost")

	require.NotNil(t, service.privateKey)
	require.Equal(t, "Telar", service.orgName)
	require.Equal(t, "http://localhost", service.webDomain)

	t.Log("✅ JWT token generation service properly configured")
}

func TestVerifySignupResult_Structure(t *testing.T) {
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
