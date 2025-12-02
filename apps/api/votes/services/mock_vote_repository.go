// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/qolzam/telar/apps/api/votes/models"
	voteRepository "github.com/qolzam/telar/apps/api/votes/repository"
)

// MockVoteRepository is a mock implementation of VoteRepository for testing
type MockVoteRepository struct {
	mock.Mock
}

// Ensure MockVoteRepository implements VoteRepository
var _ voteRepository.VoteRepository = (*MockVoteRepository)(nil)

// Upsert mocks the Upsert method
func (m *MockVoteRepository) Upsert(ctx context.Context, vote *models.Vote) (bool, int, error) {
	args := m.Called(ctx, vote)
	return args.Bool(0), args.Int(1), args.Error(2)
}

// Delete mocks the Delete method
func (m *MockVoteRepository) Delete(ctx context.Context, postID, userID uuid.UUID) (bool, int, error) {
	args := m.Called(ctx, postID, userID)
	return args.Bool(0), args.Int(1), args.Error(2)
}

// FindByUserAndPost mocks the FindByUserAndPost method
func (m *MockVoteRepository) FindByUserAndPost(ctx context.Context, userID, postID uuid.UUID) (*models.Vote, error) {
	args := m.Called(ctx, userID, postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Vote), args.Error(1)
}

// GetVotesForPosts mocks the GetVotesForPosts method
func (m *MockVoteRepository) GetVotesForPosts(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]int, error) {
	args := m.Called(ctx, postIDs, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]int), args.Error(1)
}

