package services

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts/models"
)

// PostService defines the interface for post operations
type PostService interface {
	// Create operations
	CreatePost(ctx context.Context, req *models.CreatePostRequest, user *types.UserContext) (*models.Post, error)
	CreateIndex(ctx context.Context, indexes map[string]interface{}) error
	CreateIndexes(ctx context.Context) error

	// Read operations
	GetPost(ctx context.Context, postID uuid.UUID) (*models.Post, error)
	GetPostByURLKey(ctx context.Context, urlKey string) (*models.Post, error)
	GetPostsByUser(ctx context.Context, userID uuid.UUID, filter *models.PostQueryFilter) (*models.PostsListResponse, error)
	QueryPosts(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error)
	SearchPosts(ctx context.Context, query string, filter *models.PostQueryFilter) (*models.PostsListResponse, error)
	
	// Cursor-based pagination operations (new optimized methods)
	QueryPostsWithCursor(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error)
	SearchPostsWithCursor(ctx context.Context, query string, filter *models.PostQueryFilter) (*models.PostsListResponse, error)
	GetCursorInfo(ctx context.Context, postID uuid.UUID, sortBy, sortOrder string) (*models.CursorInfo, error)

	// Update operations
	UpdatePost(ctx context.Context, postID uuid.UUID, req *models.UpdatePostRequest, user *types.UserContext) error
	UpdatePostProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error
	IncrementScore(ctx context.Context, postID uuid.UUID, delta int, user *types.UserContext) error
	IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int, user *types.UserContext) error
	SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, user *types.UserContext) error
	SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, user *types.UserContext) error
	IncrementViewCount(ctx context.Context, postID uuid.UUID, user *types.UserContext) error

	// Delete operations
	DeletePost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error
	SoftDeletePost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error
	DeleteByOwner(ctx context.Context, owner uuid.UUID, objectId uuid.UUID) error

	// Utility operations
	GenerateURLKey(ctx context.Context, postID uuid.UUID, user *types.UserContext) (string, error)
	ValidatePostOwnership(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error
	
	// Backward compatibility methods (for existing handlers)
	SetField(ctx context.Context, objectId uuid.UUID, field string, value interface{}) error
	IncrementField(ctx context.Context, objectId uuid.UUID, field string, delta int) error
	UpdateByOwner(ctx context.Context, objectId uuid.UUID, owner uuid.UUID, fields map[string]interface{}) error
	UpdateProfileForOwner(ctx context.Context, owner uuid.UUID, displayName, avatar string) error

	// Field operations for flexible updates
	UpdateFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error
	IncrementFields(ctx context.Context, postID uuid.UUID, increments map[string]interface{}) error
	UpdateAndIncrementFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error

}
