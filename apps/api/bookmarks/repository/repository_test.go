package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepository_Integration(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("postgres not available, skipping")
	}

	ctx := context.Background()
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema

	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err)
	defer client.Close()

	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err)

	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err)

	migrationSQL := `
		CREATE TABLE IF NOT EXISTS user_auths (
			id UUID PRIMARY KEY
		);

		CREATE TABLE IF NOT EXISTS posts (
			id UUID PRIMARY KEY
		);

		CREATE TABLE IF NOT EXISTS bookmarks (
			post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (owner_user_id, post_id)
		);
		CREATE INDEX IF NOT EXISTS idx_bookmarks_owner_created ON bookmarks(owner_user_id, created_at DESC);
	`
	_, err = client.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err)

	repo := NewPostgresRepositoryWithSchema(client, iso.LegacyConfig.PGSchema)

	userID := uuid.Must(uuid.NewV4())
	postID := uuid.Must(uuid.NewV4())
	otherPostID := uuid.Must(uuid.NewV4())

	_, err = client.DB().ExecContext(ctx, `INSERT INTO user_auths (id) VALUES ($1)`, userID)
	require.NoError(t, err)
	_, err = client.DB().ExecContext(ctx, `INSERT INTO posts (id) VALUES ($1), ($2)`, postID, otherPostID)
	require.NoError(t, err)

	t.Run("AddBookmark idempotent", func(t *testing.T) {
		inserted, err := repo.AddBookmark(ctx, userID, postID)
		require.NoError(t, err)
		require.True(t, inserted)

		insertedAgain, err := repo.AddBookmark(ctx, userID, postID)
		require.NoError(t, err)
		require.False(t, insertedAgain)
	})

	t.Run("RemoveBookmark idempotent", func(t *testing.T) {
		removed, err := repo.RemoveBookmark(ctx, userID, postID)
		require.NoError(t, err)
		require.True(t, removed)

		removedAgain, err := repo.RemoveBookmark(ctx, userID, postID)
		require.NoError(t, err)
		require.False(t, removedAgain)
	})

	t.Run("GetMapByUserAndPosts", func(t *testing.T) {
		// Reinsert one bookmark
		_, err := repo.AddBookmark(ctx, userID, otherPostID)
		require.NoError(t, err)

		ids := []uuid.UUID{postID, otherPostID}
		result, err := repo.GetMapByUserAndPosts(ctx, userID, ids)
		require.NoError(t, err)

		require.Equal(t, false, result[postID])
		require.Equal(t, true, result[otherPostID])
	})

	t.Run("FindMyBookmarks_Pagination", func(t *testing.T) {
		// Clean up any existing bookmarks for this user from previous test cases
		_, _ = client.DB().ExecContext(ctx, `DELETE FROM bookmarks WHERE owner_user_id = $1`, userID)
		
		// seed exactly 15 bookmarks with small delays to ensure ordering
		bookIDs := make([]uuid.UUID, 15)
		for i := 0; i < 15; i++ {
			bookIDs[i] = uuid.Must(uuid.NewV4())
			_, err := client.DB().ExecContext(ctx, `INSERT INTO posts (id) VALUES ($1)`, bookIDs[i])
			require.NoError(t, err)
			_, err = repo.AddBookmark(ctx, userID, bookIDs[i])
			require.NoError(t, err)
			// Small delay to ensure distinct timestamps for reliable ordering
			if i < 14 {
				time.Sleep(1 * time.Millisecond)
			}
		}

		firstPage, cursor, err := repo.FindMyBookmarks(ctx, userID, "", 10)
		require.NoError(t, err)
		require.Len(t, firstPage, 10, "First page should have exactly 10 items")
		require.NotEmpty(t, cursor, "Should have a cursor for next page")

		secondPage, nextCursor, err := repo.FindMyBookmarks(ctx, userID, cursor, 10)
		require.NoError(t, err)
		require.Len(t, secondPage, 5, "Second page should have exactly 5 remaining items")
		require.Equal(t, "", nextCursor, "Should have no cursor when on last page")
		
		// Verify no duplicates between pages
		firstPageIDs := make(map[uuid.UUID]bool)
		for _, entry := range firstPage {
			firstPageIDs[entry.PostID] = true
		}
		for _, entry := range secondPage {
			require.False(t, firstPageIDs[entry.PostID], "Second page should not contain items from first page")
		}
	})
}
