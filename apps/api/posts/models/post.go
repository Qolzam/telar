package models

import (
	"encoding/json"
	"errors"
	"time"

	uuid "github.com/gofrs/uuid"
)

// Post represents the complete post entity in the database
type Post struct {
	ObjectId         uuid.UUID                `json:"objectId" bson:"objectId" db:"object_id"`
	PostTypeId       int                      `json:"postTypeId" bson:"postTypeId" db:"post_type_id"`
	Score            int64                    `json:"score" bson:"score" db:"score"`
	Votes            map[string]string        `json:"votes" bson:"votes" db:"votes"`
	ViewCount        int64                    `json:"viewCount" bson:"viewCount" db:"view_count"`
	Body             string                   `json:"body" bson:"body" db:"body"`
	OwnerUserId      uuid.UUID                `json:"ownerUserId" bson:"ownerUserId" db:"owner_user_id"`
	OwnerDisplayName string                   `json:"ownerDisplayName" bson:"ownerDisplayName" db:"owner_display_name"`
	OwnerAvatar      string                   `json:"ownerAvatar" bson:"ownerAvatar" db:"owner_avatar"`
	URLKey           string                   `json:"urlKey" bson:"urlKey" db:"url_key"`
	Tags             []string                 `json:"tags" bson:"tags" db:"tags"`
	CommentCounter   int64                    `json:"commentCounter" bson:"commentCounter" db:"comment_counter"`
	Image            string                   `json:"image" bson:"image" db:"image"`
	ImageFullPath    string                   `json:"imageFullPath" bson:"imageFullPath" db:"image_full_path"`
	Video            string                   `json:"video" bson:"video" db:"video"`
	Thumbnail        string                   `json:"thumbnail" bson:"thumbnail" db:"thumbnail"`
	Album            *Album                   `json:"album" bson:"album" db:"album"`
	DisableComments  bool                     `json:"disableComments" bson:"disableComments" db:"disable_comments"`
	DisableSharing   bool                     `json:"disableSharing" bson:"disableSharing" db:"disable_sharing"`
	Deleted          bool                     `json:"deleted" bson:"deleted" db:"deleted"`
	DeletedDate      int64                    `json:"deletedDate" bson:"deletedDate" db:"deleted_date"`
	CreatedDate      int64                    `json:"createdDate" bson:"createdDate" db:"created_date"`
	LastUpdated      int64                    `json:"lastUpdated" bson:"lastUpdated" db:"last_updated"`
	AccessUserList   []string                 `json:"accessUserList" bson:"accessUserList" db:"access_user_list"`
	Permission       string                   `json:"permission" bson:"permission" db:"permission"`
	Version          string                   `json:"version" bson:"version" db:"version"`
}

// UnmarshalJSON handles both integer and string permissions during migration
func (p *Post) UnmarshalJSON(data []byte) error {
	// Create a temporary struct to unmarshal into
	type PostAlias Post
	temp := &struct {
		Permission interface{} `json:"permission"`
		*PostAlias
	}{
		PostAlias: (*PostAlias)(p),
	}

	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}

	// Handle permission field conversion
	if temp.Permission != nil {
		switch v := temp.Permission.(type) {
		case string:
			p.Permission = v
		case float64: // JSON numbers are unmarshaled as float64
			// Convert numeric permission to string
			switch int(v) {
			case 0:
				p.Permission = "Public"
			case 1:
				p.Permission = "OnlyMe"
			case 2:
				p.Permission = "Circles"
			default:
				p.Permission = "Public"
			}
		case int:
			// Convert numeric permission to string
			switch v {
			case 0:
				p.Permission = "Public"
			case 1:
				p.Permission = "OnlyMe"
			case 2:
				p.Permission = "Circles"
			default:
				p.Permission = "Public"
			}
		default:
			p.Permission = "Public"
		}
	}

	return nil
}

// Album represents the post album structure
type Album struct {
	Count   int      `json:"count" bson:"count" db:"count"`
	Cover   string   `json:"cover" bson:"cover" db:"cover"`
	CoverId uuid.UUID `json:"coverId" bson:"coverId" db:"cover_id"`
	Photos  []string `json:"photos" bson:"photos" db:"photos"`
	Title   string   `json:"title" bson:"title" db:"title"`
}

// CreatePostRequest represents the request payload for creating a post
type CreatePostRequest struct {
	ObjectId         *uuid.UUID `json:"objectId,omitempty"`          // Optional, will be generated if not provided
	PostTypeId       int        `json:"postTypeId" validate:"required"`
	Body             string     `json:"body" validate:"required,min=1,max=10000"`
	Image            string     `json:"image,omitempty"`
	ImageFullPath    string     `json:"imageFullPath,omitempty"`
	Video            string     `json:"video,omitempty"`
	Thumbnail        string     `json:"thumbnail,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
	Album            Album      `json:"album,omitempty"`
	DisableComments  bool       `json:"disableComments,omitempty"`
	DisableSharing   bool       `json:"disableSharing,omitempty"`
	AccessUserList   []string   `json:"accessUserList,omitempty"`
	Permission       string     `json:"permission,omitempty"`
	Version          string     `json:"version,omitempty"`
	// Legacy compatibility fields
	Score            int64      `json:"score,omitempty"`
	ViewCount        int64      `json:"viewCount,omitempty"`
	CommentCounter   int64      `json:"commentCounter,omitempty"`
	Deleted          bool       `json:"deleted,omitempty"`
	DeletedDate      int64      `json:"deletedDate,omitempty"`
	LastUpdated      int64      `json:"lastUpdated,omitempty"`
}

// UpdatePostRequest represents the request payload for updating a post
type UpdatePostRequest struct {
	ObjectId         *uuid.UUID `json:"objectId,omitempty" validate:"required"` // Post ID to update
	Body             *string   `json:"body,omitempty" validate:"omitempty,min=1,max=10000"`
	Image            *string   `json:"image,omitempty"`
	ImageFullPath    *string   `json:"imageFullPath,omitempty"`
	Video            *string   `json:"video,omitempty"`
	Thumbnail        *string   `json:"thumbnail,omitempty"`
	Tags             *[]string `json:"tags,omitempty"`
	Album            *Album    `json:"album,omitempty"`
	DisableComments  *bool     `json:"disableComments,omitempty"`
	DisableSharing   *bool     `json:"disableSharing,omitempty"`
	AccessUserList   *[]string `json:"accessUserList,omitempty"`
	Permission       *string   `json:"permission,omitempty"`
	Version          *string   `json:"version,omitempty"`
}

// PostQueryFilter represents query filters for posts
type PostQueryFilter struct {
	OwnerUserId    *uuid.UUID `json:"ownerUserId,omitempty"`
	PostTypeId     *int       `json:"postTypeId,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
	Search         string     `json:"search,omitempty"`
	Deleted        *bool      `json:"deleted,omitempty"`
	
	// Cursor-based pagination (new)
	Cursor         string     `json:"cursor,omitempty"`
	BeforeCursor   string     `json:"beforeCursor,omitempty"`
	AfterCursor    string     `json:"afterCursor,omitempty"`
	Limit          int        `json:"limit" validate:"min=1,max=100"`
	SortField      string     `json:"sortField,omitempty"`      // "createdDate", "score", "lastUpdated"
	SortDirection  string     `json:"sortDirection,omitempty"`  // "asc", "desc"
	
	// Legacy pagination (deprecated but maintained for backward compatibility)
	Page         int        `json:"page,omitempty" validate:"min=1"`
	SortBy       string     `json:"sortBy,omitempty"`
	SortOrder    string     `json:"sortOrder,omitempty"`
	CreatedAfter *time.Time `json:"createdAfter,omitempty"`
}

// PostResponse represents the API response for a post
type PostResponse struct {
	ObjectId         string            `json:"objectId"`
	PostTypeId       int               `json:"postTypeId"`
	Score            int64             `json:"score"`
	Votes            map[string]string `json:"votes"`
	ViewCount        int64             `json:"viewCount"`
	Body             string            `json:"body"`
	OwnerUserId      string            `json:"ownerUserId"`
	OwnerDisplayName string            `json:"ownerDisplayName"`
	OwnerAvatar      string            `json:"ownerAvatar"`
	Tags             []string          `json:"tags"`
	CommentCounter   int64             `json:"commentCounter"`
	Image            string            `json:"image,omitempty"`
	ImageFullPath    string            `json:"imageFullPath,omitempty"`
	Video            string            `json:"video,omitempty"`
	Thumbnail        string            `json:"thumbnail,omitempty"`
	URLKey           string            `json:"urlKey"`
	Album            *Album            `json:"album,omitempty"`
	DisableComments  bool              `json:"disableComments"`
	DisableSharing   bool              `json:"disableSharing"`
	Deleted          bool              `json:"deleted"`
	DeletedDate      int64             `json:"deletedDate,omitempty"`
	CreatedDate      int64             `json:"createdDate"`
	LastUpdated      int64             `json:"lastUpdated,omitempty"`
	Permission       string            `json:"permission"`
	Version          string            `json:"version,omitempty"`
	LatestComments   []CommentPreview  `json:"latestComments,omitempty"`
}

// CommentPreview is a lightweight view of a comment for feed previews
type CommentPreview struct {
	ObjectId         string `json:"objectId"`
	OwnerUserId      string `json:"ownerUserId"`
	OwnerDisplayName string `json:"ownerDisplayName"`
	OwnerAvatar      string `json:"ownerAvatar"`
	Text             string `json:"text"`
	CreatedDate      int64  `json:"createdDate"`
}
// PostsListResponse represents the response for listing posts
type PostsListResponse struct {
	Posts []PostResponse `json:"posts"`
	
	// Cursor-based pagination (new)
	NextCursor string `json:"nextCursor,omitempty"`
	PrevCursor string `json:"prevCursor,omitempty"`
	HasNext    bool   `json:"hasNext"`
	HasPrev    bool   `json:"hasPrev"`
	
	// Legacy pagination (deprecated but maintained for backward compatibility)
	TotalCount int64 `json:"totalCount,omitempty"`
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit"`
	HasMore    bool  `json:"hasMore,omitempty"`
}

// CursorPagination represents cursor-based pagination metadata
type CursorPagination struct {
	Cursor        string `json:"cursor"`
	NextCursor    string `json:"nextCursor,omitempty"`
	PrevCursor    string `json:"prevCursor,omitempty"`
	HasNext       bool   `json:"hasNext"`
	HasPrev       bool   `json:"hasPrev"`
	Limit         int    `json:"limit"`
	SortField     string `json:"sortField"`
	SortDirection string `json:"sortDirection"`
}

// CursorData represents the data encoded in a cursor
type CursorData struct {
	ID          string    `json:"id"`
	Value       interface{} `json:"value"` // The actual sort field value
	Timestamp   int64     `json:"timestamp"`
	SortField   string    `json:"sortField"`
	Direction   string    `json:"direction"`
}

// Validate validates cursor data
func (cd *CursorData) Validate() error {
	if cd.ID == "" {
		return errors.New("cursor ID cannot be empty")
	}
	if cd.SortField == "" {
		return errors.New("cursor sort field cannot be empty")
	}
	if cd.Direction != "asc" && cd.Direction != "desc" {
		return errors.New("cursor direction must be 'asc' or 'desc'")
	}
	return nil
}

// CreatePostResponse represents the response after creating a post
type CreatePostResponse struct {
	ObjectId string `json:"objectId"`
	Message  string `json:"message,omitempty"`
}

// CursorInfo represents cursor information for a specific post
type CursorInfo struct {
	PostId    string `json:"postId"`
	Cursor    string `json:"cursor"`
	Position  int    `json:"position"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}
