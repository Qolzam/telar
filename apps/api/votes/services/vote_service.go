// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/posts/repository"
	"github.com/qolzam/telar/apps/api/votes/models"
	voteRepository "github.com/qolzam/telar/apps/api/votes/repository"
)

// VoteService defines the interface for vote operations
type VoteService interface {
	// Vote creates, updates, or deletes a vote on a post
	// This is the main voting operation that handles all vote transitions atomically
	Vote(ctx context.Context, postID, userID uuid.UUID, voteType int) error
}

// voteService implements the VoteService interface
type voteService struct {
	voteRepo voteRepository.VoteRepository
	postRepo repository.PostRepository
}

// NewVoteService creates a new instance of the vote service
func NewVoteService(voteRepo voteRepository.VoteRepository, postRepo repository.PostRepository) VoteService {
	return &voteService{
		voteRepo: voteRepo,
		postRepo: postRepo,
	}
}

// Vote creates, updates, or deletes a vote on a post
// This method handles all vote transitions atomically:
// - New Vote: Creates vote and increments score
// - Toggle Off: Deletes vote and decrements score
// - Switch Vote: Updates vote type and adjusts score delta
func (s *voteService) Vote(ctx context.Context, postID, userID uuid.UUID, voteType int) error {
	// Validate vote type
	if !models.IsValidVoteType(voteType) {
		return fmt.Errorf("invalid vote type: %d (must be 1=Up or 2=Down)", voteType)
	}

	// Use PostRepository's WithTransaction to ensure atomicity
	// This ensures that both the vote table and posts.score are updated atomically
	return s.postRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		// 1. Check existing vote
		existing, err := s.voteRepo.FindByUserAndPost(txCtx, userID, postID)
		if err != nil {
			// Check if error is wrapped sql.ErrNoRows (vote not found)
			if errors.Is(err, sql.ErrNoRows) || err.Error() == "vote not found: sql: no rows in result set" || err.Error() == "vote not found" {
				// Vote not found, this is a new vote
				existing = nil
			} else {
				// Some other error occurred
				return fmt.Errorf("failed to find existing vote: %w", err)
			}
		}

		delta := 0

		if existing == nil {
			// New Vote: Create vote and increment score
			voteID, err := uuid.NewV4()
			if err != nil {
				return fmt.Errorf("failed to generate vote ID: %w", err)
			}

			newVote := &models.Vote{
				ID:          voteID,
				PostID:      postID,
				OwnerUserID: userID,
				VoteTypeID:  voteType,
			}

			created, _, err := s.voteRepo.Upsert(txCtx, newVote)
			if err != nil {
				return fmt.Errorf("failed to create vote: %w", err)
			}
			if !created {
				return fmt.Errorf("expected vote to be created but it was updated")
			}

			delta = models.GetScoreValue(voteType)
		} else if existing.VoteTypeID == voteType {
			// Toggle Off: Delete vote and reverse the score
			deleted, previousType, err := s.voteRepo.Delete(txCtx, postID, userID)
			if err != nil {
				return fmt.Errorf("failed to delete vote: %w", err)
			}
			if !deleted {
				return fmt.Errorf("expected vote to be deleted but it was not found")
			}
			if previousType != voteType {
				return fmt.Errorf("previous vote type mismatch: expected %d, got %d", voteType, previousType)
			}

			delta = -models.GetScoreValue(voteType) // Reverse the score
		} else {
			// Switch Vote: Update vote type and calculate delta
			// Create a copy to avoid mutating the original object (important for tests)
			voteToUpdate := *existing
			voteToUpdate.VoteTypeID = voteType
			created, previousType, err := s.voteRepo.Upsert(txCtx, &voteToUpdate)
			if err != nil {
				return fmt.Errorf("failed to update vote: %w", err)
			}
			if created {
				return fmt.Errorf("expected vote to be updated but it was created")
			}

			// Calculate delta: new value - old value
			// e.g., Up(+1) to Down(-1) = -1 - 1 = -2
			delta = models.GetScoreValue(voteType) - models.GetScoreValue(previousType)
		}

		// 2. Atomic Score Update on Post
		if delta != 0 {
			if err := s.postRepo.IncrementScore(txCtx, postID, delta); err != nil {
				return fmt.Errorf("failed to increment post score: %w", err)
			}
		}

		return nil
	})
}

