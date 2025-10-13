package testutil

import (
	"testing"
	
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/stretchr/testify/require"
)

// TestLoadFromEnvDirectly
func TestLoadFromEnvDirectly(t *testing.T) {
	cfg, err := platformconfig.LoadFromEnv()
	require.NoError(t, err, "LoadFromEnv should load .env file successfully")
	
	require.NotEmpty(t, cfg.JWT.PublicKey)
	require.NotEmpty(t, cfg.JWT.PrivateKey)
	
	t.Logf("Direct LoadFromEnv() test:")
	t.Logf("  JWT_PUBLIC_KEY: %d bytes", len(cfg.JWT.PublicKey))
	t.Logf("  JWT_PRIVATE_KEY: %d bytes", len(cfg.JWT.PrivateKey))
}

// TestSuiteUsesLoadFromEnv 
func TestSuiteUsesLoadFromEnv(t *testing.T) {
	suite := Setup(t)
	jwt := suite.GetTestJWTConfig()
	
	require.Greater(t, len(jwt.PublicKey), 150, "Should load from .env")
	require.Greater(t, len(jwt.PrivateKey), 200, "Should load from .env")
	
	directCfg, _ := platformconfig.LoadFromEnv()
	require.Equal(t, directCfg.JWT.PublicKey, jwt.PublicKey, "Suite should use LoadFromEnv() result")
	require.Equal(t, directCfg.JWT.PrivateKey, jwt.PrivateKey, "Suite should use LoadFromEnv() result")
	
	t.Log("✅ Suite correctly uses LoadFromEnv(), not fallback")
}
