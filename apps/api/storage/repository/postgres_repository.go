// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/storage/models"
)

type postgresRepository struct {
	client *postgres.Client
	schema string
}

// NewPostgresRepository creates a repository using the default schema.
func NewPostgresRepository(client *postgres.Client) Repository {
	return &postgresRepository{client: client, schema: ""}
}

// NewPostgresRepositoryWithSchema creates a repository using a specific schema.
func NewPostgresRepositoryWithSchema(client *postgres.Client, schema string) Repository {
	return &postgresRepository{client: client, schema: schema}
}

func (r *postgresRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.client.DB()
}

func (r *postgresRepository) prefixSchema(query string) string {
	if r.schema != "" {
		return fmt.Sprintf(query, r.schema+".")
	}
	return fmt.Sprintf(query, "")
}

// Create inserts a new file record
func (r *postgresRepository) Create(ctx context.Context, file *models.File) error {
	query := `
		INSERT INTO %sfiles (id, owner_user_id, name, path, mime_type, size_bytes, provider, bucket, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	_, err := exec.ExecContext(ctx, sqlStr,
		file.ID, file.OwnerUserID, file.Name, file.Path, file.MimeType, file.SizeBytes,
		file.Provider, file.Bucket, file.Status, file.CreatedAt, file.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

// FindByID retrieves a file by its ID
func (r *postgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.File, error) {
	query := `
		SELECT id, owner_user_id, name, path, mime_type, size_bytes, provider, bucket, status, created_at, updated_at
		FROM %sfiles
		WHERE id = $1
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	var file models.File
	err := exec.QueryRowxContext(ctx, sqlStr, id).StructScan(&file)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find file: %w", err)
	}
	return &file, nil
}

// FindByOwner retrieves files owned by a user
func (r *postgresRepository) FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.File, error) {
	query := `
		SELECT id, owner_user_id, name, path, mime_type, size_bytes, provider, bucket, status, created_at, updated_at
		FROM %sfiles
		WHERE owner_user_id = $1 AND status != 'deleted'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	sqlStr := r.prefixSchema(query)
	var files []*models.File
	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &files, sqlStr, ownerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find files by owner: %w", err)
	}
	return files, nil
}

// UpdateStatus updates the status of a file
func (r *postgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE %sfiles
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	_, err := exec.ExecContext(ctx, sqlStr, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}
	return nil
}

// Delete soft deletes a file (sets status to 'deleted')
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.UpdateStatus(ctx, id, "deleted")
}

// GetTotalSize returns the total size of all files (excluding deleted) for quota checking
func (r *postgresRepository) GetTotalSize(ctx context.Context) (int64, error) {
	query := `
		SELECT COALESCE(SUM(size_bytes), 0)
		FROM %sfiles
		WHERE status != 'deleted'
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	var totalSize int64
	err := exec.QueryRowxContext(ctx, sqlStr).Scan(&totalSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get total size: %w", err)
	}
	return totalSize, nil
}

// FindOldestFiles retrieves the oldest files (for garbage collection)
// Returns files ordered by created_at ASC, limited to the specified count
func (r *postgresRepository) FindOldestFiles(ctx context.Context, limit int) ([]*models.File, error) {
	query := `
		SELECT id, owner_user_id, name, path, mime_type, size_bytes, provider, bucket, status, created_at, updated_at
		FROM %sfiles
		WHERE status != 'deleted'
		ORDER BY created_at ASC
		LIMIT $1
	`

	sqlStr := r.prefixSchema(query)
	var files []*models.File
	err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &files, sqlStr, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find oldest files: %w", err)
	}
	return files, nil
}

// HardDelete permanently deletes a file record from the database
func (r *postgresRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM %sfiles
		WHERE id = $1
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	_, err := exec.ExecContext(ctx, sqlStr, id)
	if err != nil {
		return fmt.Errorf("failed to hard delete file: %w", err)
	}
	return nil
}

// IncrementDailyUploadCount increments the daily upload count for a user and returns the new count
func (r *postgresRepository) IncrementDailyUploadCount(ctx context.Context, userID uuid.UUID, bytesUploaded int64) (int, error) {
	query := `
		INSERT INTO %sstorage_usage_daily (user_id, day, upload_count, total_bytes_uploaded)
		VALUES ($1, CURRENT_DATE, 1, $2)
		ON CONFLICT (user_id, day) 
		DO UPDATE SET 
			upload_count = storage_usage_daily.upload_count + 1,
			total_bytes_uploaded = storage_usage_daily.total_bytes_uploaded + $2
		RETURNING upload_count
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	var count int
	err := exec.QueryRowxContext(ctx, sqlStr, userID, bytesUploaded).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to increment daily upload count: %w", err)
	}
	return count, nil
}

// GetGlobalDailyUploadCount returns the total upload count for today (approximate)
func (r *postgresRepository) GetGlobalDailyUploadCount(ctx context.Context) (int, error) {
	query := `
		SELECT COALESCE(SUM(upload_count), 0)
		FROM %sstorage_usage_daily
		WHERE day = CURRENT_DATE
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	var count int
	err := exec.QueryRowxContext(ctx, sqlStr).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get global daily upload count: %w", err)
	}
	return count, nil
}

// GetSystemStats returns the current system statistics
func (r *postgresRepository) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	query := `
		SELECT total_files, total_storage_bytes, last_gc_run
		FROM %sstorage_system_stats
		WHERE id = 1
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	var stats SystemStats
	err := exec.QueryRowxContext(ctx, sqlStr).Scan(&stats.TotalFiles, &stats.TotalStorageBytes, &stats.LastGCRun)
	if err != nil {
		if err == sql.ErrNoRows {
			// Initialize if not exists
			return &SystemStats{
				TotalFiles:        0,
				TotalStorageBytes: 0,
				LastGCRun:         time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get system stats: %w", err)
	}
	return &stats, nil
}

// UpdateSystemStats updates the system statistics
func (r *postgresRepository) UpdateSystemStats(ctx context.Context, stats *SystemStats) error {
	query := `
		UPDATE %sstorage_system_stats
		SET total_files = $1, total_storage_bytes = $2, last_gc_run = $3
		WHERE id = 1
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	_, err := exec.ExecContext(ctx, sqlStr, stats.TotalFiles, stats.TotalStorageBytes, stats.LastGCRun)
	if err != nil {
		return fmt.Errorf("failed to update system stats: %w", err)
	}
	return nil
}

