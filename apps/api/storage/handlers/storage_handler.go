// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/types"
	storageErrors "github.com/qolzam/telar/apps/api/storage/errors"
	"github.com/qolzam/telar/apps/api/storage/models"
	"github.com/qolzam/telar/apps/api/storage/services"
)

// StorageHandler handles all storage-related HTTP requests
type StorageHandler struct {
	storageService services.StorageService
}

// NewStorageHandler creates a new StorageHandler with injected dependencies
func NewStorageHandler(storageService services.StorageService) *StorageHandler {
	return &StorageHandler{
		storageService: storageService,
	}
}

// InitializeUpload handles upload initialization
// POST /storage/upload/init
func (h *StorageHandler) InitializeUpload(c *fiber.Ctx) error {
	var req models.UploadRequest
	if err := c.BodyParser(&req); err != nil {
		return storageErrors.HandleInvalidRequestError(c, "Invalid request body")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return storageErrors.HandleUserContextError(c, "Invalid user context")
	}

	// Initialize upload
	result, err := h.storageService.InitializeUpload(c.Context(), &req, user.UserID)
	if err != nil {
		return storageErrors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(result)
}

// ConfirmUpload handles upload confirmation
// POST /storage/upload/confirm
func (h *StorageHandler) ConfirmUpload(c *fiber.Ctx) error {
	var req models.ConfirmUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return storageErrors.HandleInvalidRequestError(c, "Invalid request body")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return storageErrors.HandleUserContextError(c, "Invalid user context")
	}

	// Confirm upload
	if err := h.storageService.ConfirmUpload(c.Context(), req.FileID, user.UserID); err != nil {
		return storageErrors.HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"message": "Upload confirmed successfully"})
}

// DeleteFile handles file deletion
// DELETE /storage/files/:fileId
func (h *StorageHandler) DeleteFile(c *fiber.Ctx) error {
	fileIDStr := c.Params("fileId")
	fileID, err := uuid.FromString(fileIDStr)
	if err != nil {
		return storageErrors.HandleInvalidRequestError(c, "Invalid file ID")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return storageErrors.HandleUserContextError(c, "Invalid user context")
	}

	// Delete file
	if err := h.storageService.DeleteFile(c.Context(), fileID, user.UserID); err != nil {
		return storageErrors.HandleServiceError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}

// GetFileURL handles retrieving the public URL for a file
// GET /storage/files/:fileId/url
func (h *StorageHandler) GetFileURL(c *fiber.Ctx) error {
	fileIDStr := c.Params("fileId")
	fileID, err := uuid.FromString(fileIDStr)
	if err != nil {
		return storageErrors.HandleInvalidRequestError(c, "Invalid file ID")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return storageErrors.HandleUserContextError(c, "Invalid user context")
	}

	// Get file URL
	url, err := h.storageService.GetFileURL(c.Context(), fileID, user.UserID)
	if err != nil {
		return storageErrors.HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"url": url,
	})
}

