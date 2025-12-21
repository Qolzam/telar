// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package models

import (
	"time"

	uuid "github.com/gofrs/uuid"
)

// File represents a file record in the database
type File struct {
	ID            uuid.UUID `db:"id" json:"id"`
	OwnerUserID   uuid.UUID `db:"owner_user_id" json:"ownerUserId"`
	Name          string    `db:"name" json:"name"`
	Path          string    `db:"path" json:"path"`
	MimeType      string    `db:"mime_type" json:"mimeType"`
	SizeBytes     int64     `db:"size_bytes" json:"sizeBytes"`
	Provider      string    `db:"provider" json:"provider"`
	Bucket        string    `db:"bucket" json:"bucket"`
	Status        string    `db:"status" json:"status"` // pending, uploaded, deleted
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `db:"updated_at" json:"updatedAt"`
}

// UploadRequest represents the request payload for initializing an upload
type UploadRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	ContentType string `json:"contentType" validate:"required"`
	Size        int64  `json:"size" validate:"required,min=1"`
}

// UploadResponse represents the response after initializing an upload
type UploadResponse struct {
	UploadURL string    `json:"uploadUrl"` // Presigned URL for direct upload
	FileID    uuid.UUID `json:"fileId"`    // File ID in database
	Key       string    `json:"key"`       // Storage key (path in bucket)
}

// ConfirmUploadRequest represents the request to confirm an upload
type ConfirmUploadRequest struct {
	FileID uuid.UUID `json:"fileId" validate:"required"`
}



