package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	bookmarksRepository "github.com/qolzam/telar/apps/api/bookmarks/repository"
	commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
	"github.com/qolzam/telar/apps/api/internal/cache"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
	"github.com/qolzam/telar/apps/api/posts/common"
	postsErrors "github.com/qolzam/telar/apps/api/posts/errors"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/posts/repository"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
	votesRepository "github.com/qolzam/telar/apps/api/votes/repository"
)

// postQueryBuilder has been removed as part of the architectural migration.
// All query building is now handled by the PostRepository interface methods
// (e.g., Find, Count) which accept PostFilter structs for domain-specific filtering.

// postService implements the PostService interface
type postService struct {
	repo           repository.PostRepository 
	voteRepo       votesRepository.VoteRepository
	bookmarkRepo   bookmarksRepository.Repository
	cacheService   *cache.GenericCacheService
	config         *platformconfig.Config
	commentCounter sharedInterfaces.CommentCounter
	commentRepo    commentRepository.CommentRepository 
}

// Ensure postService implements sharedInterfaces.PostStatsUpdater interface
var _ sharedInterfaces.PostStatsUpdater = (*postService)(nil)

// incrementCommentCountInternal is the internal helper that actually updates the comment counter
// This is used by both PostService.IncrementCommentCount (with ownership check) and PostStatsUpdater.IncrementCommentCountForService (without ownership check)
// Note: This method loads the post, updates CommentCounter, and saves it. For atomic increments, a dedicated repository method could be added in the future.
func (s *postService) incrementCommentCountInternal(ctx context.Context, postID uuid.UUID, delta int, checkOwnership bool, ownerID uuid.UUID) error {
	// If ownership check is required, verify ownership first
	if checkOwnership {
		post, err := s.repo.FindByID(ctx, postID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return postsErrors.ErrPostNotFound
			}
			return fmt.Errorf("failed to get post: %w", err)
		}
		if post.OwnerUserId != ownerID {
			return postsErrors.ErrPostOwnershipRequired
		}
	}

	// Use atomic repository method to prevent race conditions
	if err := s.repo.IncrementCommentCount(ctx, postID, delta); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return postsErrors.ErrPostNotFound
		}
		log.Error("Repository.IncrementCommentCount failed for post %s (delta: %d): %v", postID.String(), delta, err)
		return err
	}

	if s.cacheService != nil {
		log.Info("Invalidating posts cache after commentCounter update for post %s", postID.String())
		s.invalidateAllPosts(ctx)
	}

	return nil
}

// incrementCommentCountForService increments the comment count for a post
func (s *postService) incrementCommentCountForService(ctx context.Context, postID uuid.UUID, delta int) error {
	err := s.incrementCommentCountInternal(ctx, postID, delta, false, uuid.Nil)
	if err != nil {
		log.Error("incrementCommentCountInternal failed for post %s (delta: %d): %v", postID.String(), delta, err)
	}
	return err
}

// GetPostsByIDs retrieves posts in bulk; caller should reorder results as needed.
func (s *postService) GetPostsByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Post, error) {
	if len(ids) == 0 {
		return []*models.Post{}, nil
	}
	posts, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts by ids: %w", err)
	}
	return posts, nil
}

// NewPostService creates a new instance of the post service
// commentRepo is optional (can be nil for tests), but required for cascade soft-delete in production
func NewPostService(repo repository.PostRepository, voteRepo votesRepository.VoteRepository, bookmarkRepo bookmarksRepository.Repository, cfg *platformconfig.Config, commentCounter sharedInterfaces.CommentCounter, commentRepo commentRepository.CommentRepository) PostService {
	enableCache := true
	if cfg != nil {
		enableCache = cfg.Cache.Enabled
	}

	var cacheService *cache.GenericCacheService
	if enableCache {
		cacheService = cache.NewGenericCacheServiceFor("posts")
	}

	return &postService{
		repo:           repo,
		voteRepo:       voteRepo,
		bookmarkRepo:   bookmarkRepo,
		cacheService:   cacheService,
		config:         cfg,
		commentCounter: commentCounter,
		commentRepo:    commentRepo,
	}
}

// generateCursorCacheKey generates a cache key for cursor-based pagination
func (s *postService) generateCursorCacheKey(filter *models.PostQueryFilter) string {
	params := map[string]interface{}{
		"operation": "cursor_query",
		"limit":     filter.Limit,
		"sortField": filter.SortField,
		"sortDir":   filter.SortDirection,
	}

	if filter.OwnerUserId != nil {
		params["userId"] = filter.OwnerUserId.String()
	}
	if filter.PostTypeId != nil {
		params["postTypeId"] = *filter.PostTypeId
	}
	if filter.Deleted != nil {
		params["deleted"] = *filter.Deleted
	}
	if len(filter.Tags) > 0 {
		params["tags"] = strings.Join(filter.Tags, ",")
	}
	if filter.CreatedAfter != nil {
		params["createdAfter"] = filter.CreatedAfter.Unix()
	}
	if filter.Cursor != "" {
		params["cursor"] = filter.Cursor
	}
	if filter.AfterCursor != "" {
		params["afterCursor"] = filter.AfterCursor
	}
	if filter.BeforeCursor != "" {
		params["beforeCursor"] = filter.BeforeCursor
	}

	return s.cacheService.GenerateHashKey("cursor", params)
}

// generateQueryCacheKey generates a cache key for offset-based pagination queries
func (s *postService) generateQueryCacheKey(filter *models.PostQueryFilter) string {
	params := map[string]interface{}{
		"operation": "query",
		"limit":     filter.Limit,
		"page":      filter.Page,
	}

	if filter.OwnerUserId != nil {
		params["userId"] = filter.OwnerUserId.String()
	}
	if filter.PostTypeId != nil {
		params["postTypeId"] = *filter.PostTypeId
	}
	if filter.Deleted != nil {
		params["deleted"] = *filter.Deleted
	}
	if len(filter.Tags) > 0 {
		params["tags"] = filter.Tags
	}
	if filter.CreatedAfter != nil {
		params["createdAfter"] = filter.CreatedAfter.Unix()
	}

	return s.cacheService.GenerateHashKey("query", params)
}

// generateSearchCacheKey generates a cache key for search operations
func (s *postService) generateSearchCacheKey(searchTerm string, filter *models.PostQueryFilter) string {
	params := map[string]interface{}{
		"operation":  "search",
		"searchTerm": searchTerm,
		"limit":      filter.Limit,
		"page":       filter.Page,
	}

	if filter.OwnerUserId != nil {
		params["userId"] = filter.OwnerUserId.String()
	}
	if filter.PostTypeId != nil {
		params["postTypeId"] = *filter.PostTypeId
	}

	return s.cacheService.GenerateHashKey("search", params)
}

// getCachedPosts retrieves cached posts result
func (s *postService) getCachedPosts(ctx context.Context, cacheKey string) (*models.PostsListResponse, error) {
	var result models.PostsListResponse
	if err := s.cacheService.GetCached(ctx, cacheKey, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// cachePosts stores posts result in cache
func (s *postService) cachePosts(ctx context.Context, cacheKey string, result *models.PostsListResponse) error {
	return s.cacheService.CacheData(ctx, cacheKey, result, time.Hour)
}

// invalidateUserPosts invalidates all cache entries for a specific user
func (s *postService) invalidateUserPosts(ctx context.Context, userID string) {
	// Use pattern to invalidate all user-specific cache entries
	pattern := "cursor:*userId:" + userID + "*"
	s.cacheService.InvalidatePattern(ctx, pattern)

	pattern = "search:*userId:" + userID + "*"
	s.cacheService.InvalidatePattern(ctx, pattern)

	// Best-effort invalidation for offset-based queries
	pattern = "query:*"
	s.cacheService.InvalidatePattern(ctx, pattern)
}

// invalidateAllPosts invalidates all posts-related cache entries
func (s *postService) invalidateAllPosts(ctx context.Context) {
	// Invalidate all cursor and search cache entries
	s.cacheService.InvalidatePattern(ctx, "cursor:*")
	s.cacheService.InvalidatePattern(ctx, "search:*")
	s.cacheService.InvalidatePattern(ctx, "query:*")
}

// CreatePost creates a new post
func (s *postService) CreatePost(ctx context.Context, req *models.CreatePostRequest, user *types.UserContext) (*models.Post, error) {
	if req == nil {
		return nil, fmt.Errorf("create post request is required")
	}
	if user == nil {
		return nil, fmt.Errorf("user context is required")
	}

	// Generate UUID for the post, or use provided one for backward compatibility
	var objectId uuid.UUID
	if req.ObjectId != nil && *req.ObjectId != uuid.Nil {
		objectId = *req.ObjectId
	} else {
		var err error
		objectId, err = uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("failed to generate post ID: %w", err)
		}
	}

	// Create the post entity
	post := &models.Post{
		ObjectId:         objectId,
		PostTypeId:       req.PostTypeId,
		Score:            0,
		Votes:            make(map[string]string),
		ViewCount:        0,
		Body:             req.Body,
		OwnerUserId:      user.UserID,
		OwnerDisplayName: user.DisplayName,
		OwnerAvatar:      user.Avatar,
		URLKey:           common.GeneratePostURLKey(user.SocialName, req.Body, objectId.String()),
		Tags:             req.Tags,
		CommentCounter:   0,
		Image:            req.Image,
		ImageFullPath:    req.ImageFullPath,
		Video:            req.Video,
		Thumbnail:        req.Thumbnail,
		DisableComments:  req.DisableComments,
		DisableSharing:   req.DisableSharing,
		Deleted:          false,
		DeletedDate:      0,
		CreatedDate:      utils.UTCNowUnix(),
		LastUpdated:      0,
		AccessUserList:   req.AccessUserList,
		Permission:       req.Permission,
		Version:          req.Version,
	}

	// Handle album if provided
	if len(req.Album.Photos) > 0 {
		post.Album = &req.Album
	}

	// Set timestamps
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now
	if post.CreatedDate == 0 {
		post.CreatedDate = now.Unix()
	}
	if post.LastUpdated == 0 {
		post.LastUpdated = now.Unix()
	}

	// Save to database using new repository
	if err := s.repo.Create(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	// Invalidate relevant caches after successful creation
	if s.cacheService != nil {
		s.invalidateUserPosts(ctx, user.UserID.String())
		s.invalidateAllPosts(ctx)
	}

	return post, nil
}

// GetPost retrieves a post by ID
func (s *postService) GetPost(ctx context.Context, postID uuid.UUID) (*models.Post, error) {
	post, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, postsErrors.ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return post, nil
}

// GetPostByURLKey retrieves a post by URL key
func (s *postService) GetPostByURLKey(ctx context.Context, urlKey string) (*models.Post, error) {
	post, err := s.repo.FindByURLKey(ctx, urlKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			return nil, postsErrors.ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post by URL key: %w", err)
	}

	return post, nil
}

// GetPostsByUser retrieves posts by user ID
func (s *postService) GetPostsByUser(ctx context.Context, userID uuid.UUID, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Normalize pagination
	limit := 10
	page := 1
	if filter != nil {
		if filter.Limit > 0 {
			limit = filter.Limit
		}
		if filter.Page > 0 {
			page = filter.Page
		}
	}
	offset := (page - 1) * limit

	// Use new repository method
	posts, err := s.repo.FindByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts by user: %w", err)
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, post)
	}

	// Note: Attaching latest comment preview requires comments service integration - deferred for now

	// Get total count using repository Count method
	repoFilter := repository.PostFilter{
		OwnerUserID: &userID,
		Deleted:     ptr(false), // Only count non-deleted posts
	}
	totalCount, err := s.repo.Count(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get post count: %w", err)
	}

	return &models.PostsListResponse{
		Posts:      postResponses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
}

// ptr is a helper to get a pointer to a boolean literal
func ptr(b bool) *bool {
	return &b
}

// SearchPosts searches posts by query string
// Note: This method uses the Find method with SearchText filter for basic text search.
// For advanced full-text search with ranking, a dedicated full-text search method could be added to PostRepository in the future.
func (s *postService) SearchPosts(ctx context.Context, query string, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	if filter == nil {
		filter = &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}
	}

	// Build repository filter with search term
	repoFilter := repository.PostFilter{
		SearchText: &query,
	}
	if filter.OwnerUserId != nil {
		repoFilter.OwnerUserID = filter.OwnerUserId
	}
	if filter.PostTypeId != nil {
		repoFilter.PostTypeID = filter.PostTypeId
	}
	if len(filter.Tags) > 0 {
		repoFilter.Tags = filter.Tags
	}
	if filter.Deleted != nil {
		repoFilter.Deleted = filter.Deleted
	} else {
		repoFilter.Deleted = ptr(false) // Default to non-deleted
	}

	// Normalize pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Use repository Find method with search term
	posts, err := s.repo.Find(ctx, repoFilter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search posts: %w", err)
	}

	// Get total count
	totalCount, err := s.repo.Count(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get search count: %w", err)
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, post)
	}

	return &models.PostsListResponse{
		Posts:      postResponses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
}

// SearchPostsLite returns a small set of posts for autocomplete
func (s *postService) SearchPostsLite(ctx context.Context, query string, limit int) ([]models.PostResponse, error) {
	trimmed := strings.TrimSpace(query)
	if len(trimmed) < 3 {
		return []models.PostResponse{}, nil
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	posts, err := s.repo.Search(ctx, trimmed, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search posts: %w", err)
	}
	if len(posts) == 0 {
		return []models.PostResponse{}, nil
	}

	// Preload comment counts in bulk to match feed data
	commentCounts := map[uuid.UUID]int64{}
	postIDs := make([]uuid.UUID, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ObjectId
	}
	if s.commentRepo != nil {
		if counts, err := s.commentRepo.CountByPostIDs(ctx, postIDs); err == nil {
			commentCounts = counts
		}
	}

	// Preload vote state in bulk when user is authenticated
	voteMap := map[uuid.UUID]int{}
	if s.voteRepo != nil {
		if userCtx, ok := ctx.Value(types.UserCtxName).(types.UserContext); ok {
			if votes, err := s.voteRepo.GetVotesForPosts(ctx, postIDs, userCtx.UserID); err == nil {
				voteMap = votes
			}
		}
	}

	responses := make([]models.PostResponse, len(posts))
	ctxWithoutUser := context.WithValue(ctx, types.UserCtxName, (*types.UserContext)(nil))

	for i, post := range posts {
		if count, ok := commentCounts[post.ObjectId]; ok {
			post.CommentCounter = count
		}
		response := s.ConvertPostToResponse(ctxWithoutUser, post)
		if v, ok := voteMap[post.ObjectId]; ok {
			response.VoteType = v
		}
		responses[i] = response
	}

	return responses, nil
}

// UpdatePost updates an existing post
func (s *postService) UpdatePost(ctx context.Context, postID uuid.UUID, req *models.UpdatePostRequest, user *types.UserContext) error {
	// Load existing post
	post, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("failed to get post: %w", err)
	}

	// Verify ownership
	if post.OwnerUserId != user.UserID {
		return postsErrors.ErrPostOwnershipRequired
	}

	// Update fields on the struct
	if req.Body != nil {
		post.Body = *req.Body
	}
	if req.Image != nil {
		post.Image = *req.Image
	}
	if req.ImageFullPath != nil {
		post.ImageFullPath = *req.ImageFullPath
	}
	if req.Video != nil {
		post.Video = *req.Video
	}
	if req.Thumbnail != nil {
		post.Thumbnail = *req.Thumbnail
	}
	if req.Tags != nil {
		post.Tags = *req.Tags
	}
	if req.Album != nil {
		post.Album = req.Album
	}
	if req.DisableComments != nil {
		post.DisableComments = *req.DisableComments
	}
	if req.DisableSharing != nil {
		post.DisableSharing = *req.DisableSharing
	}
	if req.AccessUserList != nil {
		post.AccessUserList = *req.AccessUserList
	}
	if req.Permission != nil {
		post.Permission = *req.Permission
	}
	if req.Version != nil {
		post.Version = *req.Version
	}

	// Update timestamp
	post.UpdatedAt = time.Now()
	post.LastUpdated = time.Now().Unix()

	// Save using repository
	if err := s.repo.Update(ctx, post); err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	// Invalidate relevant caches after successful update
	if s.cacheService != nil {
		s.invalidateUserPosts(ctx, user.UserID.String())
		s.invalidateAllPosts(ctx)
	}

	return nil
}

// UpdatePostProfile updates post profiles for a user
func (s *postService) UpdatePostProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
	return s.repo.UpdateOwnerProfile(ctx, userID, displayName, avatar)
}

// SetField sets a single field value by objectId (for backward compatibility)
func (s *postService) SetField(ctx context.Context, objectId uuid.UUID, field string, value interface{}) error {
	updates := map[string]interface{}{field: value}
	return s.UpdateFields(ctx, objectId, updates)
}

// IncrementField increments a numeric field by delta (for backward compatibility)
func (s *postService) IncrementField(ctx context.Context, objectId uuid.UUID, field string, delta int) error {
	increments := map[string]interface{}{field: delta}
	return s.IncrementFields(ctx, objectId, increments)
}

// UpdateByOwner updates allowed fields of a post by objectId for a specific owner (SECURITY: validates ownership)
func (s *postService) UpdateByOwner(ctx context.Context, objectId uuid.UUID, owner uuid.UUID, fields map[string]interface{}) error {
	// Load post and verify ownership
	post, err := s.repo.FindByID(ctx, objectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("failed to get post: %w", err)
	}

	if post.OwnerUserId != owner {
		return postsErrors.ErrPostNotFound
	}

	// Update fields on struct (simplified - only handles common fields)
	// Note: This method supports common field types. For complex types or validation, consider using specific update methods.
	if val, ok := fields["body"].(string); ok {
		post.Body = val
	}
	if val, ok := fields["score"].(int64); ok {
		post.Score = val
	}
	// Add more field mappings as needed

	post.UpdatedAt = time.Now()
	post.LastUpdated = time.Now().Unix()

	return s.repo.Update(ctx, post)
}

// UpdateProfileForOwner updates display and avatar across all posts for an owner (SECURITY: validates ownership)
func (s *postService) UpdateProfileForOwner(ctx context.Context, owner uuid.UUID, displayName string, avatar string) error {
	return s.repo.UpdateOwnerProfile(ctx, owner, displayName, avatar)
}

// IncrementScore increments the post score
// Note: This method loads the post, updates the score, and saves it. For atomic increments at scale, a dedicated repository method could be added.
func (s *postService) IncrementScore(ctx context.Context, postID uuid.UUID, delta int, user *types.UserContext) error {
	// Use atomic repository method to prevent race conditions
	if err := s.repo.IncrementScore(ctx, postID, delta); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("failed to increment score: %w", err)
	}

	// Invalidate cache after score increment
	if s.cacheService != nil {
		s.invalidateAllPosts(ctx)
	}

	return nil
}

// IncrementCommentCount increments the comment count using native database operation with ownership validation
func (s *postService) IncrementCommentCount(ctx context.Context, postID uuid.UUID, delta int, user *types.UserContext) error {
	// Use internal helper with ownership check for PostService interface
	return s.incrementCommentCountInternal(ctx, postID, delta, true, user.UserID)
}

// IncrementCommentCountForService increments the comment count for a post (implements PostStatsUpdater interface)
// This is the microservice-safe version that doesn't require UserContext or ownership check
func (s *postService) IncrementCommentCountForService(ctx context.Context, postID uuid.UUID, delta int) error {
	return s.incrementCommentCountForService(ctx, postID, delta)
}

// IncrementViewCount increments the view count using native database operation
// This is now a single atomic SQL operation: UPDATE posts SET view_count = view_count + 1 WHERE id = $1
func (s *postService) IncrementViewCount(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	// Note: View count increments don't require ownership validation - anyone can view a post
	// The repository method handles the atomic increment efficiently
	if err := s.repo.IncrementViewCount(ctx, postID); err != nil {
		return fmt.Errorf("failed to increment view count: %w", err)
	}

	// Invalidate cache after view count increment
	if s.cacheService != nil {
		s.invalidateAllPosts(ctx)
	}

	return nil
}

// findPostForOwnershipCheck finds a post for ownership validation, regardless of deleted status.
// This is useful for operations like idempotent deletes or permanent data purges.
// It does NOT filter by deleted status, allowing it to find already-deleted posts.
func (s *postService) findPostForOwnershipCheck(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (*models.Post, error) {
	post, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, postsErrors.ErrPostNotFound
		}
		return nil, fmt.Errorf("database error during ownership check: %w", err)
	}

	// Check ownership
	if post.OwnerUserId != userID {
		return nil, postsErrors.ErrPostOwnershipRequired
	}

	return post, nil
}

// ValidatePostOwnership validates that a user owns a specific post
func (s *postService) ValidatePostOwnership(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
	post, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("failed to get post: %w", err)
	}

	// Check ownership
	if post.OwnerUserId != userID {
		return postsErrors.ErrPostOwnershipRequired
	}

	return nil
}

// validatePostExists validates that a post exists without checking ownership
func (s *postService) validatePostExists(ctx context.Context, postID uuid.UUID) error {
	_, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("database error during post existence validation: %w", err)
	}
	return nil
}

// GenerateURLKey generates a new URL key for a post
func (s *postService) GenerateURLKey(ctx context.Context, postID uuid.UUID, user *types.UserContext) (string, error) {
	// Get the post first
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return "", err
	}

	// If post already has URL key, return it
	if post.URLKey != "" {
		return post.URLKey, nil
	}

	// Generate new URL key using post owner's display name as social name (for backward compatibility)
	// In the original implementation, it calls getUserProfileByID to get the owner's profile
	// Since we don't have that function, we'll use the owner's display name as social name
	socialName := post.OwnerDisplayName
	if socialName == "" {
		socialName = "user" // fallback
	}

	urlKey := common.GeneratePostURLKey(socialName, post.Body, postID.String())

	// Update the post with new URL key using clean syntax
	updates := map[string]interface{}{
		"urlKey": urlKey,
	}

	if err := s.UpdateFields(ctx, postID, updates); err != nil {
		return "", err
	}

	return urlKey, nil
}

// Helper methods

// getPostCount counts posts matching a query
func (s *postService) getPostCount(ctx context.Context, filter repository.PostFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}

// getRootCommentCount counts root comments (non-reply comments) for a post via the CommentCounter interface
// This is used to populate commentCounter for existing posts that have 0 but actually have comments
func (s *postService) getRootCommentCount(ctx context.Context, postID uuid.UUID) (int64, error) {
	if s.commentCounter == nil {
		// If no counter is provided (e.g., during initialization), return 0
		// This allows graceful degradation when services are being wired up
		return 0, nil
	}

	count, err := s.commentCounter.GetRootCommentCount(ctx, postID)
	if err != nil {
		log.Warn("Failed to count root comments for post %s: %v", postID.String(), err)
		return 0, err
	}

	if count > 0 {
		log.Info("Lazy population: Found %d root comments for post %s", count, postID.String())
	}

	return count, nil
}

// ConvertPostToResponse converts a Post model to PostResponse
// If commentCounter is 0 or invalid, it checks actual comment count to populate it for existing posts
func (s *postService) ConvertPostToResponse(ctx context.Context, post *models.Post) models.PostResponse {
	commentCounter := post.CommentCounter

	// Lazy population: If commentCounter is 0 or missing, check actual count from comments
	// This fixes existing posts that have comments but commentCounter wasn't initialized or is incorrect
	// We only do this check if counter is 0 to avoid unnecessary queries for posts without comments
	// Use a short timeout to prevent blocking the response if comment service is slow
	// Also verify counter matches actual count if counter seems incorrect
	if commentCounter == 0 || commentCounter < 0 {
		countCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		actualCount, err := s.getRootCommentCount(countCtx, post.ObjectId)
		if err != nil {
			// Log error but don't fail - return original count (0)
			// This prevents breaking the response if comment service is unavailable or slow
			if err == context.DeadlineExceeded {
				log.Warn("Timeout getting root comment count for post %s (lazy population)", post.ObjectId.String())
			} else {
				log.Warn("Failed to get root comment count for post %s: %v", post.ObjectId.String(), err)
			}
		} else if actualCount > 0 {
			commentCounter = actualCount
			// Update post in database asynchronously (fire and forget) to persist the count
			// This ensures future fetches won't need to recalculate
			go func() {
				updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Update post's commentCounter using UpdateFields to SET it directly
				// We use UpdateFields instead of IncrementFields because we want to SET the value,
				// not increment it (since the current value might be wrong)
				updates := map[string]interface{}{
					"commentCounter": actualCount,
				}
				if err := s.UpdateFields(updateCtx, post.ObjectId, updates); err != nil {
					log.Warn("Failed to update commentCounter for post %s: %v", post.ObjectId.String(), err)
				}
			}()
		}
	}

	response := models.PostResponse{
		ObjectId:         post.ObjectId.String(),
		PostTypeId:       post.PostTypeId,
		Score:            post.Score,
		VoteType:         0, // Default to 0 (None) - will be enriched if user context available
		Votes:            post.Votes,
		ViewCount:        post.ViewCount,
		Body:             post.Body,
		OwnerUserId:      post.OwnerUserId.String(),
		OwnerDisplayName: post.OwnerDisplayName,
		OwnerAvatar:      post.OwnerAvatar,
		Tags:             post.Tags,
		CommentCounter:   commentCounter,
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

	// Enrich with vote type if user context is available
	// Note: User context must be set in context by middleware before calling service
	if userCtx, ok := ctx.Value(types.UserCtxName).(types.UserContext); ok {
		s.enrichSinglePostWithVoteType(ctx, &response, userCtx.UserID)
		s.enrichSinglePostWithBookmark(ctx, &response, userCtx.UserID)
	}

	return response
}

// QueryPosts queries posts based on filter criteria
func (s *postService) QueryPosts(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	if filter == nil {
		filter = &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}
	}

	// Build repository filter
	repoFilter := repository.PostFilter{}
	if filter.OwnerUserId != nil {
		repoFilter.OwnerUserID = filter.OwnerUserId
	}
	if filter.PostTypeId != nil {
		repoFilter.PostTypeID = filter.PostTypeId
	}
	if len(filter.Tags) > 0 {
		repoFilter.Tags = filter.Tags
	}
	if filter.Deleted != nil {
		repoFilter.Deleted = filter.Deleted
	}
	if filter.CreatedAfter != nil {
		timestamp := filter.CreatedAfter.Unix()
		repoFilter.CreatedAfter = &timestamp
	}

	// Normalize pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Query posts
	posts, err := s.repo.Find(ctx, repoFilter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}

	// Get total count
	totalCount, err := s.repo.Count(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count posts: %w", err)
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, post)
	}

	hasMore := int64(page*limit) < totalCount
	result := &models.PostsListResponse{
		Posts:      postResponses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		HasNext:    hasMore, // Set hasNext based on Limit + 1 strategy
	}

	// Generate nextCursor from the last post if there are more posts
	if hasMore && len(posts) > 0 {
		lastPost := posts[len(posts)-1]
		sortField := filter.SortField
		if sortField == "" {
			sortField = "createdDate"
		}
		sortDirection := filter.SortDirection
		if sortDirection == "" {
			sortDirection = "desc"
		}
		if cursor, err := models.CreateCursorFromPost(lastPost, sortField, sortDirection); err == nil {
			result.NextCursor = cursor
		}
	}

	// Enrich with vote types if user context is available and voteRepo is set
	if userCtx, ok := ctx.Value(types.UserCtxName).(types.UserContext); ok {
		s.enrichPostsWithVoteType(ctx, result.Posts, userCtx.UserID)
		s.enrichPostsWithBookmarks(ctx, result.Posts, userCtx.UserID)
	}

	return result, nil
}

// enrichSinglePostWithVoteType enriches a single post with vote type
func (s *postService) enrichSinglePostWithVoteType(ctx context.Context, response *models.PostResponse, userID uuid.UUID) {
	if s.voteRepo == nil {
		return
	}

	postID, err := uuid.FromString(response.ObjectId)
	if err != nil {
		return
	}

	voteMap, err := s.voteRepo.GetVotesForPosts(ctx, []uuid.UUID{postID}, userID)
	if err == nil {
		response.VoteType = voteMap[postID]
	}
}

func (s *postService) enrichSinglePostWithBookmark(ctx context.Context, response *models.PostResponse, userID uuid.UUID) {
	if s.bookmarkRepo == nil {
		return
	}

	postID, err := uuid.FromString(response.ObjectId)
	if err != nil {
		return
	}

	bookmarkMap, err := s.bookmarkRepo.GetMapByUserAndPosts(ctx, userID, []uuid.UUID{postID})
	if err == nil {
		response.IsBookmarked = bookmarkMap[postID]
	}
}

// enrichPostsWithVoteType bulk-enriches posts with current user's vote type
// This avoids N+1 queries by fetching all votes in a single query
// This method belongs in the service layer to keep handlers thin and testable
func (s *postService) enrichPostsWithVoteType(ctx context.Context, posts []models.PostResponse, userID uuid.UUID) {
	if s.voteRepo == nil || len(posts) == 0 {
		return
	}

	// Extract post IDs
	postIDs := make([]uuid.UUID, 0, len(posts))
	for i := range posts {
		id, err := uuid.FromString(posts[i].ObjectId)
		if err == nil {
			postIDs = append(postIDs, id)
		}
	}

	if len(postIDs) == 0 {
		return
	}

	// Bulk-load user votes (single query using ANY operator)
	voteMap, err := s.voteRepo.GetVotesForPosts(ctx, postIDs, userID)
	if err != nil {
		// Graceful degradation: if vote service fails, posts still return with voteType=0
		return
	}

	// Set VoteType for each post
	for i := range posts {
		id, err := uuid.FromString(posts[i].ObjectId)
		if err == nil {
			posts[i].VoteType = voteMap[id]
		}
	}
}

func (s *postService) enrichPostsWithBookmarks(ctx context.Context, posts []models.PostResponse, userID uuid.UUID) {
	if s.bookmarkRepo == nil || len(posts) == 0 {
		return
	}

	postIDs := make([]uuid.UUID, 0, len(posts))
	for i := range posts {
		id, err := uuid.FromString(posts[i].ObjectId)
		if err == nil {
			postIDs = append(postIDs, id)
		}
	}

	if len(postIDs) == 0 {
		return
	}

	bookmarkMap, err := s.bookmarkRepo.GetMapByUserAndPosts(ctx, userID, postIDs)
	if err != nil {
		return
	}

	for i := range posts {
		id, err := uuid.FromString(posts[i].ObjectId)
		if err == nil {
			posts[i].IsBookmarked = bookmarkMap[id]
		}
	}
}

// QueryPostsWithCursor retrieves posts with cursor-based pagination
// Uses Limit + 1 strategy: fetch limit+1 items, if we get limit+1, hasNext=true
func (s *postService) QueryPostsWithCursor(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	if filter == nil {
		filter = &models.PostQueryFilter{
			Limit: 10,
		}
	}

	// Build repository filter
	repoFilter := repository.PostFilter{}
	if filter.OwnerUserId != nil {
		repoFilter.OwnerUserID = filter.OwnerUserId
	}
	if filter.PostTypeId != nil {
		repoFilter.PostTypeID = filter.PostTypeId
	}
	if len(filter.Tags) > 0 {
		repoFilter.Tags = filter.Tags
	}
	if filter.Deleted != nil {
		repoFilter.Deleted = filter.Deleted
	} else {
		deleted := false
		repoFilter.Deleted = &deleted
	}
	if filter.CreatedAfter != nil {
		timestamp := filter.CreatedAfter.Unix()
		repoFilter.CreatedAfter = &timestamp
	}
	if filter.Search != "" {
		repoFilter.SearchText = &filter.Search
	}

	// Normalize limit
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Normalize sort parameters
	sortField := filter.SortField
	if sortField == "" {
		sortField = "createdDate"
	}
	sortDirection := filter.SortDirection
	if sortDirection == "" {
		sortDirection = "desc"
	}

	// Decode cursor if provided
	var cursorData *models.CursorData
	if filter.Cursor != "" {
		decoded, err := models.DecodeCursor(filter.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		cursorData = decoded
	} else if filter.AfterCursor != "" {
		decoded, err := models.DecodeCursor(filter.AfterCursor)
		if err != nil {
			return nil, fmt.Errorf("invalid after cursor: %w", err)
		}
		cursorData = decoded
	}

	// Query posts with cursor pagination (uses Limit + 1 strategy)
	posts, hasMore, err := s.repo.FindWithCursor(ctx, repoFilter, cursorData, sortField, sortDirection, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts with cursor: %w", err)
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, post)
	}

	// Generate nextCursor from the last post if there are more posts
	var nextCursor string
	if hasMore && len(posts) > 0 {
		lastPost := posts[len(posts)-1]
		if cursor, err := models.CreateCursorFromPost(lastPost, sortField, sortDirection); err == nil {
			nextCursor = cursor
		}
	}

	result := &models.PostsListResponse{
		Posts:      postResponses,
		TotalCount: 0,
		Page:       0,
		Limit:      limit,
		HasNext:    hasMore, // Set hasNext based on Limit + 1 strategy
		NextCursor: nextCursor,
	}

	// Enrich with vote types if user context is available and voteRepo is set
	if userCtx, ok := ctx.Value(types.UserCtxName).(types.UserContext); ok {
		s.enrichPostsWithVoteType(ctx, result.Posts, userCtx.UserID)
		s.enrichPostsWithBookmarks(ctx, result.Posts, userCtx.UserID)
	}

	return result, nil
}

// SearchPostsWithCursor retrieves posts matching search criteria with cursor-based pagination
// Note: Currently falls back to basic SearchPosts. For true cursor pagination, additional repository methods would be needed.
func (s *postService) SearchPostsWithCursor(ctx context.Context, searchTerm string, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// For now, fall back to basic SearchPosts - cursor pagination needs additional repository support
	return s.SearchPosts(ctx, searchTerm, filter)
}

// UpdateFields updates post fields using field-based syntax
// Note: For type safety and better validation, consider using specific update methods instead of this generic approach.
func (s *postService) UpdateFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error {
	// Load post first
	post, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to get post: %w", err)
	}

	// Apply updates to post struct
	for key, value := range updates {
		switch key {
		case "urlKey":
			if str, ok := value.(string); ok {
				post.URLKey = str
			}
		case "body":
			if str, ok := value.(string); ok {
				post.Body = str
			}
		case "score":
			if num, ok := value.(int64); ok {
				post.Score = num
			} else if num, ok := value.(int); ok {
				post.Score = int64(num)
			}
		case "commentCounter", "comment_count":
			if num, ok := value.(int64); ok {
				post.CommentCounter = num
			} else if num, ok := value.(int); ok {
				post.CommentCounter = int64(num)
			}
		case "deleted":
			if b, ok := value.(bool); ok {
				post.Deleted = b
			}
		case "deletedDate", "deleted_date":
			if num, ok := value.(int64); ok {
				post.DeletedDate = num
			}
		case "disableComments", "disable_comments":
			if b, ok := value.(bool); ok {
				post.DisableComments = b
			}
		case "disableSharing", "disable_sharing":
			if b, ok := value.(bool); ok {
				post.DisableSharing = b
			}
		}
	}

	post.UpdatedAt = time.Now()
	post.LastUpdated = time.Now().Unix()

	return s.repo.Update(ctx, post)
}

// IncrementFields increments numeric fields using field-based syntax
// Note: For type safety and better validation, consider using specific increment methods instead of this generic approach.
func (s *postService) IncrementFields(ctx context.Context, postID uuid.UUID, increments map[string]interface{}) error {
	// Load post first
	post, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to get post: %w", err)
	}

	// Apply increments to post struct
	for key, value := range increments {
		delta, ok := value.(int)
		if !ok {
			if delta64, ok := value.(int64); ok {
				delta = int(delta64)
			} else {
				continue
			}
		}

		switch key {
		case "score":
			post.Score += int64(delta)
		case "commentCounter", "comment_count":
			post.CommentCounter += int64(delta)
		case "viewCount", "view_count":
			post.ViewCount += int64(delta)
		}
	}

	post.UpdatedAt = time.Now()
	post.LastUpdated = time.Now().Unix()

	return s.repo.Update(ctx, post)
}

// UpdateAndIncrementFields performs both update and increment operations
func (s *postService) UpdateAndIncrementFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error {
	// Apply updates first
	if len(updates) > 0 {
		if err := s.UpdateFields(ctx, postID, updates); err != nil {
			return err
		}
	}

	// Then apply increments
	if len(increments) > 0 {
		if err := s.IncrementFields(ctx, postID, increments); err != nil {
			return err
		}
	}

	return nil
}

// SetCommentDisabled sets comment disabled status with ownership validation
func (s *postService) SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, user *types.UserContext) error {
	// Ownership validation is now embedded in the repository method's WHERE clause for atomicity
	return s.repo.SetCommentDisabled(ctx, postID, disabled, user.UserID)
}

// SetSharingDisabled sets sharing disabled status with ownership validation
func (s *postService) SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, user *types.UserContext) error {
	// Ownership validation is now embedded in the repository method's WHERE clause for atomicity
	return s.repo.SetSharingDisabled(ctx, postID, disabled, user.UserID)
}

// DeletePost deletes a post (soft delete)
func (s *postService) DeletePost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	// Load post to verify ownership
	post, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("failed to get post: %w", err)
	}

	// Verify ownership
	if post.OwnerUserId != user.UserID {
		return postsErrors.ErrPostOwnershipRequired
	}

	// Delete the post (soft delete - sets is_deleted = TRUE)
	if err := s.repo.Delete(ctx, postID); err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	// Invalidate relevant caches after successful deletion
	if s.cacheService != nil {
		s.invalidateUserPosts(ctx, user.UserID.String())
		s.invalidateAllPosts(ctx)
	}

	return nil
}

// SoftDeletePost marks a post as deleted (idempotent operation).
// If the post is already soft-deleted, it returns success immediately.
// This ensures DELETE operations are idempotent per REST best practices.
// Cascade soft-deletes all associated comments in the same transaction.
func (s *postService) SoftDeletePost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	// 1. Use the new helper to find the post, even if it's already deleted.
	post, err := s.findPostForOwnershipCheck(ctx, postID, user.UserID)
	if err != nil {
		// If the error is ErrPostNotFound, the post doesn't exist or the user doesn't own it.
		// From the client's perspective, the desired state (post is gone) is true.
		// Therefore, we can treat "not found" as a success case for an idempotent delete.
		if errors.Is(err, postsErrors.ErrPostNotFound) {
			return nil
		}
		// Any other error is a real failure.
		return err
	}

	// 2. If the post is found, check if it's already deleted.
	if post.Deleted {
		// The post is already in the desired state. Return success immediately.
		return nil
	}

	// 3. Perform cascade soft-delete in a transaction (post + comments)
	err = s.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		// 3a. Soft-delete the post
		updates := map[string]interface{}{
			"deleted":     true,
			"deletedDate": time.Now().Unix(),
		}
		if err := s.UpdateFields(txCtx, postID, updates); err != nil {
			return fmt.Errorf("failed to soft-delete post: %w", err)
		}

		// 3b. Cascade soft-delete all comments for this post (write-time propagation)
		if err := s.commentRepo.DeleteByPostID(txCtx, postID); err != nil {
			return fmt.Errorf("failed to cascade soft-delete comments: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 4. Invalidate caches.
	if s.cacheService != nil {
		s.invalidateUserPosts(ctx, user.UserID.String())
		s.invalidateAllPosts(ctx)
	}

	return nil
}

// DeleteByOwner deletes a post by objectId for a specific owner (SECURITY: validates ownership)
func (s *postService) DeleteByOwner(ctx context.Context, owner uuid.UUID, objectId uuid.UUID) error {
	// Verify ownership first
	post, err := s.repo.FindByID(ctx, objectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("failed to get post: %w", err)
	}

	if post.OwnerUserId != owner {
		return postsErrors.ErrPostNotFound
	}

	return s.repo.Delete(ctx, objectId)
}

// DELETED: UpdateFieldsWithOwnership, DeleteWithOwnership, IncrementFieldsWithOwnership
// These methods were removed as they violated architectural principles:
// 1. They used context.Background(), breaking context propagation
// 2. They bypassed the Query Object pattern
// 3. They created unnecessary complexity in the repository interface
//
// All operations now use the standard UpdateFields, Delete, IncrementFields methods
// with Query objects that include ownership validation criteria.

// GetCursorInfo returns cursor information for a specific post
func (s *postService) GetCursorInfo(ctx context.Context, postID uuid.UUID, sortBy, sortOrder string) (*models.CursorInfo, error) {
	// Normalize sort parameters
	sortField := models.ParseSortField(sortBy)
	direction := models.ParseSortDirection(sortOrder)

	// Ensure the post exists and load it
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}

	// Compute the sort value of the current post
	// Note: Cursor pagination position calculation requires complex SQL queries with compound OR conditions
	// that would need dedicated repository methods. The current implementation returns a simplified position
	// based on total count, which is sufficient for most use cases. For accurate position calculation
	// in large datasets, consider implementing dedicated cursor pagination methods in PostRepository.
	var sortValue interface{}
	switch sortField {
	case "createdDate":
		sortValue = post.CreatedDate
	case "lastUpdated":
		sortValue = post.LastUpdated
	case "score":
		sortValue = post.Score
	case "viewCount":
		sortValue = post.ViewCount
	case "commentCounter":
		sortValue = post.CommentCounter
	case "objectId":
		sortValue = post.ObjectId.String()
	default:
		return nil, fmt.Errorf("unsupported sort field: %s", sortField)
	}
	_ = sortValue // Suppress unused variable warning until cursor pagination is implemented

	// Count how many posts come before this one in the chosen order
	// Build a simple PostFilter for counting (cursor pagination position calculation is complex)
	// For now, return a simplified position based on total count
	deleted := false
	repoFilter := repository.PostFilter{
		Deleted: &deleted, // Only count non-deleted posts
	}
	beforeCount, err := s.getPostCount(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to compute position: %w", err)
	}
	position := int(beforeCount) + 1

	// Generate a stable cursor for this post using existing utilities
	cursor, err := models.CreateCursorFromPost(post, sortField, direction)
	if err != nil {
		return nil, fmt.Errorf("failed to create cursor: %w", err)
	}

	return &models.CursorInfo{
		PostId:    postID.String(),
		Cursor:    cursor,
		Position:  position,
		SortBy:    sortField,
		SortOrder: direction,
	}, nil
}
