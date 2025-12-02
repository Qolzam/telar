// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/qolzam/telar/apps/api/votes/models"
)

func TestVoteService_Vote(t *testing.T) {
	ctx := context.Background()
	postID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())

	t.Run("New Vote - Up", func(t *testing.T) {
		mockVoteRepo := new(MockVoteRepository)
		mockPostRepo := new(MockPostRepositoryForVotes)

		service := NewVoteService(mockVoteRepo, mockPostRepo)

		// Setup mocks
		// Service uses txCtx (transaction context), so we must use mock.Anything for context
		mockVoteRepo.On("FindByUserAndPost", mock.Anything, userID, postID).Return(nil, errors.New("vote not found: sql: no rows in result set"))
		mockVoteRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(vote *models.Vote) bool {
			return vote.PostID == postID && vote.OwnerUserID == userID && vote.VoteTypeID == models.VoteTypeUp
		})).Return(true, 0, nil) // created=true, no previous vote
		mockPostRepo.On("IncrementScore", mock.Anything, postID, 1).Return(nil) // delta = +1 for Up vote
		mockPostRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			// The service creates txCtx inside WithTransaction and uses it for repository calls
			// We need to pass a context that will be used as txCtx - use ctx for simplicity
			// The repository mocks use mock.Anything for context, so they'll match
			txCtx := ctx // Use ctx as txCtx for the transaction
			fn(txCtx)
		})

		// Execute
		err := service.Vote(ctx, postID, userID, models.VoteTypeUp)

		// Assert
		assert.NoError(t, err)
		mockVoteRepo.AssertExpectations(t)
		mockPostRepo.AssertExpectations(t)
	})

	t.Run("New Vote - Down", func(t *testing.T) {
		mockVoteRepo := new(MockVoteRepository)
		mockPostRepo := new(MockPostRepositoryForVotes)

		service := NewVoteService(mockVoteRepo, mockPostRepo)

		// Setup mocks
		// Service uses txCtx (transaction context), so we must use mock.Anything for context
		mockVoteRepo.On("FindByUserAndPost", mock.Anything, userID, postID).Return(nil, errors.New("vote not found: sql: no rows in result set"))
		mockVoteRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(vote *models.Vote) bool {
			return vote.PostID == postID && vote.OwnerUserID == userID && vote.VoteTypeID == models.VoteTypeDown
		})).Return(true, 0, nil) // created=true, no previous vote
		mockPostRepo.On("IncrementScore", mock.Anything, postID, -1).Return(nil) // delta = -1 for Down vote
		mockPostRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			// The service creates txCtx inside WithTransaction and uses it for repository calls
			// We need to pass a context that will be used as txCtx - use ctx for simplicity
			// The repository mocks use mock.Anything for context, so they'll match
			txCtx := ctx // Use ctx as txCtx for the transaction
			fn(txCtx)
		})

		// Execute
		err := service.Vote(ctx, postID, userID, models.VoteTypeDown)

		// Assert
		assert.NoError(t, err)
		mockVoteRepo.AssertExpectations(t)
		mockPostRepo.AssertExpectations(t)
	})

	t.Run("Toggle Off - Up vote", func(t *testing.T) {
		mockVoteRepo := new(MockVoteRepository)
		mockPostRepo := new(MockPostRepositoryForVotes)

		service := NewVoteService(mockVoteRepo, mockPostRepo)

		// Existing Up vote
		existingVote := &models.Vote{
			ID:          uuid.Must(uuid.NewV4()),
			PostID:      postID,
			OwnerUserID: userID,
			VoteTypeID:  models.VoteTypeUp,
			CreatedAt:   time.Now(),
		}

		// Setup mocks
		// Service uses txCtx (transaction context), so we must use mock.Anything for context
		mockVoteRepo.On("FindByUserAndPost", mock.Anything, userID, postID).Return(existingVote, nil)
		mockVoteRepo.On("Delete", mock.Anything, postID, userID).Return(true, models.VoteTypeUp, nil) // deleted=true, previousType=Up
		mockPostRepo.On("IncrementScore", mock.Anything, postID, -1).Return(nil) // delta = -1 (reverse the Up vote)
		mockPostRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			// The service creates txCtx inside WithTransaction and uses it for repository calls
			// We need to pass a context that will be used as txCtx - use ctx for simplicity
			// The repository mocks use mock.Anything for context, so they'll match
			txCtx := ctx // Use ctx as txCtx for the transaction
			fn(txCtx)
		})

		// Execute - voting Up again toggles it off
		err := service.Vote(ctx, postID, userID, models.VoteTypeUp)

		// Assert
		assert.NoError(t, err)
		mockVoteRepo.AssertExpectations(t)
		mockPostRepo.AssertExpectations(t)
	})

	t.Run("Switch Vote - Up to Down (delta = -2)", func(t *testing.T) {
		mockVoteRepo := new(MockVoteRepository)
		mockPostRepo := new(MockPostRepositoryForVotes)

		service := NewVoteService(mockVoteRepo, mockPostRepo)

		// Existing Up vote - create a fresh object to avoid pointer mutation
		// The service modifies existing.VoteTypeID on line 109, so we must return a fresh copy
		existingVoteID := uuid.Must(uuid.NewV4())
		mockVoteRepo.On("FindByUserAndPost", mock.Anything, userID, postID).Return(&models.Vote{
			ID:          existingVoteID,
			PostID:      postID,
			OwnerUserID: userID,
			VoteTypeID:  models.VoteTypeUp, // Existing vote is Up - MUST be 1
			CreatedAt:   time.Now(),
		}, nil)
		
		// IMPORTANT: Service modifies existing.VoteTypeID to Down before calling Upsert (line 109)
		// So Upsert will be called with VoteTypeID = Down
		mockVoteRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(vote *models.Vote) bool {
			// Service modifies existing.VoteTypeID to voteType (Down) before calling Upsert
			return vote.PostID == postID && vote.OwnerUserID == userID && vote.VoteTypeID == models.VoteTypeDown
		})).Return(false, models.VoteTypeUp, nil) // created=false (updated), previousType=Up
		
		// Delta calculation: Down(-1) - Up(+1) = -2
		mockPostRepo.On("IncrementScore", mock.Anything, postID, -2).Return(nil)
		mockPostRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			// The service creates txCtx inside WithTransaction and uses it for repository calls
			// We need to pass a context that will be used as txCtx - use ctx for simplicity
			// The repository mocks use mock.Anything for context, so they'll match
			txCtx := ctx // Use ctx as txCtx for the transaction
			fn(txCtx)
		})

		// Execute - switching from Up to Down
		err := service.Vote(ctx, postID, userID, models.VoteTypeDown)

		// Assert
		assert.NoError(t, err)
		mockVoteRepo.AssertExpectations(t)
		mockPostRepo.AssertExpectations(t)
	})

	t.Run("Switch Vote - Down to Up (delta = +2)", func(t *testing.T) {
		mockVoteRepo := new(MockVoteRepository)
		mockPostRepo := new(MockPostRepositoryForVotes)

		service := NewVoteService(mockVoteRepo, mockPostRepo)

		// Existing Down vote
		existingVote := &models.Vote{
			ID:          uuid.Must(uuid.NewV4()),
			PostID:      postID,
			OwnerUserID: userID,
			VoteTypeID:  models.VoteTypeDown,
			CreatedAt:   time.Now(),
		}

		// Setup mocks
		// Service uses txCtx (transaction context), so we must use mock.Anything for context
		mockVoteRepo.On("FindByUserAndPost", mock.Anything, userID, postID).Return(existingVote, nil)
		mockVoteRepo.On("Upsert", mock.Anything, mock.MatchedBy(func(vote *models.Vote) bool {
			return vote.PostID == postID && vote.OwnerUserID == userID && vote.VoteTypeID == models.VoteTypeUp
		})).Return(false, models.VoteTypeDown, nil) // created=false (updated), previousType=Down
		// Delta calculation: Up(+1) - Down(-1) = +2
		mockPostRepo.On("IncrementScore", mock.Anything, postID, 2).Return(nil)
		mockPostRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			// The service creates txCtx inside WithTransaction and uses it for repository calls
			// We need to pass a context that will be used as txCtx - use ctx for simplicity
			// The repository mocks use mock.Anything for context, so they'll match
			txCtx := ctx // Use ctx as txCtx for the transaction
			fn(txCtx)
		})

		// Execute - switching from Down to Up
		err := service.Vote(ctx, postID, userID, models.VoteTypeUp)

		// Assert
		assert.NoError(t, err)
		mockVoteRepo.AssertExpectations(t)
		mockPostRepo.AssertExpectations(t)
	})

	t.Run("Invalid vote type", func(t *testing.T) {
		mockVoteRepo := new(MockVoteRepository)
		mockPostRepo := new(MockPostRepositoryForVotes)

		service := NewVoteService(mockVoteRepo, mockPostRepo)

		// Execute with invalid vote type
		err := service.Vote(ctx, postID, userID, 99)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid vote type")
		mockVoteRepo.AssertNotCalled(t, "FindByUserAndPost")
		mockPostRepo.AssertNotCalled(t, "WithTransaction")
	})

	t.Run("Vote not found error handling", func(t *testing.T) {
		mockVoteRepo := new(MockVoteRepository)
		mockPostRepo := new(MockPostRepositoryForVotes)

		service := NewVoteService(mockVoteRepo, mockPostRepo)

		// Setup mocks - FindByUserAndPost returns sql.ErrNoRows wrapped
		// Service uses txCtx (transaction context), so we must use mock.Anything for context
		mockVoteRepo.On("FindByUserAndPost", mock.Anything, userID, postID).Return(nil, sql.ErrNoRows)
		mockVoteRepo.On("Upsert", mock.Anything, mock.Anything).Return(true, 0, nil)
		mockPostRepo.On("IncrementScore", mock.Anything, postID, 1).Return(nil)
		mockPostRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			// The service creates txCtx inside WithTransaction and uses it for repository calls
			// We need to pass a context that will be used as txCtx - use ctx for simplicity
			// The repository mocks use mock.Anything for context, so they'll match
			txCtx := ctx // Use ctx as txCtx for the transaction
			fn(txCtx)
		})

		// Execute
		err := service.Vote(ctx, postID, userID, models.VoteTypeUp)

		// Assert
		assert.NoError(t, err)
		mockVoteRepo.AssertExpectations(t)
		mockPostRepo.AssertExpectations(t)
	})
}

