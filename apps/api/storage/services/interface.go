// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/storage/models"
)

// StorageService defines the interface for storage operations
type StorageService interface {
	// InitializeUpload creates a file record and returns a presigned URL for upload
	InitializeUpload(ctx context.Context, req *models.UploadRequest, userID uuid.UUID) (*models.UploadResponse, error)

	// ConfirmUpload marks a file as uploaded after the client completes the upload
	ConfirmUpload(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error

	// EnforceQuota enforces storage quota limits by deleting oldest files if necessary
	EnforceQuota(ctx context.Context) error

	// DeleteFile deletes a file (soft delete + physical delete from storage)
	DeleteFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error

	// GetFileURL returns the public CDN URL or presigned download URL for a file
	// This is used by the frontend to display images without burning Class B operations
	GetFileURL(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (string, error)
}

