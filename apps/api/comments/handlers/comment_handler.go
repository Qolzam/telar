package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/errors"
	"github.com/qolzam/telar/apps/api/comments/models"
	"github.com/qolzam/telar/apps/api/comments/services"
	"github.com/qolzam/telar/apps/api/comments/validation"
	"github.com/qolzam/telar/apps/api/internal/types"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// CommentHandler handles all comment-related HTTP requests
type CommentHandler struct {
	commentService services.CommentService
	jwtConfig      platformconfig.JWTConfig
	hmacConfig     platformconfig.HMACConfig
}

// NewCommentHandler creates a new CommentHandler with injected dependencies
func NewCommentHandler(commentService services.CommentService, jwtConfig platformconfig.JWTConfig, hmacConfig platformconfig.HMACConfig) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
		jwtConfig:      jwtConfig,
		hmacConfig:     hmacConfig,
	}
}

// CreateComment handles comment creation
func (h *CommentHandler) CreateComment(c *fiber.Ctx) error {
	var req models.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	// Validate request
	if err := validation.ValidateCreateCommentRequest(&req); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Create comment
	result, err := h.commentService.CreateComment(c.Context(), &req, &user)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}


	response := h.convertCommentToResponse(result)
	return c.Status(http.StatusCreated).JSON(response)
}

// UpdateComment handles comment update
func (h *CommentHandler) UpdateComment(c *fiber.Ctx) error {
	var req models.UpdateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	// Validate request
	if err := validation.ValidateUpdateCommentRequest(&req); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Update comment
	err := h.commentService.UpdateComment(c.Context(), req.ObjectId, &req, &user)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Get updated comment
	comment, err := h.commentService.GetComment(c.Context(), req.ObjectId)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Convert to response format
	response := h.convertCommentToResponse(comment)
	return c.Status(http.StatusOK).JSON(response)
}

// GetCommentsByPost handles retrieving comments for a specific post
func (h *CommentHandler) GetCommentsByPost(c *fiber.Ctx) error {
	postIDStr := c.Query("postId")
	if postIDStr == "" {
		return errors.HandleInvalidRequestError(c, "postId query parameter is required")
	}

	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid post ID")
	}

	// Parse pagination parameters
	filter := &models.CommentQueryFilter{
		Page:  1,
		Limit: 10,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	// Validate filter
	if err := validation.ValidateCommentQueryFilter(filter); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	// Only root comments for initial load
	filter.RootOnly = true
	comments, err := h.commentService.GetCommentsByPost(c.Context(), postID, filter)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Enrich with reply counts
	for i := range comments.Comments {
		id, _ := uuid.FromString(comments.Comments[i].ObjectId)
		if id != [16]byte{} {
			if cnt, err := h.commentService.GetReplyCount(c.Context(), id); err == nil {
				comments.Comments[i].ReplyCount = int(cnt)
			}
		}
	}
	return c.Status(http.StatusOK).JSON(comments.Comments)
}

// GetComment handles retrieving a specific comment
func (h *CommentHandler) GetComment(c *fiber.Ctx) error {
	commentIDStr := c.Params("commentId")
	commentID, err := uuid.FromString(commentIDStr)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid comment ID")
	}

	comment, err := h.commentService.GetComment(c.Context(), commentID)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Convert to response format
	response := h.convertCommentToResponse(comment)
	return c.Status(http.StatusOK).JSON(response)
}

// DeleteComment handles comment deletion
func (h *CommentHandler) DeleteComment(c *fiber.Ctx) error {
	commentIDStr := c.Params("commentId")
	commentID, err := uuid.FromString(commentIDStr)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid comment ID")
	}

	postIDStr := c.Params("postId")
	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid post ID")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Delete comment
	err = h.commentService.DeleteComment(c.Context(), commentID, postID, &user)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}

// DeleteCommentsByPost handles deleting all comments for a specific post
func (h *CommentHandler) DeleteCommentsByPost(c *fiber.Ctx) error {
	postIDStr := c.Params("postId")
	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid post ID")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Delete all comments for the post
	err = h.commentService.DeleteCommentsByPost(c.Context(), postID, &user)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}

// IncrementScore handles incrementing the score (like) for a comment
func (h *CommentHandler) IncrementScore(c *fiber.Ctx) error {
	var req struct {
		CommentID uuid.UUID `json:"commentId"`
		Delta     int       `json:"delta"`
	}

	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "invalid request body")
	}
	if req.CommentID == uuid.Nil {
		return errors.HandleInvalidRequestError(c, "commentId is required")
	}
	if req.Delta == 0 {
		req.Delta = 1
	}

	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "invalid current user")
	}

	if err := h.commentService.IncrementScore(c.Context(), req.CommentID, req.Delta, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "score incremented"})
}

// UpdateCommentProfile handles updating comment profile information
func (h *CommentHandler) UpdateCommentProfile(c *fiber.Ctx) error {
	var req models.UpdateCommentProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	// Validate request
	if err := validation.ValidateUpdateCommentProfileRequest(&req); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Verify that the user is updating their own profile or is an admin
	if user.UserID != req.OwnerUserId {
		return errors.HandleForbiddenError(c, "Access denied")
	}

	// Update comment profile
	var displayName, avatar string
	if req.OwnerDisplayName != nil {
		displayName = *req.OwnerDisplayName
	}
	if req.OwnerAvatar != nil {
		avatar = *req.OwnerAvatar
	}

	err := h.commentService.UpdateCommentProfile(c.Context(), req.OwnerUserId, displayName, avatar)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Comment profile updated successfully",
	})
}

// convertCommentToResponse converts a Comment to CommentResponse
func (h *CommentHandler) convertCommentToResponse(comment *models.Comment) models.CommentResponse {
	response := models.CommentResponse{
		ObjectId:         comment.ObjectId.String(),
		Score:            comment.Score,
		OwnerUserId:      comment.OwnerUserId.String(),
		OwnerDisplayName: comment.OwnerDisplayName,
		OwnerAvatar:      comment.OwnerAvatar,
		PostId:           comment.PostId.String(),
		ParentCommentId: func() *string {
			if comment.ParentCommentId == nil {
				return nil
			}
			s := comment.ParentCommentId.String()
			return &s
		}(),
		Text:             comment.Text,
		Deleted:          comment.Deleted,
		DeletedDate:      comment.DeletedDate,
		CreatedDate:      comment.CreatedDate,
		LastUpdated:      comment.LastUpdated,
		ReplyCount:       0, // Default to 0 for new comments - can be enriched later if needed
	}
	return response
}

// GetReplies handles fetching replies for a specific comment with pagination
func (h *CommentHandler) GetReplies(c *fiber.Ctx) error {
	parentStr := c.Params("commentId")
	parentID, err := uuid.FromString(parentStr)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid comment ID")
	}

	filter := &models.CommentQueryFilter{
		Page:             1,
		Limit:            10,
		ParentCommentId:  &parentID,
		IncludeDeleted:   false,
	}
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}
	if err := validation.ValidateCommentQueryFilter(filter); err != nil {
	 return errors.HandleValidationError(c, err.Error())
	}
	result, err := h.commentService.QueryComments(c.Context(), filter)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.Status(http.StatusOK).JSON(result.Comments)
}
