// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/posts/repository"
)

// MockPostRepositoryForVotes is a mock implementation of PostRepository for vote service testing
type MockPostRepositoryForVotes struct {
	mock.Mock
}

// Ensure MockPostRepositoryForVotes implements PostRepository
var _ repository.PostRepository = (*MockPostRepositoryForVotes)(nil)

// WithTransaction mocks the WithTransaction method
func (m *MockPostRepositoryForVotes) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Get(0).(error)
	}
	// Execute the function if no error is expected
	if fn != nil {
		return fn(ctx)
	}
	return args.Error(0)
}

// IncrementScore mocks the IncrementScore method
func (m *MockPostRepositoryForVotes) IncrementScore(ctx context.Context, postID uuid.UUID, delta int) error {
	args := m.Called(ctx, postID, delta)
	return args.Error(0)
}

// Stub methods - these are required by PostRepository interface but not used by VoteService
func (m *MockPostRepositoryForVotes) Create(ctx context.Context, post *models.Post) error {
	args := m.Called(ctx, post)
	return args.Error(0)
}

func (m *MockPostRepositoryForVotes) FindByID(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockPostRepositoryForVotes) FindByURLKey(ctx context.Context, urlKey string) (*models.Post, error) {
	args := m.Called(ctx, urlKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockPostRepositoryForVotes) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Post, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Post), args.Error(1)
}

func (m *MockPostRepositoryForVotes) Find(ctx context.Context, filter repository.PostFilter, limit, offset int) ([]*models.Post, error) {
	args := m.Called(ctx, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Post), args.Error(1)
}

func (m *MockPostRepositoryForVotes) Count(ctx context.Context, filter repository.PostFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPostRepositoryForVotes) Update(ctx context.Context, post *models.Post) error {
	args := m.Called(ctx, post)
	return args.Error(0)
}

func (m *MockPostRepositoryForVotes) UpdateOwnerProfile(ctx context.Context, ownerID uuid.UUID, displayName, avatar string) error {
	args := m.Called(ctx, ownerID, displayName, avatar)
	return args.Error(0)
}

func (m *MockPostRepositoryForVotes) SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error {
	args := m.Called(ctx, postID, disabled, ownerID)
	return args.Error(0)
}

func (m *MockPostRepositoryForVotes) SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error {
	args := m.Called(ctx, postID, disabled, ownerID)
	return args.Error(0)
}

func (m *MockPostRepositoryForVotes) IncrementViewCount(ctx context.Context, postID uuid.UUID) error {
	args := m.Called(ctx, postID)
	return args.Error(0)
}

func (m *MockPostRepositoryForVotes) IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int) error {
	args := m.Called(ctx, postID, delta)
	return args.Error(0)
}

func (m *MockPostRepositoryForVotes) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

