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
	"github.com/stretchr/testify/require"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/votes/models"
)

// TestPostgresVoteRepository_Integration validates the new PostgresVoteRepository implementation
// This test focuses exclusively on the repository layer, bypassing the service layer.
func TestPostgresVoteRepository_Integration(t *testing.T) {
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

	// 5. Apply Schema Manually - votes table migration
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS votes (
			id UUID PRIMARY KEY,
			post_id UUID NOT NULL,
			owner_user_id UUID NOT NULL,
			vote_type_id SMALLINT NOT NULL CHECK (vote_type_id IN (1, 2)),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_unique_user_post ON votes(post_id, owner_user_id);
		CREATE INDEX IF NOT EXISTS idx_votes_post_id ON votes(post_id);
		CREATE INDEX IF NOT EXISTS idx_votes_owner_user_id ON votes(owner_user_id);
		CREATE INDEX IF NOT EXISTS idx_votes_vote_type_id ON votes(vote_type_id);
	`

	_, err = client.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply votes migration")

	// 6. Initialize Repository with schema
	repo := NewPostgresVoteRepositoryWithSchema(client, iso.LegacyConfig.PGSchema)

	// Test data
	postID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())

	// Test 1: User votes Up (Success)
	t.Run("User votes Up - Success", func(t *testing.T) {
		voteID := uuid.Must(uuid.NewV4())
		vote := &models.Vote{
			ID:          voteID,
			PostID:      postID,
			OwnerUserID: userID,
			VoteTypeID:  models.VoteTypeUp,
			CreatedAt:   time.Now(),
		}

		created, previousType, err := repo.Upsert(ctx, vote)
		require.NoError(t, err, "Failed to upsert vote")
		require.True(t, created, "Vote should be created (not updated)")
		require.Equal(t, 0, previousType, "Previous vote type should be 0 for new vote")

		// Verify the vote was created
		fetched, err := repo.FindByUserAndPost(ctx, userID, postID)
		require.NoError(t, err, "Failed to find vote")
		require.NotNil(t, fetched, "Fetched vote should not be nil")
		require.Equal(t, voteID, fetched.ID, "Vote ID should match")
		require.Equal(t, postID, fetched.PostID, "Post ID should match")
		require.Equal(t, userID, fetched.OwnerUserID, "User ID should match")
		require.Equal(t, models.VoteTypeUp, fetched.VoteTypeID, "Vote type should be Up")
	})

	// Test 2: User votes Up again (No change)
	t.Run("User votes Up again - No change", func(t *testing.T) {
		voteID := uuid.Must(uuid.NewV4())
		vote := &models.Vote{
			ID:          voteID,
			PostID:      postID,
			OwnerUserID: userID,
			VoteTypeID:  models.VoteTypeUp, // Same vote type
			CreatedAt:   time.Now(),
		}

		created, previousType, err := repo.Upsert(ctx, vote)
		require.NoError(t, err, "Failed to upsert vote")
		require.False(t, created, "Vote should be updated (not created)")
		require.Equal(t, models.VoteTypeUp, previousType, "Previous vote type should be Up")

		// Verify the vote still exists with same type
		fetched, err := repo.FindByUserAndPost(ctx, userID, postID)
		require.NoError(t, err, "Failed to find vote")
		require.NotNil(t, fetched, "Fetched vote should not be nil")
		require.Equal(t, models.VoteTypeUp, fetched.VoteTypeID, "Vote type should still be Up")
	})

	// Test 3: User switches to Down (Update works)
	t.Run("User switches to Down - Update works", func(t *testing.T) {
		voteID := uuid.Must(uuid.NewV4())
		vote := &models.Vote{
			ID:          voteID,
			PostID:      postID,
			OwnerUserID: userID,
			VoteTypeID:  models.VoteTypeDown, // Switch to Down
			CreatedAt:   time.Now(),
		}

		created, previousType, err := repo.Upsert(ctx, vote)
		require.NoError(t, err, "Failed to upsert vote")
		require.False(t, created, "Vote should be updated (not created)")
		require.Equal(t, models.VoteTypeUp, previousType, "Previous vote type should be Up")

		// Verify the vote was updated to Down
		fetched, err := repo.FindByUserAndPost(ctx, userID, postID)
		require.NoError(t, err, "Failed to find vote")
		require.NotNil(t, fetched, "Fetched vote should not be nil")
		require.Equal(t, models.VoteTypeDown, fetched.VoteTypeID, "Vote type should be Down")
	})

	// Test 4: Delete vote (Toggle off)
	t.Run("Delete vote - Toggle off", func(t *testing.T) {
		deleted, previousType, err := repo.Delete(ctx, postID, userID)
		require.NoError(t, err, "Failed to delete vote")
		require.True(t, deleted, "Vote should be deleted")
		require.Equal(t, models.VoteTypeDown, previousType, "Previous vote type should be Down")

		// Verify the vote was deleted
		_, err = repo.FindByUserAndPost(ctx, userID, postID)
		require.Error(t, err, "Vote should not be found after deletion")
	})

	// Test 5: Delete non-existent vote
	t.Run("Delete non-existent vote", func(t *testing.T) {
		deleted, previousType, err := repo.Delete(ctx, postID, userID)
		require.NoError(t, err, "Delete should not error for non-existent vote")
		require.False(t, deleted, "Vote should not be deleted (didn't exist)")
		require.Equal(t, 0, previousType, "Previous vote type should be 0 for non-existent vote")
	})

	// Test 6: Unique constraint - prevent duplicate votes
	t.Run("Unique constraint - prevent duplicate votes", func(t *testing.T) {
		// Create a vote
		voteID1 := uuid.Must(uuid.NewV4())
		vote1 := &models.Vote{
			ID:          voteID1,
			PostID:      postID,
			OwnerUserID: userID,
			VoteTypeID:  models.VoteTypeUp,
			CreatedAt:   time.Now(),
		}

		created, _, err := repo.Upsert(ctx, vote1)
		require.NoError(t, err, "Failed to upsert first vote")
		require.True(t, created, "First vote should be created")

		// Try to create another vote with same post_id and user_id (should update, not create duplicate)
		voteID2 := uuid.Must(uuid.NewV4())
		vote2 := &models.Vote{
			ID:          voteID2,
			PostID:      postID,
			OwnerUserID: userID,
			VoteTypeID:  models.VoteTypeDown,
			CreatedAt:   time.Now(),
		}

		created2, previousType, err := repo.Upsert(ctx, vote2)
		require.NoError(t, err, "Failed to upsert second vote")
		require.False(t, created2, "Second vote should update existing vote (not create duplicate)")
		require.Equal(t, models.VoteTypeUp, previousType, "Previous vote type should be Up")

		// Verify only one vote exists
		fetched, err := repo.FindByUserAndPost(ctx, userID, postID)
		require.NoError(t, err, "Failed to find vote")
		require.NotNil(t, fetched, "Fetched vote should not be nil")
		require.Equal(t, models.VoteTypeDown, fetched.VoteTypeID, "Vote type should be Down")
	})

	// Test 7: FindByUserAndPost for non-existent vote
	t.Run("FindByUserAndPost - non-existent vote", func(t *testing.T) {
		nonExistentPostID := uuid.Must(uuid.NewV4())
		nonExistentUserID := uuid.Must(uuid.NewV4())

		_, err := repo.FindByUserAndPost(ctx, nonExistentUserID, nonExistentPostID)
		require.Error(t, err, "Should error when vote not found")
		require.Contains(t, err.Error(), "vote not found", "Error message should indicate vote not found")
	})
}

