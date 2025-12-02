// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package models

import (
	"time"

	uuid "github.com/gofrs/uuid"
)

// Vote represents a user's vote on a post
// Reference: StackExchange Data Explorer (Votes, VoteTypes)
type Vote struct {
	ID          uuid.UUID `db:"id" json:"objectId"`
	PostID      uuid.UUID `db:"post_id" json:"postId"`
	OwnerUserID uuid.UUID `db:"owner_user_id" json:"ownerUserId"`
	VoteTypeID  int       `db:"vote_type_id" json:"typeId"` // 1=UpVote, 2=DownVote
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// VoteType constants
const (
	VoteTypeUp   = 1 // UpVote (+1 score)
	VoteTypeDown = 2 // DownVote (-1 score)
)

// GetScoreValue returns the score delta for a vote type
// 1 = Up (+1), 2 = Down (-1)
func GetScoreValue(voteTypeID int) int {
	if voteTypeID == VoteTypeUp {
		return 1
	}
	if voteTypeID == VoteTypeDown {
		return -1
	}
	return 0
}

// IsValidVoteType checks if the vote type is valid
func IsValidVoteType(voteTypeID int) bool {
	return voteTypeID == VoteTypeUp || voteTypeID == VoteTypeDown
}

