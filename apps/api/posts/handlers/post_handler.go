package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sync"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts/errors"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/posts/services"
	"github.com/qolzam/telar/apps/api/posts/validation"
)

// PostHandler handles all post-related HTTP requests
type PostHandler struct {
	postService services.PostService
	jwtConfig   platformconfig.JWTConfig
	hmacConfig  platformconfig.HMACConfig
	TestWg      *sync.WaitGroup
}

// NewPostHandler creates a new PostHandler with injected dependencies
func NewPostHandler(postService services.PostService, jwtConfig platformconfig.JWTConfig, hmacConfig platformconfig.HMACConfig) *PostHandler {
	return &PostHandler{
		postService: postService,
		jwtConfig:   jwtConfig,
		hmacConfig:  hmacConfig,
	}
}

// CreatePost handles post creation
func (h *PostHandler) CreatePost(c *fiber.Ctx) error {
	var req models.CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	// Validate request
	if err := validation.ValidateCreatePostRequest(&req); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Create post
	result, err := h.postService.CreatePost(c.Context(), &req, &user)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Return 201 Created for successful resource creation (REST API best practice)
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"objectId": result.ObjectId.String(),
	})
}

// GetPost handles retrieving a single post
// Note: UUID validation is handled by constraints.RequireUUID middleware
// If we reach this handler, the UUID is guaranteed to be valid
// However, we add defensive validation to prevent panic if middleware fails
func (h *PostHandler) GetPost(c *fiber.Ctx) error {
	postIDStr := c.Params("postId")
	// Defensive validation (middleware should prevent this, but safety first)
	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not Found"})
	}
	// Constraint middleware already validated UUID format, but we validate again defensively

	post, err := h.postService.GetPost(c.Context(), postID)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Increment view count asynchronously
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if ok {
		if h.TestWg != nil {
			h.TestWg.Add(1)
		}
		go func(userCtx types.UserContext, pid uuid.UUID) {
			defer func() {
				if h.TestWg != nil {
					h.TestWg.Done()
				}
			}()
			h.postService.IncrementViewCount(context.Background(), pid, &userCtx)
		}(user, post.ObjectId)
	}

	// Convert to response format (uses lazy population for commentCounter)
	response := h.postService.ConvertPostToResponse(c.Context(), post)
	return c.JSON(response)
}

// GetPostByURLKey handles retrieving a post by URL key
func (h *PostHandler) GetPostByURLKey(c *fiber.Ctx) error {
	urlKey := c.Params("urlkey")
	if urlKey == "" {
		return errors.HandleInvalidRequestError(c, "URL key is required")
	}

	post, err := h.postService.GetPostByURLKey(c.Context(), urlKey)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Increment view count asynchronously
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if ok {
		if h.TestWg != nil {
			h.TestWg.Add(1)
		}
		go func(userCtx types.UserContext, pid uuid.UUID) {
			defer func() {
				if h.TestWg != nil {
					h.TestWg.Done()
				}
			}()
			h.postService.IncrementViewCount(context.Background(), pid, &userCtx)
		}(user, post.ObjectId)
	}

	// Convert to response format (uses lazy population for commentCounter)
	response := h.postService.ConvertPostToResponse(c.Context(), post)
	return c.JSON(response)
}

// QueryPosts handles post querying with filters (now using cursor-based pagination)
func (h *PostHandler) QueryPosts(c *fiber.Ctx) error {
	// Parse query parameters for cursor-based pagination
	filter := &models.PostQueryFilter{
		SortField:     "createdDate", // Default sort field
		SortDirection: "desc",        // Default sort direction
		Limit:         20,            // Default limit (matches spec default)
	}

	// Parse cursor parameters
	if cursor := c.Query("cursor"); cursor != "" {
		filter.Cursor = cursor
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	// Parse owner filter (can be multiple)
	if ownerStr := c.Query("owner"); ownerStr != "" {
		// Handle multiple owner IDs (comma-separated)
		ownerIDs := strings.Split(ownerStr, ",")
		for _, ownerIDStr := range ownerIDs {
			if ownerID, err := uuid.FromString(strings.TrimSpace(ownerIDStr)); err == nil {
				if filter.OwnerUserId == nil {
					filter.OwnerUserId = &ownerID
				}
				// Note: Current model supports single owner, but spec allows multiple
				// This could be enhanced in the future
			}
		}
	}

	// Parse post type filter
	if postTypeStr := c.Query("postType"); postTypeStr != "" {
		if postType, err := strconv.Atoi(postTypeStr); err == nil {
			filter.PostTypeId = &postType
		}
	}

	// Parse search term
	if search := c.Query("search"); search != "" {
		filter.Search = search
	}

	// Parse sort parameters
	if sortBy := c.Query("sortBy"); sortBy != "" {
		filter.SortBy = sortBy
		// Map sortBy to SortField for consistency
		switch sortBy {
		case "createdDate":
			filter.SortField = "createdDate"
		case "score":
			filter.SortField = "score"
		case "viewCount":
			filter.SortField = "viewCount"
		case "lastUpdated":
			filter.SortField = "lastUpdated"
		}
	}

	if sortOrder := c.Query("sortOrder"); sortOrder != "" {
		filter.SortOrder = sortOrder
		// Map sortOrder to SortDirection for consistency
		switch sortOrder {
		case "asc":
			filter.SortDirection = "asc"
		case "desc":
			filter.SortDirection = "desc"
		}
	}

	// Parse tags (comma-separated)
	if tagsStr := c.Query("tags"); tagsStr != "" {
		tags := strings.Split(tagsStr, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		filter.Tags = tags
	}

	// Validate filter
	if err := validation.ValidatePostQueryFilter(filter); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	// Use cursor-based pagination instead of offset-based
	result, err := h.postService.QueryPostsWithCursor(c.Context(), filter)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(result)
}

// QueryPostsWithCursor handles post querying with cursor-based pagination
func (h *PostHandler) QueryPostsWithCursor(c *fiber.Ctx) error {
	// Parse query parameters
	filter := &models.PostQueryFilter{
		SortField:     "createdDate", // Default sort field
		SortDirection: "desc",        // Default sort direction
		Limit:         20,            // Default limit
	}

	// Parse cursor parameters
	if cursor := c.Query("cursor"); cursor != "" {
		filter.Cursor = cursor
	}
	if afterCursor := c.Query("after"); afterCursor != "" {
		filter.AfterCursor = afterCursor
	}
	if beforeCursor := c.Query("before"); beforeCursor != "" {
		filter.BeforeCursor = beforeCursor
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	// Parse sort parameters
	if sortField := c.Query("sortField"); sortField != "" {
		filter.SortField = models.ParseSortField(sortField)
	}
	if sortDirection := c.Query("sortDirection"); sortDirection != "" {
		filter.SortDirection = models.ParseSortDirection(sortDirection)
	}

	// Parse other filters
	if ownerStr := c.Query("owner"); ownerStr != "" {
		if ownerID, err := uuid.FromString(ownerStr); err == nil {
			filter.OwnerUserId = &ownerID
		}
	}

	if postTypeStr := c.Query("type"); postTypeStr != "" {
		if postType, err := strconv.Atoi(postTypeStr); err == nil {
			filter.PostTypeId = &postType
		}
	}

	// Parse tags
	if tagsStr := c.Query("tags"); tagsStr != "" {
		filter.Tags = []string{tagsStr}
	}

	// Parse created after filter
	if createdAfterStr := c.Query("createdAfter"); createdAfterStr != "" {
		if timestamp, err := strconv.ParseInt(createdAfterStr, 10, 64); err == nil {
			createdAfter := time.Unix(timestamp, 0)
			filter.CreatedAfter = &createdAfter
		}
	}

	// Validate filter
	if err := validation.ValidatePostQueryFilter(filter); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	// Query posts with cursor
	result, err := h.postService.QueryPostsWithCursor(c.Context(), filter)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(result)
}

// SearchPostsWithCursor handles post searching with cursor-based pagination
func (h *PostHandler) SearchPostsWithCursor(c *fiber.Ctx) error {
	// Get search query
	searchTerm := c.Query("q")
	if searchTerm == "" {
		return errors.HandleInvalidRequestError(c, "Search query 'q' is required")
	}

	// Parse query parameters
	filter := &models.PostQueryFilter{
		SortField:     "createdDate", // Default sort field
		SortDirection: "desc",        // Default sort direction
		Limit:         20,            // Default limit
	}

	// Parse cursor parameters
	if cursor := c.Query("cursor"); cursor != "" {
		filter.Cursor = cursor
	}
	if afterCursor := c.Query("after"); afterCursor != "" {
		filter.AfterCursor = afterCursor
	}
	if beforeCursor := c.Query("before"); beforeCursor != "" {
		filter.BeforeCursor = beforeCursor
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	// Parse sort parameters
	if sortField := c.Query("sortField"); sortField != "" {
		filter.SortField = models.ParseSortField(sortField)
	}
	if sortDirection := c.Query("sortDirection"); sortDirection != "" {
		filter.SortDirection = models.ParseSortDirection(sortDirection)
	}

	// Parse other filters
	if ownerStr := c.Query("owner"); ownerStr != "" {
		if ownerID, err := uuid.FromString(ownerStr); err == nil {
			filter.OwnerUserId = &ownerID
		}
	}

	if postTypeStr := c.Query("type"); postTypeStr != "" {
		if postType, err := strconv.Atoi(postTypeStr); err == nil {
			filter.PostTypeId = &postType
		}
	}

	// Parse tags
	if tagsStr := c.Query("tags"); tagsStr != "" {
		filter.Tags = []string{tagsStr}
	}

	// Parse created after filter
	if createdAfterStr := c.Query("createdAfter"); createdAfterStr != "" {
		if timestamp, err := strconv.ParseInt(createdAfterStr, 10, 64); err == nil {
			createdAfter := time.Unix(timestamp, 0)
			filter.CreatedAfter = &createdAfter
		}
	}

	// Validate filter
	if err := validation.ValidatePostQueryFilter(filter); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	// Search posts with cursor
	result, err := h.postService.SearchPostsWithCursor(c.Context(), searchTerm, filter)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(result)
}

// GetCursorInfo handles getting cursor information for a specific post
// Note: UUID validation is handled by constraints.RequireUUID middleware
// If we reach this handler, the UUID is guaranteed to be valid
func (h *PostHandler) GetCursorInfo(c *fiber.Ctx) error {
	postIDStr := c.Params("postId")
	if postIDStr == "" {
		return errors.HandleInvalidRequestError(c, "post ID parameter is required")
	}
	// Defensive validation (middleware should prevent this, but safety first)
	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not Found"})
	}

	// Get sort parameters (defaults match spec)
	sortBy := c.Query("sortBy")
	if sortBy == "" {
		sortBy = "createdDate"
	}
	sortOrder := c.Query("sortOrder")
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// Validate sort parameters
	validSortFields := map[string]bool{
		"createdDate": true,
		"score":       true,
		"viewCount":   true,
		"lastUpdated": true,
	}
	if !validSortFields[sortBy] {
		return errors.HandleInvalidRequestError(c, "Invalid sortBy field")
	}

	validSortOrders := map[string]bool{
		"asc":  true,
		"desc": true,
	}
	if !validSortOrders[sortOrder] {
		return errors.HandleInvalidRequestError(c, "Invalid sortOrder")
	}

	// Get cursor information for the specific post
	cursorInfo, err := h.postService.GetCursorInfo(c.Context(), postID, sortBy, sortOrder)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(cursorInfo)
}

// UpdatePost handles post updates
func (h *PostHandler) UpdatePost(c *fiber.Ctx) error {
	// Read the request body once to avoid issues with c.BodyParser consuming the stream
	body := c.Body()

	var req models.UpdatePostRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	// Validate request
	if err := validation.ValidateUpdatePostRequest(&req); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	if req.ObjectId == nil {
		return errors.HandleValidationError(c, "objectId is required")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Update post
	if err := h.postService.UpdatePost(c.Context(), *req.ObjectId, &req, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"message": "Post updated successfully"})
}

// DeletePost handles post deletion
func (h *PostHandler) DeletePost(c *fiber.Ctx) error {
	postIDStr := c.Params("postId")
	// Defensive validation (middleware should prevent this, but safety first)
	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not Found"})
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Delete post (soft delete)
	if err := h.postService.SoftDeletePost(c.Context(), postID, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}

// UpdatePostProfile handles updating post profile information
func (h *PostHandler) UpdatePostProfile(c *fiber.Ctx) error {
	var req struct {
		OwnerUserId      string `json:"ownerUserId"`
		OwnerDisplayName string `json:"ownerDisplayName"`
		OwnerAvatar      string `json:"ownerAvatar"`
	}

	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	userID, err := uuid.FromString(req.OwnerUserId)
	if err != nil {
		return errors.HandleUUIDError(c, "user ID")
	}

	if err := h.postService.UpdatePostProfile(c.Context(), userID, req.OwnerDisplayName, req.OwnerAvatar); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Post profiles updated successfully",
	})
}

// IncrementScore increments the score for a post
func (h *PostHandler) IncrementScore(c *fiber.Ctx) error {
	var req struct {
		PostID uuid.UUID `json:"postId"`
		Delta  int       `json:"delta"`
	}

	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "invalid request body")
	}

	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "invalid current user")
	}

	if err := h.postService.IncrementScore(c.Context(), req.PostID, req.Delta, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "score incremented"})
}

// IncrementCommentCount increments the comment count for a post
func (h *PostHandler) IncrementCommentCount(c *fiber.Ctx) error {
	var req struct {
		PostID uuid.UUID `json:"postId"`
		Count  int       `json:"count"`
	}

	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	if err := h.postService.IncrementCommentCount(c.Context(), req.PostID, req.Count, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "comment count incremented"})
}

// DisableComment handles disabling comments
func (h *PostHandler) DisableComment(c *fiber.Ctx) error {
	var req struct {
		ObjectId string `json:"objectId"`
		Disable  bool   `json:"disable"`
	}

	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	postID, err := uuid.FromString(req.ObjectId)
	if err != nil {
		return errors.HandleUUIDError(c, "post ID")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	if err := h.postService.SetCommentDisabled(c.Context(), postID, req.Disable, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Comment setting updated successfully",
	})
}

// DisableSharing handles disabling sharing
func (h *PostHandler) DisableSharing(c *fiber.Ctx) error {
	var req struct {
		ObjectId string `json:"objectId"`
		Disable  bool   `json:"disable"`
	}

	if err := c.BodyParser(&req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid request body")
	}

	postID, err := uuid.FromString(req.ObjectId)
	if err != nil {
		return errors.HandleUUIDError(c, "post ID")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	if err := h.postService.SetSharingDisabled(c.Context(), postID, req.Disable, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"message": "Sharing setting updated successfully",
	})
}

// GeneratePostURLKey handles URL key generation
func (h *PostHandler) GeneratePostURLKey(c *fiber.Ctx) error {
	postIDStr := c.Params("postId")

	postID, err := uuid.FromString(postIDStr)
	if err != nil {
		return errors.HandleUUIDError(c, "post ID")
	}

	// Get user context
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	// Generate URL key
	urlKey, err := h.postService.GenerateURLKey(c.Context(), postID, &user)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"urlKey": urlKey,
	})
}

// CreateIndex handles index creation
func (h *PostHandler) CreateIndex(c *fiber.Ctx) error {
	indexes := map[string]interface{}{
		"body":     "text",
		"objectId": 1,
	}

	if err := h.postService.CreateIndex(c.Context(), indexes); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Indexes created successfully",
	})
}

// IncrementViewCount increments the view count for a post
func (h *PostHandler) IncrementViewCount(c *fiber.Ctx) error {
	postID := c.Params("postId")
	if postID == "" {
		return errors.HandleInvalidRequestError(c, "postId required")
	}

	pid, err := uuid.FromString(postID)
	if err != nil {
		return errors.HandleUUIDError(c, "post ID")
	}

	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	if err := h.postService.IncrementViewCount(c.Context(), pid, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "view count incremented"})
}

// IncrementViewCountByURLKey increments the view count for a post by URL key
func (h *PostHandler) IncrementViewCountByURLKey(c *fiber.Ctx) error {
	urlKey := c.Params("urlKey")
	if urlKey == "" {
		return errors.HandleInvalidRequestError(c, "urlKey required")
	}

	post, err := h.postService.GetPostByURLKey(c.Context(), urlKey)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return errors.HandleUserContextError(c, "Invalid user context")
	}

	if err := h.postService.IncrementViewCount(c.Context(), post.ObjectId, &user); err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "view count incremented"})
}

// Helper methods

func (h *PostHandler) convertPostToResponse(post *models.Post) models.PostResponse {
	return models.PostResponse{
		ObjectId:         post.ObjectId.String(),
		PostTypeId:       post.PostTypeId,
		Score:            post.Score,
		Votes:            post.Votes,
		ViewCount:        post.ViewCount,
		Body:             post.Body,
		OwnerUserId:      post.OwnerUserId.String(),
		OwnerDisplayName: post.OwnerDisplayName,
		OwnerAvatar:      post.OwnerAvatar,
		Tags:             post.Tags,
		CommentCounter:   post.CommentCounter,
		Image:            post.Image,
		ImageFullPath:    post.ImageFullPath,
		Video:            post.Video,
		Thumbnail:        post.Thumbnail,
		URLKey:           post.URLKey,
		Album:            post.Album,
		DisableComments:  post.DisableComments,
		DisableSharing:   post.DisableSharing,
		Deleted:          post.Deleted,
		DeletedDate:      post.DeletedDate,
		CreatedDate:      post.CreatedDate,
		LastUpdated:      post.LastUpdated,
		Permission:       post.Permission,
		Version:          post.Version,
	}
}
