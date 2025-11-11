package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/types"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/posts/handlers"
	"github.com/qolzam/telar/apps/api/posts/models"
)

// createTestConfig creates a test configuration for handler tests
func createTestConfig() (platformconfig.JWTConfig, platformconfig.HMACConfig) {
	return platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		}, platformconfig.HMACConfig{
			Secret: "test-secret",
		}
}

// MockPostService implements the PostService interface for testing
type MockPostService struct {
	createPostFunc                    func(ctx context.Context, req *models.CreatePostRequest, user *types.UserContext) (*models.Post, error)
	getPostFunc                       func(ctx context.Context, postID uuid.UUID) (*models.Post, error)
	getPostByURLKeyFunc               func(ctx context.Context, urlKey string) (*models.Post, error)
	queryPostsFunc                    func(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error)
	updatePostFunc                    func(ctx context.Context, postID uuid.UUID, req *models.UpdatePostRequest, user *types.UserContext) error
	deletePostFunc                    func(ctx context.Context, postID uuid.UUID, user *types.UserContext) error
	validatePostOwnershipFunc         func(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error
	incrementViewCountFunc            func(ctx context.Context, postID uuid.UUID) error
	createIndexFunc                   func(ctx context.Context, indexes map[string]interface{}) error
	deleteWithOwnershipFunc           func(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error
	incrementFieldsFunc               func(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error
	incrementFieldsWithOwnershipFunc  func(ctx context.Context, postID uuid.UUID, userID uuid.UUID, updates map[string]interface{}) error
	updateFieldsFunc                     func(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error
	updateFieldsWithOwnershipFunc        func(ctx context.Context, postID uuid.UUID, userID uuid.UUID, updates map[string]interface{}) error
	updateAndIncrementFieldsFunc         func(ctx context.Context, postID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error
	deleteByOwnerFunc                 func(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error
	incrementFieldFunc                func(ctx context.Context, postID uuid.UUID, field string, delta int) error
	setFieldFunc                      func(ctx context.Context, postID uuid.UUID, field string, value interface{}) error
	updateByOwnerFunc                 func(ctx context.Context, postID uuid.UUID, userID uuid.UUID, updates map[string]interface{}) error
	updateProfileForOwnerFunc         func(ctx context.Context, userID uuid.UUID, displayName, avatar string) error
	
	// Mock state for testing
	posts        map[string]*models.Post
	shouldFail   bool
	failureError error
}

func (m *MockPostService) CreatePost(ctx context.Context, req *models.CreatePostRequest, user *types.UserContext) (*models.Post, error) {
	if m.createPostFunc != nil {
		return m.createPostFunc(ctx, req, user)
	}
	return nil, nil
}

func (m *MockPostService) CreateIndex(ctx context.Context, indexes map[string]interface{}) error {
	if m.createIndexFunc != nil {
		return m.createIndexFunc(ctx, indexes)
	}
	return nil
}

func (m *MockPostService) CreateIndexes(ctx context.Context) error {
	if m.createIndexFunc != nil {
		return m.createIndexFunc(ctx, map[string]interface{}{
			"body":     "text",
			"objectId": 1,
		})
	}
	return nil
}

func (m *MockPostService) GetPost(ctx context.Context, postID uuid.UUID) (*models.Post, error) {
	if m.getPostFunc != nil {
		return m.getPostFunc(ctx, postID)
	}
	return nil, nil
}

func (m *MockPostService) GetPostByURLKey(ctx context.Context, urlKey string) (*models.Post, error) {
	if m.getPostByURLKeyFunc != nil {
		return m.getPostByURLKeyFunc(ctx, urlKey)
	}
	return nil, nil
}

func (m *MockPostService) GetPostsByUser(ctx context.Context, userID uuid.UUID, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	return nil, nil
}

func (m *MockPostService) QueryPosts(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	if m.queryPostsFunc != nil {
		return m.queryPostsFunc(ctx, filter)
	}
	return nil, nil
}

func (m *MockPostService) SearchPosts(ctx context.Context, query string, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	return nil, nil
}

func (m *MockPostService) UpdatePost(ctx context.Context, postID uuid.UUID, req *models.UpdatePostRequest, user *types.UserContext) error {
	if m.updatePostFunc != nil {
		return m.updatePostFunc(ctx, postID, req, user)
	}
	return nil
}

func (m *MockPostService) UpdatePostProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
	return nil
}

func (m *MockPostService) IncrementScore(ctx context.Context, postID uuid.UUID, delta int, user *types.UserContext) error {
	return nil
}

func (m *MockPostService) IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int, user *types.UserContext) error {
	return nil
}

func (m *MockPostService) SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, user *types.UserContext) error {
	return nil
}

func (m *MockPostService) SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, user *types.UserContext) error {
	return nil
}

func (m *MockPostService) IncrementViewCount(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	if m.incrementViewCountFunc != nil {
		return m.incrementViewCountFunc(ctx, postID)
	}
	return nil
}

func (m *MockPostService) DeletePost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	if m.deletePostFunc != nil {
		return m.deletePostFunc(ctx, postID, user)
	}
	return nil
}

func (m *MockPostService) SoftDeletePost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	if m.deletePostFunc != nil {
		return m.deletePostFunc(ctx, postID, user)
	}
	return nil
}

func (m *MockPostService) GenerateURLKey(ctx context.Context, postID uuid.UUID, user *types.UserContext) (string, error) {
	return "", nil
}

func (m *MockPostService) ValidatePostOwnership(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.validatePostOwnershipFunc != nil {
		return m.validatePostOwnershipFunc(ctx, postID, userID)
	}
	
	// Default implementation: Check if post exists and user owns it
	if m.posts != nil {
		if post, exists := m.posts[postID.String()]; exists {
			if post.OwnerUserId != userID {
				return errors.New("user does not own this post")
			}
			return nil
		}
		return errors.New("post not found")
	}
	
	// Default implementation for tests without mock storage
	return nil
}

func (m *MockPostService) DeleteWithOwnership(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.deleteWithOwnershipFunc != nil {
		return m.deleteWithOwnershipFunc(ctx, postID, userID)
	}
	
	// Default implementation: Validate ownership then delete
	if err := m.ValidatePostOwnership(ctx, postID, userID); err != nil {
		return errors.New("ownership validation failed: " + err.Error())
	}
	
	// Simulate deletion from mock storage
	if m.posts != nil {
		delete(m.posts, postID.String())
	}
	
	return nil
}

func (m *MockPostService) IncrementFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.incrementFieldsFunc != nil {
		return m.incrementFieldsFunc(ctx, postID, updates)
	}
	
	// Default implementation: Validate updates format
	if len(updates) == 0 {
		return errors.New("no updates provided")
	}
	
	// Validate that all updates are numeric for increment operations
	for field, value := range updates {
		switch field {
		case "viewCount", "score", "commentCounter":
			if _, ok := value.(int64); !ok {
				if _, ok := value.(int); !ok {
					return errors.New("increment value must be numeric for field: " + field)
				}
			}
		default:
			return errors.New("unsupported field for increment: " + field)
		}
	}
	
	// Simulate updating mock storage
	if m.posts != nil {
		if post, exists := m.posts[postID.String()]; exists {
			for field, value := range updates {
				switch field {
				case "viewCount":
					if delta, ok := value.(int64); ok {
						post.ViewCount += delta
					} else if delta, ok := value.(int); ok {
						post.ViewCount += int64(delta)
					}
				case "score":
					if delta, ok := value.(int64); ok {
						post.Score += delta
					} else if delta, ok := value.(int); ok {
						post.Score += int64(delta)
					}
				case "commentCounter":
					if delta, ok := value.(int64); ok {
						post.CommentCounter += delta
					} else if delta, ok := value.(int); ok {
						post.CommentCounter += int64(delta)
					}
				}
			}
		}
	}
	
	return nil
}

func (m *MockPostService) IncrementFieldsWithOwnership(ctx context.Context, postID uuid.UUID, userID uuid.UUID, updates map[string]interface{}) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.incrementFieldsWithOwnershipFunc != nil {
		return m.incrementFieldsWithOwnershipFunc(ctx, postID, userID, updates)
	}
	
	// Default implementation: Validate ownership then increment
	if err := m.ValidatePostOwnership(ctx, postID, userID); err != nil {
		return errors.New("ownership validation failed: " + err.Error())
	}
	
	return m.IncrementFields(ctx, postID, updates)
}

func (m *MockPostService) UpdateFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.updateFieldsFunc != nil {
		return m.updateFieldsFunc(ctx, postID, updates)
	}
	
	// Default implementation: Validate updates format
	if len(updates) == 0 {
		return errors.New("no updates provided")
	}
	
	// Validate allowed fields for update
	allowedFields := map[string]bool{
		"body": true, "tags": true, "ownerDisplayName": true, "ownerAvatar": true,
		"disableComments": true, "disableSharing": true, "deletedDate": true,
	}
	
	for field := range updates {
		if !allowedFields[field] {
			return errors.New("unsupported field for update: " + field)
		}
	}
	
	// Simulate updating mock storage
	if m.posts != nil {
		if post, exists := m.posts[postID.String()]; exists {
			for field, value := range updates {
				switch field {
				case "body":
					if body, ok := value.(string); ok {
						post.Body = body
					}
				case "ownerDisplayName":
					if displayName, ok := value.(string); ok {
						post.OwnerDisplayName = displayName
					}
				case "ownerAvatar":
					if avatar, ok := value.(string); ok {
						post.OwnerAvatar = avatar
					}
				case "disableComments":
					if disabled, ok := value.(bool); ok {
						post.DisableComments = disabled
					}
				case "disableSharing":
					if disabled, ok := value.(bool); ok {
						post.DisableSharing = disabled
					}
				}
			}
		}
	}
	
	return nil
}

func (m *MockPostService) UpdateFieldsWithOwnership(ctx context.Context, postID uuid.UUID, userID uuid.UUID, updates map[string]interface{}) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.updateFieldsWithOwnershipFunc != nil {
		return m.updateFieldsWithOwnershipFunc(ctx, postID, userID, updates)
	}
	
	// Default implementation: Validate ownership then update
	if err := m.ValidatePostOwnership(ctx, postID, userID); err != nil {
		return errors.New("ownership validation failed: " + err.Error())
	}
	
	return m.UpdateFields(ctx, postID, updates)
}

func (m *MockPostService) UpdateAndIncrementFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.updateAndIncrementFieldsFunc != nil {
		return m.updateAndIncrementFieldsFunc(ctx, postID, updates, increments)
	}
	
	// Default behavior: apply updates first, then increments
	if err := m.UpdateFields(ctx, postID, updates); err != nil {
		return err
	}
	return m.IncrementFields(ctx, postID, increments)
}

func (m *MockPostService) DeleteByOwner(ctx context.Context, owner uuid.UUID, objectId uuid.UUID) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.deleteByOwnerFunc != nil {
		return m.deleteByOwnerFunc(ctx, owner, objectId)
	}
	
	// Default implementation: Validate ownership and delete
	if m.posts != nil {
		if post, exists := m.posts[objectId.String()]; exists {
			if post.OwnerUserId != owner {
				return errors.New("access denied: user does not own this post")
			}
			delete(m.posts, objectId.String())
			return nil
		} else {
			// Post doesn't exist - return not found error for non-idempotent behavior
			return errors.New("post not found")
		}
	}
	
	// No posts map - simulate successful deletion (idempotent)
	return nil
}

func (m *MockPostService) IncrementField(ctx context.Context, postID uuid.UUID, field string, delta int) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.incrementFieldFunc != nil {
		return m.incrementFieldFunc(ctx, postID, field, delta)
	}
	
	// Default implementation: Validate field and increment
	allowedFields := map[string]bool{
		"viewCount": true, "score": true, "commentCounter": true,
	}
	
	if !allowedFields[field] {
		return errors.New("unsupported field for increment: " + field)
	}
	
	// Simulate incrementing in mock storage
	if m.posts != nil {
		if post, exists := m.posts[postID.String()]; exists {
			switch field {
			case "viewCount":
				post.ViewCount += int64(delta)
			case "score":
				post.Score += int64(delta)
			case "commentCounter":
				post.CommentCounter += int64(delta)
			}
		} else {
			return errors.New("post not found")
		}
	}
	
	return nil
}

func (m *MockPostService) SetField(ctx context.Context, postID uuid.UUID, field string, value interface{}) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.setFieldFunc != nil {
		return m.setFieldFunc(ctx, postID, field, value)
	}
	
	// Default implementation: Validate field and set value
	allowedFields := map[string]bool{
		"body": true, "ownerDisplayName": true, "ownerAvatar": true,
		"disableComments": true, "disableSharing": true,
	}
	
	if !allowedFields[field] {
		return errors.New("unsupported field for set operation: " + field)
	}
	
	// Simulate setting value in mock storage
	if m.posts != nil {
		if post, exists := m.posts[postID.String()]; exists {
			switch field {
			case "body":
				if body, ok := value.(string); ok {
					post.Body = body
				} else {
					return errors.New("invalid type for body field, expected string")
				}
			case "ownerDisplayName":
				if displayName, ok := value.(string); ok {
					post.OwnerDisplayName = displayName
				} else {
					return errors.New("invalid type for ownerDisplayName field, expected string")
				}
			case "ownerAvatar":
				if avatar, ok := value.(string); ok {
					post.OwnerAvatar = avatar
				} else {
					return errors.New("invalid type for ownerAvatar field, expected string")
				}
			case "disableComments":
				if disabled, ok := value.(bool); ok {
					post.DisableComments = disabled
				} else {
					return errors.New("invalid type for disableComments field, expected bool")
				}
			case "disableSharing":
				if disabled, ok := value.(bool); ok {
					post.DisableSharing = disabled
				} else {
					return errors.New("invalid type for disableSharing field, expected bool")
				}
			}
		} else {
			return errors.New("post not found")
		}
	}
	
	return nil
}

func (m *MockPostService) UpdateByOwner(ctx context.Context, postID uuid.UUID, userID uuid.UUID, updates map[string]interface{}) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.updateByOwnerFunc != nil {
		return m.updateByOwnerFunc(ctx, postID, userID, updates)
	}
	
	// Default implementation: Validate ownership and perform batch update
	if len(updates) == 0 {
		return errors.New("no updates provided")
	}
	
	if m.posts != nil {
		if post, exists := m.posts[postID.String()]; exists {
			if post.OwnerUserId != userID {
				return errors.New("access denied: user does not own this post")
			}
			
			// Apply all updates
			for field, value := range updates {
				if err := m.SetField(ctx, postID, field, value); err != nil {
					return errors.New("failed to update field " + field + ": " + err.Error())
				}
			}
		} else {
			return errors.New("post not found")
		}
	}
	
	return nil
}

func (m *MockPostService) UpdateProfileForOwner(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
	// Check for configured failure
	if m.shouldFail {
		return m.failureError
	}
	
	// Use custom implementation if provided
	if m.updateProfileForOwnerFunc != nil {
		return m.updateProfileForOwnerFunc(ctx, userID, displayName, avatar)
	}
	
	// Default implementation: Update profile info for all posts by this user
	if displayName == "" && avatar == "" {
		return errors.New("at least one of displayName or avatar must be provided")
	}
	
	updatedCount := 0
	if m.posts != nil {
		for _, post := range m.posts {
			if post.OwnerUserId == userID {
				if displayName != "" {
					post.OwnerDisplayName = displayName
				}
				if avatar != "" {
					post.OwnerAvatar = avatar
				}
				updatedCount++
			}
		}
	}
	
	// Note: In a real implementation, you might want to return the count of updated posts
	// For testing purposes, we'll just ensure the operation completed successfully
	
	return nil
}

// New cursor-based pagination methods
func (m *MockPostService) QueryPostsWithCursor(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Check for configured failure
	if m.shouldFail {
		return nil, m.failureError
	}
	
	// Simple mock implementation - return empty results for testing
	return &models.PostsListResponse{
		Posts:      []models.PostResponse{},
		NextCursor: "",
		PrevCursor: "",
		HasNext:    false,
		HasPrev:    false,
		Limit:      filter.Limit,
	}, nil
}

func (m *MockPostService) SearchPostsWithCursor(ctx context.Context, searchTerm string, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Check for configured failure
	if m.shouldFail {
		return nil, m.failureError
	}
	
	// Simple mock implementation - return empty results for testing
	return &models.PostsListResponse{
		Posts:      []models.PostResponse{},
		NextCursor: "",
		PrevCursor: "",
		HasNext:    false,
		HasPrev:    false,
		Limit:      filter.Limit,
	}, nil
}

// GetCursorInfo returns mock cursor info for a given post (implements services.PostService)
func (m *MockPostService) GetCursorInfo(ctx context.Context, postID uuid.UUID, sortBy, sortOrder string) (*models.CursorInfo, error) {
    // Minimal mock implementation suitable for handler tests
    return &models.CursorInfo{
        PostId:    postID.String(),
        Cursor:    "",
        Position:  0,
        SortBy:    sortBy,
        SortOrder: sortOrder,
    }, nil
}

// Test cases

func TestPostHandler_CreatePost_Success(t *testing.T) {
	// Setup
	postID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()
	
	mockService := &MockPostService{
		createPostFunc: func(ctx context.Context, req *models.CreatePostRequest, user *types.UserContext) (*models.Post, error) {
			// Validate request
			if req.Body != "Test post content" {
				t.Errorf("Expected body 'Test post content', got '%s'", req.Body)
			}
			if req.PostTypeId != 1 {
				t.Errorf("Expected postTypeId 1, got %d", req.PostTypeId)
			}
			
			return &models.Post{
				ObjectId:   postID,
				PostTypeId: req.PostTypeId,
				Body:       req.Body,
				OwnerUserId: userID,
			}, nil
		},
	}
	
	jwtConfig, hmacConfig := createTestConfig()
	handler := handlers.NewPostHandler(mockService, jwtConfig, hmacConfig)
	app := fiber.New()
	
	// Create request
	reqBody := models.CreatePostRequest{
		PostTypeId: 1,
		Body:       "Test post content",
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqJSON))
	req.Header.Set(types.HeaderContentType, "application/json")
	
	// Setup user context
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID:      userID,
			DisplayName: "Test User",
			Avatar:      "avatar.jpg",
			SocialName:  "testuser",
		})
		return c.Next()
	})
	
	app.Post("/posts", handler.CreatePost)
	
	// Execute
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	
	// Verify
	if resp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
	
	var response models.CreatePostResponse
	json.NewDecoder(resp.Body).Decode(&response)
	
	if response.ObjectId != postID.String() {
		t.Errorf("Expected objectId %s, got %s", postID.String(), response.ObjectId)
	}
}

func TestPostHandler_CreatePost_ValidationError(t *testing.T) {
	mockService := &MockPostService{}
	jwtConfig, hmacConfig := createTestConfig()
	handler := handlers.NewPostHandler(mockService, jwtConfig, hmacConfig)
	app := fiber.New()
	
	// Create invalid request (missing required fields)
	reqBody := models.CreatePostRequest{
		// PostTypeId missing
		Body: "", // Empty body
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqJSON))
	req.Header.Set(types.HeaderContentType, "application/json")
	
	app.Post("/posts", handler.CreatePost)
	
	// Execute
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	
	// Verify
	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestPostHandler_GetPost_Success(t *testing.T) {
	// Setup
	postID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()
	
	expectedPost := &models.Post{
		ObjectId:         postID,
		PostTypeId:       1,
		Body:            "Test post content",
		OwnerUserId:     userID,
		OwnerDisplayName: "Test User",
		ViewCount:       5,
	}
	
	mockService := &MockPostService{
		getPostFunc: func(ctx context.Context, id uuid.UUID) (*models.Post, error) {
			if id != postID {
				t.Errorf("Expected postID %s, got %s", postID.String(), id.String())
			}
			return expectedPost, nil
		},
		incrementViewCountFunc: func(ctx context.Context, id uuid.UUID) error {
			// Verify view count increment is called
			return nil
		},
	}
	
	jwtConfig, hmacConfig := createTestConfig()
	handler := handlers.NewPostHandler(mockService, jwtConfig, hmacConfig)
	app := fiber.New()
	
	req := httptest.NewRequest("GET", "/posts/"+postID.String(), nil)
	app.Get("/posts/:postId", handler.GetPost)
	
	// Execute
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	
	// Verify
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var response models.PostResponse
	json.NewDecoder(resp.Body).Decode(&response)
	
	if response.ObjectId != postID.String() {
		t.Errorf("Expected objectId %s, got %s", postID.String(), response.ObjectId)
	}
	if response.Body != "Test post content" {
		t.Errorf("Expected body 'Test post content', got '%s'", response.Body)
	}
}

func TestPostHandler_GetPost_NotFound(t *testing.T) {
	postID, _ := uuid.NewV4()
	
	mockService := &MockPostService{
		getPostFunc: func(ctx context.Context, id uuid.UUID) (*models.Post, error) {
			return nil, errors.New("post not found")
		},
	}
	
	jwtConfig, hmacConfig := createTestConfig()
	handler := handlers.NewPostHandler(mockService, jwtConfig, hmacConfig)
	app := fiber.New()
	
	req := httptest.NewRequest("GET", "/posts/"+postID.String(), nil)
	app.Get("/posts/:postId", handler.GetPost)
	
	// Execute
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	
	// Verify
	if resp.StatusCode != 500 { // Will be 500 due to generic error handling
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestPostHandler_QueryPosts_Success(t *testing.T) {
	mockService := &MockPostService{
		queryPostsFunc: func(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
			// Verify filter parameters
			if filter.Page != 1 {
				t.Errorf("Expected page 1, got %d", filter.Page)
			}
			if filter.Limit != 20 {
				t.Errorf("Expected limit 20, got %d", filter.Limit)
			}
			
			return &models.PostsListResponse{
				Posts:      []models.PostResponse{},
				TotalCount: 0,
				Page:       1,
				Limit:      20,
				HasMore:    false,
			}, nil
		},
	}
	
	jwtConfig, hmacConfig := createTestConfig()
	handler := handlers.NewPostHandler(mockService, jwtConfig, hmacConfig)
	app := fiber.New()
	
	req := httptest.NewRequest("GET", "/posts?page=1&limit=20", nil)
	app.Get("/posts", handler.QueryPosts)
	
	// Execute
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	
	// Verify
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// Benchmark tests to verify performance improvements

func BenchmarkPostHandler_CreatePost(b *testing.B) {
	userID, _ := uuid.NewV4()
	mockService := &MockPostService{
		createPostFunc: func(ctx context.Context, req *models.CreatePostRequest, user *types.UserContext) (*models.Post, error) {
			postID, _ := uuid.NewV4()
			return &models.Post{
				ObjectId:    postID,
				PostTypeId:  req.PostTypeId,
				Body:        req.Body,
				OwnerUserId: userID,
			}, nil
		},
	}
	
	jwtConfig, hmacConfig := createTestConfig()
	handler := handlers.NewPostHandler(mockService, jwtConfig, hmacConfig)
	app := fiber.New()
	
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID:      userID,
			DisplayName: "Test User",
			SocialName:  "testuser",
		})
		return c.Next()
	})
	
	app.Post("/posts", handler.CreatePost)
	
	reqBody := models.CreatePostRequest{
		PostTypeId: 1,
		Body:       "Benchmark test content",
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqJSON))
		req.Header.Set(types.HeaderContentType, "application/json")
		
		app.Test(req)
	}
}

// Example test demonstrating advanced MockPostService capabilities
func TestMockPostService_AdvancedFeatures(t *testing.T) {
	userID, _ := uuid.NewV4()
	otherUserID, _ := uuid.NewV4()
	
	// Create mock service
	mockService := &MockPostService{}
	
	// Test 1: Basic post management
	testPost := CreateTestPost(userID, "Test post for advanced features")
	mockService.AddPost(testPost)
	
	if mockService.GetPostCount() != 1 {
		t.Errorf("Expected 1 post, got %d", mockService.GetPostCount())
	}
	
	retrieved := mockService.GetMockPost(testPost.ObjectId)
	if retrieved == nil {
		t.Error("Expected post to be retrieved, got nil")
		return // Early return to avoid nil pointer dereference
	}
	if retrieved.Body != testPost.Body {
		t.Errorf("Expected body '%s', got '%s'", testPost.Body, retrieved.Body)
	}
	
	// Test 2: Ownership validation
	var err error
	err = mockService.DeleteByOwner(context.Background(), otherUserID, testPost.ObjectId)
	if err == nil {
		t.Error("Expected ownership validation error")
	}
	if err != nil && !contains(err.Error(), "ownership validation failed") && !contains(err.Error(), "not found") && !contains(err.Error(), "access denied") {
		t.Errorf("Expected ownership validation error, got: %v", err)
	}
	
	// Test 3: Successful ownership operations
	err = mockService.DeleteByOwner(context.Background(), userID, testPost.ObjectId)
	if err != nil {
		t.Errorf("Expected successful deletion, got error: %v", err)
	}
	if mockService.GetPostCount() != 0 {
		t.Errorf("Expected 0 posts after deletion, got %d", mockService.GetPostCount())
	}
	
	// Test 4: Custom failure scenarios
	testPost2 := CreateTestPost(userID, "Another test post")
	mockService.AddPost(testPost2)
	
	mockService.SetFailure(errors.New("simulated database error"))
	err = mockService.IncrementFields(context.Background(), testPost2.ObjectId, map[string]interface{}{
		"viewCount": int64(1),
	})
	if err == nil {
		t.Error("Expected simulated database error")
	}
	if err != nil && err.Error() != "simulated database error" {
		t.Errorf("Expected 'simulated database error', got: %v", err)
	}
	
	// Test 5: Clear failure and test increment
	mockService.ClearFailure()
	err = mockService.IncrementFields(context.Background(), testPost2.ObjectId, map[string]interface{}{
		"viewCount": int64(5),
		"score":     int64(10),
	})
	if err != nil {
		t.Errorf("Expected successful increment, got error: %v", err)
	}
	
	updated := mockService.GetMockPost(testPost2.ObjectId)
	if updated == nil {
		t.Error("Expected updated post to be retrieved, got nil")
		return
	}
	if updated.ViewCount != int64(5) {
		t.Errorf("Expected ViewCount 5, got %d", updated.ViewCount)
	}
	if updated.Score != int64(10) {
		t.Errorf("Expected Score 10, got %d", updated.Score)
	}
	
	// Test 6: Custom behavior configuration
	customCalled := false
	mockService.ConfigureUpdateFields(func(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error {
		customCalled = true
		return errors.New("custom update error")
	})
	
	err = mockService.UpdateFields(context.Background(), testPost2.ObjectId, map[string]interface{}{
		"body": "Updated content",
	})
	if err == nil {
		t.Error("Expected custom update error")
	}
	if !customCalled {
		t.Error("Expected custom function to be called")
	}
	if err != nil && err.Error() != "custom update error" {
		t.Errorf("Expected 'custom update error', got: %v", err)
	}
	
	// Test 7: Reset custom behavior
	mockService.ResetAllCustomBehavior()
	err = mockService.UpdateFields(context.Background(), testPost2.ObjectId, map[string]interface{}{
		"body": "Updated content successfully",
	})
	if err != nil {
		t.Errorf("Expected successful update after reset, got error: %v", err)
	}
	
	finalUpdated := mockService.GetMockPost(testPost2.ObjectId)
	if finalUpdated == nil {
		t.Error("Expected final updated post to be retrieved, got nil")
		return
	}
	if finalUpdated.Body != "Updated content successfully" {
		t.Errorf("Expected body 'Updated content successfully', got '%s'", finalUpdated.Body)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper methods for configuring MockPostService behavior

// SetFailure configures the mock to fail with the given error
func (m *MockPostService) SetFailure(err error) {
	m.shouldFail = true
	m.failureError = err
}

// ClearFailure resets the mock to not fail
func (m *MockPostService) ClearFailure() {
	m.shouldFail = false
	m.failureError = nil
}

// AddPost adds a post to the mock storage for testing
func (m *MockPostService) AddPost(post *models.Post) {
	if m.posts == nil {
		m.posts = make(map[string]*models.Post)
	}
	m.posts[post.ObjectId.String()] = post
}

// GetMockPost retrieves a post from mock storage (helper method)
func (m *MockPostService) GetMockPost(postID uuid.UUID) *models.Post {
	if m.posts == nil {
		return nil
	}
	return m.posts[postID.String()]
}

// ClearPosts removes all posts from mock storage
func (m *MockPostService) ClearPosts() {
	m.posts = make(map[string]*models.Post)
}

// GetPostCount returns the number of posts in mock storage
func (m *MockPostService) GetPostCount() int {
	if m.posts == nil {
		return 0
	}
	return len(m.posts)
}

// ConfigureDeleteWithOwnership sets a custom implementation for DeleteWithOwnership
func (m *MockPostService) ConfigureDeleteWithOwnership(fn func(context.Context, uuid.UUID, uuid.UUID) error) {
	m.deleteWithOwnershipFunc = fn
}

// ConfigureIncrementFields sets a custom implementation for IncrementFields
func (m *MockPostService) ConfigureIncrementFields(fn func(context.Context, uuid.UUID, map[string]interface{}) error) {
	m.incrementFieldsFunc = fn
}

// ConfigureUpdateFields sets a custom implementation for UpdateFields
func (m *MockPostService) ConfigureUpdateFields(fn func(context.Context, uuid.UUID, map[string]interface{}) error) {
	m.updateFieldsFunc = fn
}

// ResetAllCustomBehavior clears all custom function implementations
func (m *MockPostService) ResetAllCustomBehavior() {
	m.deleteWithOwnershipFunc = nil
	m.incrementFieldsFunc = nil
	m.incrementFieldsWithOwnershipFunc = nil
	m.updateFieldsFunc = nil
	m.updateFieldsWithOwnershipFunc = nil
	m.updateAndIncrementFieldsFunc = nil
	m.deleteByOwnerFunc = nil
	m.incrementFieldFunc = nil
	m.setFieldFunc = nil
	m.updateByOwnerFunc = nil
	m.updateProfileForOwnerFunc = nil
}

// CreateTestPost creates a sample post for testing purposes
func CreateTestPost(ownerID uuid.UUID, body string) *models.Post {
	postID, _ := uuid.NewV4()
	return &models.Post{
		ObjectId:         postID,
		PostTypeId:       1,
		Score:            0,
		ViewCount:        0,
		Body:             body,
		OwnerUserId:      ownerID,
		OwnerDisplayName: "Test User",
		OwnerAvatar:      "avatar.jpg",
		CommentCounter:   0,
		DisableComments:  false,
		DisableSharing:   false,
		Deleted:          false,
		CreatedDate:      time.Now().Unix(),
		LastUpdated:      time.Now().Unix(),
		Permission:       "Public",
		Version:          "1.0",
	}
}
