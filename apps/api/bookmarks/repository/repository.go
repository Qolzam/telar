package repository

import (
	"context"
	"time"

	uuid "github.com/gofrs/uuid"
)

// Repository defines data access for bookmarks.
type Repository interface {
	// AddBookmark stores a bookmark; returns true when a new row was inserted.
	AddBookmark(ctx context.Context, userID, postID uuid.UUID) (bool, error)

	// RemoveBookmark deletes a bookmark; returns true when a row was deleted.
	RemoveBookmark(ctx context.Context, userID, postID uuid.UUID) (bool, error)

	// GetMapByUserAndPosts returns a presence map for the provided post IDs.
	GetMapByUserAndPosts(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)

	// FindMyBookmarks returns ordered bookmark entries with cursor pagination.
	FindMyBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]BookmarkEntry, string, error)
}

// BookmarkEntry is a lightweight projection of a bookmark row.
type BookmarkEntry struct {
	PostID    uuid.UUID `db:"post_id"`
	CreatedAt time.Time `db:"created_at"`
}
