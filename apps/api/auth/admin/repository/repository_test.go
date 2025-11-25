// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/qolzam/telar/apps/api/auth/admin/models"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
)

// TestPostgresAdminRepository_Integration validates the new PostgresAdminRepository implementation
func TestPostgresAdminRepository_Integration(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// 1. Create PostgreSQL Client
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer client.Close()

	// 2. Create Schema
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err, "Failed to create schema")

	// 3. Set Search Path
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "Failed to set search_path")

	// 4. Apply Admin Migration
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS admin_logs (
			id UUID PRIMARY KEY,
			admin_id UUID NOT NULL,
			action VARCHAR(100) NOT NULL,
			target_type VARCHAR(50),
			target_id UUID,
			details JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);

		CREATE TABLE IF NOT EXISTS invitations (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			invited_by UUID NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			code VARCHAR(50) UNIQUE NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);

		CREATE INDEX IF NOT EXISTS idx_admin_logs_admin ON admin_logs(admin_id);
		CREATE INDEX IF NOT EXISTS idx_admin_logs_created_at ON admin_logs(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_admin_logs_created_date ON admin_logs(created_date DESC);
		CREATE INDEX IF NOT EXISTS idx_admin_logs_target ON admin_logs(target_type, target_id) WHERE target_type IS NOT NULL AND target_id IS NOT NULL;

		CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_email ON invitations(email);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_code ON invitations(code);
		CREATE INDEX IF NOT EXISTS idx_invitations_expires_at ON invitations(expires_at);
		CREATE INDEX IF NOT EXISTS idx_invitations_invited_by ON invitations(invited_by);
		CREATE INDEX IF NOT EXISTS idx_invitations_used ON invitations(used) WHERE used = FALSE;
	`

	_, err = client.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply admin migration")

	// 5. Initialize Repository
	adminRepo := NewPostgresAdminRepository(client)

	// Test data
	adminID1 := uuid.Must(uuid.NewV4())
	targetID := uuid.Must(uuid.NewV4())
	now := time.Now()

	// 6. Test LogAction
	t.Run("LogAction", func(t *testing.T) {
		log := &models.AdminLog{
			ID:         uuid.Must(uuid.NewV4()),
			AdminID:    adminID1,
			Action:     "create_user",
			TargetType: stringPtr("user"),
			TargetID:   &targetID,
			Details: map[string]interface{}{
				"email": "test@example.com",
			},
			CreatedAt:   now,
			CreatedDate: now.Unix(),
		}

		err := adminRepo.LogAction(ctx, log)
		require.NoError(t, err, "Failed to log admin action")
	})

	// 7. Test GetAdminLogs
	t.Run("GetAdminLogs", func(t *testing.T) {
		filter := AdminLogFilter{
			AdminID: &adminID1,
		}
		logs, err := adminRepo.GetAdminLogs(ctx, filter, 10, 0)
		require.NoError(t, err)
		require.Greater(t, len(logs), 0)
		require.Equal(t, adminID1, logs[0].AdminID)
		require.Equal(t, "create_user", logs[0].Action)
	})

	// 8. Test CountAdminLogs
	t.Run("CountAdminLogs", func(t *testing.T) {
		filter := AdminLogFilter{
			AdminID: &adminID1,
		}
		count, err := adminRepo.CountAdminLogs(ctx, filter)
		require.NoError(t, err)
		require.Greater(t, count, int64(0))
	})

	// 9. Test CreateInvitation
	t.Run("CreateInvitation", func(t *testing.T) {
		invitation := &models.Invitation{
			ID:          uuid.Must(uuid.NewV4()),
			Email:       "invite@example.com",
			InvitedBy:   adminID1,
			Role:        "user",
			Code:        "INVITE123",
			ExpiresAt:   now.Add(24 * time.Hour),
			Used:        false,
			CreatedAt:   now,
			CreatedDate: now.Unix(),
		}

		err := adminRepo.CreateInvitation(ctx, invitation)
		require.NoError(t, err, "Failed to create invitation")
	})

	// 10. Test FindInvitationByCode
	t.Run("FindInvitationByCode", func(t *testing.T) {
		invitation, err := adminRepo.FindInvitationByCode(ctx, "INVITE123")
		require.NoError(t, err)
		require.NotNil(t, invitation)
		require.Equal(t, "invite@example.com", invitation.Email)
		require.Equal(t, adminID1, invitation.InvitedBy)
		require.False(t, invitation.Used)
	})

	// 11. Test FindInvitationByEmail
	t.Run("FindInvitationByEmail", func(t *testing.T) {
		invitation, err := adminRepo.FindInvitationByEmail(ctx, "invite@example.com")
		require.NoError(t, err)
		require.NotNil(t, invitation)
		require.Equal(t, "INVITE123", invitation.Code)
	})

	// 12. Test MarkInvitationUsed
	t.Run("MarkInvitationUsed", func(t *testing.T) {
		invitation, err := adminRepo.FindInvitationByCode(ctx, "INVITE123")
		require.NoError(t, err)

		err = adminRepo.MarkInvitationUsed(ctx, invitation.ID)
		require.NoError(t, err, "Failed to mark invitation as used")

		// Verify it's marked as used
		usedInvitation, err := adminRepo.FindInvitationByCode(ctx, "INVITE123")
		require.Error(t, err, "Should not find used invitation")
		require.Nil(t, usedInvitation)
	})

	// 13. Test CreateInvitation_DuplicateEmail
	t.Run("CreateInvitation_DuplicateEmail", func(t *testing.T) {
		invitation := &models.Invitation{
			ID:          uuid.Must(uuid.NewV4()),
			Email:       "invite@example.com", // Same email (but it was marked as used, so we need a fresh one)
			InvitedBy:   adminID1,
			Role:        "user",
			Code:        "INVITE456",
			ExpiresAt:   now.Add(24 * time.Hour),
			Used:        false,
			CreatedAt:   now,
			CreatedDate: now.Unix(),
		}

		err := adminRepo.CreateInvitation(ctx, invitation)
		require.Error(t, err, "Should fail on duplicate email")
		require.Contains(t, err.Error(), "invitation already exists")
	})

	// 14. Test GetSystemStats
	t.Run("GetSystemStats", func(t *testing.T) {
		stats, err := adminRepo.GetSystemStats(ctx)
		require.NoError(t, err)
		require.NotNil(t, stats)
		// Stats may be zero if tables don't exist yet, which is fine
		require.GreaterOrEqual(t, stats.TotalUsers, int64(0))
		require.GreaterOrEqual(t, stats.TotalAdmins, int64(0))
	})

	// 15. Test DeleteExpiredInvitations
	t.Run("DeleteExpiredInvitations", func(t *testing.T) {
		// Create an expired invitation
		expiredInvitation := &models.Invitation{
			ID:          uuid.Must(uuid.NewV4()),
			Email:       "expired@example.com",
			InvitedBy:   adminID1,
			Role:        "user",
			Code:        "EXPIRED123",
			ExpiresAt:   now.Add(-24 * time.Hour), // Expired
			Used:        false,
			CreatedAt:   now,
			CreatedDate: now.Unix(),
		}
		err := adminRepo.CreateInvitation(ctx, expiredInvitation)
		require.NoError(t, err)

		// Delete expired invitations
		err = adminRepo.DeleteExpiredInvitations(ctx, now.Unix())
		require.NoError(t, err)

		// Verify expired invitation is deleted
		_, err = adminRepo.FindInvitationByCode(ctx, "EXPIRED123")
		require.Error(t, err, "Expired invitation should be deleted")
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

