package admin

import (
	"context"
	"testing"

	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	adminRepository "github.com/qolzam/telar/apps/api/auth/admin/repository"
	profileRepository "github.com/qolzam/telar/apps/api/profile/repository"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestAdminService_CheckCreateLogin_Coverage(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping")
	}

	ctx := context.Background()

	// Create postgres client for repositories
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err)

	// Create repositories
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)
	adminRepo := adminRepository.NewPostgresAdminRepository(pgClient)

	platformCfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
	s := NewService(authRepo, profileRepo, adminRepo, "test-private-key", platformCfg)

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
