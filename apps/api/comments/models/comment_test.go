package models

import (
	"encoding/json"
	"testing"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCommentModel(t *testing.T) {
	t.Run("Comment struct", func(t *testing.T) {
		commentID := uuid.Must(uuid.NewV4())
		postID := uuid.Must(uuid.NewV4())
		ownerUserID := uuid.Must(uuid.NewV4())

		comment := &Comment{
			ObjectId:    commentID,
			PostId:      postID,
			OwnerUserId: ownerUserID,
			Text:        "This is a test comment",
			Score:       10,
			CreatedDate: 1640995200, // January 1, 2022
			LastUpdated: 1640995200,
		}

		assert.Equal(t, commentID, comment.ObjectId)
		assert.Equal(t, postID, comment.PostId)
		assert.Equal(t, ownerUserID, comment.OwnerUserId)
		assert.Equal(t, "This is a test comment", comment.Text)
		assert.Equal(t, int64(10), comment.Score)
		assert.Equal(t, int64(1640995200), comment.CreatedDate)
		assert.Equal(t, int64(1640995200), comment.LastUpdated)
	})

	t.Run("CreateCommentRequest validation", func(t *testing.T) {
		postID := uuid.Must(uuid.NewV4())

		req := &CreateCommentRequest{
			PostId: postID,
			Text:   "Test comment text",
		}

		assert.Equal(t, postID, req.PostId)
		assert.Equal(t, "Test comment text", req.Text)
	})

	t.Run("UpdateCommentRequest validation", func(t *testing.T) {
		objectID := uuid.Must(uuid.NewV4())

		req := &UpdateCommentRequest{
			ObjectId: objectID,
			Text:     "Updated comment text",
		}

		assert.Equal(t, objectID, req.ObjectId)
		assert.Equal(t, "Updated comment text", req.Text)
	})

	t.Run("Comment JSON serialization", func(t *testing.T) {
		commentID := uuid.Must(uuid.NewV4())
		postID := uuid.Must(uuid.NewV4())
		ownerUserID := uuid.Must(uuid.NewV4())

		comment := &Comment{
			ObjectId:    commentID,
			PostId:      postID,
			OwnerUserId: ownerUserID,
			Text:        "Response comment text",
			Score:       15,
			CreatedDate: 1640995200,
			LastUpdated: 1640995300,
		}

		// Test JSON serialization
		data, err := json.Marshal(comment)
		assert.NoError(t, err)

		// Test JSON deserialization
		var unmarshaled Comment
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)

		assert.Equal(t, comment.ObjectId, unmarshaled.ObjectId)
		assert.Equal(t, comment.PostId, unmarshaled.PostId)
		assert.Equal(t, comment.OwnerUserId, unmarshaled.OwnerUserId)
		assert.Equal(t, comment.Text, unmarshaled.Text)
		assert.Equal(t, comment.Score, unmarshaled.Score)
		assert.Equal(t, comment.CreatedDate, unmarshaled.CreatedDate)
		assert.Equal(t, comment.LastUpdated, unmarshaled.LastUpdated)
	})

	t.Run("CommentQueryFilter validation", func(t *testing.T) {
		postID := uuid.Must(uuid.NewV4())

		filter := &CommentQueryFilter{
			PostId: &postID,
			Limit:  20,
		}

		assert.Equal(t, &postID, filter.PostId)
		assert.Equal(t, 20, filter.Limit)
	})
}