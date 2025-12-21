// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/storage/models"
)

// Repository defines the interface for file storage database operations
type Repository interface {
	// Create inserts a new file record
	Create(ctx context.Context, file *models.File) error

	// FindByID retrieves a file by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.File, error)

	// FindByOwner retrieves files owned by a user
	FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.File, error)

	// UpdateStatus updates the status of a file
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error

	// Delete soft deletes a file (sets status to 'deleted')
	Delete(ctx context.Context, id uuid.UUID) error

	// GetTotalSize returns the total size of all files for quota checking
	GetTotalSize(ctx context.Context) (int64, error)

	// FindOldestFiles retrieves the oldest files (for garbage collection)
	// Returns files ordered by created_at ASC, limited to the specified count
	FindOldestFiles(ctx context.Context, limit int) ([]*models.File, error)

	// HardDelete permanently deletes a file record from the database
	HardDelete(ctx context.Context, id uuid.UUID) error

	// Usage tracking methods for quota enforcement
	// IncrementDailyUploadCount increments the daily upload count for a user and returns the new count
	IncrementDailyUploadCount(ctx context.Context, userID uuid.UUID, bytesUploaded int64) (int, error)

	// GetGlobalDailyUploadCount returns the total upload count for today (approximate)
	GetGlobalDailyUploadCount(ctx context.Context) (int, error)

	// GetSystemStats returns the current system statistics
	GetSystemStats(ctx context.Context) (*SystemStats, error)

	// UpdateSystemStats updates the system statistics
	UpdateSystemStats(ctx context.Context, stats *SystemStats) error
}

// SystemStats represents system-wide storage statistics
type SystemStats struct {
	TotalFiles        int64     `db:"total_files"`
	TotalStorageBytes int64     `db:"total_storage_bytes"`
	LastGCRun         time.Time `db:"last_gc_run"`
}

