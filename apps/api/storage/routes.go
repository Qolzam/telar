// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package storage

import (
	"github.com/gofiber/fiber/v2"
	constraints "github.com/qolzam/telar/apps/api/internal/middleware/constraints"
	dualauth "github.com/qolzam/telar/apps/api/internal/middleware/dualauth"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/storage/handlers"
)

// StorageHandlers holds all the handlers this router needs.
type StorageHandlers struct {
	StorageHandler *handlers.StorageHandler
}

// RouterConfig holds the configuration needed for the router's middleware.
type RouterConfig struct {
	PayloadSecret string
	PublicKey     string
}

// createDualAuthMiddleware creates dual authentication middleware using the shared helper
func createDualAuthMiddleware(cfg *RouterConfig) fiber.Handler {
	return dualauth.CreateDualAuthMiddleware(dualauth.Config{
		PayloadSecret: cfg.PayloadSecret,
		PublicKey:     cfg.PublicKey,
	})
}

// RegisterRoutes is the single entry point for setting up storage routes.
func RegisterRoutes(app *fiber.App, handlers *StorageHandlers, cfg *platformconfig.Config) {
	if handlers == nil || handlers.StorageHandler == nil {
		panic("StorageHandlers is required")
	}

	routerCfg := &RouterConfig{
		PayloadSecret: cfg.HMAC.Secret,
		PublicKey:     cfg.JWT.PublicKey,
	}

	dualAuthMiddleware := createDualAuthMiddleware(routerCfg)

	storageRoutes := app.Group("/storage")
	userGroup := storageRoutes.Group("", dualAuthMiddleware)

	// Initialize upload (requires authentication)
	// Note: No UUID constraint needed for init endpoint (file ID is generated server-side)
	userGroup.Post("/upload/init", handlers.StorageHandler.InitializeUpload)

	// Confirm upload (requires authentication)
	// Note: fileId is in request body, not URL param, so no UUID constraint needed
	userGroup.Post("/upload/confirm", handlers.StorageHandler.ConfirmUpload)

	// Get file URL (requires authentication)
	userGroup.Get("/files/:fileId/url",
		constraints.RequireUUID("fileId"),
		handlers.StorageHandler.GetFileURL,
	)

	// Delete file (requires authentication)
	userGroup.Delete("/files/:fileId",
		constraints.RequireUUID("fileId"),
		handlers.StorageHandler.DeleteFile,
	)
}

