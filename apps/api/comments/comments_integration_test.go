package comments_test

import (
	"context"
	"testing"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/qolzam/telar/apps/api/comments/models"
	"github.com/qolzam/telar/apps/api/comments/services"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// --- helpers ---

type commentGetResp struct {
	Text             string `json:"text"`
	OwnerDisplayName string `json:"ownerDisplayName"`
	PostId           string `json:"postId"`
	Deleted          bool   `json:"deleted"`
}

// runCommentsPersistenceSuite runs the persistence test suite using the service layer
func runCommentsPersistenceSuite(t *testing.T, base *platform.BaseService, cfg *platformconfig.Config) {
	t.Helper()
	commentService := services.NewCommentService(base, cfg)
	const commentCollectionName = "comment"

	// Create
	postID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()
	
	c := &models.CreateCommentRequest{
		PostId: postID,
		Text:   "integration test comment",
	}

	comment, err := commentService.CreateComment(context.Background(), c, &types.UserContext{
		UserID:      userID,
		Username:    "test@example.com",
		DisplayName: "Test User",
		SocialName:  "testuser",
		Avatar:      "test-avatar.jpg",
	})
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Count should be >= 1 for this postId
	cntRes := <-base.Repository.Count(context.Background(), commentCollectionName, map[string]interface{}{"postId": postID})
	if cntRes.Error != nil {
		t.Fatalf("count after save error: %v", cntRes.Error)
	}
	if cntRes.Count < 1 {
		t.Fatalf("expected at least 1 comment, got %d", cntRes.Count)
	}

	// Query one
	got, err := commentService.GetComment(context.Background(), comment.ObjectId)
	if err != nil {
		t.Fatalf("find one error: %v", err)
	}
	t.Logf("Expected text: %q, Got text: %q", c.Text, got.Text)
	if got.Text != c.Text {
		t.Fatalf("unexpected text: %v", got.Text)
	}

	// Update text
	newText := "updated comment text"
	if err := commentService.UpdateComment(context.Background(), comment.ObjectId, &models.UpdateCommentRequest{
		Text: newText,
	}, &types.UserContext{UserID: userID, Username: "test@example.com", SocialName: "testuser"}); err != nil {
		t.Fatalf("update error: %v", err)
	}
	
	got2, err := commentService.GetComment(context.Background(), comment.ObjectId)
	if err != nil {
		t.Fatalf("find one after update error: %v", err)
	}
	if got2.Text != newText {
		t.Fatalf("update not applied: %v", got2.Text)
	}

	// Delete
	if err := commentService.DeleteComment(context.Background(), comment.ObjectId, comment.PostId, &types.UserContext{UserID: userID, Username: "test@example.com", SocialName: "testuser"}); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	// Verify deletion
	_, err = commentService.GetComment(context.Background(), comment.ObjectId)
	if err == nil {
		t.Fatalf("expected error after deletion, got nil")
	}
}

func TestCommentsServiceLayerMongoDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping test")
	}
	
	ctx := context.Background()
	
	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)
	
	runCommentsPersistenceSuite(t, base, iso.Config)
}

func TestCommentsServiceLayerPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	ctx := context.Background()
	
	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)
	
	runCommentsPersistenceSuite(t, base, iso.Config)
}
