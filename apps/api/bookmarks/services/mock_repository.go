package services

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/bookmarks/repository"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a test double for the bookmark repository.
type MockRepository struct {
	mock.Mock
}

var _ repository.Repository = (*MockRepository)(nil)

func (m *MockRepository) AddBookmark(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, postID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) RemoveBookmark(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, postID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetMapByUserAndPosts(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	args := m.Called(ctx, userID, postIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]bool), args.Error(1)
}

func (m *MockRepository) FindMyBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]repository.BookmarkEntry, string, error) {
	args := m.Called(ctx, userID, cursor, limit)
	return args.Get(0).([]repository.BookmarkEntry), args.String(1), args.Error(2)
}
