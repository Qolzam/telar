package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/lib/pq"
)

// Post represents the complete post entity in the database
// Updated for relational schema: columns are explicit, JSONB only for dynamic data
type Post struct {
	// Primary key - maps to 'id' in the new schema
	ObjectId uuid.UUID `json:"objectId" bson:"objectId" db:"id"`
	
	// Indexed columns (The "Spine")
	OwnerUserId uuid.UUID `json:"ownerUserId" bson:"ownerUserId" db:"owner_user_id"`
	PostTypeId  int       `json:"postTypeId" bson:"postTypeId" db:"post_type_id"`
	
	// Data columns
	Body             string        `json:"body" bson:"body" db:"body"`
	Score            int64         `json:"score" bson:"score" db:"score"`
	ViewCount        int64         `json:"viewCount" bson:"viewCount" db:"view_count"`
	CommentCounter   int64         `json:"commentCounter" bson:"commentCounter" db:"comment_count"`
	Tags             pq.StringArray `json:"tags" bson:"tags" db:"tags"` // Use pq.StringArray for PostgreSQL arrays
	URLKey           string        `json:"urlKey" bson:"urlKey" db:"url_key"`
	OwnerDisplayName string        `json:"ownerDisplayName" bson:"ownerDisplayName" db:"owner_display_name"`
	OwnerAvatar      string        `json:"ownerAvatar" bson:"ownerAvatar" db:"owner_avatar"`
	Image            string        `json:"image" bson:"image" db:"image"`
	ImageFullPath    string        `json:"imageFullPath" bson:"imageFullPath" db:"image_full_path"`
	Video            string        `json:"video" bson:"video" db:"video"`
	Thumbnail        string        `json:"thumbnail" bson:"thumbnail" db:"thumbnail"`
	DisableComments  bool          `json:"disableComments" bson:"disableComments" db:"disable_comments"`
	DisableSharing   bool          `json:"disableSharing" bson:"disableSharing" db:"disable_sharing"`
	Deleted          bool          `json:"deleted" bson:"deleted" db:"is_deleted"`
	DeletedDate      int64         `json:"deletedDate" bson:"deletedDate" db:"deleted_date"`
	Permission       string        `json:"permission" bson:"permission" db:"permission"`
	Version          string        `json:"version" bson:"version" db:"version"`
	
	// Timestamps - both Unix timestamps and TIMESTAMPTZ for compatibility
	CreatedDate int64     `json:"createdDate" bson:"createdDate" db:"created_date"`
	LastUpdated int64     `json:"lastUpdated" bson:"lastUpdated" db:"last_updated"`
	CreatedAt   time.Time `json:"createdAt,omitempty" bson:"createdAt,omitempty" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty" bson:"updatedAt,omitempty" db:"updated_at"`
	
	// Unstructured data (The "Blob") - Only for truly dynamic data
	Votes          map[string]string `json:"votes" bson:"votes" db:"-"`           // Stored in metadata JSONB
	Album          *Album            `json:"album" bson:"album" db:"-"`           // Stored in metadata JSONB
	AccessUserList []string          `json:"accessUserList" bson:"accessUserList" db:"-"` // Stored in metadata JSONB
	Metadata       JSONB              `json:"metadata,omitempty" bson:"metadata,omitempty" db:"metadata"` // Custom JSONB type
}

// JSONB is a custom type for PostgreSQL JSONB that implements sql.Scanner and driver.Valuer
type JSONB map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	
	return json.Unmarshal(bytes, j)
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
	VoteType         int               `json:"voteType"` // 0=None, 1=Up, 2=Down (current user's vote)
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
