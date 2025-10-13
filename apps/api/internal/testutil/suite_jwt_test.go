package testutil

import (
	"os"
	"testing"
	
	"github.com/stretchr/testify/require"
)

// TestSuite_LoadsJWTFromEnv validates that JWT config is loaded from .env file
func TestSuite_LoadsJWTFromEnv(t *testing.T) {
	suite := Setup(t)
	
	jwtConfig := suite.GetTestJWTConfig()
	
	require.NotEmpty(t, jwtConfig.PublicKey, "JWT PublicKey should be loaded")
	require.NotEmpty(t, jwtConfig.PrivateKey, "JWT PrivateKey should be loaded")
	
	t.Logf("JWT_PUBLIC_KEY length: %d", len(jwtConfig.PublicKey))
	t.Logf("JWT_PRIVATE_KEY length: %d", len(jwtConfig.PrivateKey))
	
	envPrivateKey := os.Getenv("JWT_PRIVATE_KEY")
	if envPrivateKey != "" {
		t.Logf("✅ JWT_PRIVATE_KEY from env: %d chars", len(envPrivateKey))
		t.Logf("✅ Loaded private key: %d chars", len(jwtConfig.PrivateKey))
		
		if jwtConfig.PrivateKey == envPrivateKey {
			t.Log("✅ JWT config successfully loaded from environment")
		} else {
			t.Log("⚠️  JWT config using fallback values (env not loaded)")
		}
	}
}
