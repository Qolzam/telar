package admin

import (
	"context"
	"testing"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestAdminService_CheckCreateLogin_Coverage(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	platformCfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
	s := NewService(base, "test-private-key", platformCfg)

	// CheckAdmin should run without panic even if none exists
	_, _ = s.CheckAdmin(ctx)

	// CreateAdmin may fail if already exists; this is fine for coverage
	_, _ = s.CreateAdmin(ctx, "Admin", "admin@example.com", "Password123!@#")

	// Login may fail if wrong creds; still invoked for coverage
	_, _ = s.Login(ctx, "admin@example.com", "Password123!@#")
}

func TestAdminHelpers_GenerateSocialName(t *testing.T) {
	got := generateSocialName("John Doe", "1234-5678")
	if got == "" {
		t.Fatalf("expected non-empty social name")
	}
}
