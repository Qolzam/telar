// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/storage/models"
	"github.com/qolzam/telar/apps/api/storage/provider"
	storageRepository "github.com/qolzam/telar/apps/api/storage/repository"
)

const (
	// QuotaLimit is the total storage limit (10GB) for free tier
	QuotaLimit = 10 * 1024 * 1024 * 1024

	// QuotaThreshold is the threshold (9GB) at which we start enforcing quota
	QuotaThreshold = 9 * 1024 * 1024 * 1024

	// QuotaTarget is the target size (8GB) after quota enforcement
	QuotaTarget = 8 * 1024 * 1024 * 1024

	// DefaultProvider is the default storage provider
	DefaultProvider = "r2"
)

var (
	ErrFileTooLarge        = fmt.Errorf("file too large: max size exceeded. Please compress client-side")
	ErrFileNotFound        = fmt.Errorf("file not found")
	ErrQuotaExceeded       = fmt.Errorf("quota exceeded")
	ErrInvalidMimeType     = fmt.Errorf("invalid MIME type: file type not allowed")
	ErrDailyLimitReached   = fmt.Errorf("daily upload limit reached")
	ErrGlobalLimitReached  = fmt.Errorf("system storage busy, try again later")
)

type service struct {
	repo     storageRepository.Repository
	provider provider.BlobProvider
	bucket   string
	config   *platformconfig.StorageConfig
}

// NewStorageService creates a new storage service
func NewStorageService(repo storageRepository.Repository, blobProvider provider.BlobProvider, bucket string, config *platformconfig.StorageConfig) StorageService {
	return &service{
		repo:     repo,
		provider: blobProvider,
		bucket:   bucket,
		config:   config,
	}
}

// InitializeUpload creates a file record and returns a presigned URL for upload
func (s *service) InitializeUpload(ctx context.Context, req *models.UploadRequest, userID uuid.UUID) (*models.UploadResponse, error) {
	// 1. Config Check: File Size (hard cap, requires client-side compression)
	maxSize := int64(s.config.MaxFileSizeMB) * 1024 * 1024
	if req.Size > maxSize {
		return nil, fmt.Errorf("%w: max %d MB. Please compress client-side", ErrFileTooLarge, s.config.MaxFileSizeMB)
	}

	// 2. MIME Type Validation
	if !s.isMimeTypeAllowed(req.ContentType) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMimeType, req.ContentType)
	}

	// 3. Quota Check: User Daily Limit (Postgres, NOT R2)
	userCount, err := s.repo.IncrementDailyUploadCount(ctx, userID, req.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to check user quota: %w", err)
	}
	if userCount > s.config.UserDailyUploadLimit {
		return nil, ErrDailyLimitReached
	}

	// 4. Quota Check: Global Safety Net
	globalCount, err := s.repo.GetGlobalDailyUploadCount(ctx)
	if err != nil {
		log.Error("Failed to get global upload count: %v", err)
		// Continue anyway - don't block on stats failure
	} else if globalCount > s.config.GlobalDailyUploadLimit {
		return nil, ErrGlobalLimitReached
	}

	// 5. Generate File ID and Path: users/{userID}/{uuid}.{ext}
	fileID := uuid.Must(uuid.NewV4())
	ext := filepath.Ext(req.Name)
	key := fmt.Sprintf("users/%s/%s%s", userID.String(), fileID.String(), ext)

	// 6. Save Metadata to DB as 'pending'
	now := time.Now()
	file := &models.File{
		ID:          fileID,
		OwnerUserID: userID,
		Name:        req.Name,
		Path:        key,
		MimeType:    req.ContentType,
		SizeBytes:   req.Size,
		Provider:    DefaultProvider,
		Bucket:      s.bucket,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, file); err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	// 7. Generate R2 Presigned URL with STRICT Content-Length constraints
	uploadURL, err := s.provider.GeneratePresignedUploadURL(ctx, key, req.ContentType, req.Size, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return &models.UploadResponse{
		UploadURL: uploadURL,
		FileID:    fileID,
		Key:       key,
	}, nil
}

// isMimeTypeAllowed checks if the MIME type is in the allowed list
func (s *service) isMimeTypeAllowed(mimeType string) bool {
	if len(s.config.AllowedMimeTypes) == 0 {
		return true // If no restrictions, allow all
	}
	for _, allowed := range s.config.AllowedMimeTypes {
		if strings.EqualFold(mimeType, allowed) {
			return true
		}
	}
	return false
}

// ConfirmUpload marks a file as uploaded after the client completes the upload
func (s *service) ConfirmUpload(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	// 1. Verify file exists and belongs to user
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return ErrFileNotFound
	}

	if file.OwnerUserID != userID {
		return fmt.Errorf("unauthorized: file does not belong to user")
	}

	// 2. Verify file is in pending status
	if file.Status != "pending" {
		return fmt.Errorf("file is not in pending status")
	}

	// 3. Optionally verify file exists in storage (metadata check)
	// For now, we trust the client confirmation
	// In production, you might want to verify:
	// _, err = s.provider.GetMetadata(ctx, file.Path)
	// if err != nil {
	//     return fmt.Errorf("file not found in storage: %w", err)
	// }

	// 4. Update status to 'uploaded'
	if err := s.repo.UpdateStatus(ctx, fileID, "uploaded"); err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}

	return nil
}

// EnforceQuota enforces storage quota limits by deleting oldest files if necessary
func (s *service) EnforceQuota(ctx context.Context) error {
	// 1. Check Total Usage
	totalSize, err := s.repo.GetTotalSize(ctx)
	if err != nil {
		return fmt.Errorf("failed to get total size: %w", err)
	}

	if totalSize < QuotaThreshold {
		return nil // < 9GB is fine
	}

	log.Info("Quota enforcement triggered: currentSize=%d, threshold=%d", totalSize, QuotaThreshold)

	// 2. Panic Mode: Delete oldest files until we are safe
	bytesToDelete := totalSize - QuotaTarget

	// Fetch oldest files (fetch in batches to avoid memory issues)
	batchSize := 100
	totalDeleted := int64(0)

	for totalDeleted < bytesToDelete {
		oldFiles, err := s.repo.FindOldestFiles(ctx, batchSize)
		if err != nil {
			return fmt.Errorf("failed to find oldest files: %w", err)
		}

		if len(oldFiles) == 0 {
			// No more files to delete
			break
		}

		// Delete files until we reach the target
		for _, file := range oldFiles {
			if totalDeleted >= bytesToDelete {
				break
			}

			// Delete from storage provider
			if err := s.provider.Delete(ctx, file.Path); err != nil {
				log.Error("Failed to delete file from storage: error=%v, path=%s", err, file.Path)
				// Continue with next file even if delete fails
			}

			// Delete from DB
			if err := s.repo.HardDelete(ctx, file.ID); err != nil {
				log.Error("Failed to delete file from DB: error=%v, fileID=%s", err, file.ID)
				// Continue with next file
			} else {
				totalDeleted += file.SizeBytes
			}
		}

		// If we got fewer files than requested, we've reached the end
		if len(oldFiles) < batchSize {
			break
		}
	}

	log.Info("Quota enforcement completed: bytesDeleted=%d", totalDeleted)
	return nil
}

// DeleteFile deletes a file (soft delete + physical delete from storage)
func (s *service) DeleteFile(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	// 1. Verify file exists and belongs to user
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return ErrFileNotFound
	}

	if file.OwnerUserID != userID {
		return fmt.Errorf("unauthorized: file does not belong to user")
	}

	// 2. Delete from storage provider
	if err := s.provider.Delete(ctx, file.Path); err != nil {
		log.Error("Failed to delete file from storage: error=%v, path=%s", err, file.Path)
		// Continue with DB delete even if storage delete fails
	}

	// 3. Soft delete in DB
	if err := s.repo.Delete(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	return nil
}

// GetFileURL returns the public CDN URL or presigned download URL for a file
func (s *service) GetFileURL(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (string, error) {
	// 1. Verify file exists and belongs to user
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return "", ErrFileNotFound
	}

	if file.OwnerUserID != userID {
		return "", fmt.Errorf("unauthorized: file does not belong to user")
	}

	// 2. Verify file is uploaded (not pending or deleted)
	if file.Status != "uploaded" {
		return "", fmt.Errorf("file is not available (status: %s)", file.Status)
	}

	// 3. Generate public CDN URL or presigned download URL
	// The provider will return CDN URL if PublicURL is configured, otherwise presigned URL
	url, err := s.provider.GeneratePresignedDownloadURL(ctx, file.Path, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate file URL: %w", err)
	}

	return url, nil
}

