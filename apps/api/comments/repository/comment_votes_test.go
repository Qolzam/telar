// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

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
	"golang.org/x/crypto/bcrypt"
)

// TestCommentVotes_Integration tests the comment voting functionality
func TestCommentVotes_Integration(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// Setup postgres client
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer client.Close()

	// Create schema and set search_path
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err)
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err)

	// Apply migrations
	applyMigrations(t, ctx, client.DB(), iso.LegacyConfig.PGSchema)

	// Create repositories
	commentRepo := NewPostgresCommentRepositoryWithSchema(client, iso.LegacyConfig.PGSchema)
	authRepo := authRepository.NewPostgresAuthRepository(client)
	postRepo := postsRepository.NewPostgresRepositoryWithSchema(client, iso.LegacyConfig.PGSchema)

	// Test 1: Cascade Delete (User)
	t.Run("Cascade_Delete_User", func(t *testing.T) {
		// Create User A
		userAID := uuid.Must(uuid.NewV4())
		passHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		userA := &authModels.UserAuth{
			ObjectId:      userAID,
			Username:      fmt.Sprintf("user-a-%s@test.com", userAID.String()[:8]),
			Password:      passHash,
			Role:          "user",
			EmailVerified: true,
		}
		err := authRepo.CreateUser(ctx, userA)
		require.NoError(t, err)

		// Create Post P
		postID := uuid.Must(uuid.NewV4())
		post := &postsModels.Post{
			ObjectId:     postID,
		OwnerUserId:  userAID,
		PostTypeId:   1,
		Body:         "Test post",
		Score:        0,
		CommentCounter: 0,
		}
		err = postRepo.Create(ctx, post)
		require.NoError(t, err)

		// Create Comment C
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:    commentID,
			PostId:      postID,
			OwnerUserId: userAID,
			Text:        "Test comment",
			Score:       0,
		}
		err = commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// User A likes Comment C
		created, err := commentRepo.AddVote(ctx, commentID, userAID)
		require.NoError(t, err)
		require.True(t, created, "Vote should be created")

		// Verify vote exists and score is 1
		err = commentRepo.IncrementScore(ctx, commentID, 1)
		require.NoError(t, err)
		updatedComment, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, int64(1), updatedComment.Score, "Score should be 1 after like")

		// Verify vote row exists
		var voteCount int
		err = client.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM comment_votes WHERE comment_id = $1 AND owner_user_id = $2`,
			commentID, userAID).Scan(&voteCount)
		require.NoError(t, err)
		require.Equal(t, 1, voteCount, "Vote row should exist")

		// Delete User A
		err = authRepo.Delete(ctx, userAID)
		require.NoError(t, err)

		// Verify vote row is GONE (cascade delete)
		err = client.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM comment_votes WHERE comment_id = $1 AND owner_user_id = $2`,
			commentID, userAID).Scan(&voteCount)
		require.NoError(t, err)
		require.Equal(t, 0, voteCount, "Vote row should be deleted when user is deleted")
	})

	// Test 2: Cascade Delete (Comment)
	t.Run("Cascade_Delete_Comment", func(t *testing.T) {
		// Create User B
		userBID := uuid.Must(uuid.NewV4())
		passHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		userB := &authModels.UserAuth{
			ObjectId:      userBID,
			Username:      fmt.Sprintf("user-b-%s@test.com", userBID.String()[:8]),
			Password:      passHash,
			Role:          "user",
			EmailVerified: true,
		}
		err := authRepo.CreateUser(ctx, userB)
		require.NoError(t, err)

		// Create Post P
		postID := uuid.Must(uuid.NewV4())
		post := &postsModels.Post{
			ObjectId:     postID,
			OwnerUserId:  userBID,
			PostTypeId:   1,
			Body:         "Test post",
			Score:        0,
			CommentCounter: 0,
		}
		err = postRepo.Create(ctx, post)
		require.NoError(t, err)

		// Create Comment C
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:    commentID,
			PostId:      postID,
			OwnerUserId: userBID,
			Text:        "Test comment",
			Score:       0,
		}
		err = commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// User B likes Comment C
		created, err := commentRepo.AddVote(ctx, commentID, userBID)
		require.NoError(t, err)
		require.True(t, created)

		// Verify vote row exists
		var voteCount int
		err = client.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM comment_votes WHERE comment_id = $1 AND owner_user_id = $2`,
			commentID, userBID).Scan(&voteCount)
		require.NoError(t, err)
		require.Equal(t, 1, voteCount, "Vote row should exist")

		// Hard delete Comment C (ON DELETE CASCADE only works on hard deletes, not soft deletes)
		// Use direct SQL to test the cascade behavior
		_, err = client.DB().ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, commentID)
		require.NoError(t, err)

		// Verify vote row is GONE (cascade delete)
		err = client.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM comment_votes WHERE comment_id = $1 AND owner_user_id = $2`,
			commentID, userBID).Scan(&voteCount)
		require.NoError(t, err)
		require.Equal(t, 0, voteCount, "Vote row should be deleted when comment is hard-deleted")
	})

	// Test 3: Idempotency (Double Click Fix)
	t.Run("Idempotency_DoubleClick", func(t *testing.T) {
		// Create User A
		userAID := uuid.Must(uuid.NewV4())
		passHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		userA := &authModels.UserAuth{
			ObjectId:      userAID,
			Username:      fmt.Sprintf("user-idemp-%s@test.com", userAID.String()[:8]),
			Password:      passHash,
			Role:          "user",
			EmailVerified: true,
		}
		err := authRepo.CreateUser(ctx, userA)
		require.NoError(t, err)

		// Create Post P
		postID := uuid.Must(uuid.NewV4())
		post := &postsModels.Post{
			ObjectId:     postID,
		OwnerUserId:  userAID,
		PostTypeId:   1,
		Body:         "Test post",
		Score:        0,
		CommentCounter: 0,
		}
		err = postRepo.Create(ctx, post)
		require.NoError(t, err)

		// Create Comment C
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:    commentID,
			PostId:      postID,
			OwnerUserId: userAID,
			Text:        "Test comment",
			Score:       0,
		}
		err = commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// User A likes Comment C (first time)
		created, err := commentRepo.AddVote(ctx, commentID, userAID)
		require.NoError(t, err)
		require.True(t, created, "First vote should be created")

		// Increment score
		err = commentRepo.IncrementScore(ctx, commentID, 1)
		require.NoError(t, err)

		// Verify score is 1
		updatedComment, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, int64(1), updatedComment.Score, "Score should be 1 after first like")

		// User A likes Comment C again (simulate network retry)
		created2, err := commentRepo.AddVote(ctx, commentID, userAID)
		require.NoError(t, err)
		require.False(t, created2, "Second vote should return false (already exists)")

		// Verify score remains 1 (not double counted)
		updatedComment2, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, int64(1), updatedComment2.Score, "Score should remain 1 (not double counted)")

		// Verify only one vote row exists
		var voteCount int
		err = client.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM comment_votes WHERE comment_id = $1 AND owner_user_id = $2`,
			commentID, userAID).Scan(&voteCount)
		require.NoError(t, err)
		require.Equal(t, 1, voteCount, "Only one vote row should exist")
	})

	// Test 4: Concurrent Stress Test (Race Condition Safety)
	t.Run("Concurrent_Stress_Test_NoScoreDrift", func(t *testing.T) {
		// Create User A
		userAID := uuid.Must(uuid.NewV4())
		passHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
		userA := &authModels.UserAuth{
			ObjectId:      userAID,
			Username:      fmt.Sprintf("user-concurrent-%s@test.com", userAID.String()[:8]),
			Password:      passHash,
			Role:          "user",
			EmailVerified: true,
		}
		err := authRepo.CreateUser(ctx, userA)
		require.NoError(t, err)

		// Create Post P
		postID := uuid.Must(uuid.NewV4())
		post := &postsModels.Post{
			ObjectId:     postID,
			OwnerUserId:  userAID,
			PostTypeId:   1,
			Body:         "Test post",
			Score:        0,
			CommentCounter: 0,
		}
		err = postRepo.Create(ctx, post)
		require.NoError(t, err)

		// Create Comment C
		commentID := uuid.Must(uuid.NewV4())
		comment := &models.Comment{
			ObjectId:    commentID,
			PostId:      postID,
			OwnerUserId: userAID,
			Text:        "Test comment",
			Score:       0,
		}
		err = commentRepo.Create(ctx, comment)
		require.NoError(t, err)

		// Create 10 different users to simulate concurrent voting
		userIDs := make([]uuid.UUID, 10)
		for i := 0; i < 10; i++ {
			userID := uuid.Must(uuid.NewV4())
			userIDs[i] = userID
			passHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
			user := &authModels.UserAuth{
				ObjectId:      userID,
				Username:      fmt.Sprintf("user-concurrent-%d-%s@test.com", i, userID.String()[:8]),
				Password:      passHash,
				Role:          "user",
				EmailVerified: true,
			}
			err := authRepo.CreateUser(ctx, user)
			require.NoError(t, err)
		}

		// Launch 10 goroutines to add votes concurrently
		// Each goroutine will try to add a vote for a different user
		var wg sync.WaitGroup
		errors := make(chan error, 10)
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(userIdx int) {
				defer wg.Done()
				created, err := commentRepo.AddVote(ctx, commentID, userIDs[userIdx])
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: AddVote failed: %w", userIdx, err)
					return
				}
				if !created {
					errors <- fmt.Errorf("goroutine %d: AddVote returned false (unexpected)", userIdx)
					return
				}
				// Increment score atomically
				if err := commentRepo.IncrementScore(ctx, commentID, 1); err != nil {
					errors <- fmt.Errorf("goroutine %d: IncrementScore failed: %w", userIdx, err)
					return
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			require.NoError(t, err, "Concurrent vote operations should not fail")
		}

		// Verify final state: All 10 votes should exist, score should be exactly 10
		updatedComment, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, int64(10), updatedComment.Score, "Score should be exactly 10 after 10 concurrent likes (no drift)")

		// Verify all 10 vote rows exist
		var voteCount int
		err = client.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM comment_votes WHERE comment_id = $1`,
			commentID).Scan(&voteCount)
		require.NoError(t, err)
		require.Equal(t, 10, voteCount, "All 10 vote rows should exist")

		// Now launch 5 goroutines to remove votes concurrently (toggle off)
		// This tests the RemoveVote + DecrementScore atomicity
		var wg2 sync.WaitGroup
		errors2 := make(chan error, 5)
		
		for i := 0; i < 5; i++ {
			wg2.Add(1)
			go func(userIdx int) {
				defer wg2.Done()
				deleted, err := commentRepo.RemoveVote(ctx, commentID, userIDs[userIdx])
				if err != nil {
					errors2 <- fmt.Errorf("goroutine %d: RemoveVote failed: %w", userIdx, err)
					return
				}
				if !deleted {
					errors2 <- fmt.Errorf("goroutine %d: RemoveVote returned false (unexpected)", userIdx)
					return
				}
				// Decrement score atomically
				if err := commentRepo.IncrementScore(ctx, commentID, -1); err != nil {
					errors2 <- fmt.Errorf("goroutine %d: IncrementScore(-1) failed: %w", userIdx, err)
					return
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg2.Wait()
		close(errors2)

		// Check for errors
		for err := range errors2 {
			require.NoError(t, err, "Concurrent vote removal operations should not fail")
		}

		// Verify final state: 5 votes removed, score should be exactly 5
		updatedComment2, err := commentRepo.FindByID(ctx, commentID)
		require.NoError(t, err)
		require.Equal(t, int64(5), updatedComment2.Score, "Score should be exactly 5 after 5 concurrent unlikes (10 - 5 = 5, no drift)")

		// Verify exactly 5 vote rows remain
		err = client.DB().QueryRowContext(ctx,
			`SELECT COUNT(*) FROM comment_votes WHERE comment_id = $1`,
			commentID).Scan(&voteCount)
		require.NoError(t, err)
		require.Equal(t, 5, voteCount, "Exactly 5 vote rows should remain")
	})
}

// applyMigrations applies all necessary migrations for comment votes tests
func applyMigrations(t *testing.T, ctx context.Context, db interface{}, schema string) {
	// Apply auth migration first (required for user_auths FK)
	authSQL := `
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

	// Apply profiles migration (required for FindReplies LEFT JOIN)
	profilesSQL := `
		CREATE TABLE IF NOT EXISTS profiles (
			user_id UUID PRIMARY KEY REFERENCES user_auths(id) ON DELETE CASCADE,
			full_name VARCHAR(255),
			social_name VARCHAR(255),
			email VARCHAR(255),
			avatar VARCHAR(512),
			banner VARCHAR(512),
			tagline VARCHAR(500),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`

	// Apply posts migration
	postsSQL := `
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

	// Apply comments migration (005_create_comments_table.sql + 008_add_reply_to_user.sql)
	commentsSQL := `
		CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY,
			post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
			parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
			reply_to_user_id UUID REFERENCES user_auths(id) ON DELETE SET NULL,
			reply_to_display_name VARCHAR(255),
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
		CREATE INDEX IF NOT EXISTS idx_comments_reply_to_user ON comments(reply_to_user_id) WHERE reply_to_user_id IS NOT NULL;
	`

	// Apply comment_votes migration
	votesSQL := `
		CREATE TABLE IF NOT EXISTS comment_votes (
			comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
			owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (comment_id, owner_user_id)
		);
		CREATE INDEX IF NOT EXISTS idx_comment_votes_owner ON comment_votes(owner_user_id);
	`

	// Execute migrations in correct order (auth first, then posts, then comments, then votes)
	// client.DB() returns *sqlx.DB which implements ExecContext
	type execContext interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	}
	if execer, ok := db.(execContext); ok {
		_, err := execer.ExecContext(ctx, authSQL)
		require.NoError(t, err, "Failed to apply auth migration")
		_, err = execer.ExecContext(ctx, profilesSQL)
		require.NoError(t, err, "Failed to apply profiles migration")
		_, err = execer.ExecContext(ctx, postsSQL)
		require.NoError(t, err, "Failed to apply posts migration")
		_, err = execer.ExecContext(ctx, commentsSQL)
		require.NoError(t, err, "Failed to apply comments migration")
		_, err = execer.ExecContext(ctx, votesSQL)
		require.NoError(t, err, "Failed to apply comment_votes migration")
	} else {
		t.Fatalf("db does not implement ExecContext, got %T", db)
	}
}

