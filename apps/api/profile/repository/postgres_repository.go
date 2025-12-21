// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/profile/models"
)

// postgresProfileRepository implements ProfileRepository using raw SQL queries
type postgresProfileRepository struct {
	client *postgres.Client
}

// getExecutor returns either the transaction from context or the DB connection
// Uses the same transaction key pattern as auth repository for cross-package transactions
func (r *postgresProfileRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	// Check for transaction in context (shared key with auth repository)
	// Using type assertion on interface{} to work across packages
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.client.DB()
}

// NewPostgresProfileRepository creates a new PostgreSQL repository for profiles
func NewPostgresProfileRepository(client *postgres.Client) ProfileRepository {
	return &postgresProfileRepository{
		client: client,
	}
}

// Create inserts a new profile
func (r *postgresProfileRepository) Create(ctx context.Context, profile *models.Profile) error {
	// Set timestamps if not set
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = time.Now()
	}
	if profile.CreatedDate == 0 {
		profile.CreatedDate = time.Now().Unix()
	}
	if profile.LastUpdated == 0 {
		profile.LastUpdated = time.Now().Unix()
	}

	query := `
		INSERT INTO profiles (
			user_id, full_name, social_name, email, avatar, banner, tagline,
			created_at, updated_at, created_date, last_updated, last_seen,
			birthday, web_url, company_name, country, address, phone,
			vote_count, share_count, follow_count, follower_count, post_count,
			facebook_id, instagram_id, twitter_id, linkedin_id,
			access_user_list, permission
		) VALUES (
			:user_id, :full_name, :social_name, :email, :avatar, :banner, :tagline,
			:created_at, :updated_at, :created_date, :last_updated, :last_seen,
			:birthday, :web_url, :company_name, :country, :address, :phone,
			:vote_count, :share_count, :follow_count, :follower_count, :post_count,
			:facebook_id, :instagram_id, :twitter_id, :linkedin_id,
			:access_user_list, :permission
		)`

	insertData := struct {
		UserID         uuid.UUID   `db:"user_id"`
		FullName       string      `db:"full_name"`
		SocialName     string      `db:"social_name"`
		Email          string      `db:"email"`
		Avatar         string      `db:"avatar"`
		Banner         string      `db:"banner"`
		Tagline        string      `db:"tagline"`
		CreatedAt      time.Time   `db:"created_at"`
		UpdatedAt      time.Time   `db:"updated_at"`
		CreatedDate    int64       `db:"created_date"`
		LastUpdated    int64       `db:"last_updated"`
		LastSeen       int64       `db:"last_seen"`
		Birthday       int64       `db:"birthday"`
		WebUrl         string      `db:"web_url"`
		CompanyName    string      `db:"company_name"`
		Country        string      `db:"country"`
		Address        string      `db:"address"`
		Phone          string      `db:"phone"`
		VoteCount      int64       `db:"vote_count"`
		ShareCount     int64       `db:"share_count"`
		FollowCount    int64       `db:"follow_count"`
		FollowerCount  int64       `db:"follower_count"`
		PostCount      int64       `db:"post_count"`
		FacebookId     string      `db:"facebook_id"`
		InstagramId    string      `db:"instagram_id"`
		TwitterId      string      `db:"twitter_id"`
		LinkedinId     string      `db:"linkedin_id"`
		AccessUserList interface{} `db:"access_user_list"`
		Permission     string      `db:"permission"`
	}{
		UserID:        profile.ObjectId,
		FullName:      profile.FullName,
		SocialName:    profile.SocialName,
		Email:         profile.Email,
		Avatar:        profile.Avatar,
		Banner:        profile.Banner,
		Tagline:       profile.Tagline,
		CreatedAt:     profile.CreatedAt,
		UpdatedAt:     profile.UpdatedAt,
		CreatedDate:   profile.CreatedDate,
		LastUpdated:   profile.LastUpdated,
		LastSeen:      profile.LastSeen,
		Birthday:      profile.Birthday,
		WebUrl:        profile.WebUrl,
		CompanyName:   profile.CompanyName,
		Country:       profile.Country,
		Address:       profile.Address,
		Phone:         profile.Phone,
		VoteCount:     profile.VoteCount,
		ShareCount:    profile.ShareCount,
		FollowCount:   profile.FollowCount,
		FollowerCount: profile.FollowerCount,
		PostCount:     profile.PostCount,
		FacebookId:    profile.FacebookId,
		InstagramId:   profile.InstagramId,
		TwitterId:     profile.TwitterId,
		LinkedinId:    profile.LinkedInId,
		Permission:    profile.Permission,
	}

	// Handle AccessUserList - ensure it's a pq.StringArray (not nil)
	if profile.AccessUserList == nil {
		insertData.AccessUserList = pq.StringArray{}
	} else {
		insertData.AccessUserList = profile.AccessUserList
	}

	_, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, insertData)
	if err != nil {
		// Check for unique constraint violation on social_name
		if strings.Contains(err.Error(), "idx_profiles_social_name") || strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("social name already exists: %w", err)
		}
		return fmt.Errorf("failed to create profile: %w", err)
	}

	return nil
}

// FindByID retrieves a profile by user ID
func (r *postgresProfileRepository) FindByID(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	query := `
		SELECT 
			user_id, full_name, social_name, email, avatar, banner, tagline,
			created_at, updated_at, created_date, last_updated, last_seen,
			birthday, web_url, company_name, country, address, phone,
			vote_count, share_count, follow_count, follower_count, post_count,
			facebook_id, instagram_id, twitter_id, linkedin_id,
			access_user_list, permission
		FROM profiles
		WHERE user_id = $1
	`

	var profile models.Profile
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &profile, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find profile: %w", err)
	}

	return &profile, nil
}

// FindBySocialName retrieves a profile by social name
func (r *postgresProfileRepository) FindBySocialName(ctx context.Context, socialName string) (*models.Profile, error) {
	query := `
		SELECT 
			user_id, full_name, social_name, email, avatar, banner, tagline,
			created_at, updated_at, created_date, last_updated, last_seen,
			birthday, web_url, company_name, country, address, phone,
			vote_count, share_count, follow_count, follower_count, post_count,
			facebook_id, instagram_id, twitter_id, linkedin_id,
			access_user_list, permission
		FROM profiles
		WHERE social_name = $1
	`

	var profile models.Profile
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &profile, query, socialName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find profile: %w", err)
	}

	return &profile, nil
}

// FindByIDs retrieves multiple profiles by user IDs
func (r *postgresProfileRepository) FindByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*models.Profile, error) {
	if len(userIDs) == 0 {
		return []*models.Profile{}, nil
	}

	// Convert UUIDs to strings for pq.Array (most reliable with PostgreSQL UUID arrays)
	idStrings := make([]string, len(userIDs))
	for i, id := range userIDs {
		idStrings[i] = id.String()
	}

	// Use ANY($1) which takes a single array argument - more performant than IN with multiple parameters
	// pq.Array properly encodes the string slice as a PostgreSQL array
	query := `
		SELECT 
			user_id, full_name, social_name, email, avatar, banner, tagline,
			created_at, updated_at, created_date, last_updated, last_seen,
			birthday, web_url, company_name, country, address, phone,
			vote_count, share_count, follow_count, follower_count, post_count,
			facebook_id, instagram_id, twitter_id, linkedin_id,
			access_user_list, permission
		FROM profiles
		WHERE user_id::text = ANY($1::text[])
		ORDER BY created_at DESC
	`

	var profiles []models.Profile
	// No sqlx.In needed here - it's a single argument (the array)
	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &profiles, query, pq.Array(idStrings))
	if err != nil {
		return nil, fmt.Errorf("failed to find profiles by IDs: %w", err)
	}

	result := make([]*models.Profile, len(profiles))
	for i := range profiles {
		result[i] = &profiles[i]
	}

	return result, nil
}

// Find retrieves profiles matching the filter criteria with pagination
func (r *postgresProfileRepository) Find(ctx context.Context, filter ProfileFilter, limit, offset int) ([]*models.Profile, error) {
	query, args := r.buildFindQuery(filter, limit, offset)

	var profiles []models.Profile
	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &profiles, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find profiles: %w", err)
	}

	result := make([]*models.Profile, len(profiles))
	for i := range profiles {
		result[i] = &profiles[i]
	}

	return result, nil
}

// Count returns the number of profiles matching the filter criteria
func (r *postgresProfileRepository) Count(ctx context.Context, filter ProfileFilter) (int64, error) {
	query, args := r.buildCountQuery(filter)

	var count int64
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count profiles: %w", err)
	}

	return count, nil
}

// Search finds profiles by full name, social name, or tagline using full-text search
func (r *postgresProfileRepository) Search(ctx context.Context, query string, limit int) ([]*models.Profile, error) {
	searchTerm := strings.TrimSpace(query)
	if searchTerm == "" {
		return []*models.Profile{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	sqlQuery := `
		SELECT
			user_id, full_name, social_name, email, avatar, banner, tagline,
			created_at, updated_at, created_date, last_updated, last_seen,
			birthday, web_url, company_name, country, address, phone,
			vote_count, share_count, follow_count, follower_count, post_count,
			facebook_id, instagram_id, twitter_id, linkedin_id,
			access_user_list, permission
		FROM profiles
		WHERE
			to_tsvector('english',
				COALESCE(full_name, '') || ' ' ||
				COALESCE(social_name, '') || ' ' ||
				COALESCE(tagline, '')
			) @@ plainto_tsquery('english', $1)
		ORDER BY
			ts_rank_cd(
				to_tsvector('english',
					COALESCE(full_name, '') || ' ' ||
					COALESCE(social_name, '') || ' ' ||
					COALESCE(tagline, '')
				), plainto_tsquery('english', $1)
			) DESC,
			created_at DESC
		LIMIT $2
	`

	var profiles []models.Profile
	if err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &profiles, sqlQuery, searchTerm, limit); err != nil {
		return nil, fmt.Errorf("failed to search profiles: %w", err)
	}

	results := make([]*models.Profile, len(profiles))
	for i := range profiles {
		results[i] = &profiles[i]
	}

	return results, nil
}

// Update updates an existing profile
func (r *postgresProfileRepository) Update(ctx context.Context, profile *models.Profile) error {
	profile.UpdatedAt = time.Now()
	profile.LastUpdated = time.Now().Unix()

	query := `
		UPDATE profiles SET
			full_name = :full_name,
			social_name = :social_name,
			email = :email,
			avatar = :avatar,
			banner = :banner,
			tagline = :tagline,
			updated_at = :updated_at,
			last_updated = :last_updated,
			last_seen = :last_seen,
			birthday = :birthday,
			web_url = :web_url,
			company_name = :company_name,
			country = :country,
			address = :address,
			phone = :phone,
			vote_count = :vote_count,
			share_count = :share_count,
			follow_count = :follow_count,
			follower_count = :follower_count,
			post_count = :post_count,
			facebook_id = :facebook_id,
			instagram_id = :instagram_id,
			twitter_id = :twitter_id,
			linkedin_id = :linkedin_id,
			access_user_list = :access_user_list,
			permission = :permission
		WHERE user_id = :user_id
	`

	updateData := struct {
		UserID         uuid.UUID   `db:"user_id"`
		FullName       string      `db:"full_name"`
		SocialName     string      `db:"social_name"`
		Email          string      `db:"email"`
		Avatar         string      `db:"avatar"`
		Banner         string      `db:"banner"`
		Tagline        string      `db:"tagline"`
		UpdatedAt      time.Time   `db:"updated_at"`
		LastUpdated    int64       `db:"last_updated"`
		LastSeen       int64       `db:"last_seen"`
		Birthday       int64       `db:"birthday"`
		WebUrl         string      `db:"web_url"`
		CompanyName    string      `db:"company_name"`
		Country        string      `db:"country"`
		Address        string      `db:"address"`
		Phone          string      `db:"phone"`
		VoteCount      int64       `db:"vote_count"`
		ShareCount     int64       `db:"share_count"`
		FollowCount    int64       `db:"follow_count"`
		FollowerCount  int64       `db:"follower_count"`
		PostCount      int64       `db:"post_count"`
		FacebookId     string      `db:"facebook_id"`
		InstagramId    string      `db:"instagram_id"`
		TwitterId      string      `db:"twitter_id"`
		LinkedinId     string      `db:"linkedin_id"`
		AccessUserList interface{} `db:"access_user_list"`
		Permission     string      `db:"permission"`
	}{
		UserID:         profile.ObjectId,
		FullName:       profile.FullName,
		SocialName:     profile.SocialName,
		Email:          profile.Email,
		Avatar:         profile.Avatar,
		Banner:         profile.Banner,
		Tagline:        profile.Tagline,
		UpdatedAt:      profile.UpdatedAt,
		LastUpdated:    profile.LastUpdated,
		LastSeen:       profile.LastSeen,
		Birthday:       profile.Birthday,
		WebUrl:         profile.WebUrl,
		CompanyName:    profile.CompanyName,
		Country:        profile.Country,
		Address:        profile.Address,
		Phone:          profile.Phone,
		VoteCount:      profile.VoteCount,
		ShareCount:     profile.ShareCount,
		FollowCount:    profile.FollowCount,
		FollowerCount:  profile.FollowerCount,
		PostCount:      profile.PostCount,
		FacebookId:     profile.FacebookId,
		InstagramId:    profile.InstagramId,
		TwitterId:      profile.TwitterId,
		LinkedinId:     profile.LinkedInId,
		AccessUserList: profile.AccessUserList,
		Permission:     profile.Permission,
	}

	result, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, updateData)
	if err != nil {
		// Check for unique constraint violation on social_name
		if strings.Contains(err.Error(), "idx_profiles_social_name") || strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("social name already exists: %w", err)
		}
		return fmt.Errorf("failed to update profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("profile not found")
	}

	return nil
}

// UpdateLastSeen updates the last_seen timestamp for a profile
func (r *postgresProfileRepository) UpdateLastSeen(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE profiles
		SET last_seen = EXTRACT(EPOCH FROM NOW())::BIGINT, updated_at = NOW()
		WHERE user_id = $1
	`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("profile not found")
	}

	return nil
}

// UpdateOwnerProfile updates display name and avatar for a profile
func (r *postgresProfileRepository) UpdateOwnerProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
	query := `
		UPDATE profiles
		SET full_name = $1, avatar = $2, updated_at = NOW(), last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE user_id = $3
	`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, displayName, avatar, userID)
	if err != nil {
		return fmt.Errorf("failed to update owner profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("profile not found")
	}

	return nil
}

// Delete deletes a profile by user ID
// Note: Profiles use hard delete (no soft delete field in schema)
func (r *postgresProfileRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM profiles WHERE user_id = $1`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("profile not found")
	}

	return nil
}

// buildFindQuery constructs a SQL query with WHERE clause based on filter criteria
func (r *postgresProfileRepository) buildFindQuery(filter ProfileFilter, limit, offset int) (string, []interface{}) {
	query := `
		SELECT 
			user_id, full_name, social_name, email, avatar, banner, tagline,
			created_at, updated_at, created_date, last_updated, last_seen,
			birthday, web_url, company_name, country, address, phone,
			vote_count, share_count, follow_count, follower_count, post_count,
			facebook_id, instagram_id, twitter_id, linkedin_id,
			access_user_list, permission
		FROM profiles
		WHERE 1=1`

	var args []interface{}
	argIndex := 1

	if filter.SocialName != nil && *filter.SocialName != "" {
		query += fmt.Sprintf(" AND social_name = $%d", argIndex)
		args = append(args, *filter.SocialName)
		argIndex++
	}

	if filter.Email != nil && *filter.Email != "" {
		query += fmt.Sprintf(" AND email = $%d", argIndex)
		args = append(args, *filter.Email)
		argIndex++
	}

	if filter.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_date >= $%d", argIndex)
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.SearchText != nil && *filter.SearchText != "" {
		// Use full-text search for better performance and relevance
		query += fmt.Sprintf(` AND to_tsvector('english',
			COALESCE(full_name, '') || ' ' ||
			COALESCE(social_name, '') || ' ' ||
			COALESCE(tagline, '')
		) @@ plainto_tsquery('english', $%d)`, argIndex)
		args = append(args, *filter.SearchText)
		argIndex++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	return query, args
}

// buildCountQuery constructs a COUNT query with WHERE clause based on filter criteria
func (r *postgresProfileRepository) buildCountQuery(filter ProfileFilter) (string, []interface{}) {
	query := "SELECT COUNT(*) FROM profiles WHERE 1=1"

	var args []interface{}
	argIndex := 1

	if filter.SocialName != nil && *filter.SocialName != "" {
		query += fmt.Sprintf(" AND social_name = $%d", argIndex)
		args = append(args, *filter.SocialName)
		argIndex++
	}

	if filter.Email != nil && *filter.Email != "" {
		query += fmt.Sprintf(" AND email = $%d", argIndex)
		args = append(args, *filter.Email)
		argIndex++
	}

	if filter.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_date >= $%d", argIndex)
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.SearchText != nil && *filter.SearchText != "" {
		query += fmt.Sprintf(` AND to_tsvector('english',
			COALESCE(full_name, '') || ' ' ||
			COALESCE(social_name, '') || ' ' ||
			COALESCE(tagline, '')
		) @@ plainto_tsquery('english', $%d)`, argIndex)
		args = append(args, *filter.SearchText)
		argIndex++
	}

	return query, args
}
