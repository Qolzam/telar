package models

import (
	"time"

	uuid "github.com/gofrs/uuid"
)

// Comment represents the complete comment entity in the database
type Comment struct {
	ObjectId         uuid.UUID `json:"objectId" bson:"objectId" db:"objectId"`
	Score            int64     `json:"score" bson:"score" db:"score"`
	OwnerUserId      uuid.UUID `json:"ownerUserId" bson:"ownerUserId" db:"ownerUserId"`
	OwnerDisplayName string    `json:"ownerDisplayName" bson:"ownerDisplayName" db:"ownerDisplayName"`
	OwnerAvatar      string    `json:"ownerAvatar" bson:"ownerAvatar" db:"ownerAvatar"`
	PostId           uuid.UUID `json:"postId" bson:"postId" db:"postId"`
	ParentCommentId  *uuid.UUID `json:"parentCommentId,omitempty" bson:"parentCommentId,omitempty" db:"parentCommentId"`
	Text             string    `json:"text" bson:"text" db:"text"`
	Deleted          bool      `json:"deleted" bson:"deleted" db:"deleted"`
	DeletedDate      int64     `json:"deletedDate" bson:"deletedDate" db:"deletedDate"`
	CreatedDate      int64     `json:"createdDate" bson:"createdDate" db:"createdDate"`
	LastUpdated      int64     `json:"lastUpdated" bson:"lastUpdated" db:"lastUpdated"`
}

// CreateCommentRequest represents the request payload for creating a comment
type CreateCommentRequest struct {
	PostId uuid.UUID `json:"postId" validate:"required"`
	Text   string    `json:"text" validate:"required,min=1,max=1000"`
	ParentCommentId *uuid.UUID `json:"parentCommentId,omitempty"`
}

// UpdateCommentRequest represents the request payload for updating a comment
type UpdateCommentRequest struct {
	ObjectId uuid.UUID `json:"objectId" validate:"required"`
	Text     string    `json:"text" validate:"required,min=1,max=1000"`
}

// UpdateCommentProfileRequest represents the request payload for updating comment profile info
type UpdateCommentProfileRequest struct {
	OwnerUserId      uuid.UUID `json:"ownerUserId" validate:"required"`
	OwnerDisplayName *string   `json:"ownerDisplayName,omitempty"`
	OwnerAvatar      *string   `json:"ownerAvatar,omitempty"`
}

// CommentQueryFilter represents query filters for comments
type CommentQueryFilter struct {
	PostId          *uuid.UUID `json:"postId,omitempty"`
	OwnerUserId     *uuid.UUID `json:"ownerUserId,omitempty"`
	ParentCommentId *uuid.UUID `json:"parentCommentId,omitempty"`
	RootOnly        bool       `json:"rootOnly,omitempty"`
	IncludeDeleted  bool       `json:"includeDeleted,omitempty"`
	Deleted         *bool      `json:"deleted,omitempty"`
	CreatedAfter    *time.Time `json:"createdAfter,omitempty"`
	CreatedBefore   *time.Time `json:"createdBefore,omitempty"`
	Limit          int        `json:"limit" validate:"min=1,max=100"`
	Page           int        `json:"page,omitempty" validate:"min=1"`
	SortField      string     `json:"sortField,omitempty"`
	SortDirection  string     `json:"sortDirection,omitempty"`
	SortBy         string     `json:"sortBy,omitempty"`
	SortOrder      string     `json:"sortOrder,omitempty"`
}

// CommentResponse represents the response format for comment data
type CommentResponse struct {
	ObjectId         string `json:"objectId"`
	Score            int64  `json:"score"`
	OwnerUserId      string `json:"ownerUserId"`
	OwnerDisplayName string `json:"ownerDisplayName"`
	OwnerAvatar      string `json:"ownerAvatar"`
	PostId           string `json:"postId"`
	ParentCommentId  *string `json:"parentCommentId,omitempty"`
	ReplyCount       int    `json:"replyCount,omitempty"`
	Text             string `json:"text"`
	Deleted          bool   `json:"deleted"`
	DeletedDate      int64  `json:"deletedDate,omitempty"`
	CreatedDate      int64  `json:"createdDate"`
	LastUpdated      int64  `json:"lastUpdated,omitempty"`
}

// CommentsListResponse represents the response for listing comments
type CommentsListResponse struct {
	Comments []CommentResponse `json:"comments"`
	Count    int               `json:"count"`
	Page     int               `json:"page,omitempty"`
	Limit    int               `json:"limit,omitempty"`
	HasMore  bool              `json:"hasMore,omitempty"`
}
