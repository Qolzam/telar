package login

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/require"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	authModels "github.com/qolzam/telar/apps/api/auth/models"
)

func TestLoginService_FindAndReadProfile_Coverage(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// Create postgres client and repository
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer pgClient.Close()

	// Create schema and set search_path
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err, "Failed to create schema")
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "Failed to set search_path")

	// Apply auth migration
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS user_auths (
			id UUID PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash BYTEA NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth migration")

	authRepo := authRepository.NewPostgresAuthRepository(pgClient)

	// Create service config for testing
	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
	svc := NewService(authRepo, serviceConfig)
	uid := uuid.Must(uuid.NewV4())

	// Seed userAuth using repository
	userAuth := &authModels.UserAuth{
		ObjectId:      uid,
		Username:      "find@example.com",
		Password:      []byte("p"),
		EmailVerified: true,
		PhoneVerified: false,
		Role:          "user",
		CreatedDate:   1,
		LastUpdated:   1,
	}
	err = authRepo.CreateUser(ctx, userAuth)
	require.NoError(t, err, "Failed to create user")

	_, _ = svc.FindUserByUsername(ctx, "find@example.com")
	
}
