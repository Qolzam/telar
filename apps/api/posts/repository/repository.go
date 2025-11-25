// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/posts/models"
)

// PostFilter represents filtering criteria for querying posts
type PostFilter struct {
	OwnerUserID  *uuid.UUID
	PostTypeID   *int
	Tags         []string
	Deleted      *bool
	CreatedAfter *int64
	URLKey       *string
	SearchText   *string
}

// PostRepository defines the interface for post-specific database operations
// This is a domain-specific repository that knows exactly what a "Post" is
// and how to execute optimized SQL queries for that specific domain.
type PostRepository interface {
	// Create inserts a new post
	Create(ctx context.Context, post *models.Post) error

	// FindByID retrieves a post by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.Post, error)

	// FindByURLKey retrieves a post by its URL key
	FindByURLKey(ctx context.Context, urlKey string) (*models.Post, error)

	// FindByUser retrieves posts by owner user ID with pagination
	FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Post, error)

	// Find retrieves posts matching the filter criteria with pagination
	Find(ctx context.Context, filter PostFilter, limit, offset int) ([]*models.Post, error)

	// Count returns the number of posts matching the filter criteria
	Count(ctx context.Context, filter PostFilter) (int64, error)

	// Update updates an existing post
	Update(ctx context.Context, post *models.Post) error

	// UpdateOwnerProfile updates display name and avatar for all posts by an owner
	UpdateOwnerProfile(ctx context.Context, ownerID uuid.UUID, displayName, avatar string) error

	// SetCommentDisabled sets the comment disabled flag for a post with ownership validation
	SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error

	// SetSharingDisabled sets the sharing disabled flag for a post with ownership validation
	SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error

	// IncrementViewCount atomically increments the view count for a post
	IncrementViewCount(ctx context.Context, postID uuid.UUID) error

	// IncrementCommentCount atomically increments the comment count for a post
	// This is used for denormalized count updates when comments are created/deleted
	IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int) error

	// IncrementScore atomically increments the score for a post
	// This prevents race conditions when multiple users vote simultaneously
	IncrementScore(ctx context.Context, postID uuid.UUID, delta int) error

	// WithTransaction executes a function within a database transaction
	// This is critical for atomic operations (e.g., comment creation + count increment)
	WithTransaction(ctx context.Context, fn func(context.Context) error) error

	// Delete deletes a post by ID (soft delete)
	Delete(ctx context.Context, id uuid.UUID) error
}


