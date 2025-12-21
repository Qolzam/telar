package handlers

import (
	"fmt"
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

	if result == nil {
		return errors.HandleServiceError(c, fmt.Errorf("comment creation returned nil"))
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

	// Update 
	updatedComment, err := h.commentService.UpdateComment(c.Context(), req.ObjectId, &req, &user)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	if updatedComment == nil {
		return errors.HandleServiceError(c, fmt.Errorf("comment not found after update"))
	}

	response := h.convertCommentToResponse(updatedComment)
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

	// Parse pagination parameters (prefer cursor-based pagination for performance)
	filter := &models.CommentQueryFilter{
		Limit: 10,
	}

	// Cursor-based pagination (required for 1M+ users)
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		filter.Cursor = cursorStr
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
	filter.PostId = &postID

	ctx := c.Context()


	// Use cursor-based pagination (always)
	comments, err := h.commentService.QueryCommentsWithCursor(ctx, filter)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}


	// Enrich with reply counts and user votes (BULK OPERATIONS - avoids N+1 queries)
	commentIDs := make([]uuid.UUID, 0, len(comments.Comments))
	for i := range comments.Comments {
		id, _ := uuid.FromString(comments.Comments[i].ObjectId)
		if id != [16]byte{} {
			commentIDs = append(commentIDs, id)
		}
	}

	// Bulk-load reply counts (single query instead of N queries)
	if len(commentIDs) > 0 {

		replyCountMap, err := h.commentService.GetReplyCountsBulk(ctx, commentIDs)
		if err == nil {

			for i := range comments.Comments {
				id, _ := uuid.FromString(comments.Comments[i].ObjectId)
				if id != [16]byte{} {
					if count, exists := replyCountMap[id]; exists {
						comments.Comments[i].ReplyCount = int(count)
					}
				}
			}
		}
	}

	// Bulk-load user votes if user is authenticated (single query using ANY operator)
	if user, ok := c.Locals(types.UserCtxName).(types.UserContext); ok && len(commentIDs) > 0 {

		voteMap, err := h.commentService.GetUserVotesForComments(ctx, commentIDs, user.UserID)
		if err == nil {

			// Set IsLiked for each comment
			for i := range comments.Comments {
				id, _ := uuid.FromString(comments.Comments[i].ObjectId)
				if id != [16]byte{} {
					comments.Comments[i].IsLiked = voteMap[id]
				}
			}
		}
	}

	// Return full CommentsListResponse object (includes nextCursor, hasNext for cursor pagination)
	return c.Status(http.StatusOK).JSON(comments)
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

	// Enrich with reply count (always set, even if 0)
	replyCountMap, err := h.commentService.GetReplyCountsBulk(c.Context(), []uuid.UUID{commentID})
	if err == nil {
		if count, exists := replyCountMap[commentID]; exists {
			response.ReplyCount = int(count)
		} else {
			// If not in map, default to 0 (no replies)
			response.ReplyCount = 0
		}
	} else {
		// If error occurs, default to 0
		response.ReplyCount = 0
	}
	// Always set replyCount (removed omitempty from model)

	// Enrich with user vote status if user is authenticated
	if user, ok := c.Locals(types.UserCtxName).(types.UserContext); ok {
		voteMap, err := h.commentService.GetUserVotesForComments(c.Context(), []uuid.UUID{commentID}, user.UserID)
		if err == nil {
			response.IsLiked = voteMap[commentID]
		} else {
			// If error occurs, default to false (user hasn't liked)
			response.IsLiked = false
		}
	}

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
		ReplyToUserId: func() *string {
			if comment.ReplyToUserId == nil {
				return nil
			}
			s := comment.ReplyToUserId.String()
			return &s
		}(),
		ReplyToDisplayName: comment.ReplyToDisplayName,
		Text:             comment.Text,
		Deleted:          comment.Deleted,
		DeletedDate:      comment.DeletedDate,
		CreatedDate:      comment.CreatedDate,
		LastUpdated:      comment.LastUpdated,
		ReplyCount:       0, // Default to 0 for new comments - can be enriched later if needed
		IsLiked:          false, // Default to false - should be enriched by caller if needed
	}
	return response
}

// ToggleLike handles toggling a user's like on a comment
func (h *CommentHandler) ToggleLike(c *fiber.Ctx) error {
	commentIDStr := c.Params("commentId")
	commentID, err := uuid.FromString(commentIDStr)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid comment ID")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Toggle like and get the result (optimized: returns comment, score, and isLiked without re-fetching)
	// The comment is returned from the transaction to avoid a second database query
	comment, newScore, isLiked, err := h.commentService.ToggleLike(c.Context(), commentID, user.UserID)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Construct response from the comment returned by ToggleLike (no re-fetch needed)
	response := h.convertCommentToResponse(comment)
	response.Score = newScore
	response.IsLiked = isLiked

	return c.Status(http.StatusOK).JSON(response)
}

// GetReplies handles fetching replies for a specific comment with cursor-based pagination
func (h *CommentHandler) GetReplies(c *fiber.Ctx) error {
	parentStr := c.Params("commentId")
	parentID, err := uuid.FromString(parentStr)
	if err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid comment ID")
	}

	// Parse pagination parameters (cursor-based pagination only for 1M+ users performance)
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	cursor := c.Query("cursor")

	// Use cursor-based pagination (always)
	result, err := h.commentService.QueryRepliesWithCursor(c.Context(), parentID, cursor, limit)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Enrich with user votes if user is authenticated (BULK OPERATIONS - avoids N+1 queries)
	commentIDs := make([]uuid.UUID, 0, len(result.Comments))
	for i := range result.Comments {
		id, _ := uuid.FromString(result.Comments[i].ObjectId)
		if id != [16]byte{} {
			commentIDs = append(commentIDs, id)
		}
	}

	// Bulk-load user votes if user is authenticated (single query using ANY operator)
	if user, ok := c.Locals(types.UserCtxName).(types.UserContext); ok && len(commentIDs) > 0 {
		voteMap, err := h.commentService.GetUserVotesForComments(c.Context(), commentIDs, user.UserID)
		if err == nil {
			// Set IsLiked for each reply
			for i := range result.Comments {
				id, _ := uuid.FromString(result.Comments[i].ObjectId)
				if id != [16]byte{} {
					result.Comments[i].IsLiked = voteMap[id]
				}
			}
		}
	}

	// Return full CommentsListResponse object (includes nextCursor, hasNext for cursor pagination)
	return c.Status(http.StatusOK).JSON(result)
}
