// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/votes/models"
)

// postgresVoteRepository implements VoteRepository using raw SQL queries
type postgresVoteRepository struct {
	client *postgres.Client
	schema string // Schema name for search_path isolation
}

// NewPostgresVoteRepository creates a new PostgreSQL repository for votes
func NewPostgresVoteRepository(client *postgres.Client) VoteRepository {
	return &postgresVoteRepository{
		client: client,
		schema: "", // Default to empty (uses default schema)
	}
}

// NewPostgresVoteRepositoryWithSchema creates a new PostgreSQL repository with explicit schema
func NewPostgresVoteRepositoryWithSchema(client *postgres.Client, schema string) VoteRepository {
	return &postgresVoteRepository{
		client: client,
		schema: schema,
	}
}

// getExecutor returns either the transaction from context or the DB connection
func (r *postgresVoteRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	// Check for transaction in context (shared key for cross-package transactions)
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.client.DB()
}

// FindByUserAndPost retrieves a user's vote on a specific post
func (r *postgresVoteRepository) FindByUserAndPost(ctx context.Context, userID, postID uuid.UUID) (*models.Vote, error) {
	query := `
		SELECT id, post_id, owner_user_id, vote_type_id, created_at
		FROM votes
		WHERE post_id = $1 AND owner_user_id = $2
	`

	var vote models.Vote
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &vote, query, postID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find vote: %w", err)
	}

	return &vote, nil
}

// GetVotesForPosts bulk retrieves user's votes for multiple posts
// Returns a map of postID -> voteTypeID (0 if no vote exists)
// This avoids N+1 queries when enriching post lists with vote status
func (r *postgresVoteRepository) GetVotesForPosts(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]int, error) {
	if len(postIDs) == 0 {
		return make(map[uuid.UUID]int), nil
	}

	// Convert []uuid.UUID to pq.Array for PostgreSQL ANY operator
	postIDsArray := make([]string, len(postIDs))
	for i, id := range postIDs {
		postIDsArray[i] = id.String()
	}

	query := `
		SELECT post_id, vote_type_id
		FROM votes
		WHERE owner_user_id = $1 AND post_id = ANY($2::uuid[])
	`

	type voteResult struct {
		PostID     uuid.UUID `db:"post_id"`
		VoteTypeID int       `db:"vote_type_id"`
	}

	var results []voteResult
	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &results, query, userID, pq.Array(postIDsArray))
	if err != nil {
		return nil, fmt.Errorf("failed to get votes for posts: %w", err)
	}

	// Build map for O(1) lookup
	voteMap := make(map[uuid.UUID]int, len(results))
	for _, result := range results {
		voteMap[result.PostID] = result.VoteTypeID
	}

	// Ensure all requested IDs are in the map (with 0 if no vote)
	for _, postID := range postIDs {
		if _, exists := voteMap[postID]; !exists {
			voteMap[postID] = 0
		}
	}

	return voteMap, nil
}

// Upsert inserts a new vote or updates an existing vote
// Returns: (created bool, previousVoteType int, err error)
// created=true means a new vote was inserted, created=false means an existing vote was updated
// previousVoteType is the vote type before the operation (0 if no previous vote existed)
func (r *postgresVoteRepository) Upsert(ctx context.Context, vote *models.Vote) (bool, int, error) {
	// Set timestamps if not set
	if vote.CreatedAt.IsZero() {
		vote.CreatedAt = time.Now()
	}

	// First, try to find existing vote
	existing, err := r.FindByUserAndPost(ctx, vote.OwnerUserID, vote.PostID)
	previousVoteType := 0

	if err != nil {
		// Check if the error is a wrapped sql.ErrNoRows (from FindByUserAndPost)
		// FindByUserAndPost wraps sql.ErrNoRows, so we check both the wrapped error and error message
		if errors.Is(err, sql.ErrNoRows) || (err.Error() != "" && (err.Error() == "vote not found: sql: no rows in result set" || err.Error() == "vote not found")) {
			// Insert new vote
			query := `
				INSERT INTO votes (id, post_id, owner_user_id, vote_type_id, created_at)
				VALUES (:id, :post_id, :owner_user_id, :vote_type_id, :created_at)
			`

			insertData := struct {
				ID          uuid.UUID `db:"id"`
				PostID      uuid.UUID `db:"post_id"`
				OwnerUserID uuid.UUID `db:"owner_user_id"`
				VoteTypeID  int       `db:"vote_type_id"`
				CreatedAt   time.Time `db:"created_at"`
			}{
				ID:          vote.ID,
				PostID:      vote.PostID,
				OwnerUserID: vote.OwnerUserID,
				VoteTypeID:  vote.VoteTypeID,
				CreatedAt:   vote.CreatedAt,
			}

			_, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, insertData)
			if err != nil {
				return false, 0, fmt.Errorf("failed to insert vote: %w", err)
			}

			return true, 0, nil // created=true, no previous vote
		}
		// Some other error occurred
		return false, 0, fmt.Errorf("failed to check existing vote: %w", err)
	}

	// Vote exists, update it
	previousVoteType = existing.VoteTypeID

	query := `
		UPDATE votes
		SET vote_type_id = :vote_type_id
		WHERE post_id = :post_id AND owner_user_id = :owner_user_id
	`

	updateData := struct {
		PostID     uuid.UUID `db:"post_id"`
		OwnerUserID uuid.UUID `db:"owner_user_id"`
		VoteTypeID int       `db:"vote_type_id"`
	}{
		PostID:      vote.PostID,
		OwnerUserID: vote.OwnerUserID,
		VoteTypeID:  vote.VoteTypeID,
	}

	result, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, updateData)
	if err != nil {
		return false, previousVoteType, fmt.Errorf("failed to update vote: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, previousVoteType, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return false, previousVoteType, fmt.Errorf("vote not found for update")
	}

	return false, previousVoteType, nil // created=false, previous vote existed
}

// Delete removes a vote (toggle off)
// Returns: (deleted bool, previousVoteType int, err error)
// deleted=true means a vote was found and removed, deleted=false means no vote existed
// previousVoteType is the vote type that was removed (0 if no vote existed)
func (r *postgresVoteRepository) Delete(ctx context.Context, postID, userID uuid.UUID) (bool, int, error) {
	// First, find the existing vote to get the previous vote type
	existing, err := r.FindByUserAndPost(ctx, userID, postID)
	if err != nil {
		// Check if the error is a wrapped sql.ErrNoRows (from FindByUserAndPost)
		if errors.Is(err, sql.ErrNoRows) || (err.Error() != "" && (err.Error() == "vote not found: sql: no rows in result set" || err.Error() == "vote not found")) {
			// Vote not found
			return false, 0, nil // deleted=false, no previous vote
		}
		// Some other error occurred
		return false, 0, fmt.Errorf("failed to find vote: %w", err)
	}

	previousVoteType := existing.VoteTypeID

	// Delete the vote
	query := `
		DELETE FROM votes
		WHERE post_id = $1 AND owner_user_id = $2
	`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, postID, userID)
	if err != nil {
		return false, previousVoteType, fmt.Errorf("failed to delete vote: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, previousVoteType, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return false, previousVoteType, fmt.Errorf("vote not found for deletion")
	}

	return true, previousVoteType, nil // deleted=true, previous vote existed
}

