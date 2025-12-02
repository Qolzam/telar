// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/votes/errors"
	"github.com/qolzam/telar/apps/api/votes/services"
)

// VoteHandler handles all vote-related HTTP requests
type VoteHandler struct {
	voteService services.VoteService
	jwtConfig   platformconfig.JWTConfig
	hmacConfig  platformconfig.HMACConfig
}

// NewVoteHandler creates a new VoteHandler with injected dependencies
func NewVoteHandler(voteService services.VoteService, jwtConfig platformconfig.JWTConfig, hmacConfig platformconfig.HMACConfig) *VoteHandler {
	return &VoteHandler{
		voteService: voteService,
		jwtConfig:   jwtConfig,
		hmacConfig:  hmacConfig,
	}
}

// VoteRequest represents the request body for voting
type VoteRequest struct {
	PostID  string `json:"postId"`  // UUID as string
	TypeID  int    `json:"typeId"`  // 1=Up, 2=Down
}

// Vote handles vote creation/update/deletion
// Endpoint: POST /votes
// Body: {"postId": "uuid", "typeId": 1}
func (h *VoteHandler) Vote(c *fiber.Ctx) error {
	var req VoteRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	// Validate postId
	if req.PostID == "" {
		return errors.HandleValidationError(c, "postId is required")
	}

	postID, err := uuid.FromString(req.PostID)
	if err != nil {
		return errors.HandleUUIDError(c, "postId")
	}

	// Validate typeId
	if req.TypeID != 1 && req.TypeID != 2 {
		return errors.HandleValidationError(c, "typeId must be 1 (Up) or 2 (Down)")
	}

	// Get user context from JWT
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Call service
	if err := h.voteService.Vote(c.Context(), postID, user.UserID, req.TypeID); err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Return 200 OK for successful vote operation
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Vote recorded successfully",
	})
}

