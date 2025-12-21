// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	commentsErrors "github.com/qolzam/telar/apps/api/comments/errors"
	"github.com/qolzam/telar/apps/api/comments/models"
	commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
	"github.com/qolzam/telar/apps/api/comments/services/mocks"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
)

func createTestUserContext() *types.UserContext {
	userID := uuid.Must(uuid.NewV4())
	return &types.UserContext{
		UserID:      userID,
		Username:    "test@example.com",
		DisplayName: "Test User",
		Avatar:      "https://example.com/avatar.jpg",
		SocialName:  "testuser",
		SystemRole:  "user",
		CreatedDate: time.Now().Unix(),
	}
}

func createTestCreateCommentRequest() *models.CreateCommentRequest {
	postID := uuid.Must(uuid.NewV4())
	return &models.CreateCommentRequest{
		PostId:          postID,
		Text:            "This is a test comment",
		ParentCommentId: nil, // Root comment
	}
}

func createTestComment() models.Comment {
	commentID := uuid.Must(uuid.NewV4())
	postID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	now := time.Now().Unix()
	return models.Comment{
		ObjectId:         commentID,
		PostId:           postID,
		OwnerUserId:      userID,
		OwnerDisplayName: "Test User",
		OwnerAvatar:      "https://example.com/avatar.jpg",
		Text:             "Test comment text",
		Score:            0,
		Deleted:          false,
		DeletedDate:      0,
		ParentCommentId:  nil,
		CreatedDate:      now,
		LastUpdated:      now,
	}
}

func setupTestService() (*commentService, *mocks.MockCommentRepository, *mocks.MockPostRepository) {
	mockCommentRepo := &mocks.MockCommentRepository{}
	mockPostRepo := &mocks.MockPostRepository{}
	cfg := &platformconfig.Config{}
	svc := &commentService{
		commentRepo:      mockCommentRepo,
		postRepo:         mockPostRepo,
		cacheService:     nil,
		config:           cfg,
		postStatsUpdater: nil,
	}
	return svc, mockCommentRepo, mockPostRepo
}

// Test CreateComment with valid request (root comment)
func TestCreateComment_ValidRequest_Success(t *testing.T) {
	service, mockCommentRepo, mockPostRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreateCommentRequest()
	req.ParentCommentId = nil // Root comment

	// Setup mock expectations for transaction
	mockPostRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		// Execute the transaction function
		fn(ctx)
	})

	// Setup expectations for Create (called within transaction)
	mockCommentRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Comment")).Return(nil).Run(func(args mock.Arguments) {
		comment := args.Get(1).(*models.Comment)
		assert.Equal(t, req.PostId, comment.PostId)
		assert.Equal(t, req.Text, comment.Text)
		assert.Equal(t, user.UserID, comment.OwnerUserId)
		assert.Equal(t, user.DisplayName, comment.OwnerDisplayName)
		assert.Equal(t, user.Avatar, comment.OwnerAvatar)
		assert.False(t, comment.Deleted)
		assert.Nil(t, comment.ParentCommentId)
	})

	// Setup expectations for IncrementCommentCount (called within transaction)
	mockPostRepo.On("IncrementCommentCount", mock.Anything, req.PostId, 1).Return(nil)

	// Execute
	result, err := service.CreateComment(ctx, req, user)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.PostId, result.PostId)
	assert.Equal(t, req.Text, result.Text)
	assert.Equal(t, user.UserID, result.OwnerUserId)
	assert.Equal(t, user.DisplayName, result.OwnerDisplayName)
	assert.Equal(t, user.Avatar, result.OwnerAvatar)
	assert.Equal(t, int64(0), result.Score)
	assert.False(t, result.Deleted)
	assert.NotEqual(t, uuid.Nil, result.ObjectId)
	assert.Greater(t, result.CreatedDate, int64(0))

	mockCommentRepo.AssertExpectations(t)
	mockPostRepo.AssertExpectations(t)
}

// Test CreateComment with reply (no count increment)
func TestCreateComment_Reply_NoCountIncrement(t *testing.T) {
	service, mockCommentRepo, mockPostRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreateCommentRequest()
	parentID := uuid.Must(uuid.NewV4())
	req.ParentCommentId = &parentID // Reply comment

	mockCommentRepo.On("FindByID", ctx, parentID).Return(&models.Comment{
		ObjectId:         parentID,
		PostId:           req.PostId,
		OwnerUserId:      user.UserID,
		OwnerDisplayName: user.DisplayName,
		Deleted:          false,
	}, nil)

	// Setup expectations for Create (no transaction for replies)
	mockCommentRepo.On("Create", ctx, mock.AnythingOfType("*models.Comment")).Return(nil).Run(func(args mock.Arguments) {
		comment := args.Get(1).(*models.Comment)
		assert.Equal(t, req.PostId, comment.PostId)
		assert.Equal(t, req.Text, comment.Text)
		assert.NotNil(t, comment.ParentCommentId)
		assert.Equal(t, parentID, *comment.ParentCommentId)
	})

	// Execute
	result, err := service.CreateComment(ctx, req, user)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.ParentCommentId)
	assert.Equal(t, parentID, *result.ParentCommentId)

	// Verify IncrementCommentCount was NOT called for replies
	mockPostRepo.AssertNotCalled(t, "IncrementCommentCount")
	mockPostRepo.AssertNotCalled(t, "WithTransaction")
	mockCommentRepo.AssertExpectations(t)
}

// Test CreateComment with nil request
func TestCreateComment_NilRequest_ReturnsError(t *testing.T) {
	service, _, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()

	// Execute
	result, err := service.CreateComment(ctx, nil, user)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "create comment request is required")
}

// Test CreateComment with nil user context
func TestCreateComment_NilUserContext_ReturnsError(t *testing.T) {
	service, _, _ := setupTestService()
	ctx := context.Background()
	req := createTestCreateCommentRequest()

	// Execute
	result, err := service.CreateComment(ctx, req, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user context is required")
}

// Test GetComment with valid ID
func TestGetComment_ValidId_ReturnsComment(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	testComment := createTestComment()
	testComment.Deleted = false
	commentID := testComment.ObjectId

	// Setup mock expectations
	mockCommentRepo.On("FindByID", ctx, commentID).Return(&testComment, nil)

	// Execute
	result, err := service.GetComment(ctx, commentID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, commentID, result.ObjectId)
	assert.Equal(t, testComment.Text, result.Text)
	assert.Equal(t, testComment.Score, result.Score)

	mockCommentRepo.AssertExpectations(t)
}

// Test GetComment with non-existent ID
func TestGetComment_NonExistentId_ReturnsError(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	commentID := uuid.Must(uuid.NewV4())

	// Setup mock expectations - return not found error
	mockCommentRepo.On("FindByID", ctx, commentID).Return(nil, errors.New("comment not found"))

	// Execute
	result, err := service.GetComment(ctx, commentID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, commentsErrors.ErrCommentNotFound, err)

	mockCommentRepo.AssertExpectations(t)
}

// Test GetComment with deleted comment
func TestGetComment_DeletedComment_ReturnsError(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	testComment := createTestComment()
	testComment.Deleted = true
	commentID := testComment.ObjectId

	// Setup mock expectations
	mockCommentRepo.On("FindByID", ctx, commentID).Return(&testComment, nil)

	// Execute
	result, err := service.GetComment(ctx, commentID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, commentsErrors.ErrCommentNotFound, err)

	mockCommentRepo.AssertExpectations(t)
}

// Test UpdateComment with valid request
func TestUpdateComment_ValidRequest_Success(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID
	testComment.Deleted = false
	commentID := testComment.ObjectId

	req := &models.UpdateCommentRequest{
		ObjectId: commentID,
		Text:     "Updated comment text",
	}

	// Setup mock expectations
	mockCommentRepo.On("FindByID", ctx, commentID).Return(&testComment, nil)
	mockCommentRepo.On("Update", ctx, mock.AnythingOfType("*models.Comment")).Return(nil).Run(func(args mock.Arguments) {
		comment := args.Get(1).(*models.Comment)
		assert.Equal(t, "Updated comment text", comment.Text)
		assert.Greater(t, comment.LastUpdated, int64(0))
	})

	// Execute
	updatedComment, err := service.UpdateComment(ctx, commentID, req, user)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, updatedComment)
	assert.Equal(t, "Updated comment text", updatedComment.Text)
	assert.Greater(t, updatedComment.LastUpdated, int64(0))
	mockCommentRepo.AssertExpectations(t)
}

// Test UpdateComment with unauthorized user
func TestUpdateComment_UnauthorizedUser_ReturnsError(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = uuid.Must(uuid.NewV4()) // Different user
	testComment.Deleted = false
	commentID := testComment.ObjectId

	req := &models.UpdateCommentRequest{
		ObjectId: commentID,
		Text:     "Updated comment text",
	}

	// Setup mock expectations
	mockCommentRepo.On("FindByID", ctx, commentID).Return(&testComment, nil)

	// Execute
	updatedComment, err := service.UpdateComment(ctx, commentID, req, user)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedComment)
	assert.Equal(t, commentsErrors.ErrCommentOwnershipRequired, err)

	mockCommentRepo.AssertExpectations(t)
}

// Test IncrementScore with valid user
func TestIncrementScore_ValidUser_Success(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	commentID := uuid.Must(uuid.NewV4())
	delta := 5

	// Setup mock expectations
	mockCommentRepo.On("IncrementScore", ctx, commentID, delta).Return(nil)

	// Execute
	err := service.IncrementScore(ctx, commentID, delta, user)

	// Assert
	assert.NoError(t, err)
	mockCommentRepo.AssertExpectations(t)
}

// Test IncrementScore with database error
func TestIncrementScore_DatabaseError_ReturnsError(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	commentID := uuid.Must(uuid.NewV4())
	delta := 5

	// Setup mock expectations with error
	mockCommentRepo.On("IncrementScore", ctx, commentID, delta).Return(errors.New("database connection failed"))

	// Execute
	err := service.IncrementScore(ctx, commentID, delta, user)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment score")

	mockCommentRepo.AssertExpectations(t)
}

// Test DeleteComment with valid ownership (root comment)
func TestDeleteComment_ValidOwnership_Success(t *testing.T) {
	service, mockCommentRepo, mockPostRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID
	testComment.Deleted = false
	testComment.ParentCommentId = nil // Root comment
	commentID := testComment.ObjectId
	postID := testComment.PostId

	// Setup mock expectations for transaction
	mockPostRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		fn(ctx)
	})

	// Setup expectations for FindByID (to verify ownership)
	mockCommentRepo.On("FindByID", ctx, commentID).Return(&testComment, nil)

	// Setup expectations for Delete (called within transaction)
	mockCommentRepo.On("Delete", mock.Anything, commentID).Return(nil)
	mockCommentRepo.On("DeleteRepliesByParentID", mock.Anything, commentID).Return(nil)

	// Setup expectations for IncrementCommentCount (called within transaction)
	mockPostRepo.On("IncrementCommentCount", mock.Anything, postID, -1).Return(nil)

	// Execute
	err := service.DeleteComment(ctx, commentID, postID, user)

	// Assert
	assert.NoError(t, err)
	mockCommentRepo.AssertExpectations(t)
	mockPostRepo.AssertExpectations(t)
}

// Test DeleteComment with reply (no count decrement)
func TestDeleteComment_Reply_NoCountDecrement(t *testing.T) {
	service, mockCommentRepo, mockPostRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID
	testComment.Deleted = false
	parentID := uuid.Must(uuid.NewV4())
	testComment.ParentCommentId = &parentID // Reply comment
	commentID := testComment.ObjectId
	postID := testComment.PostId

	// Setup expectations for FindByID (to verify ownership)
	mockCommentRepo.On("FindByID", ctx, commentID).Return(&testComment, nil)

	// Setup expectations for Delete (no transaction for replies)
	mockCommentRepo.On("Delete", ctx, commentID).Return(nil)

	// Execute
	err := service.DeleteComment(ctx, commentID, postID, user)

	// Assert
	assert.NoError(t, err)

	// Verify IncrementCommentCount was NOT called for replies
	mockPostRepo.AssertNotCalled(t, "IncrementCommentCount")
	mockPostRepo.AssertNotCalled(t, "WithTransaction")
	mockCommentRepo.AssertExpectations(t)
}

// Test DeleteComment with unauthorized user
func TestDeleteComment_UnauthorizedUser_ReturnsError(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = uuid.Must(uuid.NewV4()) // Different user
	testComment.Deleted = false
	commentID := testComment.ObjectId
	postID := testComment.PostId

	// Setup mock expectations
	mockCommentRepo.On("FindByID", ctx, commentID).Return(&testComment, nil)

	// Execute
	err := service.DeleteComment(ctx, commentID, postID, user)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, commentsErrors.ErrCommentOwnershipRequired, err)

	mockCommentRepo.AssertExpectations(t)
}

// Test GetCommentsByPost with filter
func TestGetCommentsByPost_WithFilter_ReturnsResults(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	postID := uuid.Must(uuid.NewV4())
	filter := &models.CommentQueryFilter{
		PostId:   &postID,
		Limit:    10,
		Page:     1,
		RootOnly: true,
	}

	expectedComment := createTestComment()
	expectedComment.PostId = postID
	expectedComment.ParentCommentId = nil
	expectedComments := []*models.Comment{&expectedComment}

	// Setup mock expectations
	mockCommentRepo.On("Find", ctx, mock.MatchedBy(func(f commentRepository.CommentFilter) bool {
		return f.PostID != nil && *f.PostID == postID && f.RootOnly == true
	}), 10, 0).Return(expectedComments, nil)

	mockCommentRepo.On("Count", ctx, mock.MatchedBy(func(f commentRepository.CommentFilter) bool {
		return f.PostID != nil && *f.PostID == postID && f.RootOnly == true
	})).Return(int64(1), nil)

	// Execute
	result, err := service.GetCommentsByPost(ctx, postID, filter)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Count)
	assert.Len(t, result.Comments, 1)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)

	mockCommentRepo.AssertExpectations(t)
}

// Test GetReplyCount
func TestGetReplyCount_Success(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	parentID := uuid.Must(uuid.NewV4())

	// Setup mock expectations
	mockCommentRepo.On("CountReplies", ctx, parentID).Return(int64(5), nil)

	// Execute
	count, err := service.GetReplyCount(ctx, parentID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)

	mockCommentRepo.AssertExpectations(t)
}

// Note: Concurrent atomicity testing for ToggleLike is handled by the integration test
// in repository/comment_votes_test.go, which tests against the real database (thread-safe).
// Unit tests with mocks are not suitable for concurrent testing when the race detector is enabled.

// Test UpdateCommentProfile
func TestUpdateCommentProfile_ValidParameters_Success(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	displayName := "Updated Display Name"
	avatar := "https://example.com/new-avatar.jpg"

	// Setup mock expectations
	mockCommentRepo.On("UpdateOwnerProfile", ctx, userID, displayName, avatar).Return(nil)

	// Execute
	err := service.UpdateCommentProfile(ctx, userID, displayName, avatar)

	// Assert
	assert.NoError(t, err)
	mockCommentRepo.AssertExpectations(t)
}

// Test GetRootCommentCount
func TestGetRootCommentCount_Success(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	postID := uuid.Must(uuid.NewV4())

	// Setup mock expectations
	mockCommentRepo.On("CountByPostID", ctx, postID).Return(int64(10), nil)

	// Execute
	count, err := service.GetRootCommentCount(ctx, postID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, int64(10), count)

	mockCommentRepo.AssertExpectations(t)
}

// Test ValidateCommentOwnership with valid owner
func TestValidateCommentOwnership_ValidOwner_Success(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID
	testComment.Deleted = false

	// Setup mock expectations
	mockCommentRepo.On("FindByID", ctx, testComment.ObjectId).Return(&testComment, nil)

	// Execute
	err := service.ValidateCommentOwnership(ctx, testComment.ObjectId, user.UserID)

	// Assert
	assert.NoError(t, err)
	mockCommentRepo.AssertExpectations(t)
}

// Test ValidateCommentOwnership with invalid owner
func TestValidateCommentOwnership_InvalidOwner_ReturnsError(t *testing.T) {
	service, mockCommentRepo, _ := setupTestService()
	ctx := context.Background()
	commentID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	testComment := createTestComment()
	testComment.OwnerUserId = uuid.Must(uuid.NewV4()) // Different user
	testComment.Deleted = false

	// Setup mock expectations
	mockCommentRepo.On("FindByID", ctx, commentID).Return(&testComment, nil)

	// Execute
	err := service.ValidateCommentOwnership(ctx, commentID, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, commentsErrors.ErrCommentNotFound, err)

	mockCommentRepo.AssertExpectations(t)
}
