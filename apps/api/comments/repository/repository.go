// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/models"
)

// CommentFilter represents filtering criteria for querying comments
type CommentFilter struct {
	PostID         *uuid.UUID
	OwnerUserID    *uuid.UUID
	ParentCommentID *uuid.UUID
	RootOnly       bool // If true, only return root comments (parent_comment_id IS NULL)
	IncludeDeleted bool // If false, filter out deleted comments
	Deleted        *bool
	CreatedAfter   *int64
	CreatedBefore  *int64
}

// CommentRepository defines the interface for comment-specific database operations
// This is a domain-specific repository that knows exactly what a "Comment" is
// and how to execute optimized SQL queries for that specific domain.
type CommentRepository interface {
	// Create inserts a new comment
	Create(ctx context.Context, comment *models.Comment) error

	// FindByID retrieves a comment by its ID
	FindByID(ctx context.Context, commentID uuid.UUID) (*models.Comment, error)

	// FindByPostID retrieves comments for a specific post with pagination
	// Returns root comments (parent_comment_id IS NULL) ordered by created_date DESC
	FindByPostID(ctx context.Context, postID uuid.UUID, limit, offset int) ([]*models.Comment, error)

	// FindByPostIDWithCursor retrieves comments for a specific post with cursor-based pagination
	// Returns root comments (parent_comment_id IS NULL) ordered by created_date DESC, id DESC
	// cursor is a base64-encoded string containing created_date and id
	// Returns comments, nextCursor (empty if no more), and error
	FindByPostIDWithCursor(ctx context.Context, postID uuid.UUID, cursor string, limit int) ([]*models.Comment, string, error)

	// FindByUserID retrieves comments created by a specific user with pagination
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Comment, error)

	// FindReplies retrieves replies to a specific comment with pagination
	FindReplies(ctx context.Context, parentID uuid.UUID, limit, offset int) ([]*models.Comment, error)

	// FindRepliesWithCursor retrieves replies to a specific comment with cursor-based pagination
	// Returns replies ordered by created_date ASC, id ASC for stable ordering
	// cursor is a base64-encoded string containing created_date and id
	// Returns replies, nextCursor (empty if no more), and error
	FindRepliesWithCursor(ctx context.Context, parentID uuid.UUID, cursor string, limit int) ([]*models.Comment, string, error)

	// CountByPostID counts root comments (not replies) for a post
	// This is used for the denormalized comment_count on posts
	CountByPostID(ctx context.Context, postID uuid.UUID) (int64, error)

	// CountReplies counts replies to a specific comment
	CountReplies(ctx context.Context, parentID uuid.UUID) (int64, error)

	// Find retrieves comments matching the filter criteria with pagination
	Find(ctx context.Context, filter CommentFilter, limit, offset int) ([]*models.Comment, error)

	// Count returns the number of comments matching the filter criteria
	Count(ctx context.Context, filter CommentFilter) (int64, error)

	// Update updates an existing comment
	Update(ctx context.Context, comment *models.Comment) error

	// UpdateOwnerProfile updates display name and avatar for all comments by an owner
	UpdateOwnerProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error

	// IncrementScore atomically increments the score for a comment
	IncrementScore(ctx context.Context, commentID uuid.UUID, delta int) error

	// Delete deletes a comment by ID (soft delete)
	Delete(ctx context.Context, commentID uuid.UUID) error

	// DeleteByPostID soft deletes all comments for a post (batch operation)
	DeleteByPostID(ctx context.Context, postID uuid.UUID) error

	// AddVote attempts to add a vote (like) for a comment
	// Returns true if a new row was inserted, false if it already existed
	AddVote(ctx context.Context, commentID, userID uuid.UUID) (bool, error)

	// RemoveVote removes a vote (like) for a comment
	// Returns true if a row was deleted, false if no vote existed
	RemoveVote(ctx context.Context, commentID, userID uuid.UUID) (bool, error)

	// GetUserVotesForComments bulk checks which comments the user has liked
	// Returns a map of CommentID -> bool (true if user liked it)
	GetUserVotesForComments(ctx context.Context, commentIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error)

	// CountRepliesBulk counts replies for multiple comments in a single query
	// Returns a map of parentCommentID -> replyCount
	// This avoids N+1 queries when loading comment lists
	CountRepliesBulk(ctx context.Context, parentIDs []uuid.UUID) (map[uuid.UUID]int64, error)

	// WithTransaction executes a function within a transaction
	// This is needed for atomic vote operations that update both comment_votes and comments.score
	WithTransaction(ctx context.Context, fn func(context.Context) error) error
}

