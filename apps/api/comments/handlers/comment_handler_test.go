package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	commentErrors "github.com/qolzam/telar/apps/api/comments/errors"
	"github.com/qolzam/telar/apps/api/comments/handlers"
	"github.com/qolzam/telar/apps/api/comments/models"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommentService implements the CommentService interface for testing
type MockCommentService struct {
	createCommentFunc                func(ctx context.Context, req *models.CreateCommentRequest, user *types.UserContext) (*models.Comment, error)
	getCommentFunc                   func(ctx context.Context, commentID uuid.UUID) (*models.Comment, error)
	getCommentsByPostFunc            func(ctx context.Context, postID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error)
	getCommentsByUserFunc            func(ctx context.Context, userID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error)
	queryCommentsFunc                func(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error)
	queryCommentsWithCursorFunc      func(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error)
	updateCommentFunc                func(ctx context.Context, commentID uuid.UUID, req *models.UpdateCommentRequest, user *types.UserContext) error
	updateCommentProfileFunc         func(ctx context.Context, userID uuid.UUID, displayName, avatar string) error
	incrementScoreFunc               func(ctx context.Context, commentID uuid.UUID, delta int, user *types.UserContext) error
	deleteCommentFunc                func(ctx context.Context, commentID uuid.UUID, postID uuid.UUID, user *types.UserContext) error
	deleteCommentsByPostFunc         func(ctx context.Context, postID uuid.UUID, user *types.UserContext) error
	softDeleteCommentFunc            func(ctx context.Context, commentID uuid.UUID, user *types.UserContext) error
	validateCommentOwnershipFunc     func(ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error
	createIndexFunc                  func(ctx context.Context, indexes map[string]interface{}) error
	deleteByOwnerFunc                func(ctx context.Context, owner uuid.UUID, objectId uuid.UUID) error
	setFieldFunc                     func(ctx context.Context, objectId uuid.UUID, field string, value interface{}) error
	incrementFieldFunc               func(ctx context.Context, objectId uuid.UUID, field string, delta int) error
	updateByOwnerFunc                func(ctx context.Context, objectId uuid.UUID, owner uuid.UUID, fields map[string]interface{}) error
	updateProfileForOwnerFunc        func(ctx context.Context, owner uuid.UUID, displayName, avatar string) error
	updateFieldsFunc                 func(ctx context.Context, commentID uuid.UUID, updates map[string]interface{}) error
	incrementFieldsFunc              func(ctx context.Context, commentID uuid.UUID, increments map[string]interface{}) error
	updateAndIncrementFieldsFunc     func(ctx context.Context, commentID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error
	updateFieldsWithOwnershipFunc    func(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID, updates map[string]interface{}) error
	deleteWithOwnershipFunc          func(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID) error
	incrementFieldsWithOwnershipFunc func(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID, increments map[string]interface{}) error

	// Mock state for testing
	shouldFail   bool
	failureError error
}

func (m *MockCommentService) CreateComment(ctx context.Context, req *models.CreateCommentRequest, user *types.UserContext) (*models.Comment, error) {
	if m.createCommentFunc != nil {
		return m.createCommentFunc(ctx, req, user)
	}
	if m.shouldFail {
		return nil, m.failureError
	}
	return nil, nil
}

func (m *MockCommentService) CreateIndex(ctx context.Context, indexes map[string]interface{}) error {
	if m.createIndexFunc != nil {
		return m.createIndexFunc(ctx, indexes)
	}
	return nil
}

func (m *MockCommentService) GetComment(ctx context.Context, commentID uuid.UUID) (*models.Comment, error) {
	if m.getCommentFunc != nil {
		return m.getCommentFunc(ctx, commentID)
	}
	if m.shouldFail {
		return nil, m.failureError
	}
	return nil, nil
}

func (m *MockCommentService) GetCommentsByPost(ctx context.Context, postID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
	if m.getCommentsByPostFunc != nil {
		return m.getCommentsByPostFunc(ctx, postID, filter)
	}
	if m.shouldFail {
		return nil, m.failureError
	}
	return nil, nil
}

func (m *MockCommentService) GetCommentsByUser(ctx context.Context, userID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
	if m.getCommentsByUserFunc != nil {
		return m.getCommentsByUserFunc(ctx, userID, filter)
	}
	return nil, nil
}

func (m *MockCommentService) QueryComments(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
	if m.queryCommentsFunc != nil {
		return m.queryCommentsFunc(ctx, filter)
	}
	return nil, nil
}

func (m *MockCommentService) QueryCommentsWithCursor(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
	if m.queryCommentsWithCursorFunc != nil {
		return m.queryCommentsWithCursorFunc(ctx, filter)
	}
	return nil, nil
}

func (m *MockCommentService) UpdateComment(ctx context.Context, commentID uuid.UUID, req *models.UpdateCommentRequest, user *types.UserContext) error {
	if m.updateCommentFunc != nil {
		return m.updateCommentFunc(ctx, commentID, req, user)
	}
	if m.shouldFail {
		return m.failureError
	}
	return nil
}

func (m *MockCommentService) UpdateCommentProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
	if m.updateCommentProfileFunc != nil {
		return m.updateCommentProfileFunc(ctx, userID, displayName, avatar)
	}
	return nil
}

func (m *MockCommentService) IncrementScore(ctx context.Context, commentID uuid.UUID, delta int, user *types.UserContext) error {
	if m.incrementScoreFunc != nil {
		return m.incrementScoreFunc(ctx, commentID, delta, user)
	}
	return nil
}

func (m *MockCommentService) DeleteComment(ctx context.Context, commentID uuid.UUID, postID uuid.UUID, user *types.UserContext) error {
	if m.deleteCommentFunc != nil {
		return m.deleteCommentFunc(ctx, commentID, postID, user)
	}
	if m.shouldFail {
		return m.failureError
	}
	return nil
}

func (m *MockCommentService) DeleteCommentsByPost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	if m.deleteCommentsByPostFunc != nil {
		return m.deleteCommentsByPostFunc(ctx, postID, user)
	}
	return nil
}

func (m *MockCommentService) SoftDeleteComment(ctx context.Context, commentID uuid.UUID, user *types.UserContext) error {
	if m.softDeleteCommentFunc != nil {
		return m.softDeleteCommentFunc(ctx, commentID, user)
	}
	return nil
}

func (m *MockCommentService) DeleteByOwner(ctx context.Context, owner uuid.UUID, objectId uuid.UUID) error {
	if m.deleteByOwnerFunc != nil {
		return m.deleteByOwnerFunc(ctx, owner, objectId)
	}
	return nil
}

func (m *MockCommentService) ValidateCommentOwnership(ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error {
	if m.validateCommentOwnershipFunc != nil {
		return m.validateCommentOwnershipFunc(ctx, commentID, userID)
	}
	if m.shouldFail {
		return m.failureError
	}
	return nil
}

func (m *MockCommentService) SetField(ctx context.Context, objectId uuid.UUID, field string, value interface{}) error {
	if m.setFieldFunc != nil {
		return m.setFieldFunc(ctx, objectId, field, value)
	}
	return nil
}

func (m *MockCommentService) IncrementField(ctx context.Context, objectId uuid.UUID, field string, delta int) error {
	if m.incrementFieldFunc != nil {
		return m.incrementFieldFunc(ctx, objectId, field, delta)
	}
	return nil
}

func (m *MockCommentService) UpdateByOwner(ctx context.Context, objectId uuid.UUID, owner uuid.UUID, fields map[string]interface{}) error {
	if m.updateByOwnerFunc != nil {
		return m.updateByOwnerFunc(ctx, objectId, owner, fields)
	}
	return nil
}

func (m *MockCommentService) UpdateProfileForOwner(ctx context.Context, owner uuid.UUID, displayName, avatar string) error {
	if m.updateProfileForOwnerFunc != nil {
		return m.updateProfileForOwnerFunc(ctx, owner, displayName, avatar)
	}
	return nil
}

func (m *MockCommentService) UpdateFields(ctx context.Context, commentID uuid.UUID, updates map[string]interface{}) error {
	if m.updateFieldsFunc != nil {
		return m.updateFieldsFunc(ctx, commentID, updates)
	}
	return nil
}

func (m *MockCommentService) IncrementFields(ctx context.Context, commentID uuid.UUID, increments map[string]interface{}) error {
	if m.incrementFieldsFunc != nil {
		return m.incrementFieldsFunc(ctx, commentID, increments)
	}
	return nil
}

func (m *MockCommentService) UpdateAndIncrementFields(ctx context.Context, commentID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error {
	if m.updateAndIncrementFieldsFunc != nil {
		return m.updateAndIncrementFieldsFunc(ctx, commentID, updates, increments)
	}
	return nil
}

func (m *MockCommentService) UpdateFieldsWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID, updates map[string]interface{}) error {
	if m.updateFieldsWithOwnershipFunc != nil {
		return m.updateFieldsWithOwnershipFunc(ctx, commentID, ownerID, updates)
	}
	return nil
}

func (m *MockCommentService) DeleteWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID) error {
	if m.deleteWithOwnershipFunc != nil {
		return m.deleteWithOwnershipFunc(ctx, commentID, ownerID)
	}
	return nil
}

func (m *MockCommentService) IncrementFieldsWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID, increments map[string]interface{}) error {
	if m.incrementFieldsWithOwnershipFunc != nil {
		return m.incrementFieldsWithOwnershipFunc(ctx, commentID, ownerID, increments)
	}
	return nil
}

// Test cases

func TestCommentHandler_CreateComment_Success(t *testing.T) {
	// Setup
	commentID, _ := uuid.NewV4()
	postID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()

	mockService := &MockCommentService{
		createCommentFunc: func(ctx context.Context, req *models.CreateCommentRequest, user *types.UserContext) (*models.Comment, error) {
			// Validate request
			assert.Equal(t, "Test comment content", req.Text)
			assert.Equal(t, postID, req.PostId)
			assert.Equal(t, userID, user.UserID)

			return &models.Comment{
				ObjectId:    commentID,
				PostId:      req.PostId,
				Text:        req.Text,
				OwnerUserId: userID,
				Score:       0,
				CreatedDate: time.Now().Unix(),
				LastUpdated: time.Now().Unix(),
			}, nil
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Create request
	reqBody := models.CreateCommentRequest{
		PostId: postID,
		Text:   "Test comment content",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/comments", bytes.NewReader(reqJSON))
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

	app.Post("/comments", handler.CreateComment)

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 201, resp.StatusCode)

	var response struct {
		ObjectId string `json:"objectId"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, commentID.String(), response.ObjectId)
}

func TestCommentHandler_CreateComment_ValidationError(t *testing.T) {
	mockService := &MockCommentService{}
	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Create request with missing required fields
	reqBody := models.CreateCommentRequest{
		Text: "", // Empty text should fail validation
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/comments", bytes.NewReader(reqJSON))
	req.Header.Set(types.HeaderContentType, "application/json")

	// Setup user context
	userID, _ := uuid.NewV4()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID:      userID,
			DisplayName: "Test User",
		})
		return c.Next()
	})

	app.Post("/comments", handler.CreateComment)

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 400, resp.StatusCode)
}

func TestCommentHandler_CreateComment_ServiceError(t *testing.T) {
	mockService := &MockCommentService{
		createCommentFunc: func(ctx context.Context, req *models.CreateCommentRequest, user *types.UserContext) (*models.Comment, error) {
			return nil, errors.New("service error")
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Create request
	postID, _ := uuid.NewV4()
	reqBody := models.CreateCommentRequest{
		PostId: postID,
		Text:   "Test comment content",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/comments", bytes.NewReader(reqJSON))
	req.Header.Set(types.HeaderContentType, "application/json")

	// Setup user context
	userID, _ := uuid.NewV4()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID:      userID,
			DisplayName: "Test User",
		})
		return c.Next()
	})

	app.Post("/comments", handler.CreateComment)

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 500, resp.StatusCode)
}

func TestCommentHandler_GetComment_Success(t *testing.T) {
	// Setup
	commentID, _ := uuid.NewV4()
	postID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()

	expectedComment := &models.Comment{
		ObjectId:    commentID,
		PostId:      postID,
		Text:        "Test comment",
		OwnerUserId: userID,
		Score:       5,
		CreatedDate: time.Now().Unix(),
	}

	mockService := &MockCommentService{
		getCommentFunc: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			assert.Equal(t, commentID, id)
			return expectedComment, nil
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	app.Get("/comments/:commentId", handler.GetComment)

	// Execute request
	req := httptest.NewRequest("GET", "/comments/"+commentID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)

	var response models.Comment
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, expectedComment.ObjectId, response.ObjectId)
	assert.Equal(t, expectedComment.Text, response.Text)
}

func TestCommentHandler_GetComment_NotFound(t *testing.T) {
	commentID, _ := uuid.NewV4()

	mockService := &MockCommentService{
		getCommentFunc: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			return nil, errors.New("comment not found")
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	app.Get("/comments/:commentId", handler.GetComment)

	// Execute request
	req := httptest.NewRequest("GET", "/comments/"+commentID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 500, resp.StatusCode)
}

func TestCommentHandler_GetComment_InvalidID(t *testing.T) {
	mockService := &MockCommentService{}
	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	app.Get("/comments/:commentId", handler.GetComment)

	// Execute request with invalid UUID
	req := httptest.NewRequest("GET", "/comments/invalid-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 400, resp.StatusCode)
}

func TestCommentHandler_UpdateComment_Success(t *testing.T) {
	// Setup
	commentID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()

	updatedComment := &models.Comment{
		ObjectId:    commentID,
		Text:        "Updated comment text",
		OwnerUserId: userID,
		Score:       0,
		CreatedDate: time.Now().Unix(),
		LastUpdated: time.Now().Unix(),
	}

	mockService := &MockCommentService{
		updateCommentFunc: func(ctx context.Context, id uuid.UUID, req *models.UpdateCommentRequest, user *types.UserContext) error {
			assert.Equal(t, commentID, id)
			assert.Equal(t, commentID, req.ObjectId)
			assert.Equal(t, "Updated comment text", req.Text)
			assert.Equal(t, userID, user.UserID)
			return nil
		},
		getCommentFunc: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			assert.Equal(t, commentID, id)
			return updatedComment, nil
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Create request
	reqBody := models.UpdateCommentRequest{
		ObjectId: commentID,
		Text:     "Updated comment text",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/comments", bytes.NewReader(reqJSON))
	req.Header.Set(types.HeaderContentType, "application/json")

	// Setup user context
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID:      userID,
			DisplayName: "Test User",
		})
		return c.Next()
	})

	app.Put("/comments", handler.UpdateComment)

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCommentHandler_UpdateComment_ValidationError(t *testing.T) {
	commentID, _ := uuid.NewV4()

	mockService := &MockCommentService{}
	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Create request with invalid data (empty text)
	reqBody := models.UpdateCommentRequest{
		ObjectId: commentID,
		Text:     "", // Empty text should fail validation
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/comments", bytes.NewReader(reqJSON))
	req.Header.Set(types.HeaderContentType, "application/json")

	// Setup user context
	userID, _ := uuid.NewV4()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID: userID,
		})
		return c.Next()
	})

	app.Put("/comments", handler.UpdateComment)

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 400, resp.StatusCode)
}

func TestCommentHandler_UpdateComment_AuthorizationError(t *testing.T) {
	commentID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()

	mockService := &MockCommentService{
		updateCommentFunc: func(ctx context.Context, id uuid.UUID, req *models.UpdateCommentRequest, user *types.UserContext) error {
			return errors.New("unauthorized")
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Create request
	reqBody := models.UpdateCommentRequest{
		ObjectId: commentID,
		Text:     "Updated comment text",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/comments", bytes.NewReader(reqJSON))
	req.Header.Set(types.HeaderContentType, "application/json")

	// Setup user context
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID: userID,
		})
		return c.Next()
	})

	app.Put("/comments", handler.UpdateComment)

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 500, resp.StatusCode)
}

func TestCommentHandler_DeleteComment_Success(t *testing.T) {
	// Setup
	commentID, _ := uuid.NewV4()
	postID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()

	mockService := &MockCommentService{
		deleteCommentFunc: func(ctx context.Context, cID uuid.UUID, pID uuid.UUID, user *types.UserContext) error {
			assert.Equal(t, commentID, cID)
			assert.Equal(t, postID, pID)
			assert.Equal(t, userID, user.UserID)
			return nil
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Setup user context
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID: userID,
		})
		return c.Next()
	})

	app.Delete("/comments/:commentId/:postId", handler.DeleteComment)

	// Execute request
	req := httptest.NewRequest("DELETE", "/comments/"+commentID.String()+"/"+postID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 204, resp.StatusCode)
}

func TestCommentHandler_DeleteComment_OwnershipError(t *testing.T) {
	commentID, _ := uuid.NewV4()
	postID, _ := uuid.NewV4()
	userID, _ := uuid.NewV4()

	mockService := &MockCommentService{
		deleteCommentFunc: func(ctx context.Context, cID uuid.UUID, pID uuid.UUID, user *types.UserContext) error {
			return errors.New("comment ownership required")
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Setup user context
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID: userID,
		})
		return c.Next()
	})

	app.Delete("/comments/:commentId/:postId", handler.DeleteComment)

	// Execute request
	req := httptest.NewRequest("DELETE", "/comments/"+commentID.String()+"/"+postID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 500, resp.StatusCode)
}

func TestCommentHandler_GetCommentsByPost_Success(t *testing.T) {
	// Setup
	postID, _ := uuid.NewV4()
	commentID1, _ := uuid.NewV4()
	commentID2, _ := uuid.NewV4()

	expectedComments := &models.CommentsListResponse{
		Comments: []models.CommentResponse{
			{
				ObjectId: commentID1.String(),
				PostId:   postID.String(),
				Text:     "First comment",
			},
			{
				ObjectId: commentID2.String(),
				PostId:   postID.String(),
				Text:     "Second comment",
			},
		},
		Count: 2,
	}

	mockService := &MockCommentService{
		getCommentsByPostFunc: func(ctx context.Context, pID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
			assert.Equal(t, postID, pID)
			return expectedComments, nil
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	app.Get("/comments", handler.GetCommentsByPost)

	// Execute request
	req := httptest.NewRequest("GET", "/comments?postId="+postID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)

	var response []models.CommentResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, 2, len(response))
}

func TestCommentHandler_GetCommentsByPost_WithPagination(t *testing.T) {
	// Setup
	postID, _ := uuid.NewV4()

	mockService := &MockCommentService{
		getCommentsByPostFunc: func(ctx context.Context, pID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
			assert.Equal(t, postID, pID)
			assert.NotNil(t, filter)
			assert.Equal(t, 1, filter.Page)
			assert.Equal(t, 10, filter.Limit)
			return &models.CommentsListResponse{
				Comments: []models.CommentResponse{},
				Count:    0,
			}, nil
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	app.Get("/comments", handler.GetCommentsByPost)

	// Execute request with pagination parameters
	req := httptest.NewRequest("GET", "/comments?postId="+postID.String()+"&page=1&limit=10", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCommentHandler_MissingUserContext(t *testing.T) {
	mockService := &MockCommentService{}
	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Create request without user context
	postID, _ := uuid.NewV4()
	reqBody := models.CreateCommentRequest{
		PostId: postID,
		Text:   "Test comment content",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/comments", bytes.NewReader(reqJSON))
	req.Header.Set(types.HeaderContentType, "application/json")

	app.Post("/comments", handler.CreateComment)

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 401, resp.StatusCode)
}

func TestCommentHandler_InvalidRequestBody(t *testing.T) {
	mockService := &MockCommentService{}
	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	// Create request with invalid JSON
	req := httptest.NewRequest("POST", "/comments", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(types.HeaderContentType, "application/json")

	// Setup user context
	userID, _ := uuid.NewV4()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(types.UserCtxName, types.UserContext{
			UserID: userID,
		})
		return c.Next()
	})

	app.Post("/comments", handler.CreateComment)

	// Execute request
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 400, resp.StatusCode)
}

func TestCommentHandler_GetCommentsByPost_WithFiltering(t *testing.T) {
	// Setup
	postID, _ := uuid.NewV4()

	mockService := &MockCommentService{
		getCommentsByPostFunc: func(ctx context.Context, pID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
			assert.Equal(t, postID, pID)
			assert.NotNil(t, filter)
			// Test that filtering parameters were parsed correctly
			return &models.CommentsListResponse{
				Comments: []models.CommentResponse{},
				Count:    0,
			}, nil
		},
	}

	handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
	app := fiber.New()

	app.Get("/comments", handler.GetCommentsByPost)

	// Execute request with search filter
	req := httptest.NewRequest("GET", "/comments?postId="+postID.String()+"&page=1&limit=5", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)
}

func TestCommentHandler_ErrorResponseFormats(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func() *MockCommentService
		expectedStatus int
		endpoint       string
		method         string
		body           interface{}
	}{
		{
			name: "Service returns comment not found error",
			setupMock: func() *MockCommentService {
				return &MockCommentService{
					getCommentFunc: func(ctx context.Context, commentID uuid.UUID) (*models.Comment, error) {
						return nil, errors.New("comment not found")
					},
				}
			},
			expectedStatus: 500,
			endpoint:       "/comments/550e8400-e29b-41d4-a716-446655440000",
			method:         "GET",
		},
		{
			name: "Service returns authorization error",
			setupMock: func() *MockCommentService {
				return &MockCommentService{
					updateCommentFunc: func(ctx context.Context, commentID uuid.UUID, req *models.UpdateCommentRequest, user *types.UserContext) error {
						return errors.New("unauthorized")
					},
				}
			},
			expectedStatus: 500,
			endpoint:       "/comments",
			method:         "PUT",
			body: models.UpdateCommentRequest{
				ObjectId: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				Text:     "Updated text",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := tt.setupMock()
			handler := handlers.NewCommentHandler(mockService, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})
			app := fiber.New()

			// Setup user context for requests that need it
			if tt.method != "GET" {
				userID, _ := uuid.NewV4()
				app.Use(func(c *fiber.Ctx) error {
					c.Locals(types.UserCtxName, types.UserContext{
						UserID: userID,
					})
					return c.Next()
				})
			}

			// Setup routes
			switch tt.method {
			case "GET":
				app.Get("/comments/:commentId", handler.GetComment)
			case "PUT":
				app.Put("/comments", handler.UpdateComment)
			}

			// Create request
			var req *http.Request
			if tt.body != nil {
				bodyJSON, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.endpoint, bytes.NewReader(bodyJSON))
				req.Header.Set(types.HeaderContentType, "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.endpoint, nil)
			}

			// Execute request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Validate error response format (flat structure)
			var errResp commentErrors.ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&errResp)
			require.NoError(t, err)

			// Verify flat error response structure
			require.NotEmpty(t, errResp.Code, "Error response should have a code")
			require.NotEmpty(t, errResp.Message, "Error response should have a message")
			// Details field is optional, so we don't require it to be non-empty
		})
	}
}
