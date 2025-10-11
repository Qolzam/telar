package testutil

import (
	"testing"
	
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/stretchr/testify/require"
)

// TestJWTConfigLoadsFromEnv validates that LoadFromEnv() correctly loads JWT keys
func TestJWTConfigLoadsFromEnv(t *testing.T) {
	cfg, err := platformconfig.LoadFromEnv()
	require.NoError(t, err, "LoadFromEnv should succeed")
	
	require.NotEmpty(t, cfg.JWT.PublicKey, "JWT_PUBLIC_KEY should be loaded from .env")
	require.NotEmpty(t, cfg.JWT.PrivateKey, "JWT_PRIVATE_KEY should be loaded from .env")
	require.NotEmpty(t, cfg.HMAC.Secret, "HMAC_SECRET should be loaded from .env")
	
	t.Logf("✅ JWT_PUBLIC_KEY loaded: %d bytes", len(cfg.JWT.PublicKey))
	t.Logf("✅ JWT_PRIVATE_KEY loaded: %d bytes", len(cfg.JWT.PrivateKey))
	t.Logf("✅ HMAC_SECRET loaded: %s", cfg.HMAC.Secret)
}
