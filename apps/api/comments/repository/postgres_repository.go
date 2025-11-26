// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qolzam/telar/apps/api/comments/models"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
)

// postgresCommentRepository implements CommentRepository using raw SQL queries
type postgresCommentRepository struct {
	client *postgres.Client
	schema string // Schema name for search_path isolation
}

// NewPostgresCommentRepository creates a new PostgreSQL repository for comments
func NewPostgresCommentRepository(client *postgres.Client) CommentRepository {
	return &postgresCommentRepository{
		client: client,
		schema: "", // Default to empty (uses default schema)
	}
}

// NewPostgresCommentRepositoryWithSchema creates a new PostgreSQL repository with explicit schema
func NewPostgresCommentRepositoryWithSchema(client *postgres.Client, schema string) CommentRepository {
	return &postgresCommentRepository{
		client: client,
		schema: schema,
	}
}

// getExecutor returns either the transaction from context or the DB connection
// If not in a transaction and schema is set, ensures search_path is set before returning executor
func (r *postgresCommentRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	// Check for transaction in context (shared key for cross-package transactions)
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			// Transaction already has search_path set in PostRepository.WithTransaction
			return tx
		}
	}
	
	// Not in transaction - ensure search_path is set on connection if schema is specified
	// This is necessary because connection pooling means each connection needs search_path set
	if r.schema != "" {
		setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, r.schema)
		// Execute on the connection to set search_path for subsequent queries
		// Note: This is per-connection, so it will persist for this connection's lifetime
		_, _ = r.client.DB().ExecContext(ctx, setSearchPathSQL)
	}
	
	return r.client.DB()
}

// Create inserts a new comment
func (r *postgresCommentRepository) Create(ctx context.Context, comment *models.Comment) error {
	// Set timestamps if not set
	now := time.Now()
	nowUnix := now.Unix()
	if comment.CreatedDate == 0 {
		comment.CreatedDate = nowUnix
	}
	if comment.LastUpdated == 0 {
		comment.LastUpdated = nowUnix
	}

	query := `
		INSERT INTO comments (
			id, post_id, owner_user_id, parent_comment_id, text, score,
			owner_display_name, owner_avatar, is_deleted, deleted_date,
			created_at, updated_at, created_date, last_updated
		) VALUES (
			:id, :post_id, :owner_user_id, :parent_comment_id, :text, :score,
			:owner_display_name, :owner_avatar, :is_deleted, :deleted_date,
			:created_at, :updated_at, :created_date, :last_updated
		)`

	insertData := struct {
		ID              uuid.UUID   `db:"id"`
		PostID          uuid.UUID   `db:"post_id"`
		OwnerUserID     uuid.UUID   `db:"owner_user_id"`
		ParentCommentID *uuid.UUID  `db:"parent_comment_id"`
		Text            string      `db:"text"`
		Score           int64       `db:"score"`
		OwnerDisplayName string      `db:"owner_display_name"`
		OwnerAvatar     string      `db:"owner_avatar"`
		IsDeleted       bool        `db:"is_deleted"`
		DeletedDate     int64       `db:"deleted_date"`
		CreatedAt       time.Time   `db:"created_at"`
		UpdatedAt       time.Time   `db:"updated_at"`
		CreatedDate     int64       `db:"created_date"`
		LastUpdated     int64       `db:"last_updated"`
	}{
		ID:              comment.ObjectId,
		PostID:          comment.PostId,
		OwnerUserID:     comment.OwnerUserId,
		ParentCommentID: comment.ParentCommentId,
		Text:            comment.Text,
		Score:           comment.Score,
		OwnerDisplayName: comment.OwnerDisplayName,
		OwnerAvatar:     comment.OwnerAvatar,
		IsDeleted:       comment.Deleted,
		DeletedDate:     comment.DeletedDate,
		CreatedAt:       now,
		UpdatedAt:       now,
		CreatedDate:     comment.CreatedDate,
		LastUpdated:     comment.LastUpdated,
	}

	_, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, insertData)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// FindByID retrieves a comment by its ID
func (r *postgresCommentRepository) FindByID(ctx context.Context, commentID uuid.UUID) (*models.Comment, error) {
	query := `
		SELECT 
			id, post_id, owner_user_id, parent_comment_id, text, score,
			owner_display_name, owner_avatar, is_deleted, deleted_date,
			created_at, updated_at, created_date, last_updated
		FROM comments 
		WHERE id = $1`

	var result struct {
		ID              uuid.UUID   `db:"id"`
		PostID          uuid.UUID   `db:"post_id"`
		OwnerUserID     uuid.UUID   `db:"owner_user_id"`
		ParentCommentID *uuid.UUID  `db:"parent_comment_id"`
		Text            string      `db:"text"`
		Score           int64       `db:"score"`
		OwnerDisplayName string      `db:"owner_display_name"`
		OwnerAvatar     string      `db:"owner_avatar"`
		IsDeleted       bool        `db:"is_deleted"`
		DeletedDate     int64       `db:"deleted_date"`
		CreatedAt       time.Time   `db:"created_at"`
		UpdatedAt       time.Time   `db:"updated_at"`
		CreatedDate     int64       `db:"created_date"`
		LastUpdated     int64       `db:"last_updated"`
	}

	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &result, query, commentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("comment not found")
		}
		return nil, fmt.Errorf("failed to find comment by ID: %w", err)
	}

	return &models.Comment{
		ObjectId:         result.ID,
		PostId:           result.PostID,
		OwnerUserId:      result.OwnerUserID,
		ParentCommentId:  result.ParentCommentID,
		Text:             result.Text,
		Score:            result.Score,
		OwnerDisplayName: result.OwnerDisplayName,
		OwnerAvatar:      result.OwnerAvatar,
		Deleted:          result.IsDeleted,
		DeletedDate:      result.DeletedDate,
		CreatedDate:      result.CreatedDate,
		LastUpdated:      result.LastUpdated,
	}, nil
}

// FindByPostID retrieves root comments for a specific post with pagination
func (r *postgresCommentRepository) FindByPostID(ctx context.Context, postID uuid.UUID, limit, offset int) ([]*models.Comment, error) {
	query := `
		SELECT 
			id, post_id, owner_user_id, parent_comment_id, text, score,
			owner_display_name, owner_avatar, is_deleted, deleted_date,
			created_at, updated_at, created_date, last_updated
		FROM comments 
		WHERE post_id = $1 AND parent_comment_id IS NULL AND is_deleted = FALSE
		ORDER BY created_date DESC
		LIMIT $2 OFFSET $3`

	var results []struct {
		ID              uuid.UUID   `db:"id"`
		PostID          uuid.UUID   `db:"post_id"`
		OwnerUserID     uuid.UUID   `db:"owner_user_id"`
		ParentCommentID *uuid.UUID  `db:"parent_comment_id"`
		Text            string      `db:"text"`
		Score           int64       `db:"score"`
		OwnerDisplayName string      `db:"owner_display_name"`
		OwnerAvatar     string      `db:"owner_avatar"`
		IsDeleted       bool        `db:"is_deleted"`
		DeletedDate     int64       `db:"deleted_date"`
		CreatedAt       time.Time   `db:"created_at"`
		UpdatedAt       time.Time   `db:"updated_at"`
		CreatedDate     int64       `db:"created_date"`
		LastUpdated     int64       `db:"last_updated"`
	}

	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &results, query, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find comments by post ID: %w", err)
	}

	comments := make([]*models.Comment, len(results))
	for i, result := range results {
		comments[i] = &models.Comment{
			ObjectId:         result.ID,
			PostId:           result.PostID,
			OwnerUserId:      result.OwnerUserID,
			ParentCommentId:  result.ParentCommentID,
			Text:             result.Text,
			Score:            result.Score,
			OwnerDisplayName: result.OwnerDisplayName,
			OwnerAvatar:      result.OwnerAvatar,
			Deleted:          result.IsDeleted,
			DeletedDate:      result.DeletedDate,
			CreatedDate:      result.CreatedDate,
			LastUpdated:      result.LastUpdated,
		}
	}

	return comments, nil
}

// FindByPostIDWithCursor retrieves root comments for a specific post with cursor-based pagination
// Uses keyset pagination with created_date DESC, id DESC for stable ordering
func (r *postgresCommentRepository) FindByPostIDWithCursor(ctx context.Context, postID uuid.UUID, cursor string, limit int) ([]*models.Comment, string, error) {
	// Parse cursor if provided
	var cursorCreatedDate int64
	var cursorID uuid.UUID
	hasCursor := false
	
	if cursor != "" {
		// Simple cursor format: base64(created_date:uuid)
		// Decode cursor
		decoded, err := base64.URLEncoding.DecodeString(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor format: %w", err)
		}
		
		// Parse format: "created_date:uuid"
		parts := strings.Split(string(decoded), ":")
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid cursor format: expected created_date:uuid")
		}
		
		cursorCreatedDate, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor created_date: %w", err)
		}
		
		cursorID, err = uuid.FromString(parts[1])
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor id: %w", err)
		}
		
		hasCursor = true
	}
	
	// Build query with cursor condition
	query := `
		SELECT 
			id, post_id, owner_user_id, parent_comment_id, text, score,
			owner_display_name, owner_avatar, is_deleted, deleted_date,
			created_at, updated_at, created_date, last_updated
		FROM comments 
		WHERE post_id = $1 AND parent_comment_id IS NULL AND is_deleted = FALSE`
	
	args := []interface{}{postID}
	argIndex := 2
	
	if hasCursor {
		// Keyset pagination: (created_date < cursorCreatedDate) OR (created_date = cursorCreatedDate AND id < cursorID)
		query += fmt.Sprintf(` AND ((created_date < $%d) OR (created_date = $%d AND id < $%d))`, argIndex, argIndex, argIndex+1)
		args = append(args, cursorCreatedDate, cursorID)
		argIndex += 2
	}
	
	query += ` ORDER BY created_date DESC, id DESC LIMIT $` + fmt.Sprintf("%d", argIndex)
	args = append(args, limit+1) // Fetch one extra to determine if there's a next page
	
	var results []struct {
		ID              uuid.UUID   `db:"id"`
		PostID          uuid.UUID   `db:"post_id"`
		OwnerUserID     uuid.UUID   `db:"owner_user_id"`
		ParentCommentID *uuid.UUID  `db:"parent_comment_id"`
		Text            string      `db:"text"`
		Score           int64       `db:"score"`
		OwnerDisplayName string      `db:"owner_display_name"`
		OwnerAvatar     string      `db:"owner_avatar"`
		IsDeleted       bool        `db:"is_deleted"`
		DeletedDate     int64       `db:"deleted_date"`
		CreatedAt       time.Time   `db:"created_at"`
		UpdatedAt       time.Time   `db:"updated_at"`
		CreatedDate     int64       `db:"created_date"`
		LastUpdated     int64       `db:"last_updated"`
	}
	
	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &results, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find comments by post ID with cursor: %w", err)
	}
	
	// Check if there's a next page
	hasNext := len(results) > limit
	if hasNext {
		results = results[:limit] // Remove the extra item
	}
	
	comments := make([]*models.Comment, len(results))
	for i, result := range results {
		comments[i] = &models.Comment{
			ObjectId:         result.ID,
			PostId:           result.PostID,
			OwnerUserId:      result.OwnerUserID,
			ParentCommentId:  result.ParentCommentID,
			Text:             result.Text,
			Score:            result.Score,
			OwnerDisplayName: result.OwnerDisplayName,
			OwnerAvatar:      result.OwnerAvatar,
			Deleted:          result.IsDeleted,
			DeletedDate:      result.DeletedDate,
			CreatedDate:      result.CreatedDate,
			LastUpdated:      result.LastUpdated,
		}
	}
	
	// Generate next cursor from last comment
	var nextCursor string
	if hasNext && len(comments) > 0 {
		lastComment := comments[len(comments)-1]
		cursorData := fmt.Sprintf("%d:%s", lastComment.CreatedDate, lastComment.ObjectId.String())
		nextCursor = base64.URLEncoding.EncodeToString([]byte(cursorData))
	}
	
	return comments, nextCursor, nil
}

// FindByUserID retrieves comments created by a specific user with pagination
func (r *postgresCommentRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Comment, error) {
	query := `
		SELECT 
			id, post_id, owner_user_id, parent_comment_id, text, score,
			owner_display_name, owner_avatar, is_deleted, deleted_date,
			created_at, updated_at, created_date, last_updated
		FROM comments 
		WHERE owner_user_id = $1 AND is_deleted = FALSE
		ORDER BY created_date DESC
		LIMIT $2 OFFSET $3`

	var results []struct {
		ID              uuid.UUID   `db:"id"`
		PostID          uuid.UUID   `db:"post_id"`
		OwnerUserID     uuid.UUID   `db:"owner_user_id"`
		ParentCommentID *uuid.UUID  `db:"parent_comment_id"`
		Text            string      `db:"text"`
		Score           int64       `db:"score"`
		OwnerDisplayName string      `db:"owner_display_name"`
		OwnerAvatar     string      `db:"owner_avatar"`
		IsDeleted       bool        `db:"is_deleted"`
		DeletedDate     int64       `db:"deleted_date"`
		CreatedAt       time.Time   `db:"created_at"`
		UpdatedAt       time.Time   `db:"updated_at"`
		CreatedDate     int64       `db:"created_date"`
		LastUpdated     int64       `db:"last_updated"`
	}

	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &results, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find comments by user ID: %w", err)
	}

	comments := make([]*models.Comment, len(results))
	for i, result := range results {
		comments[i] = &models.Comment{
			ObjectId:         result.ID,
			PostId:           result.PostID,
			OwnerUserId:      result.OwnerUserID,
			ParentCommentId:  result.ParentCommentID,
			Text:             result.Text,
			Score:            result.Score,
			OwnerDisplayName: result.OwnerDisplayName,
			OwnerAvatar:      result.OwnerAvatar,
			Deleted:          result.IsDeleted,
			DeletedDate:      result.DeletedDate,
			CreatedDate:      result.CreatedDate,
			LastUpdated:      result.LastUpdated,
		}
	}

	return comments, nil
}

// FindReplies retrieves replies to a specific comment with pagination
func (r *postgresCommentRepository) FindReplies(ctx context.Context, parentID uuid.UUID, limit, offset int) ([]*models.Comment, error) {
	query := `
		SELECT 
			id, post_id, owner_user_id, parent_comment_id, text, score,
			owner_display_name, owner_avatar, is_deleted, deleted_date,
			created_at, updated_at, created_date, last_updated
		FROM comments 
		WHERE parent_comment_id = $1 AND is_deleted = FALSE
		ORDER BY created_date ASC
		LIMIT $2 OFFSET $3`

	var results []struct {
		ID              uuid.UUID   `db:"id"`
		PostID          uuid.UUID   `db:"post_id"`
		OwnerUserID     uuid.UUID   `db:"owner_user_id"`
		ParentCommentID *uuid.UUID  `db:"parent_comment_id"`
		Text            string      `db:"text"`
		Score           int64       `db:"score"`
		OwnerDisplayName string      `db:"owner_display_name"`
		OwnerAvatar     string      `db:"owner_avatar"`
		IsDeleted       bool        `db:"is_deleted"`
		DeletedDate     int64       `db:"deleted_date"`
		CreatedAt       time.Time   `db:"created_at"`
		UpdatedAt       time.Time   `db:"updated_at"`
		CreatedDate     int64       `db:"created_date"`
		LastUpdated     int64       `db:"last_updated"`
	}

	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &results, query, parentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find replies: %w", err)
	}

	comments := make([]*models.Comment, len(results))
	for i, result := range results {
		comments[i] = &models.Comment{
			ObjectId:         result.ID,
			PostId:           result.PostID,
			OwnerUserId:      result.OwnerUserID,
			ParentCommentId:  result.ParentCommentID,
			Text:             result.Text,
			Score:            result.Score,
			OwnerDisplayName: result.OwnerDisplayName,
			OwnerAvatar:      result.OwnerAvatar,
			Deleted:          result.IsDeleted,
			DeletedDate:      result.DeletedDate,
			CreatedDate:      result.CreatedDate,
			LastUpdated:      result.LastUpdated,
		}
	}

	return comments, nil
}

// CountByPostID counts root comments (not replies) for a post
func (r *postgresCommentRepository) CountByPostID(ctx context.Context, postID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM comments WHERE post_id = $1 AND parent_comment_id IS NULL AND is_deleted = FALSE`

	var count int64
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &count, query, postID)
	if err != nil {
		return 0, fmt.Errorf("failed to count comments by post ID: %w", err)
	}

	return count, nil
}

// CountReplies counts replies to a specific comment
func (r *postgresCommentRepository) CountReplies(ctx context.Context, parentID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM comments WHERE parent_comment_id = $1 AND is_deleted = FALSE`

	var count int64
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &count, query, parentID)
	if err != nil {
		return 0, fmt.Errorf("failed to count replies: %w", err)
	}

	return count, nil
}

// Find retrieves comments matching the filter criteria with pagination
func (r *postgresCommentRepository) Find(ctx context.Context, filter CommentFilter, limit, offset int) ([]*models.Comment, error) {
	query := `SELECT 
		id, post_id, owner_user_id, parent_comment_id, text, score,
		owner_display_name, owner_avatar, is_deleted, deleted_date,
		created_at, updated_at, created_date, last_updated
		FROM comments WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.PostID != nil {
		query += fmt.Sprintf(` AND post_id = $%d`, argIndex)
		args = append(args, *filter.PostID)
		argIndex++
	}
	if filter.OwnerUserID != nil {
		query += fmt.Sprintf(` AND owner_user_id = $%d`, argIndex)
		args = append(args, *filter.OwnerUserID)
		argIndex++
	}
	if filter.ParentCommentID != nil {
		query += fmt.Sprintf(` AND parent_comment_id = $%d`, argIndex)
		args = append(args, *filter.ParentCommentID)
		argIndex++
	} else if filter.RootOnly {
		query += ` AND parent_comment_id IS NULL`
	}
	if !filter.IncludeDeleted {
		if filter.Deleted != nil {
			query += fmt.Sprintf(` AND is_deleted = $%d`, argIndex)
			args = append(args, *filter.Deleted)
			argIndex++
		} else {
			query += ` AND is_deleted = FALSE`
		}
	}
	if filter.CreatedAfter != nil {
		query += fmt.Sprintf(` AND created_date >= $%d`, argIndex)
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}
	if filter.CreatedBefore != nil {
		query += fmt.Sprintf(` AND created_date <= $%d`, argIndex)
		args = append(args, *filter.CreatedBefore)
		argIndex++
	}

	query += ` ORDER BY created_date DESC LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)
	args = append(args, limit, offset)

	var results []struct {
		ID              uuid.UUID   `db:"id"`
		PostID          uuid.UUID   `db:"post_id"`
		OwnerUserID     uuid.UUID   `db:"owner_user_id"`
		ParentCommentID *uuid.UUID  `db:"parent_comment_id"`
		Text            string      `db:"text"`
		Score           int64       `db:"score"`
		OwnerDisplayName string      `db:"owner_display_name"`
		OwnerAvatar     string      `db:"owner_avatar"`
		IsDeleted       bool        `db:"is_deleted"`
		DeletedDate     int64       `db:"deleted_date"`
		CreatedAt       time.Time   `db:"created_at"`
		UpdatedAt       time.Time   `db:"updated_at"`
		CreatedDate     int64       `db:"created_date"`
		LastUpdated     int64       `db:"last_updated"`
	}

	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &results, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find comments: %w", err)
	}

	comments := make([]*models.Comment, len(results))
	for i, result := range results {
		comments[i] = &models.Comment{
			ObjectId:         result.ID,
			PostId:           result.PostID,
			OwnerUserId:      result.OwnerUserID,
			ParentCommentId:  result.ParentCommentID,
			Text:             result.Text,
			Score:            result.Score,
			OwnerDisplayName: result.OwnerDisplayName,
			OwnerAvatar:      result.OwnerAvatar,
			Deleted:          result.IsDeleted,
			DeletedDate:      result.DeletedDate,
			CreatedDate:      result.CreatedDate,
			LastUpdated:      result.LastUpdated,
		}
	}

	return comments, nil
}

// Count returns the number of comments matching the filter criteria
func (r *postgresCommentRepository) Count(ctx context.Context, filter CommentFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM comments WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.PostID != nil {
		query += fmt.Sprintf(` AND post_id = $%d`, argIndex)
		args = append(args, *filter.PostID)
		argIndex++
	}
	if filter.OwnerUserID != nil {
		query += fmt.Sprintf(` AND owner_user_id = $%d`, argIndex)
		args = append(args, *filter.OwnerUserID)
		argIndex++
	}
	if filter.ParentCommentID != nil {
		query += fmt.Sprintf(` AND parent_comment_id = $%d`, argIndex)
		args = append(args, *filter.ParentCommentID)
		argIndex++
	} else if filter.RootOnly {
		query += ` AND parent_comment_id IS NULL`
	}
	if !filter.IncludeDeleted {
		if filter.Deleted != nil {
			query += fmt.Sprintf(` AND is_deleted = $%d`, argIndex)
			args = append(args, *filter.Deleted)
			argIndex++
		} else {
			query += ` AND is_deleted = FALSE`
		}
	}
	if filter.CreatedAfter != nil {
		query += fmt.Sprintf(` AND created_date >= $%d`, argIndex)
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}
	if filter.CreatedBefore != nil {
		query += fmt.Sprintf(` AND created_date <= $%d`, argIndex)
		args = append(args, *filter.CreatedBefore)
		argIndex++
	}

	var count int64
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count comments: %w", err)
	}

	return count, nil
}

// Update updates an existing comment
func (r *postgresCommentRepository) Update(ctx context.Context, comment *models.Comment) error {
	now := time.Now()
	nowUnix := now.Unix()
	if comment.LastUpdated == 0 {
		comment.LastUpdated = nowUnix
	}

	query := `
		UPDATE comments SET
			text = :text,
			score = :score,
			owner_display_name = :owner_display_name,
			owner_avatar = :owner_avatar,
			is_deleted = :is_deleted,
			deleted_date = :deleted_date,
			updated_at = :updated_at,
			last_updated = :last_updated
		WHERE id = :id`

	updateData := struct {
		ID              uuid.UUID   `db:"id"`
		Text            string      `db:"text"`
		Score           int64       `db:"score"`
		OwnerDisplayName string      `db:"owner_display_name"`
		OwnerAvatar     string      `db:"owner_avatar"`
		IsDeleted       bool        `db:"is_deleted"`
		DeletedDate     int64       `db:"deleted_date"`
		UpdatedAt       time.Time   `db:"updated_at"`
		LastUpdated     int64       `db:"last_updated"`
	}{
		ID:              comment.ObjectId,
		Text:            comment.Text,
		Score:           comment.Score,
		OwnerDisplayName: comment.OwnerDisplayName,
		OwnerAvatar:     comment.OwnerAvatar,
		IsDeleted:       comment.Deleted,
		DeletedDate:     comment.DeletedDate,
		UpdatedAt:       now,
		LastUpdated:     comment.LastUpdated,
	}

	result, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, updateData)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found")
	}

	return nil
}

// UpdateOwnerProfile updates display name and avatar for all comments by an owner
func (r *postgresCommentRepository) UpdateOwnerProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
	query := `
		UPDATE comments SET
			owner_display_name = $1,
			owner_avatar = $2,
			updated_at = NOW(),
			last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE owner_user_id = $3`

	_, err := r.getExecutor(ctx).ExecContext(ctx, query, displayName, avatar, userID)
	if err != nil {
		return fmt.Errorf("failed to update owner profile: %w", err)
	}

	return nil
}

// IncrementScore atomically increments the score for a comment
func (r *postgresCommentRepository) IncrementScore(ctx context.Context, commentID uuid.UUID, delta int) error {
	query := `UPDATE comments SET score = score + $1, updated_at = NOW(), last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT WHERE id = $2`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, delta, commentID)
	if err != nil {
		return fmt.Errorf("failed to increment score: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found")
	}

	return nil
}

// Delete soft deletes a comment by ID
func (r *postgresCommentRepository) Delete(ctx context.Context, commentID uuid.UUID) error {
	nowUnix := time.Now().Unix()
	query := `UPDATE comments SET is_deleted = TRUE, deleted_date = $1, updated_at = NOW(), last_updated = $1 WHERE id = $2`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, nowUnix, commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found")
	}

	return nil
}

// DeleteByPostID soft deletes all comments for a post
func (r *postgresCommentRepository) DeleteByPostID(ctx context.Context, postID uuid.UUID) error {
	nowUnix := time.Now().Unix()
	query := `UPDATE comments SET is_deleted = TRUE, deleted_date = $1, updated_at = NOW(), last_updated = $1 WHERE post_id = $2`

	_, err := r.getExecutor(ctx).ExecContext(ctx, query, nowUnix, postID)
	if err != nil {
		return fmt.Errorf("failed to delete comments by post ID: %w", err)
	}

	return nil
}

