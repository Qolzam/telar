// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package provider

import (
	"context"
	"time"
)

// BlobProvider defines the interface for blob storage providers
// This interface is provider-agnostic, allowing easy switching between
// Cloudflare R2, AWS S3, Google Cloud Storage, etc.
type BlobProvider interface {
	// GeneratePresignedUploadURL generates a URL for the frontend to upload file directly (PUT)
	// The URL expires after the specified duration
	// contentLength enforces the exact file size at the R2 level (prevents size manipulation)
	GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, contentLength int64, expiresIn time.Duration) (string, error)

	// GeneratePresignedDownloadURL generates a URL for the frontend to view/download the file (GET)
	// For public files, this might just return the CDN URL
	// The URL expires after the specified duration
	GeneratePresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)

	// Delete physically deletes the file from the storage provider
	Delete(ctx context.Context, key string) error

	// GetMetadata checks if file exists and returns its size (for validation)
	GetMetadata(ctx context.Context, key string) (size int64, err error)
}

