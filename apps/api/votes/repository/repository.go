// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/votes/models"
)

// VoteRepository defines the interface for vote-specific database operations
// This is a domain-specific repository that knows exactly what a "Vote" is
// and how to execute optimized SQL queries for that specific domain.
type VoteRepository interface {
	// Upsert inserts a new vote or updates an existing vote
	// If vote exists, update type. If not, insert.
	// Returns: (created bool, previousVoteType int, err error)
	// We need previousType to calculate the Score delta for the Post.
	// created=true means a new vote was inserted, created=false means an existing vote was updated
	// previousVoteType is the vote type before the operation (0 if no previous vote existed)
	Upsert(ctx context.Context, vote *models.Vote) (bool, int, error)

	// Delete removes a vote (toggle off)
	// Returns: (deleted bool, previousVoteType int, err error)
	// deleted=true means a vote was found and removed, deleted=false means no vote existed
	// previousVoteType is the vote type that was removed (0 if no vote existed)
	Delete(ctx context.Context, postID, userID uuid.UUID) (bool, int, error)

	// FindByUserAndPost retrieves a user's vote on a specific post
	// Returns the vote if found, or ErrNotFound if no vote exists
	FindByUserAndPost(ctx context.Context, userID, postID uuid.UUID) (*models.Vote, error)

	// GetVotesForPosts bulk retrieves user's votes for multiple posts
	// Returns a map of postID -> voteTypeID (0 if no vote exists)
	// This avoids N+1 queries when enriching post lists with vote status
	GetVotesForPosts(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]int, error)
}

