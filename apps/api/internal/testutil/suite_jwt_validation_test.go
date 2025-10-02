package testutil

import (
	"testing"
	
	"github.com/stretchr/testify/require"
)

// TestSuiteLoadsJWTFromEnv validates that Suite.Setup() loads JWT keys from .env
func TestSuiteLoadsJWTFromEnv(t *testing.T) {
	suite := Setup(t)
	
	jwtConfig := suite.GetTestJWTConfig()
	
	require.NotEmpty(t, jwtConfig.PublicKey, "JWT PublicKey should be loaded from .env")
	require.NotEmpty(t, jwtConfig.PrivateKey, "JWT PrivateKey should be loaded from .env")
	

	require.Greater(t, len(jwtConfig.PublicKey), 100, "PublicKey should be from .env (not empty fallback)")
	require.Greater(t, len(jwtConfig.PrivateKey), 100, "PrivateKey should be from .env (not empty fallback)")
	
	t.Logf("✅ Suite correctly loads JWT keys from .env")
	t.Logf("✅ JWT_PUBLIC_KEY: %d bytes", len(jwtConfig.PublicKey))
	t.Logf("✅ JWT_PRIVATE_KEY: %d bytes", len(jwtConfig.PrivateKey))
}

// TestSuiteGenerateUniqueKeys validates the unique key generation method
func TestSuiteGenerateUniqueKeys(t *testing.T) {
	suite := Setup(t)
	
	pub1, priv1 := suite.GenerateUniqueJWTKeys(t)
	pub2, priv2 := suite.GenerateUniqueJWTKeys(t)
	
	require.NotEqual(t, priv1, priv2, "Generated private keys should be unique")
	require.NotEqual(t, pub1, pub2, "Generated public keys should be unique")
	
	require.Contains(t, priv1, "BEGIN PRIVATE KEY", "Private key should be PEM formatted")
	require.Contains(t, pub1, "BEGIN PUBLIC KEY", "Public key should be PEM formatted")
	
	t.Log("✅ GenerateUniqueJWTKeys() produces unique keys each call")
}
