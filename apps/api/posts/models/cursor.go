package models

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// CursorType represents the type of cursor
type CursorType string

const (
	CursorTypeTimestamp CursorType = "timestamp"
	CursorTypeID        CursorType = "id"
	CursorTypeComposite CursorType = "composite"
	CursorTypeScore     CursorType = "score"
)

// DefaultSortFields defines the default sort fields for different cursor types
var DefaultSortFields = map[CursorType]string{
	CursorTypeTimestamp: "createdDate",
	CursorTypeID:        "objectId",
	CursorTypeComposite: "createdDate",
	CursorTypeScore:     "score",
}

// EncodeCursor encodes cursor data into a base64 string
func EncodeCursor(data *CursorData) (string, error) {
	if data == nil {
		return "", nil
	}

	if err := data.Validate(); err != nil {
		return "", fmt.Errorf("invalid cursor data: %w", err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor data: %w", err)
	}

	return base64.URLEncoding.EncodeToString(jsonData), nil
}

// DecodeCursor decodes a base64 cursor string into cursor data
func DecodeCursor(cursor string) (*CursorData, error) {
	if cursor == "" {
		return nil, nil
	}

	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %w", err)
	}

	var data CursorData
	if err := json.Unmarshal(decoded, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor data: %w", err)
	}

	if err := data.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cursor data: %w", err)
	}

	return &data, nil
}

// CreateCursorFromPost creates a cursor from a post based on the sort field
func CreateCursorFromPost(post *Post, sortField, direction string) (string, error) {
	if post == nil {
		return "", errors.New("post cannot be nil")
	}

	if sortField == "" {
		sortField = "createdDate" // Default sort field
	}

	if direction == "" {
		direction = "desc" // Default direction
	}

	cursorData := &CursorData{
		ID:        post.ObjectId.String(),
		Timestamp: time.Now().Unix(),
		SortField: sortField,
		Direction: direction,
	}

	// Set the value based on the sort field
	switch sortField {
	case "createdDate":
		cursorData.Value = post.CreatedDate
	case "lastUpdated":
		cursorData.Value = post.LastUpdated
	case "score":
		cursorData.Value = post.Score
	case "viewCount":
		cursorData.Value = post.ViewCount
	case "commentCounter":
		cursorData.Value = post.CommentCounter
	case "objectId":
		cursorData.Value = post.ObjectId.String()
	default:
		return "", fmt.Errorf("unsupported sort field: %s", sortField)
	}

	return EncodeCursor(cursorData)
}

// ParseSortField validates and returns the sort field
func ParseSortField(sortField string) string {
	validFields := map[string]bool{
		"createdDate":    true,
		"lastUpdated":    true,
		"score":          true,
		"viewCount":      true,
		"commentCounter": true,
		"objectId":       true,
	}

	if sortField == "" || !validFields[sortField] {
		return "createdDate" // Default
	}

	return sortField
}

// ParseSortDirection validates and returns the sort direction
func ParseSortDirection(direction string) string {
	if direction == "asc" || direction == "desc" {
		return direction
	}
	return "desc" // Default
}

// CompareCursorValues compares two cursor values based on the sort field and direction
func CompareCursorValues(value1, value2 interface{}, sortField, direction string) int {
	switch sortField {
	case "createdDate", "lastUpdated", "deletedDate":
		return compareInt64(value1, value2, direction)
	case "score", "viewCount", "commentCounter":
		return compareInt64(value1, value2, direction)
	case "objectId":
		return compareString(value1, value2, direction)
	default:
		return 0
	}
}

// compareInt64 compares two int64 values
func compareInt64(value1, value2 interface{}, direction string) int {
	var v1, v2 int64

	switch v := value1.(type) {
	case int64:
		v1 = v
	case float64:
		v1 = int64(v)
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			v1 = parsed
		}
	}

	switch v := value2.(type) {
	case int64:
		v2 = v
	case float64:
		v2 = int64(v)
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			v2 = parsed
		}
	}

	if direction == "asc" {
		if v1 < v2 {
			return -1
		} else if v1 > v2 {
			return 1
		}
		return 0
	}

	// desc direction
	if v1 > v2 {
		return -1
	} else if v1 < v2 {
		return 1
	}
	return 0
}

// compareString compares two string values
func compareString(value1, value2 interface{}, direction string) int {
	v1, ok1 := value1.(string)
	v2, ok2 := value2.(string)

	if !ok1 || !ok2 {
		return 0
	}

	if direction == "asc" {
		if v1 < v2 {
			return -1
		} else if v1 > v2 {
			return 1
		}
		return 0
	}

	// desc direction
	if v1 > v2 {
		return -1
	} else if v1 < v2 {
		return 1
	}
	return 0
}

// BuildCursorQuery builds ONLY the cursor-based pagination query conditions
// Base filters (deleted, ownerUserId, etc.) are handled separately in the service layer
func BuildCursorQuery(filter *PostQueryFilter, cursorData *CursorData) map[string]interface{} {
	query := make(map[string]interface{})

	// Only add cursor-based filters, NOT base filters
	if cursorData != nil {
		sortField := cursorData.SortField
		direction := cursorData.Direction
		value := cursorData.Value

		var operator string
		// filter.Cursor is the standard parameter for forward pagination (treat same as AfterCursor)
		// AfterCursor and BeforeCursor are explicit directional cursors
		hasForwardCursor := filter.Cursor != "" || filter.AfterCursor != ""
		
		if direction == "desc" {
			if hasForwardCursor {
				operator = "$lt" // Less than for desc after cursor (forward pagination)
			} else if filter.BeforeCursor != "" {
				operator = "$gt" // Greater than for desc before cursor (backward pagination)
			}
		} else {
			if hasForwardCursor {
				operator = "$gt" // Greater than for asc after cursor (forward pagination)
			} else if filter.BeforeCursor != "" {
				operator = "$lt" // Less than for asc before cursor (backward pagination)
			}
		}

		if operator != "" {
			// Convert sortField to snake_case column name
			dbColumnName := toSnakeCaseColumn(sortField)
			
			// For composite queries, we need to handle ID tiebreakers
			if sortField == "objectId" {
				query["object_id"] = map[string]interface{}{
					operator: value,
				}
			} else {
				// Use compound query for non-ID fields with ID as tiebreaker
				// For DESC with forward pagination ($lt), we want:
				// - created_date < cursorValue OR
				// - (created_date == cursorValue AND object_id < cursorId)
				// This excludes the cursor post itself and all posts after it
				cursorCondition := []map[string]interface{}{
					{
						dbColumnName: map[string]interface{}{
							operator: value,
						},
					},
					{
						dbColumnName: value,
						"object_id": map[string]interface{}{
							operator: cursorData.ID,
						},
					},
				}
				query["$or"] = cursorCondition
			}
		}
	}

	return query
}

// ValidateLimit validates and normalizes the limit value
func ValidateLimit(limit int) int {
	if limit <= 0 {
		return 20 // Default limit
	}
	if limit > 100 {
		return 100 // Maximum limit
	}
	return limit
}

// toSnakeCaseColumn converts camelCase field names to snake_case column names
func toSnakeCaseColumn(fieldName string) string {
	switch fieldName {
	case "objectId", "ObjectId", "ObjectID":
		return "object_id"
	case "createdDate", "CreatedDate":
		return "created_date"
	case "lastUpdated", "LastUpdated":
		return "last_updated"
	case "deletedDate", "DeletedDate":
		return "deleted_date"
	case "ownerUserId", "OwnerUserId":
		return "owner_user_id"
	case "postTypeId", "PostTypeId":
		return "post_type_id"
	case "commentCounter", "CommentCounter":
		return "comment_counter"
	case "viewCount", "ViewCount":
		return "view_count"
	default:
		// For fields that don't have a direct column mapping, assume they're in JSONB
		// The query builder will handle JSONB paths
		return fieldName
	}
}
