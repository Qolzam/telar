// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/auth/models"
	profileRepo "github.com/qolzam/telar/apps/api/profile/repository"
	profileModels "github.com/qolzam/telar/apps/api/profile/models"
)

// TestPostgresAuthRepository_Integration validates the new PostgresAuthRepository implementation
// This test focuses exclusively on the repository layer, bypassing the service layer.
func TestPostgresAuthRepository_Integration(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	// 1. Setup Isolated DB
	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// 2. Create postgres.Client from isolated test config
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema

	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer client.Close()

	// 3. Create schema if it doesn't exist
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err, "Failed to create schema")

	// 4. Set search_path to the isolated schema
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "Failed to set search_path")

	// 5. Apply Auth Schema Migration
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

		CREATE TABLE IF NOT EXISTS verifications (
			id UUID PRIMARY KEY,
			user_id UUID REFERENCES user_auths(id) ON DELETE CASCADE,
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

		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
		CREATE INDEX IF NOT EXISTS idx_user_auths_role ON user_auths(role);
		CREATE INDEX IF NOT EXISTS idx_user_auths_created_at ON user_auths(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_user_auths_created_date ON user_auths(created_date DESC);

		CREATE INDEX IF NOT EXISTS idx_verifications_user_type ON verifications(user_id, target_type) WHERE user_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_verifications_code ON verifications(code) WHERE used = FALSE;
		CREATE INDEX IF NOT EXISTS idx_verifications_target ON verifications(target, target_type) WHERE user_id IS NULL;
		CREATE INDEX IF NOT EXISTS idx_verifications_expires_at ON verifications(expires_at) WHERE used = FALSE;
		CREATE INDEX IF NOT EXISTS idx_verifications_created_at ON verifications(created_date DESC);
	`

	_, err = client.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth migration")

	// 6. Initialize Repositories
	authRepo := NewPostgresAuthRepository(client)
	verificationRepo := NewPostgresVerificationRepository(client)

	// Test data
	userID1 := uuid.Must(uuid.NewV4())
	userID2 := uuid.Must(uuid.NewV4())
	now := time.Now()
	passwordHash := []byte("hashed_password_123")

	// 7. Test CreateUser
	t.Run("CreateUser", func(t *testing.T) {
		userAuth := &models.UserAuth{
			ObjectId:      userID1,
			Username:      "test@example.com",
			Password:      passwordHash,
			Role:          "user",
			EmailVerified: false,
			PhoneVerified: false,
			CreatedDate:   now.Unix(),
			LastUpdated:   now.Unix(),
		}

		err := authRepo.CreateUser(ctx, userAuth)
		require.NoError(t, err, "Failed to create user")
	})

	// 8. Test FindByUsername
	t.Run("FindByUsername", func(t *testing.T) {
		fetched, err := authRepo.FindByUsername(ctx, "test@example.com")
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.Equal(t, userID1, fetched.ObjectId)
		require.Equal(t, "test@example.com", fetched.Username)
		require.Equal(t, passwordHash, fetched.Password)
		require.Equal(t, "user", fetched.Role)
		require.False(t, fetched.EmailVerified)
		require.False(t, fetched.PhoneVerified)
	})

	// 9. Test FindByID
	t.Run("FindByID", func(t *testing.T) {
		fetched, err := authRepo.FindByID(ctx, userID1)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.Equal(t, userID1, fetched.ObjectId)
		require.Equal(t, "test@example.com", fetched.Username)
	})

	// 10. Test CreateUser_DuplicateUsername (unique constraint)
	t.Run("CreateUser_DuplicateUsername", func(t *testing.T) {
		userAuth := &models.UserAuth{
			ObjectId:   uuid.Must(uuid.NewV4()),
			Username:   "test@example.com", // Same as above
			Password:   []byte("another_hash"),
			Role:       "user",
			CreatedDate: now.Unix(),
			LastUpdated: now.Unix(),
		}

		err := authRepo.CreateUser(ctx, userAuth)
		require.Error(t, err)
		require.Contains(t, err.Error(), "username already exists")
	})

	// 11. Test UpdatePassword
	t.Run("UpdatePassword", func(t *testing.T) {
		newPasswordHash := []byte("new_hashed_password")
		err := authRepo.UpdatePassword(ctx, userID1, newPasswordHash)
		require.NoError(t, err)

		fetched, err := authRepo.FindByID(ctx, userID1)
		require.NoError(t, err)
		require.Equal(t, newPasswordHash, fetched.Password)
	})

	// 12. Test UpdateEmailVerified
	t.Run("UpdateEmailVerified", func(t *testing.T) {
		err := authRepo.UpdateEmailVerified(ctx, userID1, true)
		require.NoError(t, err)

		fetched, err := authRepo.FindByID(ctx, userID1)
		require.NoError(t, err)
		require.True(t, fetched.EmailVerified)
	})

	// 13. Test UpdatePhoneVerified
	t.Run("UpdatePhoneVerified", func(t *testing.T) {
		err := authRepo.UpdatePhoneVerified(ctx, userID1, true)
		require.NoError(t, err)

		fetched, err := authRepo.FindByID(ctx, userID1)
		require.NoError(t, err)
		require.True(t, fetched.PhoneVerified)
	})

	// 14. Test VerificationRepository - Create with valid user_id
	t.Run("Verification_CreateWithUserID", func(t *testing.T) {
		verification := &models.UserVerification{
			ObjectId:        uuid.Must(uuid.NewV4()),
			UserId:          userID1,
			Code:            "123456",
			Target:          "test@example.com",
			TargetType:      "email",
			Counter:         1,
			CreatedDate:     now.Unix(),
			LastUpdated:     now.Unix(),
			RemoteIpAddress: "192.168.1.1",
			IsVerified:      false,
			ExpiresAt:       now.Add(15 * time.Minute).Unix(),
			Used:            false,
			FullName:        "Test User",
		}

		err := verificationRepo.SaveVerification(ctx, verification)
		require.NoError(t, err, "Failed to save verification with user_id")
	})

	// 15. Test VerificationRepository - Create with nil user_id (password reset)
	t.Run("Verification_CreateWithNilUserID", func(t *testing.T) {
		verification := &models.UserVerification{
			ObjectId:        uuid.Must(uuid.NewV4()),
			UserId:          uuid.Nil, // No user_id for password reset
			Code:            "654321",
			Target:          "reset@example.com",
			TargetType:      "password_reset",
			Counter:         1,
			CreatedDate:     now.Unix(),
			LastUpdated:     now.Unix(),
			RemoteIpAddress: "192.168.1.2",
			IsVerified:      false,
			HashedPassword:  []byte("reset_password_hash"),
			ExpiresAt:       now.Add(1 * time.Hour).Unix(),
			Used:            false,
			FullName:        "Reset User",
		}

		err := verificationRepo.SaveVerification(ctx, verification)
		require.NoError(t, err, "Failed to save verification with nil user_id")
	})

	// 16. Test FindVerification by code
	t.Run("Verification_FindByCode", func(t *testing.T) {
		fetched, err := verificationRepo.FindVerification(ctx, "123456", "email")
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.Equal(t, "123456", fetched.Code)
		require.Equal(t, userID1, fetched.UserId)
		require.Equal(t, "test@example.com", fetched.Target)
		require.Equal(t, "email", fetched.TargetType)
		require.False(t, fetched.Used)
	})

	// 17. Test FindVerificationByUser
	t.Run("Verification_FindByUser", func(t *testing.T) {
		fetched, err := verificationRepo.FindVerificationByUser(ctx, userID1, "email")
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.Equal(t, userID1, fetched.UserId)
		require.Equal(t, "email", fetched.TargetType)
	})

	// 18. Test FindVerificationByTarget (for password reset)
	t.Run("Verification_FindByTarget", func(t *testing.T) {
		fetched, err := verificationRepo.FindVerificationByTarget(ctx, "reset@example.com", "password_reset")
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.Equal(t, "reset@example.com", fetched.Target)
		require.Equal(t, "password_reset", fetched.TargetType)
		require.Equal(t, uuid.Nil, fetched.UserId) // Should be nil for password reset
		require.NotNil(t, fetched.HashedPassword)
	})

	// 19. Test MarkVerified
	t.Run("Verification_MarkVerified", func(t *testing.T) {
		verification, err := verificationRepo.FindVerification(ctx, "123456", "email")
		require.NoError(t, err)

		err = verificationRepo.MarkVerified(ctx, verification.ObjectId)
		require.NoError(t, err)

		fetched, err := verificationRepo.FindVerificationByUser(ctx, userID1, "email")
		require.NoError(t, err)
		require.True(t, fetched.IsVerified)
	})

	// 20. Test MarkUsed
	t.Run("Verification_MarkUsed", func(t *testing.T) {
		verification, err := verificationRepo.FindVerificationByTarget(ctx, "reset@example.com", "password_reset")
		require.NoError(t, err)

		err = verificationRepo.MarkUsed(ctx, verification.ObjectId)
		require.NoError(t, err)

		// Should not find it again (used = TRUE)
		_, err = verificationRepo.FindVerification(ctx, "654321", "password_reset")
		require.Error(t, err)
		require.Contains(t, err.Error(), "verification not found")
	})

	// 21. Test DeleteExpired
	t.Run("Verification_DeleteExpired", func(t *testing.T) {
		// Create an expired verification
		expiredVerification := &models.UserVerification{
			ObjectId:    uuid.Must(uuid.NewV4()),
			UserId:      userID1,
			Code:        "999999",
			Target:      "expired@example.com",
			TargetType:  "email",
			Counter:     1,
			CreatedDate: now.Add(-2 * time.Hour).Unix(),
			LastUpdated: now.Add(-2 * time.Hour).Unix(),
			ExpiresAt:   now.Add(-1 * time.Hour).Unix(), // Expired
			Used:        false,
		}

		err := verificationRepo.SaveVerification(ctx, expiredVerification)
		require.NoError(t, err)

		// Delete expired verifications
		err = verificationRepo.DeleteExpired(ctx, now.Unix())
		require.NoError(t, err)

		// Should not find the expired verification
		_, err = verificationRepo.FindVerification(ctx, "999999", "email")
		require.Error(t, err)
	})

	// 22. Test Transaction - Atomic User+Profile Creation
	t.Run("Transaction_AtomicUserAndProfileCreation", func(t *testing.T) {
		// Apply profiles migration for this test (before transaction)
		profileMigrationSQL := `
			CREATE TABLE IF NOT EXISTS profiles (
				user_id UUID PRIMARY KEY,
				full_name VARCHAR(255),
				social_name VARCHAR(255),
				email VARCHAR(255),
				avatar VARCHAR(512),
				banner VARCHAR(512),
				tagline VARCHAR(500),
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
				last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
				last_seen BIGINT DEFAULT 0,
				birthday BIGINT DEFAULT 0,
				web_url VARCHAR(512),
				company_name VARCHAR(255),
				country VARCHAR(100),
				address TEXT,
				phone VARCHAR(50),
				vote_count BIGINT DEFAULT 0,
				share_count BIGINT DEFAULT 0,
				follow_count BIGINT DEFAULT 0,
				follower_count BIGINT DEFAULT 0,
				post_count BIGINT DEFAULT 0,
				facebook_id VARCHAR(255),
				instagram_id VARCHAR(255),
				twitter_id VARCHAR(255),
				linkedin_id VARCHAR(255),
				access_user_list TEXT[],
				permission VARCHAR(50) DEFAULT 'Public'
			);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_social_name ON profiles(social_name) WHERE social_name IS NOT NULL;
		`
		// Ensure we're in the right schema
		setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
		_, err = client.DB().ExecContext(ctx, setSearchPathSQL)
		require.NoError(t, err, "Failed to set search_path")
		
		_, err = client.DB().ExecContext(ctx, profileMigrationSQL)
		require.NoError(t, err, "Failed to apply profiles migration")

		// Ensure search_path is set on the DB connection before creating profile repo
		setSearchPathSQL2 := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
		_, err = client.DB().ExecContext(ctx, setSearchPathSQL2)
		require.NoError(t, err, "Failed to set search_path before transaction")

		profileRepo := profileRepo.NewPostgresProfileRepository(client)

		newUserID := uuid.Must(uuid.NewV4())
		newUserAuth := &models.UserAuth{
			ObjectId:    newUserID,
			Username:    "atomic@example.com",
			Password:    []byte("atomic_password"),
			Role:        "user",
			CreatedDate: now.Unix(),
			LastUpdated: now.Unix(),
		}

		newProfile := &profileModels.Profile{
			ObjectId:    newUserID,
			FullName:    "Atomic User",
			SocialName:  "atomicuser",
			Email:       "atomic@example.com",
			CreatedDate: now.Unix(),
			LastUpdated: now.Unix(),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Test successful transaction
		// Note: Profile repository doesn't support transactions via context yet,
		// so we'll test that user creation works in transaction, and profile creation works separately
		// For a true atomic test, we'd need to update ProfileRepository to support transactions
		err = authRepo.WithTransaction(ctx, func(txCtx context.Context) error {
			// Set search_path within transaction
			tx, ok := txCtx.Value("tx").(*sqlx.Tx)
			if !ok {
				return fmt.Errorf("transaction not found in context")
			}
			setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
			_, err := tx.ExecContext(txCtx, setSearchPathSQL)
			if err != nil {
				return fmt.Errorf("failed to set search_path in transaction: %w", err)
			}

			if err := authRepo.CreateUser(txCtx, newUserAuth); err != nil {
				return err
			}
			// Profile repository doesn't support transaction context, so we create it after
			// In a real implementation, ProfileRepository would also support transactions
			return nil
		})
		require.NoError(t, err, "Atomic user creation should succeed")

		// Create profile after transaction (since ProfileRepository doesn't support transactions yet)
		// This tests that the user was created successfully
		err = profileRepo.Create(ctx, newProfile)
		require.NoError(t, err, "Profile creation should succeed")
		require.NoError(t, err, "Atomic creation should succeed")

		// Verify both were created
		user, err := authRepo.FindByID(ctx, newUserID)
		require.NoError(t, err)
		require.NotNil(t, user)

		profile, err := profileRepo.FindByID(ctx, newUserID)
		require.NoError(t, err)
		require.NotNil(t, profile)
		require.Equal(t, "Atomic User", profile.FullName)
	})

	// 23. Test Transaction - Rollback on Failure
	t.Run("Transaction_RollbackOnFailure", func(t *testing.T) {
		rollbackUserID := uuid.Must(uuid.NewV4())
		rollbackUserAuth := &models.UserAuth{
			ObjectId:    rollbackUserID,
			Username:    "rollback@example.com",
			Password:    []byte("rollback_password"),
			Role:        "user",
			CreatedDate: now.Unix(),
			LastUpdated: now.Unix(),
		}

		// This profile will have a duplicate social_name to force a failure
		rollbackProfile := &profileModels.Profile{
			ObjectId:    rollbackUserID,
			FullName:    "Rollback User",
			SocialName:  "atomicuser", // Duplicate - will fail
			Email:       "rollback@example.com",
			CreatedDate: now.Unix(),
			LastUpdated: now.Unix(),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		profileRepo := profileRepo.NewPostgresProfileRepository(client)

		// Test transaction rollback
		err = authRepo.WithTransaction(ctx, func(txCtx context.Context) error {
			if err := authRepo.CreateUser(txCtx, rollbackUserAuth); err != nil {
				return err
			}
			// This should fail due to duplicate social_name
			if err := profileRepo.Create(txCtx, rollbackProfile); err != nil {
				return err
			}
			return nil
		})
		require.Error(t, err, "Transaction should fail")

		// Verify user was NOT created (rolled back)
		_, err = authRepo.FindByID(ctx, rollbackUserID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "user not found")
	})

	// 24. Test Delete
	t.Run("Delete", func(t *testing.T) {
		// Create a user first to delete
		deleteUserAuth := &models.UserAuth{
			ObjectId:    userID2,
			Username:    "delete@example.com",
			Password:    []byte("delete_password"),
			Role:        "user",
			CreatedDate: now.Unix(),
			LastUpdated: now.Unix(),
		}
		err := authRepo.CreateUser(ctx, deleteUserAuth)
		require.NoError(t, err, "Failed to create user for deletion test")

		// Verify it exists
		_, err = authRepo.FindByID(ctx, userID2)
		require.NoError(t, err, "User should exist before deletion")

		// Delete it
		err = authRepo.Delete(ctx, userID2)
		require.NoError(t, err)

		// Verify it's gone
		_, err = authRepo.FindByID(ctx, userID2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "user not found")
	})
}

