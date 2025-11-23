package validation

import (
	"testing"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/models"
)

func TestValidateCreateCommentRequest(t *testing.T) {
	validPostID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name    string
		req     *models.CreateCommentRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &models.CreateCommentRequest{
				PostId: validPostID,
				Text:   "This is a valid comment",
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty text",
			req: &models.CreateCommentRequest{
				PostId: validPostID,
				Text:   "",
			},
			wantErr: true,
		},
		{
			name: "whitespace only text",
			req: &models.CreateCommentRequest{
				PostId: validPostID,
				Text:   "   ",
			},
			wantErr: true,
		},
		{
			name: "text too long",
			req: &models.CreateCommentRequest{
				PostId: validPostID,
				Text:   string(make([]byte, 1001)), // 1001 characters
			},
			wantErr: true,
		},
		{
			name: "empty post ID",
			req: &models.CreateCommentRequest{
				PostId: [16]byte{},
				Text:   "Valid comment text",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateCommentRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateCommentRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateCommentRequest(t *testing.T) {
	validObjectID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name    string
		req     *models.UpdateCommentRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &models.UpdateCommentRequest{
				ObjectId: validObjectID,
				Text:     "Updated comment text",
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty text",
			req: &models.UpdateCommentRequest{
				ObjectId: validObjectID,
				Text:     "",
			},
			wantErr: true,
		},
		{
			name: "text too long",
			req: &models.UpdateCommentRequest{
				ObjectId: validObjectID,
				Text:     string(make([]byte, 1001)),
			},
			wantErr: true,
		},
		{
			name: "empty object ID",
			req: &models.UpdateCommentRequest{
				ObjectId: [16]byte{},
				Text:     "Valid text",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateCommentRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateCommentRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCommentQueryFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  *models.CommentQueryFilter
		wantErr bool
	}{
		{
			name: "valid filter",
			filter: &models.CommentQueryFilter{
				Page:  1,
				Limit: 10,
			},
			wantErr: false,
		},
		{
			name:    "nil filter",
			filter:  nil,
			wantErr: true,
		},
		{
			name: "auto-correct page and limit",
			filter: &models.CommentQueryFilter{
				Page:  0,
				Limit: 0,
			},
			wantErr: false,
		},
		{
			name: "limit too high",
			filter: &models.CommentQueryFilter{
				Page:  1,
				Limit: 101,
			},
			wantErr: false, // Should auto-correct to 100
		},
		{
			name: "invalid sort order",
			filter: &models.CommentQueryFilter{
				Page:      1,
				Limit:     10,
				SortOrder: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid sort field",
			filter: &models.CommentQueryFilter{
				Page:    1,
				Limit:   10,
				SortBy:  "invalid_field",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommentQueryFilter(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommentQueryFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
