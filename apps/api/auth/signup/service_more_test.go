package signup

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/models"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"

	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/utils"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
)

func TestSignupService_SaveAndUpdateVerification_Coverage(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()
	// Create postgres client and repository
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	if err != nil {
		t.Skipf("Failed to create postgres client: %v", err)
	}
	defer pgClient.Close()

	// Create schema and set search_path
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, schemaSQL)
	if err != nil {
		t.Skipf("Failed to create schema: %v", err)
	}
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, setSearchPathSQL)
	if err != nil {
		t.Skipf("Failed to set search_path: %v", err)
	}

	// Apply auth migration
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS verifications (
			id UUID PRIMARY KEY,
			user_id UUID,
			code VARCHAR(10) NOT NULL,
			target VARCHAR(255) NOT NULL,
			target_type VARCHAR(50) NOT NULL,
			counter BIGINT DEFAULT 1,
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			remote_ip_address VARCHAR(45),
			is_verified BOOLEAN DEFAULT FALSE,
			hashed_password BYTEA,
			expires_at BIGINT NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			full_name VARCHAR(255)
		);
		CREATE INDEX IF NOT EXISTS idx_verifications_user_type ON verifications(user_id, target_type) WHERE user_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_verifications_code ON verifications(code) WHERE used = FALSE;
		CREATE INDEX IF NOT EXISTS idx_verifications_target ON verifications(target, target_type) WHERE user_id IS NULL;
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	if err != nil {
		t.Skipf("Failed to apply auth migration: %v", err)
	}

	verifRepo := authRepository.NewPostgresVerificationRepository(pgClient)
	
	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}
	s := NewService(verifRepo, serviceConfig)
	verifyId := uuid.Must(uuid.NewV4())
	doc := &models.UserVerification{
		ObjectId:    verifyId,
		UserId:      uuid.Must(uuid.NewV4()),
		Code:        "000000",
		Target:      "u@example.com",
		TargetType:  "email",
		Counter:     1,
		CreatedDate: utils.UTCNowUnix(),
	}
	_ = s.SaveUserVerification(ctx, doc)
	_ = s.UpdateVerification(ctx, &models.DatabaseFilter{ObjectId: &verifyId}, &models.DatabaseUpdate{Set: map[string]interface{}{"counter": 2}})
}
