// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package mocks

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/qolzam/telar/apps/api/posts/models"
	postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
)

// MockPostRepository is a mock implementation of PostRepository
type MockPostRepository struct {
	mock.Mock
}

var _ postsRepository.PostRepository = (*MockPostRepository)(nil)

func (m *MockPostRepository) Create(ctx context.Context, post *models.Post) error {
	args := m.Called(ctx, post)
	return args.Error(0)
}

func (m *MockPostRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockPostRepository) FindByURLKey(ctx context.Context, urlKey string) (*models.Post, error) {
	args := m.Called(ctx, urlKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockPostRepository) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Post, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Post), args.Error(1)
}

func (m *MockPostRepository) Find(ctx context.Context, filter postsRepository.PostFilter, limit, offset int) ([]*models.Post, error) {
	args := m.Called(ctx, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Post), args.Error(1)
}

func (m *MockPostRepository) Count(ctx context.Context, filter postsRepository.PostFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPostRepository) Update(ctx context.Context, post *models.Post) error {
	args := m.Called(ctx, post)
	return args.Error(0)
}

func (m *MockPostRepository) UpdateOwnerProfile(ctx context.Context, ownerID uuid.UUID, displayName, avatar string) error {
	args := m.Called(ctx, ownerID, displayName, avatar)
	return args.Error(0)
}

func (m *MockPostRepository) SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error {
	args := m.Called(ctx, postID, disabled, ownerID)
	return args.Error(0)
}

func (m *MockPostRepository) SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error {
	args := m.Called(ctx, postID, disabled, ownerID)
	return args.Error(0)
}

func (m *MockPostRepository) IncrementViewCount(ctx context.Context, postID uuid.UUID) error {
	args := m.Called(ctx, postID)
	return args.Error(0)
}

func (m *MockPostRepository) IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int) error {
	args := m.Called(ctx, postID, delta)
	return args.Error(0)
}

func (m *MockPostRepository) IncrementScore(ctx context.Context, postID uuid.UUID, delta int) error {
	args := m.Called(ctx, postID, delta)
	return args.Error(0)
}

func (m *MockPostRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	// Execute the function within the mock
	if fn != nil {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return args.Error(0)
}

func (m *MockPostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

