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
	"github.com/qolzam/telar/apps/api/posts/models"
)

// TestPostgresRepository_Integration validates the new PostgresRepository implementation
// This test focuses exclusively on the repository layer, bypassing the service layer.
func TestPostgresRepository_Integration(t *testing.T) {
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
	// Use the schema from isolated test config
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema // Ensure we use the isolated schema
	
	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer client.Close()

	// 3. Create schema if it doesn't exist (isolated test creates unique schema per test)
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err, "Failed to create schema")

	// 4. Set search_path to the isolated schema
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "Failed to set search_path")

	// 5. Apply Schema Manually
	// Read and execute the migration SQL
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS posts (
			id UUID PRIMARY KEY,
			owner_user_id UUID NOT NULL,
			post_type_id INT NOT NULL,
			body TEXT,
			score BIGINT DEFAULT 0,
			view_count BIGINT DEFAULT 0,
			comment_count BIGINT DEFAULT 0,
			is_deleted BOOLEAN DEFAULT FALSE,
			deleted_date BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			tags TEXT[],
			url_key VARCHAR(255),
			owner_display_name VARCHAR(255),
			owner_avatar VARCHAR(512),
			image VARCHAR(512),
			image_full_path VARCHAR(512),
			video VARCHAR(512),
			thumbnail VARCHAR(512),
			disable_comments BOOLEAN DEFAULT FALSE,
			disable_sharing BOOLEAN DEFAULT FALSE,
			permission VARCHAR(50) DEFAULT 'Public',
			version VARCHAR(50),
			metadata JSONB DEFAULT '{}'::jsonb
		);

		CREATE INDEX IF NOT EXISTS idx_posts_owner ON posts(owner_user_id);
		CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_posts_created_date ON posts(created_date DESC);
		CREATE INDEX IF NOT EXISTS idx_posts_tags ON posts USING GIN(tags);
		CREATE INDEX IF NOT EXISTS idx_posts_post_type ON posts(post_type_id);
		CREATE INDEX IF NOT EXISTS idx_posts_deleted ON posts(is_deleted) WHERE is_deleted = FALSE;
		CREATE INDEX IF NOT EXISTS idx_posts_url_key ON posts(url_key) WHERE url_key IS NOT NULL;
	`

	_, err = client.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply migration")

	// 6. Initialize Repository
	repo := NewPostgresRepository(client)

	// 7. Test Create
	t.Run("Create", func(t *testing.T) {
		postID := uuid.Must(uuid.NewV4())
		ownerID := uuid.Must(uuid.NewV4())
		now := time.Now()

		post := &models.Post{
			ObjectId:         postID,
			OwnerUserId:      ownerID,
			PostTypeId:       1,
			Body:             "Test post body",
			Score:            0,
			ViewCount:        0,
			CommentCounter:   0,
			Tags:             pq.StringArray{"test", "integration"},
			URLKey:           "test-post-url-key",
			OwnerDisplayName: "Test User",
			OwnerAvatar:      "https://example.com/avatar.jpg",
			Image:            "https://example.com/image.jpg",
			ImageFullPath:    "/full/path/image.jpg",
			Video:            "",
			Thumbnail:        "https://example.com/thumb.jpg",
			DisableComments:  false,
			DisableSharing:   false,
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      now.Unix(),
			LastUpdated:      now.Unix(),
			CreatedAt:        now,
			UpdatedAt:        now,
			Permission:       "Public",
			Version:          "1.0",
			Votes:            map[string]string{"user1": "up"},
			Album: &models.Album{
				Count:   3,
				Cover:   "cover.jpg",
				CoverId: uuid.Must(uuid.NewV4()),
				Photos:  []string{"photo1.jpg", "photo2.jpg", "photo3.jpg"},
				Title:   "Test Album",
			},
			AccessUserList: []string{"user1", "user2"},
		}

		err := repo.Create(ctx, post)
		require.NoError(t, err, "Failed to create post")
	})

	// 8. Test FindByID
	t.Run("FindByID", func(t *testing.T) {
		postID := uuid.Must(uuid.NewV4())
		ownerID := uuid.Must(uuid.NewV4())
		now := time.Now()

		// Create a post first
		post := &models.Post{
			ObjectId:         postID,
			OwnerUserId:      ownerID,
			PostTypeId:       1,
			Body:             "FindByID test post",
			Score:            10,
			ViewCount:        5,
			CommentCounter:   2,
			Tags:             pq.StringArray{"find", "by", "id"},
			URLKey:           "findbyid-test",
			OwnerDisplayName: "Find User",
			OwnerAvatar:      "avatar.jpg",
			CreatedDate:      now.Unix(),
			LastUpdated:      now.Unix(),
			CreatedAt:        now,
			UpdatedAt:        now,
			Permission:       "Public",
			Votes:            map[string]string{"user1": "up", "user2": "down"},
		}

		err := repo.Create(ctx, post)
		require.NoError(t, err)

		// Find it back
		fetched, err := repo.FindByID(ctx, postID)
		require.NoError(t, err, "Failed to find post by ID")
		require.NotNil(t, fetched, "Fetched post should not be nil")
		require.Equal(t, post.ObjectId, fetched.ObjectId, "Post ID should match")
		require.Equal(t, post.Body, fetched.Body, "Post body should match")
		require.Equal(t, post.Score, fetched.Score, "Post score should match")
		require.Equal(t, post.ViewCount, fetched.ViewCount, "Post view count should match")
		require.Equal(t, post.CommentCounter, fetched.CommentCounter, "Post comment count should match")
		require.Equal(t, len(post.Tags), len(fetched.Tags), "Tags length should match")
		require.Equal(t, post.Tags[0], fetched.Tags[0], "First tag should match")
		require.Equal(t, post.OwnerDisplayName, fetched.OwnerDisplayName, "Owner display name should match")
		require.Equal(t, post.Permission, fetched.Permission, "Permission should match")

		// Verify metadata fields (Votes, Album, AccessUserList) are populated
		require.NotNil(t, fetched.Votes, "Votes should not be nil")
		require.Equal(t, len(post.Votes), len(fetched.Votes), "Votes length should match")
		require.Equal(t, post.Votes["user1"], fetched.Votes["user1"], "Vote should match")
	})

	// 9. Test FindByUser
	t.Run("FindByUser", func(t *testing.T) {
		ownerID := uuid.Must(uuid.NewV4())
		now := time.Now()

		// Create multiple posts for the same user
		post1 := &models.Post{
			ObjectId:      uuid.Must(uuid.NewV4()),
			OwnerUserId:   ownerID,
			PostTypeId:    1,
			Body:          "User post 1",
			CreatedDate:   now.Unix(),
			LastUpdated:   now.Unix(),
			CreatedAt:     now,
			UpdatedAt:     now,
			Permission:    "Public",
		}

		post2 := &models.Post{
			ObjectId:      uuid.Must(uuid.NewV4()),
			OwnerUserId:   ownerID,
			PostTypeId:    1,
			Body:          "User post 2",
			CreatedDate:   now.Add(1 * time.Second).Unix(),
			LastUpdated:   now.Add(1 * time.Second).Unix(),
			CreatedAt:     now.Add(1 * time.Second),
			UpdatedAt:     now.Add(1 * time.Second),
			Permission:    "Public",
		}

		err := repo.Create(ctx, post1)
		require.NoError(t, err)
		err = repo.Create(ctx, post2)
		require.NoError(t, err)

		// Find posts by user
		posts, err := repo.FindByUser(ctx, ownerID, 10, 0)
		require.NoError(t, err, "Failed to find posts by user")
		require.GreaterOrEqual(t, len(posts), 2, "Should find at least 2 posts")

		// Verify posts are ordered by created_at DESC
		if len(posts) >= 2 {
			require.True(t, posts[0].CreatedAt.After(posts[1].CreatedAt) || posts[0].CreatedAt.Equal(posts[1].CreatedAt),
				"Posts should be ordered by created_at DESC")
		}
	})

	// 10. Test IncrementViewCount
	t.Run("IncrementViewCount", func(t *testing.T) {
		postID := uuid.Must(uuid.NewV4())
		ownerID := uuid.Must(uuid.NewV4())
		now := time.Now()

		post := &models.Post{
			ObjectId:      postID,
			OwnerUserId:   ownerID,
			PostTypeId:    1,
			Body:          "Increment test post",
			ViewCount:     5,
			CreatedDate:   now.Unix(),
			LastUpdated:   now.Unix(),
			CreatedAt:     now,
			UpdatedAt:     now,
			Permission:    "Public",
		}

		err := repo.Create(ctx, post)
		require.NoError(t, err)

		// Increment view count
		err = repo.IncrementViewCount(ctx, postID)
		require.NoError(t, err, "Failed to increment view count")

		// Verify increment
		fetched, err := repo.FindByID(ctx, postID)
		require.NoError(t, err)
		require.Equal(t, int64(6), fetched.ViewCount, "View count should be incremented from 5 to 6")

		// Increment again
		err = repo.IncrementViewCount(ctx, postID)
		require.NoError(t, err)
		fetched, err = repo.FindByID(ctx, postID)
		require.NoError(t, err)
		require.Equal(t, int64(7), fetched.ViewCount, "View count should be incremented to 7")
	})

	// 11. Test Update
	t.Run("Update", func(t *testing.T) {
		postID := uuid.Must(uuid.NewV4())
		ownerID := uuid.Must(uuid.NewV4())
		now := time.Now()

		post := &models.Post{
			ObjectId:      postID,
			OwnerUserId:   ownerID,
			PostTypeId:    1,
			Body:          "Original body",
			Score:         0,
			CreatedDate:   now.Unix(),
			LastUpdated:   now.Unix(),
			CreatedAt:     now,
			UpdatedAt:     now,
			Permission:    "Public",
		}

		err := repo.Create(ctx, post)
		require.NoError(t, err)

		// Update the post
		post.Body = "Updated body"
		post.Score = 100
		post.Tags = pq.StringArray{"updated", "tags"}

		err = repo.Update(ctx, post)
		require.NoError(t, err, "Failed to update post")

		// Verify update
		fetched, err := repo.FindByID(ctx, postID)
		require.NoError(t, err)
		require.Equal(t, "Updated body", fetched.Body, "Body should be updated")
		require.Equal(t, int64(100), fetched.Score, "Score should be updated")
		require.Equal(t, 2, len(fetched.Tags), "Tags should be updated")
		require.True(t, fetched.UpdatedAt.After(now), "UpdatedAt should be after creation time")
	})

	// 12. Test Delete (soft delete)
	t.Run("Delete", func(t *testing.T) {
		postID := uuid.Must(uuid.NewV4())
		ownerID := uuid.Must(uuid.NewV4())
		now := time.Now()

		post := &models.Post{
			ObjectId:      postID,
			OwnerUserId:   ownerID,
			PostTypeId:    1,
			Body:          "Post to delete",
			CreatedDate:   now.Unix(),
			LastUpdated:   now.Unix(),
			CreatedAt:     now,
			UpdatedAt:     now,
			Permission:    "Public",
		}

		err := repo.Create(ctx, post)
		require.NoError(t, err)

		// Delete the post
		err = repo.Delete(ctx, postID)
		require.NoError(t, err, "Failed to delete post")

		// Verify it's soft deleted (FindByID should not find it)
		_, err = repo.FindByID(ctx, postID)
		require.Error(t, err, "FindByID should fail for deleted post")
		require.Contains(t, err.Error(), "not found", "Error should indicate post not found")
	})

	// 13. Test FindByID with non-existent post
	t.Run("FindByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.Must(uuid.NewV4())
		_, err := repo.FindByID(ctx, nonExistentID)
		require.Error(t, err, "Should return error for non-existent post")
		require.Contains(t, err.Error(), "not found", "Error should indicate post not found")
	})
}

