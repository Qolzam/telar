package comments_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments"
	"github.com/qolzam/telar/apps/api/comments/handlers"
	commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
	"github.com/qolzam/telar/apps/api/comments/services"
	commentModels "github.com/qolzam/telar/apps/api/comments/models"
	postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	authModels "github.com/qolzam/telar/apps/api/auth/models"
	postsModels "github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// verifyPostgresConnection tests if we can actually connect to PostgreSQL using testutil
func verifyPostgresConnection() error {
	suite := testutil.Setup(&testing.T{})
	iso := testutil.NewIsolatedTest(&testing.T{}, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		return fmt.Errorf("PostgreSQL not available")
	}
	return nil
}

// Legacy helper functions removed - PostgreSQL only

// DELETED: Redundant signHMAC helper removed per g-sol10.md Step 2
// All HMAC signing now uses the centralized testutil.signHMAC with SHA256

// addHMACHeaders creates HMAC authentication headers using canonical signing format
func addHMACHeaders(req *http.Request, body []byte, secret string, uid string) {
	// Generate timestamp for canonical signing
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	
	// Extract request details for canonical signing
	method := req.Method
	path := req.URL.Path
	query := req.URL.RawQuery
	
	// Generate canonical HMAC signature
	sig := testutil.SignHMAC(method, path, query, body, uid, timestamp, secret)
	req.Header.Set(types.HeaderHMACAuthenticate, sig)
	req.Header.Set(types.HeaderUID, uid)
	req.Header.Set(types.HeaderTimestamp, timestamp)
	req.Header.Set("username", "test@example.com")
	req.Header.Set("displayName", "Tester")
	req.Header.Set("socialName", "tester")
	req.Header.Set("systemRole", "user")
}

// newTestApp creates a new test Fiber app with comments routes
// Returns the app and the secret used for HMAC signing
func newTestApp(t *testing.T, commentRepo commentRepository.CommentRepository, postRepo postsRepository.PostRepository, cfg *platformconfig.Config) (*fiber.App, string) {
	app := fiber.New()

	// Add test middleware to set user context
	app.Use(func(c *fiber.Ctx) error {
		// Extract user info from headers (simulating HMAC middleware)
		uid := c.Get(types.HeaderUID)
		if uid != "" {
			userID, err := uuid.FromString(uid)
			if err != nil {
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid user ID"})
			}
			createdDate, _ := strconv.ParseInt(c.Get("createdDate"), 10, 64)
			if createdDate == 0 {
				createdDate = time.Now().Unix()
			}
			user := types.UserContext{
				UserID:      userID, // This MUST match the user created in setupTestData
				Username:    c.Get("email"),
				DisplayName: c.Get("displayName"),
				SocialName:  c.Get("socialName"),
				Avatar:      "",
				Banner:      "",
				TagLine:     "",
				SystemRole:  c.Get("role"),
				CreatedDate: createdDate,
			}
			c.Locals(types.UserCtxName, user)
		}
		return c.Next()
	})

	// Create handlers and config using the new repository-based service
	commentService := services.NewCommentService(commentRepo, postRepo, cfg, nil)
	commentHandler := handlers.NewCommentHandler(commentService, cfg.JWT, cfg.HMAC)

	commentsHandlers := &comments.CommentsHandlers{
		CommentHandler: commentHandler,
	}

	// Use the new dependency injection pattern
	comments.RegisterRoutes(app, commentsHandlers, cfg)
	return app, cfg.HMAC.Secret
}

// Test data structures
type createCommentRequest struct {
	PostId string `json:"postId"`
	Text   string `json:"text"`
}

type updateCommentRequest struct {
	ObjectId string `json:"objectId"`
	Text     string `json:"text"`
}

type commentResponse struct {
	ObjectId         string `json:"objectId"`
	Score            int64  `json:"score"`
	OwnerUserId      string `json:"ownerUserId"`
	OwnerDisplayName string `json:"ownerDisplayName"`
	OwnerAvatar      string `json:"ownerAvatar"`
	PostId           string `json:"postId"`
	Text             string `json:"text"`
	Deleted          bool   `json:"deleted"`
	DeletedDate      int64  `json:"deletedDate,omitempty"`
	CreatedDate      int64  `json:"createdDate"`
	LastUpdated      int64  `json:"lastUpdated,omitempty"`
}

// TestCommentsHTTPCompatibilityPostgreSQL tests HTTP compatibility with PostgreSQL
func TestCommentsHTTPCompatibilityPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// Create PostRepository FIRST (applies posts migration, required for comments foreign key)
	postRepo, err := postsRepository.NewPostgresRepositoryForTest(ctx, iso)
	require.NoError(t, err, "failed to create PostRepository")
	
	// PostRepository client needs search_path set (ApplyPostsMigration sets it, but we need to ensure it persists)
	// The client returned by NewPostgresRepositoryForTest should already have search_path set via ApplyPostsMigration

	// Create CommentRepository AFTER posts migration (comments table has FK to posts)
	commentRepo, err := commentRepository.NewPostgresCommentRepositoryForTest(ctx, iso)
	require.NoError(t, err, "failed to create CommentRepository")

	// Create AuthRepository for test user creation
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "failed to create postgres client for auth")
	
	// Set search_path to isolated schema (critical for foreign key constraints)
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "failed to set search_path for auth client")
	
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)

	app, secret := newTestApp(t, commentRepo, postRepo, iso.Config)
	
	// Create test user and post FIRST, then use that user's ID for tests
	userUUID := uuid.Must(uuid.NewV4())
	uid := userUUID.String()
	
	// Create HTTP helper
	httpHelper := testutil.NewHTTPHelper(t, app)

	// Test basic CRUD operations (setupTestData will create user with this uid)
	runCommentsHTTPCompatibilityTests(t, "PostgreSQL", app, secret, uid, httpHelper, authRepo, postRepo)
}

// runCommentsHTTPCompatibilityTests runs the main HTTP compatibility test suite
func runCommentsHTTPCompatibilityTests(t *testing.T, dbType string, app *fiber.App, secret string, uid string, httpHelper *testutil.HTTPHelper, authRepo authRepository.AuthRepository, postRepo postsRepository.PostRepository) {
	ctx := context.Background()

	// Create test user for all tests
	userID := setupTestData(t, ctx, authRepo, uid)

	t.Run(fmt.Sprintf("Create_and_Get_Comment_%s", dbType), func(t *testing.T) {
		// Each subtest gets its own unique post to avoid data pollution
		postID := createTestPost(t, ctx, postRepo, userID)
		testCreateAndGetComment(t, app, secret, uid, httpHelper, postID)
	})

	t.Run(fmt.Sprintf("Update_Comment_%s", dbType), func(t *testing.T) {
		// Each subtest gets its own unique post to avoid data pollution
		postID := createTestPost(t, ctx, postRepo, userID)
		testUpdateComment(t, app, secret, uid, httpHelper, postID)
	})

	t.Run(fmt.Sprintf("Delete_Comment_%s", dbType), func(t *testing.T) {
		// Each subtest gets its own unique post to avoid data pollution
		postID := createTestPost(t, ctx, postRepo, userID)
		testDeleteComment(t, app, secret, uid, httpHelper, postID)
	})

	t.Run(fmt.Sprintf("Get_Comments_By_Post_%s", dbType), func(t *testing.T) {
		// Each subtest gets its own unique post to avoid data pollution
		postID := createTestPost(t, ctx, postRepo, userID)
		testGetCommentsByPost(t, app, secret, uid, httpHelper, postID)
	})
}

// setupTestData creates a test user, returning the user ID
func setupTestData(t *testing.T, ctx context.Context, authRepo authRepository.AuthRepository, uid string) uuid.UUID {
	// Create test user with unique username
	userUUID := uuid.Must(uuid.FromString(uid))
	hashedPassword := []byte("test_password_hash")
	now := time.Now()
	
	// Generate unique username to avoid conflicts
	uniqueID := uuid.Must(uuid.NewV4()).String()[:8]
	username := fmt.Sprintf("testuser_%s@example.com", uniqueID)

	userAuth := &authModels.UserAuth{
		ObjectId:      userUUID,
		Username:      username,
		Password:      hashedPassword,
		Role:          "user",
		EmailVerified: true,
		PhoneVerified: false,
		CreatedDate:   now.Unix(),
		LastUpdated:   now.Unix(),
	}

	err := authRepo.CreateUser(ctx, userAuth)
	require.NoError(t, err, "Failed to create test user")

	return userUUID
}

// createTestPost creates a unique test post for a user, returning the post ID
func createTestPost(t *testing.T, ctx context.Context, postRepo postsRepository.PostRepository, userID uuid.UUID) uuid.UUID {
	now := time.Now()
	postUUID := uuid.Must(uuid.NewV4())
	post := &postsModels.Post{
		ObjectId:         postUUID,
		OwnerUserId:      userID,
		PostTypeId:       1,
		Body:             fmt.Sprintf("Test post %s", postUUID.String()[:8]),
		Score:            0,
		ViewCount:        0,
		CommentCounter:   0,
		Tags:             pq.StringArray{"test"},
		OwnerDisplayName: "Test User",
		OwnerAvatar:      "",
		Deleted:          false,
		DeletedDate:      0,
		CreatedDate:      now.Unix(),
		LastUpdated:      now.Unix(),
		CreatedAt:        now,
		UpdatedAt:        now,
		Permission:       "Public",
	}

	err := postRepo.Create(ctx, post)
	require.NoError(t, err, "Failed to create test post")

	return postUUID
}

func testCreateAndGetComment(t *testing.T, app *fiber.App, secret string, uid string, httpHelper *testutil.HTTPHelper, postID uuid.UUID) {
	postId := postID.String()

	// Create comment
	createReq := createCommentRequest{
		PostId: postId,
		Text:   "This is a test comment",
	}

	resp := httpHelper.NewRequest(http.MethodPost, "/comments/", createReq).
		WithHMACAuth(secret, uid).Send()
	
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Create comment should return 201 Created")

	var createResp struct {
		ObjectId string `json:"objectId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate create response
	if createResp.ObjectId == "" {
		t.Error("Expected objectId to be set")
	}

	// Get the created comment to validate full details
	resp = httpHelper.NewRequest(http.MethodGet, "/comments/"+createResp.ObjectId, nil).
		WithHMACAuth(secret, uid).Send()
	
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get comment should return 200 OK")

	var commentResp commentResponse
	if err := json.NewDecoder(resp.Body).Decode(&commentResp); err != nil {
		t.Fatalf("Failed to decode get response: %v", err)
	}

	// Validate comment fields
	if commentResp.ObjectId != createResp.ObjectId {
		t.Errorf("Expected objectId %s, got %s", createResp.ObjectId, commentResp.ObjectId)
	}
	if commentResp.PostId != postId {
		t.Errorf("Expected postId %s, got %s", postId, commentResp.PostId)
	}
	if commentResp.Text != "This is a test comment" {
		t.Errorf("Expected text 'This is a test comment', got %s", commentResp.Text)
	}
	if commentResp.Deleted {
		t.Error("Expected deleted to be false")
	}
}

func testUpdateComment(t *testing.T, app *fiber.App, secret string, uid string, httpHelper *testutil.HTTPHelper, postID uuid.UUID) {
	postId := postID.String()

	// Create comment first
	createReq := createCommentRequest{
		PostId: postId,
		Text:   "Original comment text",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/comments/", bytes.NewReader(body))
	req.Header.Set(types.HeaderContentType, "application/json")
	addHMACHeaders(req, body, secret, uid)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}
	defer resp.Body.Close()

	var commentResp commentResponse
	if err := json.NewDecoder(resp.Body).Decode(&commentResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	// Update comment
	updateReq := updateCommentRequest{
		ObjectId: commentResp.ObjectId,
		Text:     "Updated comment text",
	}

	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, "/comments/", bytes.NewReader(body))
	req.Header.Set(types.HeaderContentType, "application/json")
	addHMACHeaders(req, body, secret, uid)

	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to update comment: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var updateResp commentResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		t.Fatalf("Failed to decode update response: %v", err)
	}

	// Validate updated comment
	if updateResp.Text != "Updated comment text" {
		t.Errorf("Expected text 'Updated comment text', got %s", updateResp.Text)
	}
	if updateResp.LastUpdated == 0 {
		t.Error("Expected lastUpdated to be set")
	}
}

func testDeleteComment(t *testing.T, app *fiber.App, secret string, uid string, httpHelper *testutil.HTTPHelper, postID uuid.UUID) {
	postId := postID.String()

	// Create comment first
	createReq := createCommentRequest{
		PostId: postId,
		Text:   "Comment to be deleted",
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/comments/", bytes.NewReader(body))
	req.Header.Set(types.HeaderContentType, "application/json")
	addHMACHeaders(req, body, secret, uid)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to create comment: %v", err)
	}
	defer resp.Body.Close()

	var commentResp commentResponse
	if err := json.NewDecoder(resp.Body).Decode(&commentResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	// Delete comment
	deleteURL := fmt.Sprintf("/comments/id/%s/post/%s", commentResp.ObjectId, postId)
	req = httptest.NewRequest(http.MethodDelete, deleteURL, nil)
	addHMACHeaders(req, nil, secret, uid)

	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to delete comment: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify comment is deleted by trying to get it
	req = httptest.NewRequest(http.MethodGet, "/comments/"+commentResp.ObjectId, nil)
	addHMACHeaders(req, nil, secret, uid)

	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to execute get request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 404 since comment is deleted
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404 for deleted comment, got %d", resp.StatusCode)
	}
}

func testGetCommentsByPost(t *testing.T, app *fiber.App, secret string, uid string, httpHelper *testutil.HTTPHelper, postID uuid.UUID) {
	postId := postID.String()

	// Create multiple comments for the same post
	commentTexts := []string{"First comment", "Second comment", "Third comment"}

	for _, text := range commentTexts {
		createReq := createCommentRequest{
			PostId: postId,
			Text:   text,
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest(http.MethodPost, "/comments/", bytes.NewReader(body))
		req.Header.Set(types.HeaderContentType, "application/json")
		addHMACHeaders(req, body, secret, uid)

		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", resp.StatusCode)
		}
	}

	// Get comments by post
	getURL := fmt.Sprintf("/comments/?postId=%s&limit=10", postId)
	req := httptest.NewRequest(http.MethodGet, getURL, nil)
	addHMACHeaders(req, nil, secret, uid)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d, response body: %s", resp.StatusCode, string(bodyBytes))
	}

	var response commentModels.CommentsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode get comments response: %v", err)
	}

	// Validate we got the expected number of comments
	if len(response.Comments) != len(commentTexts) {
		t.Errorf("Expected %d comments, got %d", len(commentTexts), len(response.Comments))
	}

	// Validate each comment belongs to the correct post
	for _, comment := range response.Comments {
		if comment.PostId != postId {
			t.Errorf("Expected postId %s, got %s", postId, comment.PostId)
		}
	}
}
