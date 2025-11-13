package validation

import (
	"fmt"
	"strings"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/posts/models"
)

// ValidateCreatePostRequest validates the create post request
func ValidateCreatePostRequest(req *models.CreatePostRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.PostTypeId <= 0 {
		return fmt.Errorf("postTypeId must be greater than 0")
	}

	if req.Body == "" {
		return fmt.Errorf("body is required")
	}

	if len(req.Body) > 10000 {
		return fmt.Errorf("body must be less than 10000 characters")
	}

	if req.Permission != "" {
		validPermissions := []string{"Public", "OnlyMe", "Circles"}
		isValid := false
		for _, perm := range validPermissions {
			if req.Permission == perm {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("permission must be one of: %v", validPermissions)
		}
	}

	return nil
}

// ValidateUpdatePostRequest validates update post request
func ValidateUpdatePostRequest(req *models.UpdatePostRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	if req.ObjectId == nil {
		return fmt.Errorf("objectId is required")
	}
	if *req.ObjectId == uuid.Nil {
		return fmt.Errorf("objectId must be a valid UUID")
	}

	// Validate body if provided
	if req.Body != nil {
		if strings.TrimSpace(*req.Body) == "" {
			return fmt.Errorf("body cannot be empty or whitespace only")
		}
		if len(*req.Body) > 10000 {
			return fmt.Errorf("body cannot exceed 10000 characters")
		}
		if len(*req.Body) < 1 {
			return fmt.Errorf("body must be at least 1 character")
		}
	}

	// Validate tags if provided
	if req.Tags != nil {
		if len(*req.Tags) > 10 {
			return fmt.Errorf("maximum 10 tags allowed")
		}
		for i, tag := range *req.Tags {
			if strings.TrimSpace(tag) == "" {
				return fmt.Errorf("tag at index %d cannot be empty", i)
			}
			if len(tag) > 50 {
				return fmt.Errorf("tag at index %d cannot exceed 50 characters", i)
			}
		}
	}

	// Validate access user list if provided
	if req.AccessUserList != nil && len(*req.AccessUserList) > 100 {
		return fmt.Errorf("maximum 100 users allowed in access list")
	}

	// Validate permission if provided
	if req.Permission != nil && *req.Permission != "" {
		validPermissions := map[string]bool{
			"Public":  true,
			"OnlyMe":  true,
			"Circles": true,
		}
		if !validPermissions[*req.Permission] {
			return fmt.Errorf("permission must be one of: Public, OnlyMe, Circles")
		}
	}

	return nil
}

// ValidatePostQueryFilter validates query filter
func ValidatePostQueryFilter(filter *models.PostQueryFilter) error {
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

	// Validate sort field
	validSortFields := map[string]bool{
		"created_date":   true,
		"last_updated":   true,
		"score":          true,
		"viewCount":      true,
		"commentCounter": true,
	}

	if filter.SortBy != "" && !validSortFields[filter.SortBy] {
		return fmt.Errorf("invalid sortBy field: %s", filter.SortBy)
	}

	return nil
}
