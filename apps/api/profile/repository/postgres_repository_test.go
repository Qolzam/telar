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
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/profile/models"
)

// TestPostgresProfileRepository_Integration validates the new PostgresProfileRepository implementation
// This test focuses exclusively on the repository layer, bypassing the service layer.
func TestPostgresProfileRepository_Integration(t *testing.T) {
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

	// 5. Apply Schema Manually
	migrationSQL := `
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
		CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles(email) WHERE email IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON profiles(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_profiles_created_date ON profiles(created_date DESC);
	`

	_, err = client.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply migration")

	// 6. Initialize Repository
	repo := NewPostgresProfileRepository(client)

	// Test data
	userID1 := uuid.Must(uuid.NewV4())
	userID2 := uuid.Must(uuid.NewV4())
	now := time.Now()

	// 7. Test Create
	t.Run("Create", func(t *testing.T) {
		profile := &models.Profile{
			ObjectId:      userID1,
			FullName:      "Test User",
			SocialName:    "testuser",
			Email:         "test@example.com",
			Avatar:        "https://example.com/avatar.jpg",
			Banner:        "https://example.com/banner.jpg",
			Tagline:       "Test tagline",
			CreatedDate:    now.Unix(),
			LastUpdated:   now.Unix(),
			LastSeen:      now.Unix(),
			VoteCount:     10,
			ShareCount:    5,
			FollowCount:   20,
			FollowerCount: 15,
			PostCount:     3,
			AccessUserList: pq.StringArray{"user1", "user2"},
			Permission:    "Public",
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		err := repo.Create(ctx, profile)
		require.NoError(t, err, "Failed to create profile")
	})

	// 8. Test FindByID
	t.Run("FindByID", func(t *testing.T) {
		fetched, err := repo.FindByID(ctx, userID1)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.Equal(t, userID1, fetched.ObjectId)
		require.Equal(t, "Test User", fetched.FullName)
		require.Equal(t, "testuser", fetched.SocialName)
		require.Equal(t, int64(10), fetched.VoteCount)
		require.Equal(t, []string{"user1", "user2"}, []string(fetched.AccessUserList))
	})

	// 9. Test FindBySocialName
	t.Run("FindBySocialName", func(t *testing.T) {
		fetched, err := repo.FindBySocialName(ctx, "testuser")
		require.NoError(t, err)
		require.NotNil(t, fetched)
		require.Equal(t, userID1, fetched.ObjectId)
		require.Equal(t, "testuser", fetched.SocialName)
	})

	// 10. Test Create with duplicate social_name (unique constraint)
	t.Run("Create_DuplicateSocialName", func(t *testing.T) {
		profile := &models.Profile{
			ObjectId:   uuid.Must(uuid.NewV4()),
			SocialName: "testuser", // Same as above
			FullName:   "Another User",
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		err := repo.Create(ctx, profile)
		require.Error(t, err)
		require.Contains(t, err.Error(), "social name already exists")
	})

	// 11. Test FindByIDs
	t.Run("FindByIDs", func(t *testing.T) {
		// Create second profile
		profile2 := &models.Profile{
			ObjectId:   userID2,
			FullName:   "Second User",
			SocialName: "seconduser",
			Email:      "second@example.com",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		err := repo.Create(ctx, profile2)
		require.NoError(t, err)

		profiles, err := repo.FindByIDs(ctx, []uuid.UUID{userID1, userID2})
		require.NoError(t, err)
		require.Len(t, profiles, 2)
	})

	// 12. Test Update
	t.Run("Update", func(t *testing.T) {
		profile, err := repo.FindByID(ctx, userID1)
		require.NoError(t, err)

		profile.FullName = "Updated Name"
		profile.Tagline = "Updated tagline"
		profile.VoteCount = 20

		err = repo.Update(ctx, profile)
		require.NoError(t, err)

		fetched, err := repo.FindByID(ctx, userID1)
		require.NoError(t, err)
		require.Equal(t, "Updated Name", fetched.FullName)
		require.Equal(t, "Updated tagline", fetched.Tagline)
		require.Equal(t, int64(20), fetched.VoteCount)
	})

	// 13. Test UpdateLastSeen
	t.Run("UpdateLastSeen", func(t *testing.T) {
		// Get current last_seen
		profile, err := repo.FindByID(ctx, userID1)
		require.NoError(t, err)
		originalLastSeen := profile.LastSeen

		// Wait a moment to ensure timestamp difference
		time.Sleep(100 * time.Millisecond)

		err = repo.UpdateLastSeen(ctx, userID1)
		require.NoError(t, err)

		fetched, err := repo.FindByID(ctx, userID1)
		require.NoError(t, err)
		require.GreaterOrEqual(t, fetched.LastSeen, originalLastSeen)
	})

	// 14. Test UpdateOwnerProfile
	t.Run("UpdateOwnerProfile", func(t *testing.T) {
		err := repo.UpdateOwnerProfile(ctx, userID1, "New Display Name", "https://example.com/new-avatar.jpg")
		require.NoError(t, err)

		fetched, err := repo.FindByID(ctx, userID1)
		require.NoError(t, err)
		require.Equal(t, "New Display Name", fetched.FullName)
		require.Equal(t, "https://example.com/new-avatar.jpg", fetched.Avatar)
	})

	// 15. Test Find with search
	t.Run("Find_WithSearch", func(t *testing.T) {
		searchText := "Updated"
		filter := ProfileFilter{
			SearchText: &searchText,
		}

		profiles, err := repo.Find(ctx, filter, 10, 0)
		require.NoError(t, err)
		require.Greater(t, len(profiles), 0)
	})

	// 16. Test Count
	t.Run("Count", func(t *testing.T) {
		filter := ProfileFilter{}
		count, err := repo.Count(ctx, filter)
		require.NoError(t, err)
		require.GreaterOrEqual(t, count, int64(2))
	})

	// 17. Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, userID2)
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, userID2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "profile not found")
	})
}

