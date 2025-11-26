package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/models"
	commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
	postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	authModels "github.com/qolzam/telar/apps/api/auth/models"
	postsModels "github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommentRepository_CRUD tests basic CRUD operations using CommentRepository
func TestCommentRepository_CRUD(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// Create PostRepository and apply migrations
	postRepo, err := postsRepository.NewPostgresRepositoryForTest(ctx, iso)
	require.NoError(t, err, "failed to create PostRepository")

	// Create AuthRepository for test user
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "failed to create postgres client for auth")
	
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "failed to set search_path for auth client")
	
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)

	// Create CommentRepository
	commentRepo, err := commentRepository.NewPostgresCommentRepositoryForTest(ctx, iso)
	require.NoError(t, err, "failed to create CommentRepository")

	// Create test user and post
	userID := uuid.Must(uuid.NewV4())
	postID := uuid.Must(uuid.NewV4())
	now := time.Now()

	userAuth := &authModels.UserAuth{
		ObjectId:      userID,
		Username:      "testuser@example.com",
		Password:      []byte("test_password"),
		Role:          "user",
		EmailVerified: true,
		CreatedDate:   now.Unix(),
		LastUpdated:   now.Unix(),
	}
	err = authRepo.CreateUser(ctx, userAuth)
	require.NoError(t, err, "failed to create test user")

	post := &postsModels.Post{
		ObjectId:         postID,
		OwnerUserId:      userID,
		PostTypeId:       1,
		Body:             "Test post",
		Tags:             pq.StringArray{"test"},
		CreatedDate:      now.Unix(),
		LastUpdated:      now.Unix(),
		CreatedAt:        now,
		UpdatedAt:        now,
		Permission:       "Public",
	}
	err = postRepo.Create(ctx, post)
	require.NoError(t, err, "failed to create test post")

	// Test CRUD operations
	t.Run("BasicCRUDOperations", func(t *testing.T) {
		commentID := uuid.Must(uuid.NewV4())
		
		// Create comment
		comment := &models.Comment{
			ObjectId:    commentID,
			PostId:      postID,
			OwnerUserId: userID,
			Text:        "This is a test comment",
			Score:       0,
			CreatedDate: now.Unix(),
			LastUpdated: now.Unix(),
		}
		err := commentRepo.Create(ctx, comment)
		assert.NoError(t, err, "failed to create comment")
		
		// Read comment
		found, err := commentRepo.FindByID(ctx, commentID)
		assert.NoError(t, err, "failed to find comment")
		assert.NotNil(t, found)
		assert.Equal(t, comment.Text, found.Text)
		
		// Update comment
		found.Text = "Updated test comment"
		found.LastUpdated = time.Now().Unix()
		err = commentRepo.Update(ctx, found)
		assert.NoError(t, err, "failed to update comment")
		
		// Verify update
		updated, err := commentRepo.FindByID(ctx, commentID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated test comment", updated.Text)
		
		// Delete comment (soft delete)
		err = commentRepo.Delete(ctx, commentID)
		assert.NoError(t, err, "failed to delete comment")
		
		// Verify deletion (should return deleted comment but marked as deleted)
		deleted, err := commentRepo.FindByID(ctx, commentID)
		// Note: Repository may return deleted comments, service layer filters them
		if err == nil {
			assert.True(t, deleted.Deleted, "comment should be marked as deleted")
		}
	})
}
