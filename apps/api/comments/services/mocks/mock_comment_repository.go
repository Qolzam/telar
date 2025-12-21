// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package mocks

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/models"
	commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
	"github.com/stretchr/testify/mock"
)

// MockCommentRepository is a mock implementation of CommentRepository
type MockCommentRepository struct {
	mock.Mock
}

var _ commentRepository.CommentRepository = (*MockCommentRepository)(nil)

func (m *MockCommentRepository) Create(ctx context.Context, comment *models.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}

func (m *MockCommentRepository) FindByID(ctx context.Context, commentID uuid.UUID) (*models.Comment, error) {
	args := m.Called(ctx, commentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Comment), args.Error(1)
}

func (m *MockCommentRepository) FindByPostID(ctx context.Context, postID uuid.UUID, limit, offset int) ([]*models.Comment, error) {
	args := m.Called(ctx, postID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Comment), args.Error(1)
}

func (m *MockCommentRepository) FindByPostIDWithCursor(ctx context.Context, postID uuid.UUID, cursor string, limit int) ([]*models.Comment, string, error) {
	args := m.Called(ctx, postID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).([]*models.Comment), args.String(1), args.Error(2)
}

func (m *MockCommentRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Comment, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Comment), args.Error(1)
}

func (m *MockCommentRepository) FindReplies(ctx context.Context, parentID uuid.UUID, limit, offset int) ([]*models.Comment, error) {
	args := m.Called(ctx, parentID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Comment), args.Error(1)
}

func (m *MockCommentRepository) FindRepliesWithCursor(ctx context.Context, parentID uuid.UUID, cursor string, limit int) ([]*models.Comment, string, error) {
	args := m.Called(ctx, parentID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).([]*models.Comment), args.String(1), args.Error(2)
}

func (m *MockCommentRepository) CountByPostID(ctx context.Context, postID uuid.UUID) (int64, error) {
	args := m.Called(ctx, postID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCommentRepository) CountByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	args := m.Called(ctx, postIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]int64), args.Error(1)
}

func (m *MockCommentRepository) CountReplies(ctx context.Context, parentID uuid.UUID) (int64, error) {
	args := m.Called(ctx, parentID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCommentRepository) CountRepliesBulk(ctx context.Context, parentIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	args := m.Called(ctx, parentIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]int64), args.Error(1)
}

func (m *MockCommentRepository) Find(ctx context.Context, filter commentRepository.CommentFilter, limit, offset int) ([]*models.Comment, error) {
	args := m.Called(ctx, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Comment), args.Error(1)
}

func (m *MockCommentRepository) Count(ctx context.Context, filter commentRepository.CommentFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCommentRepository) Update(ctx context.Context, comment *models.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}

func (m *MockCommentRepository) UpdateOwnerProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
	args := m.Called(ctx, userID, displayName, avatar)
	return args.Error(0)
}

func (m *MockCommentRepository) IncrementScore(ctx context.Context, commentID uuid.UUID, delta int) error {
	args := m.Called(ctx, commentID, delta)
	return args.Error(0)
}

func (m *MockCommentRepository) Delete(ctx context.Context, commentID uuid.UUID) error {
	args := m.Called(ctx, commentID)
	return args.Error(0)
}

func (m *MockCommentRepository) DeleteByPostID(ctx context.Context, postID uuid.UUID) error {
	args := m.Called(ctx, postID)
	return args.Error(0)
}

func (m *MockCommentRepository) DeleteRepliesByParentID(ctx context.Context, parentID uuid.UUID) error {
	args := m.Called(ctx, parentID)
	return args.Error(0)
}

func (m *MockCommentRepository) AddVote(ctx context.Context, commentID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, commentID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCommentRepository) RemoveVote(ctx context.Context, commentID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, commentID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCommentRepository) GetUserVotesForComments(ctx context.Context, commentIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	args := m.Called(ctx, commentIDs, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]bool), args.Error(1)
}

func (m *MockCommentRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	// Execute the function within the mock
	if fn != nil {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return args.Error(0)
}
