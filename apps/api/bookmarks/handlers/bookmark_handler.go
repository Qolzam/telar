package handlers

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/bookmarks/errors"
	"github.com/qolzam/telar/apps/api/bookmarks/services"
	"github.com/qolzam/telar/apps/api/internal/types"
)

type BookmarkHandler struct {
	service services.Service
}

func NewBookmarkHandler(service services.Service) *BookmarkHandler {
	return &BookmarkHandler{service: service}
}

// Toggle toggles bookmark state for a post.
// Endpoint: POST /bookmarks/:postId/toggle
func (h *BookmarkHandler) Toggle(c *fiber.Ctx) error {
	postIDParam := c.Params("postId")
	if postIDParam == "" {
		return errors.HandleValidationError(c, "postId is required")
	}

	postID, err := uuid.FromString(postIDParam)
	if err != nil {
		return errors.HandleUUIDError(c, "postId")
	}

	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "invalid user context")
	}

	bookmarked, err := h.service.ToggleBookmark(c.Context(), user.UserID, postID)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"isBookmarked": bookmarked})
}

// List returns bookmarked posts for the current user with cursor pagination.
// Endpoint: GET /bookmarks?cursor=...&limit=...
func (h *BookmarkHandler) List(c *fiber.Ctx) error {
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "invalid user context")
	}

	cursor := c.Query("cursor")
	limit := c.QueryInt("limit", 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	ctxWithUser := context.WithValue(c.Context(), types.UserCtxName, user)
	resp, err := h.service.ListBookmarks(ctxWithUser, user.UserID, cursor, limit)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusOK).JSON(resp)
}
