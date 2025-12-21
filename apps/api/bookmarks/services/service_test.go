package services

import (
	"context"
	"errors"
	"testing"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/bookmarks/repository"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestToggleBookmark(t *testing.T) {
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	postID := uuid.Must(uuid.NewV4())

	t.Run("creates bookmark when absent", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockRepo.On("AddBookmark", ctx, userID, postID).Return(true, nil).Once()

		svc := NewService(mockRepo, nil)
		state, err := svc.ToggleBookmark(ctx, userID, postID)

		require.NoError(t, err)
		require.True(t, state)
		mockRepo.AssertExpectations(t)
	})

	t.Run("removes bookmark when present", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockRepo.On("AddBookmark", ctx, userID, postID).Return(false, nil).Once()
		mockRepo.On("RemoveBookmark", ctx, userID, postID).Return(true, nil).Once()

		svc := NewService(mockRepo, nil)
		state, err := svc.ToggleBookmark(ctx, userID, postID)

		require.NoError(t, err)
		require.False(t, state)
		mockRepo.AssertExpectations(t)
	})

	t.Run("propagates add errors", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockRepo.On("AddBookmark", ctx, userID, postID).Return(false, errors.New("db down")).Once()

		svc := NewService(mockRepo, nil)
		_, err := svc.ToggleBookmark(ctx, userID, postID)

		require.Error(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestListBookmarks(t *testing.T) {
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	postID := uuid.Must(uuid.NewV4())

	mockRepo := new(MockRepository)
	mockPostSvc := new(MockPostService)

	entry := repository.BookmarkEntry{PostID: postID}
	mockRepo.On("FindMyBookmarks", ctx, userID, "", 10).Return([]repository.BookmarkEntry{entry}, "", nil).Once()

	mockPost := &models.Post{ObjectId: postID}
	mockPostSvc.On("GetPostsByIDs", ctx, []uuid.UUID{postID}).Return([]*models.Post{mockPost}, nil).Once()
	mockPostSvc.On("ConvertPostToResponse", ctx, mockPost).Return(models.PostResponse{ObjectId: postID.String()}).Once()

	svc := NewService(mockRepo, mockPostSvc)
	resp, err := svc.ListBookmarks(ctx, userID, "", 10)

	require.NoError(t, err)
	require.Len(t, resp.Posts, 1)
	require.True(t, resp.Posts[0].IsBookmarked)
	mockRepo.AssertExpectations(t)
	mockPostSvc.AssertExpectations(t)
}

type MockPostService struct {
	mock.Mock
}

func (m *MockPostService) GetPostsByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Post, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]*models.Post), args.Error(1)
}

func (m *MockPostService) ConvertPostToResponse(ctx context.Context, post *models.Post) models.PostResponse {
	args := m.Called(ctx, post)
	return args.Get(0).(models.PostResponse)
}
