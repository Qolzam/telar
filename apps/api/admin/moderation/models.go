package moderation

import (
	uuid "github.com/gofrs/uuid"
)

// FlaggedPost represents a post waiting in the queue
type FlaggedPost struct {
	ID                uuid.UUID   `json:"id"`
	Content           string      `json:"content"`
	AuthorID          uuid.UUID   `json:"authorId"`
	AuthorName        string      `json:"authorName"` // From owner_display_name field
	ModerationDetails interface{} `json:"moderationDetails"` // Raw JSONB from AI
	CreatedAt         int64       `json:"createdAt"`
}

type ModerationList struct {
	Items []FlaggedPost `json:"items"`
	Count int64         `json:"count"`
}





