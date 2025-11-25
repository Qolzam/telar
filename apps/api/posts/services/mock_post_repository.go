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

// MockPostRepository is a mock implementation of PostRepository for testing
type MockPostRepository struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockPostRepository) Create(ctx context.Context, post *models.Post) error {
	args := m.Called(ctx, post)
	return args.Error(0)
}

// FindByID mocks the FindByID method
func (m *MockPostRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

// FindByURLKey mocks the FindByURLKey method
func (m *MockPostRepository) FindByURLKey(ctx context.Context, urlKey string) (*models.Post, error) {
	args := m.Called(ctx, urlKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

// FindByUser mocks the FindByUser method
func (m *MockPostRepository) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Post, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Post), args.Error(1)
}

// Find mocks the Find method
func (m *MockPostRepository) Find(ctx context.Context, filter repository.PostFilter, limit, offset int) ([]*models.Post, error) {
	args := m.Called(ctx, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Post), args.Error(1)
}

// Count mocks the Count method
func (m *MockPostRepository) Count(ctx context.Context, filter repository.PostFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

// Update mocks the Update method
func (m *MockPostRepository) Update(ctx context.Context, post *models.Post) error {
	args := m.Called(ctx, post)
	return args.Error(0)
}

// UpdateOwnerProfile mocks the UpdateOwnerProfile method
func (m *MockPostRepository) UpdateOwnerProfile(ctx context.Context, ownerID uuid.UUID, displayName, avatar string) error {
	args := m.Called(ctx, ownerID, displayName, avatar)
	return args.Error(0)
}

// SetCommentDisabled mocks the SetCommentDisabled method
func (m *MockPostRepository) SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error {
	args := m.Called(ctx, postID, disabled, ownerID)
	return args.Error(0)
}

// SetSharingDisabled mocks the SetSharingDisabled method
func (m *MockPostRepository) SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error {
	args := m.Called(ctx, postID, disabled, ownerID)
	return args.Error(0)
}

// IncrementViewCount mocks the IncrementViewCount method
func (m *MockPostRepository) IncrementViewCount(ctx context.Context, postID uuid.UUID) error {
	args := m.Called(ctx, postID)
	return args.Error(0)
}

// IncrementCommentCount mocks the IncrementCommentCount method
func (m *MockPostRepository) IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int) error {
	args := m.Called(ctx, postID, delta)
	return args.Error(0)
}

// IncrementScore mocks the IncrementScore method
func (m *MockPostRepository) IncrementScore(ctx context.Context, postID uuid.UUID, delta int) error {
	args := m.Called(ctx, postID, delta)
	return args.Error(0)
}

// WithTransaction mocks the WithTransaction method
func (m *MockPostRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
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

// Delete mocks the Delete method
func (m *MockPostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

