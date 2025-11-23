package validation

import (
	"fmt"
	"strings"

	"github.com/qolzam/telar/apps/api/comments/models"
)

// ValidateCreateCommentRequest validates the create comment request
func ValidateCreateCommentRequest(req *models.CreateCommentRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.PostId == [16]byte{} {
		return fmt.Errorf("postId is required")
	}

	if req.Text == "" {
		return fmt.Errorf("text is required")
	}

	if len(req.Text) > 1000 {
		return fmt.Errorf("text must be less than 1000 characters")
	}

	if len(strings.TrimSpace(req.Text)) < 1 {
		return fmt.Errorf("text cannot be empty or whitespace only")
	}

	if req.ParentCommentId != nil && *req.ParentCommentId == [16]byte{} {
		return fmt.Errorf("parentCommentId, if provided, must be a valid UUID")
	}

	return nil
}

// ValidateUpdateCommentRequest validates update comment request
func ValidateUpdateCommentRequest(req *models.UpdateCommentRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.ObjectId == [16]byte{} {
		return fmt.Errorf("objectId is required")
	}

	if req.Text == "" {
		return fmt.Errorf("text is required")
	}

	if len(req.Text) > 1000 {
		return fmt.Errorf("text must be less than 1000 characters")
	}

	if len(strings.TrimSpace(req.Text)) < 1 {
		return fmt.Errorf("text cannot be empty or whitespace only")
	}

	return nil
}

// ValidateUpdateCommentProfileRequest validates update comment profile request
func ValidateUpdateCommentProfileRequest(req *models.UpdateCommentProfileRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.OwnerUserId == [16]byte{} {
		return fmt.Errorf("ownerUserId is required")
	}

	// Validate display name if provided
	if req.OwnerDisplayName != nil {
		if strings.TrimSpace(*req.OwnerDisplayName) == "" {
			return fmt.Errorf("ownerDisplayName cannot be empty or whitespace only")
		}
		if len(*req.OwnerDisplayName) > 100 {
			return fmt.Errorf("ownerDisplayName cannot exceed 100 characters")
		}
	}

	// Validate avatar if provided
	if req.OwnerAvatar != nil && len(*req.OwnerAvatar) > 500 {
		return fmt.Errorf("ownerAvatar URL cannot exceed 500 characters")
	}

	return nil
}

// ValidateCommentQueryFilter validates query filter
func ValidateCommentQueryFilter(filter *models.CommentQueryFilter) error {
	if filter == nil {
		return fmt.Errorf("filter is required")
	}

	// Set defaults and validate pagination
	if filter.Page < 1 {
		filter.Page = 1
	}

	if filter.Limit < 1 {
		filter.Limit = 10
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	// Validate sort order
	if filter.SortOrder != "" && filter.SortOrder != "asc" && filter.SortOrder != "desc" {
		return fmt.Errorf("sortOrder must be 'asc' or 'desc'")
	}

	// Validate sort direction
	if filter.SortDirection != "" && filter.SortDirection != "asc" && filter.SortDirection != "desc" {
		return fmt.Errorf("sortDirection must be 'asc' or 'desc'")
	}

	// Validate sort field
	validSortFields := map[string]bool{
		"created_date": true,
		"last_updated": true,
		"score":        true,
	}

	if filter.SortBy != "" && !validSortFields[filter.SortBy] {
		return fmt.Errorf("invalid sortBy field: %s", filter.SortBy)
	}

	if filter.SortField != "" && !validSortFields[filter.SortField] {
		return fmt.Errorf("invalid sortField: %s", filter.SortField)
	}

	return nil
}
