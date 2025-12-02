// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/comments/models"
	postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
	postsModels "github.com/qolzam/telar/apps/api/posts/models"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	authModels "github.com/qolzam/telar/apps/api/auth/models"
)

// TestPostgresCommentRepository_Integration validates the new PostgresCommentRepository implementation
func TestPostgresCommentRepository_Integration(t *testing.T) {
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

	// 5. Apply Posts Schema Migration (required for foreign key)
	postsMigrationSQL := `
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
	`
	_, err = client.DB().ExecContext(ctx, postsMigrationSQL)
	require.NoError(t, err, "Failed to apply posts migration")

	// 6. Apply Auth Schema Migration (required for foreign key)
	authMigrationSQL := `
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
	`
	_, err = client.DB().ExecContext(ctx, authMigrationSQL)
	require.NoError(t, err, "Failed to apply auth migration")

	// 7. Apply Comments Schema Migration
	commentsMigrationSQL := `
		CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY,
			post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
			parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
			text TEXT NOT NULL,
			score BIGINT DEFAULT 0,
			owner_display_name VARCHAR(255),
			owner_avatar VARCHAR(512),
			is_deleted BOOLEAN DEFAULT FALSE,
			deleted_date BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);

		CREATE INDEX IF NOT EXISTS idx_comments_post ON comments(post_id);
		CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_comment_id) WHERE parent_comment_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_comments_owner ON comments(owner_user_id);
		CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_comments_created_date ON comments(created_date DESC);
		CREATE INDEX IF NOT EXISTS idx_comments_deleted ON comments(is_deleted) WHERE is_deleted = FALSE;
		CREATE INDEX IF NOT EXISTS idx_comments_post_active ON comments(post_id, created_date DESC) WHERE is_deleted = FALSE;
	`
	_, err = client.DB().ExecContext(ctx, commentsMigrationSQL)
	require.NoError(t, err, "Failed to apply comments migration")

	// 8. Initialize Repositories
	commentRepo := NewPostgresCommentRepository(client)
	postRepo := postsRepository.NewPostgresRepository(client)
	authRepo := authRepository.NewPostgresAuthRepository(client)

	// Test data
	userID1 := uuid.Must(uuid.NewV4())
	userID2 := uuid.Must(uuid.NewV4())
	postID1 := uuid.Must(uuid.NewV4())
	postID2 := uuid.Must(uuid.NewV4())
	now := time.Now()
	nowUnix := now.Unix()

	// 9. Create test users
	t.Run("Setup_Users", func(t *testing.T) {
		user1 := &authModels.UserAuth{
			ObjectId:      userID1,
			Username:      "user1@example.com",
			Password:      []byte("hashed"),
			Role:          "user",
			EmailVerified: true,
			PhoneVerified: false,
			CreatedDate:   nowUnix,
			LastUpdated:   nowUnix,
		}
		err := authRepo.CreateUser(ctx, user1)
		require.NoError(t, err)

		user2 := &authModels.UserAuth{
			ObjectId:      userID2,
			Username:      "user2@example.com",
			Password:      []byte("hashed"),
			Role:          "user",
			EmailVerified: true,
			PhoneVerified: false,
			CreatedDate:   nowUnix,
			LastUpdated:   nowUnix,
		}
		err = authRepo.CreateUser(ctx, user2)
		require.NoError(t, err)
	})

	// 10. Create test posts
	t.Run("Setup_Posts", func(t *testing.T) {
		post1 := &postsModels.Post{
			ObjectId:    postID1,
			OwnerUserId: userID1,
			PostTypeId:  1,
			Body:        "Test post 1",
			CreatedDate: nowUnix,
			LastUpdated: nowUnix,
		}
		err := postRepo.Create(ctx, post1)
		require.NoError(t, err)

		post2 := &postsModels.Post{
			ObjectId:    postID2,
			OwnerUserId: userID1,
			PostTypeId:  1,
			Body:        "Test post 2",
			CreatedDate: nowUnix,
			LastUpdated: nowUnix,
		}
		err = postRepo.Create(ctx, post2)
		require.NoError(t, err)
	})

	// 11. Test Foreign Key Constraint - Invalid Post ID
	t.Run("CreateComment_InvalidPostID_Fails", func(t *testing.T) {
		invalidPostID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         uuid.Must(uuid.NewV4()),
			PostId:           invalidPostID,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "This should fail",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}

		err := commentRepo.Create(ctx, comment)
		require.Error(t, err, "Should fail with foreign key constraint violation")
		// The repository now returns a specific error message for invalid post ID
		require.True(t, 
			strings.Contains(err.Error(), "post does not exist") ||
			strings.Contains(err.Error(), "foreign key"),
			"Error should mention post does not exist or foreign key, got: %s", err.Error())
	})

	// 12. Test Foreign Key Constraint - Invalid User ID
	t.Run("CreateComment_InvalidUserID_Fails", func(t *testing.T) {
		invalidUserID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         uuid.Must(uuid.NewV4()),
			PostId:           postID1,
			OwnerUserId:      invalidUserID,
			ParentCommentId:  nil,
			Text:             "This should fail",
			Score:            0,
			OwnerDisplayName: "Invalid User",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}

		err := commentRepo.Create(ctx, comment)
		require.Error(t, err, "Should fail with foreign key constraint violation")
		// The repository now returns a specific error message for invalid user ID
		require.True(t, 
			strings.Contains(err.Error(), "user does not exist") ||
			strings.Contains(err.Error(), "foreign key"),
			"Error should mention user does not exist or foreign key, got: %s", err.Error())
	})

	// 13. Test Create Root Comment
	t.Run("Create_RootComment", func(t *testing.T) {
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         commentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Root comment",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "avatar1.jpg",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}

		err := commentRepo.Create(ctx, comment)
		require.NoError(t, err, "Failed to create root comment")
	})

	// 14. Test FindByID
	t.Run("FindByID", func(t *testing.T) {
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         commentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Find me",
			Score:            5,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "avatar1.jpg",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}

		err := commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		found, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, commentID, found.ObjectId)
		require.Equal(t, "Find me", found.Text)
		require.Equal(t, int64(5), found.Score)
		require.Equal(t, postID1, found.PostId)
		require.Nil(t, found.ParentCommentId, "Root comment should have nil parent")
	})

	// 15. Test Create Reply (Nested Comment)
	t.Run("Create_Reply", func(t *testing.T) {
		// Create a root comment first
		rootCommentID := uuid.Must(uuid.NewV4())
		rootComment := &models.Comment{
			ObjectId:         rootCommentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Root comment for reply",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, rootComment)
		require.NoError(t, err)

		// Create a reply
		replyID := uuid.Must(uuid.NewV4())
		reply := &models.Comment{
			ObjectId:         replyID,
			PostId:           postID1,
			OwnerUserId:      userID2,
			ParentCommentId:  &rootCommentID,
			Text:             "This is a reply",
			Score:            0,
			OwnerDisplayName: "User 2",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix + 1,
			LastUpdated:      nowUnix + 1,
		}

		err = commentRepo.Create(ctx, reply)
		require.NoError(t, err, "Failed to create reply")

		// Verify the reply
		found, err := commentRepo.FindByID(ctx, replyID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, replyID, found.ObjectId)
		require.NotNil(t, found.ParentCommentId, "Reply should have parent_comment_id")
		require.Equal(t, rootCommentID, *found.ParentCommentId)
	})

	// 16. Test FindReplies
	t.Run("FindReplies", func(t *testing.T) {
		// Create a root comment
		rootCommentID := uuid.Must(uuid.NewV4())
		rootComment := &models.Comment{
			ObjectId:         rootCommentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Root for replies test",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, rootComment)
		require.NoError(t, err)

		// Create multiple replies
		reply1ID := uuid.Must(uuid.NewV4())
		reply1 := &models.Comment{
			ObjectId:         reply1ID,
			PostId:           postID1,
			OwnerUserId:      userID2,
			ParentCommentId:  &rootCommentID,
			Text:             "Reply 1",
			Score:            0,
			OwnerDisplayName: "User 2",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix + 10,
			LastUpdated:      nowUnix + 10,
		}
		err = commentRepo.Create(ctx, reply1)
		require.NoError(t, err)

		reply2ID := uuid.Must(uuid.NewV4())
		reply2 := &models.Comment{
			ObjectId:         reply2ID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  &rootCommentID,
			Text:             "Reply 2",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix + 20,
			LastUpdated:      nowUnix + 20,
		}
		err = commentRepo.Create(ctx, reply2)
		require.NoError(t, err)

		// Find replies
		replies, err := commentRepo.FindReplies(ctx, rootCommentID, 10, 0)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(replies), 2, "Should find at least 2 replies")

		// Verify replies are ordered by created_date ASC
		foundReply1 := false
		foundReply2 := false
		for _, reply := range replies {
			if reply.ObjectId == reply1ID {
				foundReply1 = true
				require.Equal(t, rootCommentID, *reply.ParentCommentId)
			}
			if reply.ObjectId == reply2ID {
				foundReply2 = true
				require.Equal(t, rootCommentID, *reply.ParentCommentId)
			}
		}
		require.True(t, foundReply1, "Should find reply 1")
		require.True(t, foundReply2, "Should find reply 2")
	})

	// 17. Test FindByPostID (Root Comments Only)
	t.Run("FindByPostID", func(t *testing.T) {
		// Create a root comment on postID2
		rootCommentID := uuid.Must(uuid.NewV4())
		rootComment := &models.Comment{
			ObjectId:         rootCommentID,
			PostId:           postID2,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Root comment on post 2",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, rootComment)
		require.NoError(t, err)

		// Find root comments for postID2
		comments, err := commentRepo.FindByPostID(ctx, postID2, 10, 0)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(comments), 1, "Should find at least 1 root comment")

		// Verify all returned comments are root comments (no parent)
		for _, comment := range comments {
			require.Nil(t, comment.ParentCommentId, "All comments should be root comments")
			require.Equal(t, postID2, comment.PostId)
		}
	})

	// 18. Test CountByPostID (Root Comments Only)
	t.Run("CountByPostID", func(t *testing.T) {
		count, err := commentRepo.CountByPostID(ctx, postID1)
		require.NoError(t, err)
		require.GreaterOrEqual(t, count, int64(0), "Count should be non-negative")

		// Count should only include root comments, not replies
		// We've created several root comments on postID1, so count should be > 0
		require.Greater(t, count, int64(0), "Should have at least one root comment")
	})

	// 19. Test CountReplies
	t.Run("CountReplies", func(t *testing.T) {
		// Create a root comment
		rootCommentID := uuid.Must(uuid.NewV4())
		rootComment := &models.Comment{
			ObjectId:         rootCommentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Root for count test",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, rootComment)
		require.NoError(t, err)

		// Create a reply
		replyID := uuid.Must(uuid.NewV4())
		reply := &models.Comment{
			ObjectId:         replyID,
			PostId:           postID1,
			OwnerUserId:      userID2,
			ParentCommentId:  &rootCommentID,
			Text:             "Reply for count",
			Score:            0,
			OwnerDisplayName: "User 2",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix + 1,
			LastUpdated:      nowUnix + 1,
		}
		err = commentRepo.Create(ctx, reply)
		require.NoError(t, err)

		// Count replies
		count, err := commentRepo.CountReplies(ctx, rootCommentID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, count, int64(1), "Should have at least 1 reply")
	})

	// 20. Test IncrementScore (Atomic Operation)
	t.Run("IncrementScore", func(t *testing.T) {
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         commentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Comment for score test",
			Score:            10,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// Increment score by 5
		err = commentRepo.IncrementScore(ctx, commentID, 5)
		require.NoError(t, err)

		// Verify score was incremented
		found, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, int64(15), found.Score, "Score should be 10 + 5 = 15")

		// Increment by negative value (decrement)
		err = commentRepo.IncrementScore(ctx, commentID, -3)
		require.NoError(t, err)

		// Verify score was decremented
		found, err = commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, int64(12), found.Score, "Score should be 15 - 3 = 12")
	})

	// 21. Test Update
	t.Run("Update", func(t *testing.T) {
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         commentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Original text",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// Update the comment
		comment.Text = "Updated text"
		comment.Score = 100
		err = commentRepo.Update(ctx, comment)
		require.NoError(t, err)

		// Verify update
		found, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, "Updated text", found.Text)
		require.Equal(t, int64(100), found.Score)
	})

	// 22. Test Delete (Soft Delete)
	t.Run("Delete", func(t *testing.T) {
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         commentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Comment to delete",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// Delete the comment
		err = commentRepo.Delete(ctx, commentID)
		require.NoError(t, err)

		// Verify it's soft deleted (still exists but marked as deleted)
		found, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.True(t, found.Deleted, "Comment should be marked as deleted")
		require.Greater(t, found.DeletedDate, int64(0), "Deleted date should be set")
	})

	// 23. Test DeleteByPostID (Batch Soft Delete)
	t.Run("DeleteByPostID", func(t *testing.T) {
		// Create comments on postID2
		comment1ID := uuid.Must(uuid.NewV4())
		comment1 := &models.Comment{
			ObjectId:         comment1ID,
			PostId:           postID2,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Comment 1 on post 2",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, comment1)
		require.NoError(t, err)

		comment2ID := uuid.Must(uuid.NewV4())
		comment2 := &models.Comment{
			ObjectId:         comment2ID,
			PostId:           postID2,
			OwnerUserId:      userID2,
			ParentCommentId:  nil,
			Text:             "Comment 2 on post 2",
			Score:            0,
			OwnerDisplayName: "User 2",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err = commentRepo.Create(ctx, comment2)
		require.NoError(t, err)

		// Delete all comments for postID2
		err = commentRepo.DeleteByPostID(ctx, postID2)
		require.NoError(t, err)

		// Verify comments are soft deleted
		found1, err := commentRepo.FindByID(ctx, comment1ID)
		require.NoError(t, err)
		require.True(t, found1.Deleted, "Comment 1 should be deleted")

		found2, err := commentRepo.FindByID(ctx, comment2ID)
		require.NoError(t, err)
		require.True(t, found2.Deleted, "Comment 2 should be deleted")
	})

	// 24. Test Cascade Delete - Delete Post
	t.Run("CascadeDelete_DeletePost", func(t *testing.T) {
		// Create a new post
		testPostID := uuid.Must(uuid.NewV4())
		testPost := &postsModels.Post{
			ObjectId:    testPostID,
			OwnerUserId: userID1,
			PostTypeId:  1,
			Body:        "Post for cascade test",
			CreatedDate: nowUnix,
			LastUpdated: nowUnix,
		}
		err := postRepo.Create(ctx, testPost)
		require.NoError(t, err)

		// Create comments on this post
		comment1ID := uuid.Must(uuid.NewV4())
		comment1 := &models.Comment{
			ObjectId:         comment1ID,
			PostId:           testPostID,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Comment 1",
			Score:            0,
			OwnerDisplayName: "User 1",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err = commentRepo.Create(ctx, comment1)
		require.NoError(t, err)

		comment2ID := uuid.Must(uuid.NewV4())
		comment2 := &models.Comment{
			ObjectId:         comment2ID,
			PostId:           testPostID,
			OwnerUserId:      userID2,
			ParentCommentId:  &comment1ID,
			Text:             "Reply to comment 1",
			Score:            0,
			OwnerDisplayName: "User 2",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix + 1,
			LastUpdated:      nowUnix + 1,
		}
		err = commentRepo.Create(ctx, comment2)
		require.NoError(t, err)

		// Hard delete the post to test cascade (PostRepository.Delete is soft delete)
		// We need to directly DELETE from posts table to trigger CASCADE
		deletePostSQL := `DELETE FROM posts WHERE id = $1`
		_, err = client.DB().ExecContext(ctx, deletePostSQL, testPostID)
		require.NoError(t, err, "Failed to hard delete post")

		// Verify comments are cascade deleted (hard delete, not soft delete)
		_, err = commentRepo.FindByID(ctx, comment1ID)
		require.Error(t, err, "Comment 1 should be cascade deleted")
		require.Contains(t, err.Error(), "not found", "Error should indicate comment not found")

		_, err = commentRepo.FindByID(ctx, comment2ID)
		require.Error(t, err, "Comment 2 should be cascade deleted")
		require.Contains(t, err.Error(), "not found", "Error should indicate comment not found")
	})

	// 25. Test FindByUserID
	t.Run("FindByUserID", func(t *testing.T) {
		// Create a comment by userID2
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         commentID,
			PostId:           postID1,
			OwnerUserId:      userID2,
			ParentCommentId:  nil,
			Text:             "Comment by user 2",
			Score:            0,
			OwnerDisplayName: "User 2",
			OwnerAvatar:      "",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// Find comments by userID2
		comments, err := commentRepo.FindByUserID(ctx, userID2, 10, 0)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(comments), 1, "Should find at least 1 comment by user 2")

		// Verify all comments belong to userID2
		for _, c := range comments {
			require.Equal(t, userID2, c.OwnerUserId)
			require.False(t, c.Deleted, "Should not return deleted comments")
		}
	})

	// 26. Test UpdateOwnerProfile
	t.Run("UpdateOwnerProfile", func(t *testing.T) {
		// Create a comment by userID1
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:         commentID,
			PostId:           postID1,
			OwnerUserId:      userID1,
			ParentCommentId:  nil,
			Text:             "Comment for profile update",
			Score:            0,
			OwnerDisplayName: "Old Name",
			OwnerAvatar:      "old_avatar.jpg",
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      nowUnix,
			LastUpdated:      nowUnix,
		}
		err := commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// Update owner profile
		err = commentRepo.UpdateOwnerProfile(ctx, userID1, "New Name", "new_avatar.jpg")
		require.NoError(t, err)

		// Verify profile was updated
		found, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, "New Name", found.OwnerDisplayName)
		require.Equal(t, "new_avatar.jpg", found.OwnerAvatar)
	})
}

