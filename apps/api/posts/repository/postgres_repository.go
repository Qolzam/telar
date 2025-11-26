// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/posts/models"
)

// postgresRepository implements PostRepository using raw SQL queries
type postgresRepository struct {
	client *postgres.Client
	schema string // Schema name for search_path isolation
}

// NewPostgresRepository creates a new PostgreSQL repository for posts
func NewPostgresRepository(client *postgres.Client) PostRepository {
	return &postgresRepository{
		client: client,
		schema: "", // Default to empty (uses default schema)
	}
}

// NewPostgresRepositoryWithSchema creates a new PostgreSQL repository with explicit schema
func NewPostgresRepositoryWithSchema(client *postgres.Client, schema string) PostRepository {
	return &postgresRepository{
		client: client,
		schema: schema,
	}
}

// getExecutor returns either the transaction from context or the DB connection
func (r *postgresRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	// Check for transaction in context (shared key for cross-package transactions)
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.client.DB()
}

// Create inserts a new post
func (r *postgresRepository) Create(ctx context.Context, post *models.Post) error {
	// Build metadata JSONB from dynamic fields
	metadata := r.buildMetadata(post)

	query := `
		INSERT INTO posts (
		id, owner_user_id, post_type_id, body, score, view_count, 
		comment_count, is_deleted, deleted_date, created_at, updated_at,
		created_date, last_updated, tags, url_key, owner_display_name,
		owner_avatar, image, image_full_path, video, thumbnail,
		disable_comments, disable_sharing, permission, version, metadata
	) VALUES (
		:id, :owner_user_id, :post_type_id, :body, :score, :view_count,
		:comment_count, :is_deleted, :deleted_date, :created_at, :updated_at,
		:created_date, :last_updated, :tags, :url_key, :owner_display_name,
		:owner_avatar, :image, :image_full_path, :video, :thumbnail,
		:disable_comments, :disable_sharing, :permission, :version, :metadata
	)`

	// Set timestamps if not set
	if post.CreatedAt.IsZero() {
		post.CreatedAt = time.Now()
	}
	if post.UpdatedAt.IsZero() {
		post.UpdatedAt = time.Now()
	}
	if post.CreatedDate == 0 {
		post.CreatedDate = time.Now().Unix()
	}
	if post.LastUpdated == 0 {
		post.LastUpdated = time.Now().Unix()
	}

	// Prepare the struct for insertion
	insertData := struct {
		ID              uuid.UUID       `db:"id"`
		OwnerUserID     uuid.UUID       `db:"owner_user_id"`
		PostTypeID      int             `db:"post_type_id"`
		Body            string          `db:"body"`
		Score           int64           `db:"score"`
		ViewCount       int64           `db:"view_count"`
		CommentCount    int64           `db:"comment_count"`
		IsDeleted       bool            `db:"is_deleted"`
		DeletedDate     int64           `db:"deleted_date"`
		CreatedAt       time.Time       `db:"created_at"`
		UpdatedAt       time.Time       `db:"updated_at"`
		CreatedDate     int64           `db:"created_date"`
		LastUpdated     int64           `db:"last_updated"`
		Tags            interface{}     `db:"tags"`
		URLKey          string          `db:"url_key"`
		OwnerDisplayName string          `db:"owner_display_name"`
		OwnerAvatar     string          `db:"owner_avatar"`
		Image           string          `db:"image"`
		ImageFullPath   string          `db:"image_full_path"`
		Video           string          `db:"video"`
		Thumbnail       string          `db:"thumbnail"`
		DisableComments bool            `db:"disable_comments"`
		DisableSharing  bool            `db:"disable_sharing"`
		Permission      string          `db:"permission"`
		Version         string          `db:"version"`
		Metadata        json.RawMessage `db:"metadata"`
	}{
		ID:              post.ObjectId,
		OwnerUserID:     post.OwnerUserId,
		PostTypeID:      post.PostTypeId,
		Body:            post.Body,
		Score:           post.Score,
		ViewCount:       post.ViewCount,
		CommentCount:    post.CommentCounter,
		IsDeleted:       post.Deleted,
		DeletedDate:     post.DeletedDate,
		CreatedAt:       post.CreatedAt,
		UpdatedAt:       post.UpdatedAt,
		CreatedDate:     post.CreatedDate,
		LastUpdated:     post.LastUpdated,
		Tags:            post.Tags,
		URLKey:          post.URLKey,
		OwnerDisplayName: post.OwnerDisplayName,
		OwnerAvatar:     post.OwnerAvatar,
		Image:           post.Image,
		ImageFullPath:   post.ImageFullPath,
		Video:           post.Video,
		Thumbnail:       post.Thumbnail,
		DisableComments: post.DisableComments,
		DisableSharing:  post.DisableSharing,
		Permission:      post.Permission,
		Version:         post.Version,
		Metadata:        metadata,
	}

	executor := r.getExecutor(ctx)
	_, err := sqlx.NamedExecContext(ctx, executor, query, insertData)
	if err != nil {
		return err
	}
	return nil
}

// FindByID retrieves a post by its ID
func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	query := `
		SELECT 
			id, owner_user_id, post_type_id, body, score, view_count,
			comment_count, is_deleted, deleted_date, created_at, updated_at,
			created_date, last_updated, tags, url_key, owner_display_name,
			owner_avatar, image, image_full_path, video, thumbnail,
			disable_comments, disable_sharing, permission, version, metadata
		FROM posts
		WHERE id = $1 AND is_deleted = FALSE
	`

	// Scan into Post struct - sqlx will handle the metadata field via the db tag
	var post models.Post
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &post, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find post: %w", err)
	}

	// Populate dynamic fields from metadata
	if post.Metadata != nil {
		metadataJSON, _ := json.Marshal(post.Metadata)
		r.populateMetadata(&post, metadataJSON)
	}

	return &post, nil
}

// FindByUser retrieves posts by owner user ID with pagination
func (r *postgresRepository) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Post, error) {
	query := `
		SELECT 
			id, owner_user_id, post_type_id, body, score, view_count,
			comment_count, is_deleted, deleted_date, created_at, updated_at,
			created_date, last_updated, tags, url_key, owner_display_name,
			owner_avatar, image, image_full_path, video, thumbnail,
			disable_comments, disable_sharing, permission, version, metadata
		FROM posts
		WHERE owner_user_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`

	var posts []models.Post
	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &posts, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts by user: %w", err)
	}

	// Populate metadata for each post
	result := make([]*models.Post, len(posts))
	for i := range posts {
		post := &posts[i]
		if post.Metadata != nil {
			metadataJSON, _ := json.Marshal(post.Metadata)
			r.populateMetadata(post, metadataJSON)
		}
		result[i] = post
	}

	return result, nil
}

// Update updates an existing post
func (r *postgresRepository) Update(ctx context.Context, post *models.Post) error {
	// Build metadata JSONB from dynamic fields
	metadata := r.buildMetadata(post)

	// Set updated timestamp
	post.UpdatedAt = time.Now()
	post.LastUpdated = time.Now().Unix()

	query := `
		UPDATE posts SET
			owner_user_id = :owner_user_id,
			post_type_id = :post_type_id,
			body = :body,
			score = :score,
			view_count = :view_count,
			comment_count = :comment_count,
			is_deleted = :is_deleted,
			deleted_date = :deleted_date,
			updated_at = :updated_at,
			last_updated = :last_updated,
			tags = :tags,
			url_key = :url_key,
			owner_display_name = :owner_display_name,
			owner_avatar = :owner_avatar,
			image = :image,
			image_full_path = :image_full_path,
			video = :video,
			thumbnail = :thumbnail,
			disable_comments = :disable_comments,
			disable_sharing = :disable_sharing,
			permission = :permission,
			version = :version,
			metadata = :metadata
		WHERE id = :id
	`

	updateData := struct {
		ID              uuid.UUID       `db:"id"`
		OwnerUserID     uuid.UUID       `db:"owner_user_id"`
		PostTypeID      int             `db:"post_type_id"`
		Body            string          `db:"body"`
		Score           int64           `db:"score"`
		ViewCount       int64           `db:"view_count"`
		CommentCount    int64           `db:"comment_count"`
		IsDeleted       bool            `db:"is_deleted"`
		DeletedDate     int64           `db:"deleted_date"`
		UpdatedAt       time.Time       `db:"updated_at"`
		LastUpdated     int64           `db:"last_updated"`
		Tags            interface{}     `db:"tags"`
		URLKey          string          `db:"url_key"`
		OwnerDisplayName string          `db:"owner_display_name"`
		OwnerAvatar     string          `db:"owner_avatar"`
		Image           string          `db:"image"`
		ImageFullPath   string          `db:"image_full_path"`
		Video           string          `db:"video"`
		Thumbnail       string          `db:"thumbnail"`
		DisableComments bool            `db:"disable_comments"`
		DisableSharing  bool            `db:"disable_sharing"`
		Permission      string          `db:"permission"`
		Version         string          `db:"version"`
		Metadata        json.RawMessage `db:"metadata"`
	}{
		ID:              post.ObjectId,
		OwnerUserID:     post.OwnerUserId,
		PostTypeID:      post.PostTypeId,
		Body:            post.Body,
		Score:           post.Score,
		ViewCount:       post.ViewCount,
		CommentCount:    post.CommentCounter,
		IsDeleted:       post.Deleted,
		DeletedDate:     post.DeletedDate,
		UpdatedAt:       post.UpdatedAt,
		LastUpdated:     post.LastUpdated,
		Tags:            post.Tags,
		URLKey:          post.URLKey,
		OwnerDisplayName: post.OwnerDisplayName,
		OwnerAvatar:     post.OwnerAvatar,
		Image:           post.Image,
		ImageFullPath:   post.ImageFullPath,
		Video:           post.Video,
		Thumbnail:       post.Thumbnail,
		DisableComments: post.DisableComments,
		DisableSharing:  post.DisableSharing,
		Permission:      post.Permission,
		Version:         post.Version,
		Metadata:        metadata,
	}

	result, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, updateData)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

// IncrementViewCount atomically increments the view count for a post
func (r *postgresRepository) IncrementViewCount(ctx context.Context, postID uuid.UUID) error {
	query := `UPDATE posts SET view_count = view_count + 1 WHERE id = $1 AND is_deleted = FALSE`

	result, err := r.client.DB().ExecContext(ctx, query, postID)
	if err != nil {
		return fmt.Errorf("failed to increment view count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

// IncrementCommentCount atomically increments the comment count for a post
func (r *postgresRepository) IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int) error {
	query := `UPDATE posts SET comment_count = comment_count + $1, updated_at = NOW(), last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT WHERE id = $2 AND is_deleted = FALSE`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, delta, postID)
	if err != nil {
		return fmt.Errorf("failed to increment comment count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

// IncrementScore atomically increments the score for a post
func (r *postgresRepository) IncrementScore(ctx context.Context, postID uuid.UUID, delta int) error {
	executor := r.getExecutor(ctx)
	
	query := `
		UPDATE posts 
		SET score = score + $1,
		    updated_at = NOW(),
		    last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2 AND is_deleted = FALSE
	`
	
	result, err := executor.ExecContext(ctx, query, delta, postID)
	if err != nil {
		return fmt.Errorf("failed to increment score: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}
	
	return nil
}

// Delete deletes a post by ID (soft delete by setting is_deleted = TRUE)
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE posts 
		SET is_deleted = TRUE, deleted_date = $1, updated_at = NOW(), last_updated = $1
		WHERE id = $2 AND is_deleted = FALSE
	`

	deletedDate := time.Now().Unix()
	result, err := r.getExecutor(ctx).ExecContext(ctx, query, deletedDate, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

// buildMetadata builds the metadata JSONB from dynamic fields (Votes, Album, AccessUserList)
func (r *postgresRepository) buildMetadata(post *models.Post) json.RawMessage {
	metadata := make(map[string]interface{})
	
	if post.Votes != nil && len(post.Votes) > 0 {
		metadata["votes"] = post.Votes
	}
	if post.Album != nil {
		metadata["album"] = post.Album
	}
	if post.AccessUserList != nil && len(post.AccessUserList) > 0 {
		metadata["accessUserList"] = post.AccessUserList
	}

	if len(metadata) == 0 {
		return json.RawMessage("{}")
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return json.RawMessage("{}")
	}

	return json.RawMessage(jsonData)
}

// populateMetadata populates dynamic fields (Votes, Album, AccessUserList) from metadata JSONB
func (r *postgresRepository) populateMetadata(post *models.Post, metadataJSON json.RawMessage) {
	if len(metadataJSON) == 0 {
		return
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return
	}

	if votes, ok := metadata["votes"].(map[string]interface{}); ok {
		post.Votes = make(map[string]string)
		for k, v := range votes {
			if str, ok := v.(string); ok {
				post.Votes[k] = str
			}
		}
	}

	if albumData, ok := metadata["album"]; ok {
		albumJSON, err := json.Marshal(albumData)
		if err == nil {
			var album models.Album
			if err := json.Unmarshal(albumJSON, &album); err == nil {
				post.Album = &album
			}
		}
	}

	if accessList, ok := metadata["accessUserList"].([]interface{}); ok {
		post.AccessUserList = make([]string, 0, len(accessList))
		for _, item := range accessList {
			if str, ok := item.(string); ok {
				post.AccessUserList = append(post.AccessUserList, str)
			}
		}
	}
}

// FindByURLKey retrieves a post by its URL key
func (r *postgresRepository) FindByURLKey(ctx context.Context, urlKey string) (*models.Post, error) {
	query := `
		SELECT 
			id, owner_user_id, post_type_id, body, score, view_count,
			comment_count, is_deleted, deleted_date, created_at, updated_at,
			created_date, last_updated, tags, url_key, owner_display_name,
			owner_avatar, image, image_full_path, video, thumbnail,
			disable_comments, disable_sharing, permission, version, metadata
		FROM posts
		WHERE url_key = $1 AND is_deleted = FALSE
		LIMIT 1`

	var post models.Post
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &post, query, urlKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found with url_key: %s", urlKey)
		}
		return nil, fmt.Errorf("failed to find post by url_key: %w", err)
	}

	if post.Metadata != nil {
		metadataJSON, _ := json.Marshal(post.Metadata)
		r.populateMetadata(&post, metadataJSON)
	}

	return &post, nil
}

// Find retrieves posts matching the filter criteria with pagination
func (r *postgresRepository) Find(ctx context.Context, filter PostFilter, limit, offset int) ([]*models.Post, error) {
	query, args := r.buildFindQuery(filter, limit, offset)

	var posts []models.Post
	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &posts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}

	result := make([]*models.Post, len(posts))
	for i := range posts {
		post := &posts[i]
		if post.Metadata != nil {
			metadataJSON, _ := json.Marshal(post.Metadata)
			r.populateMetadata(post, metadataJSON)
		}
		result[i] = post
	}

	return result, nil
}

// Count returns the number of posts matching the filter criteria
func (r *postgresRepository) Count(ctx context.Context, filter PostFilter) (int64, error) {
	query, args := r.buildCountQuery(filter)

	var count int64
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return count, nil
}

// UpdateOwnerProfile updates display name and avatar for all posts by an owner
func (r *postgresRepository) UpdateOwnerProfile(ctx context.Context, ownerID uuid.UUID, displayName, avatar string) error {
	query := `
		UPDATE posts
		SET owner_display_name = $1, owner_avatar = $2, updated_at = NOW(), last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE owner_user_id = $3 AND is_deleted = FALSE`

	result, err := r.client.DB().ExecContext(ctx, query, displayName, avatar, ownerID)
	if err != nil {
		return fmt.Errorf("failed to update owner profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no posts found for owner: %s", ownerID.String())
	}

	return nil
}

// SetCommentDisabled sets the comment disabled flag for a post with ownership validation
// Ownership validation is embedded in the WHERE clause for atomicity and security
func (r *postgresRepository) SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error {
	query := `
		UPDATE posts
		SET disable_comments = $1, updated_at = NOW(), last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2 AND owner_user_id = $3 AND is_deleted = FALSE`

	result, err := r.client.DB().ExecContext(ctx, query, disabled, postID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to set comment disabled: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found, already deleted, or user does not own the post: %s", postID.String())
	}

	return nil
}

// SetSharingDisabled sets the sharing disabled flag for a post with ownership validation
// Ownership validation is embedded in the WHERE clause for atomicity and security
func (r *postgresRepository) SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, ownerID uuid.UUID) error {
	query := `
		UPDATE posts
		SET disable_sharing = $1, updated_at = NOW(), last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2 AND owner_user_id = $3 AND is_deleted = FALSE`

	result, err := r.client.DB().ExecContext(ctx, query, disabled, postID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to set sharing disabled: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found, already deleted, or user does not own the post: %s", postID.String())
	}

	return nil
}

// buildFindQuery constructs a SQL query with WHERE clause based on filter criteria
func (r *postgresRepository) buildFindQuery(filter PostFilter, limit, offset int) (string, []interface{}) {
	query := `
		SELECT id, owner_user_id, post_type_id, body, score, view_count,
			comment_count, is_deleted, deleted_date, created_at, updated_at,
			created_date, last_updated, tags, url_key, owner_display_name,
			owner_avatar, image, image_full_path, video, thumbnail,
			disable_comments, disable_sharing, permission, version, metadata
		FROM posts
		WHERE 1=1`

	var args []interface{}
	argIndex := 1

	if filter.OwnerUserID != nil {
		query += fmt.Sprintf(" AND owner_user_id = $%d", argIndex)
		args = append(args, *filter.OwnerUserID)
		argIndex++
	}

	if filter.PostTypeID != nil {
		query += fmt.Sprintf(" AND post_type_id = $%d", argIndex)
		args = append(args, *filter.PostTypeID)
		argIndex++
	}

	if len(filter.Tags) > 0 {
		query += fmt.Sprintf(" AND tags && $%d", argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}

	if filter.Deleted != nil {
		query += fmt.Sprintf(" AND is_deleted = $%d", argIndex)
		args = append(args, *filter.Deleted)
		argIndex++
	} else {
		query += " AND is_deleted = FALSE"
	}

	if filter.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_date >= $%d", argIndex)
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.URLKey != nil {
		query += fmt.Sprintf(" AND url_key = $%d", argIndex)
		args = append(args, *filter.URLKey)
		argIndex++
	}

	if filter.SearchText != nil && *filter.SearchText != "" {
		searchPattern := "%" + *filter.SearchText + "%"
		query += fmt.Sprintf(" AND (body ILIKE $%d OR owner_display_name ILIKE $%d)", argIndex, argIndex)
		args = append(args, searchPattern)
		argIndex++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	return query, args
}

// buildCountQuery constructs a COUNT query with WHERE clause based on filter criteria
func (r *postgresRepository) buildCountQuery(filter PostFilter) (string, []interface{}) {
	query := "SELECT COUNT(*) FROM posts WHERE 1=1"

	var args []interface{}
	argIndex := 1

	if filter.OwnerUserID != nil {
		query += fmt.Sprintf(" AND owner_user_id = $%d", argIndex)
		args = append(args, *filter.OwnerUserID)
		argIndex++
	}

	if filter.PostTypeID != nil {
		query += fmt.Sprintf(" AND post_type_id = $%d", argIndex)
		args = append(args, *filter.PostTypeID)
		argIndex++
	}

	if len(filter.Tags) > 0 {
		query += fmt.Sprintf(" AND tags && $%d", argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}

	if filter.Deleted != nil {
		query += fmt.Sprintf(" AND is_deleted = $%d", argIndex)
		args = append(args, *filter.Deleted)
		argIndex++
	} else {
		query += " AND is_deleted = FALSE"
	}

	if filter.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_date >= $%d", argIndex)
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.URLKey != nil {
		query += fmt.Sprintf(" AND url_key = $%d", argIndex)
		args = append(args, *filter.URLKey)
		argIndex++
	}

	if filter.SearchText != nil && *filter.SearchText != "" {
		searchPattern := "%" + *filter.SearchText + "%"
		query += fmt.Sprintf(" AND (body ILIKE $%d OR owner_display_name ILIKE $%d)", argIndex, argIndex)
		args = append(args, searchPattern)
		argIndex++
	}

	return query, args
}

// WithTransaction executes a function within a database transaction
func (r *postgresRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := r.client.DB().BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Set search_path for this transaction if schema is specified
	// This is critical for schema isolation in tests and multi-tenant scenarios
	if r.schema != "" {
		setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, r.schema)
		// Schema is set correctly - debug log removed for production
		_, err = tx.ExecContext(ctx, setSearchPathSQL)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to set search_path in transaction (schema=%s): %w", r.schema, err)
		}
	} else {
		// DEBUG: Log when schema is empty (this should not happen in isolated tests)
		fmt.Printf("[PostRepository.WithTransaction] ERROR: schema is empty, transaction will use default search_path\n")
	}

	// Inject transaction into context using shared key
	txCtx := context.WithValue(ctx, "tx", tx)

	// Execute the function
	if err := fn(txCtx); err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	// Commit on success
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

