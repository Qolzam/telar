package services

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/models"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// CommentService defines the interface for comment operations
type CommentService interface {
	// Create operations
	CreateComment(ctx context.Context, req *models.CreateCommentRequest, user *types.UserContext) (*models.Comment, error)
	CreateIndex(ctx context.Context, indexes map[string]interface{}) error

	// Read operations
	GetComment(ctx context.Context, commentID uuid.UUID) (*models.Comment, error)
	GetCommentsByPost(ctx context.Context, postID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error)
	GetCommentsByUser(ctx context.Context, userID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error)
	QueryComments(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error)
	GetReplyCount(ctx context.Context, parentID uuid.UUID) (int64, error)
	
	// Cursor-based pagination operations (new optimized methods)
	QueryCommentsWithCursor(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error)

	// Update operations
	UpdateComment(ctx context.Context, commentID uuid.UUID, req *models.UpdateCommentRequest, user *types.UserContext) error
	UpdateCommentProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error
	IncrementScore(ctx context.Context, commentID uuid.UUID, delta int, user *types.UserContext) error

	// Delete operations
	DeleteComment(ctx context.Context, commentID uuid.UUID, postID uuid.UUID, user *types.UserContext) error
	DeleteCommentsByPost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error
	SoftDeleteComment(ctx context.Context, commentID uuid.UUID, user *types.UserContext) error
	DeleteByOwner(ctx context.Context, owner uuid.UUID, objectId uuid.UUID) error

	// Utility operations
	ValidateCommentOwnership(ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error

	// GetRootCommentCount counts root comments (non-reply comments) for a post
	GetRootCommentCount(ctx context.Context, postID uuid.UUID) (int64, error)
}

