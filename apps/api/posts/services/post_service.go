package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/cache"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
	commentsModels "github.com/qolzam/telar/apps/api/comments/models"
	"github.com/qolzam/telar/apps/api/posts/common"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
	postsErrors "github.com/qolzam/telar/apps/api/posts/errors"
	"github.com/qolzam/telar/apps/api/posts/models"
)

const postCollectionName = "post"

// postQueryBuilder is a helper struct for building posts-specific queries.
// It knows the schema of the `posts` table and provides fluent methods for query construction.
// This pattern moves schema knowledge from the generic repository to the service layer.
type postQueryBuilder struct {
	query *dbi.Query
}

// newPostQueryBuilder creates a new postQueryBuilder instance.
func newPostQueryBuilder() *postQueryBuilder {
	return &postQueryBuilder{
		query: &dbi.Query{
			Conditions: []dbi.Field{},
			OrGroups:   [][]dbi.Field{},
		},
	}
}

// WhereObjectID adds a filter for the object_id (indexed column).
func (b *postQueryBuilder) WhereObjectID(objectID uuid.UUID) *postQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "object_id", // Indexed column - direct access
		Value:    objectID,
		Operator: "=",
		IsJSONB:  false,
	})
	return b
}

// WhereOwner adds a filter for the owner_user_id (indexed column).
func (b *postQueryBuilder) WhereOwner(userID uuid.UUID) *postQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "owner_user_id", // Indexed column - direct access
		Value:    userID,
		Operator: "=",
		IsJSONB:  false,
	})
	return b
}

// WhereNotDeleted adds a filter to exclude deleted posts (JSONB field).
func (b *postQueryBuilder) WhereNotDeleted() *postQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:      "data->>'deleted'", // JSONB field in data column
		Value:     false,
		Operator:  "=",
		IsJSONB:   true,
		JSONBCast: "::boolean",
	})
	return b
}

// WhereDeleted adds a filter for the deleted status (JSONB field).
func (b *postQueryBuilder) WhereDeleted(deleted bool) *postQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:      "data->>'deleted'", // JSONB field in data column
		Value:     deleted,
		Operator:  "=",
		IsJSONB:   true,
		JSONBCast: "::boolean",
	})
	return b
}

// WherePostType adds a filter for post_type_id (indexed column).
func (b *postQueryBuilder) WherePostType(postTypeID int) *postQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "post_type_id", // Indexed column - direct access
		Value:    postTypeID,
		Operator: "=",
		IsJSONB:  false,
	})
	return b
}

// WhereTagsIn adds a filter for tags using abstract CONTAINS_ANY operator.
// Matches posts that have any of the specified tags.
func (b *postQueryBuilder) WhereTagsIn(tags []string) *postQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->'tags'", // JSONB array field
		Value:    tags,
		Operator: "CONTAINS_ANY", // Abstract operator - repository translates to PostgreSQL ?|
		IsJSONB:  true,
	})
	return b
}

// WhereTagsAll adds a filter for tags using the $all operator (JSONB array).
// Matches posts that have all of the specified tags.
func (b *postQueryBuilder) WhereTagsAll(tags []string) *postQueryBuilder {
	// Convert tags slice to JSON array string for @> operator
	tagsJSON, _ := json.Marshal(tags)
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->'tags'", // JSONB array field
		Value:    string(tagsJSON),
		Operator: "@>", // PostgreSQL JSONB contains operator (contains all)
		IsJSONB:  true,
	})
	return b
}

// WhereCreatedAfter adds a filter for created_date (indexed column) with >= operator.
func (b *postQueryBuilder) WhereCreatedAfter(timestamp int64) *postQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "created_date", // Indexed column - direct access
		Value:    timestamp,
		Operator: ">=",
		IsJSONB:  false,
	})
	return b
}

// WhereURLKey adds a filter for urlKey (JSONB field).
func (b *postQueryBuilder) WhereURLKey(urlKey string) *postQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'urlKey'", // JSONB field
		Value:    urlKey,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereSearchText adds a search filter using abstract REGEX_I operator across multiple fields.
// This creates an OR group for searching in body, tags, and ownerDisplayName.
func (b *postQueryBuilder) WhereSearchText(searchTerm string) *postQueryBuilder {
	// Create an OR group for search across multiple fields
	orFields := []dbi.Field{
		{
			Name:     "data->>'body'",
			Value:    searchTerm,
			Operator: "REGEX_I", // Abstract operator - repository translates to PostgreSQL ~*
			IsJSONB:  true,
		},
		{
			Name:     "data->>'tags'",
			Value:    searchTerm,
			Operator: "REGEX_I", // Abstract operator - repository translates to PostgreSQL ~*
			IsJSONB:  true,
		},
		{
			Name:     "data->>'ownerDisplayName'",
			Value:    searchTerm,
			Operator: "REGEX_I", // Abstract operator - repository translates to PostgreSQL ~*
			IsJSONB:  true,
		},
	}
	b.query.OrGroups = append(b.query.OrGroups, orFields)
	return b
}

// WhereCursor applies the complex OR logic for cursor pagination.
// sortField: the database column/path to sort by (e.g., "created_date" or "data->>'score'")
// sortValue: the cursor value to compare against
// tieBreakerID: the object_id value for tie-breaking
// direction: "asc" or "desc"
func (b *postQueryBuilder) WhereCursor(sortField string, sortValue interface{}, tieBreakerID uuid.UUID, direction string, isBefore bool) *postQueryBuilder {
	primaryOp := ">"
	tieOp := ">"

	if direction == "desc" {
		primaryOp = "<"
		tieOp = "<"
	}

	if isBefore {
		if primaryOp == "<" {
			primaryOp = ">"
		} else {
			primaryOp = "<"
		}

		if tieOp == "<" {
			tieOp = ">"
		} else {
			tieOp = "<"
		}
	}

	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     sortField,
		Operator: "CURSOR_PAGINATION",
		Value: map[string]interface{}{
			"sortValue": sortValue,
			"tieBreaker": tieBreakerID,
			"primaryOp": primaryOp,
			"tieOp":     tieOp,
		},
	})

	return b
}

// DELETED: WhereCursorCondition and mapToField - Removed as part of architectural cleanup
// Cursor logic is now handled directly by WhereCursor method.

// Build returns the constructed Query object.
func (b *postQueryBuilder) Build() *dbi.Query {
	return b.query
}

// postService implements the PostService interface
type postService struct {
	base           *platform.BaseService
	cacheService   *cache.GenericCacheService
	config         *platformconfig.Config
	commentCounter sharedInterfaces.CommentCounter // For counting comments
}

// Ensure postService implements sharedInterfaces.PostStatsUpdater interface
var _ sharedInterfaces.PostStatsUpdater = (*postService)(nil)

// incrementCommentCountInternal is the internal helper that actually updates the comment counter
// This is used by both PostService.IncrementCommentCount (with ownership check) and PostStatsUpdater.IncrementCommentCountForService (without ownership check)
func (s *postService) incrementCommentCountInternal(ctx context.Context, postID uuid.UUID, delta int, checkOwnership bool, ownerID uuid.UUID) error {
	qb := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereNotDeleted()
	
	if checkOwnership {
		qb = qb.WhereOwner(ownerID)
	}
	
	query := qb.Build()

	// Atomically increment/decrement commentCounter using IncrementFields
	increments := map[string]interface{}{
		"commentCounter": delta,
	}
	result := <-s.base.Repository.IncrementFields(ctx, postCollectionName, query, increments)
	if result.Error != nil {
		log.Error("Repository.IncrementFields failed for post %s (delta: %d): %v", postID.String(), delta, result.Error)
		return result.Error
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

// NewPostService creates a new instance of the post service
func NewPostService(base *platform.BaseService, cfg *platformconfig.Config, commentCounter sharedInterfaces.CommentCounter) PostService {
	enableCache := true
	if cfg != nil {
		enableCache = cfg.Cache.Enabled
	}

	var cacheService *cache.GenericCacheService
	if enableCache {
		cacheService = cache.NewGenericCacheServiceFor("posts")
	}

	return &postService{
		base:           base,
		cacheService:   cacheService,
		config:         cfg,
		commentCounter: commentCounter,
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

	// Save to database
	res := <-s.base.Repository.Save(
		ctx,
		postCollectionName,
		post.ObjectId,
		post.OwnerUserId,
		post.CreatedDate,
		post.LastUpdated,
		post,
	)
	if err := res.Error; err != nil {
		return nil, fmt.Errorf("failed to save post: %w", err)
	}

	// Invalidate relevant caches after successful creation
	if s.cacheService != nil {
		s.invalidateUserPosts(ctx, user.UserID.String())
		s.invalidateAllPosts(ctx)
	}

	return post, nil
}

// CreateIndex creates database indexes
func (s *postService) CreateIndex(ctx context.Context, indexes map[string]interface{}) error {
	return <-s.base.Repository.CreateIndex(ctx, postCollectionName, indexes)
}

// CreateIndexes creates default database indexes for posts collection
func (s *postService) CreateIndexes(ctx context.Context) error {
	indexes := map[string]interface{}{
		"body":     "text",
		"objectId": 1,
	}
	return s.CreateIndex(ctx, indexes)
}

// GetPost retrieves a post by ID
func (s *postService) GetPost(ctx context.Context, postID uuid.UUID) (*models.Post, error) {
	query := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereNotDeleted().
		Build()

	single := <-s.base.Repository.FindOne(ctx, postCollectionName, query)

	// Use the robust "decode-then-check" pattern
	var post models.Post
	if err := single.Decode(&post); err != nil {
		if errors.Is(err, dbi.ErrNoDocuments) {
			return nil, postsErrors.ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to decode post: %w", err)
	}

	return &post, nil
}

// GetPostByURLKey retrieves a post by URL key
func (s *postService) GetPostByURLKey(ctx context.Context, urlKey string) (*models.Post, error) {
	query := newPostQueryBuilder().
		WhereURLKey(urlKey).
		WhereNotDeleted().
		Build()

	single := <-s.base.Repository.FindOne(ctx, postCollectionName, query)

	// Use the robust "decode-then-check" pattern
	var post models.Post
	if err := single.Decode(&post); err != nil {
		if errors.Is(err, dbi.ErrNoDocuments) {
			return nil, postsErrors.ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to decode post: %w", err)
	}

	return &post, nil
}

// GetPostsByUser retrieves posts by user ID
func (s *postService) GetPostsByUser(ctx context.Context, userID uuid.UUID, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Build query using query builder
	qb := newPostQueryBuilder().
		WhereOwner(userID).
		WhereNotDeleted()

	// Add additional filters
	if filter != nil {
		if filter.PostTypeId != nil {
			qb.WherePostType(*filter.PostTypeId)
		}
		if len(filter.Tags) > 0 {
			qb.WhereTagsIn(filter.Tags)
		}
	}
	query := qb.Build()

	// Build find options
	limit := int64(filter.Limit)
	skip := int64((filter.Page - 1) * filter.Limit)
	findOptions := &dbi.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  map[string]int{"created_date": -1}, // Default sort - use snake_case
	}

	// Query posts
	cursor := <-s.base.Repository.Find(ctx, postCollectionName, query, findOptions)
	if err := cursor.Error(); err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}
	defer cursor.Close()

	var posts []models.Post
	for cursor.Next() {
		var post models.Post
		if err := cursor.Decode(&post); err != nil {
			return nil, fmt.Errorf("failed to decode post: %w", err)
		}
		posts = append(posts, post)
	}

	// Get total count for pagination
	totalCount, err := s.getPostCount(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, &post)
	}

	// Attach latest comment preview for each post (limit 1)
	for i, post := range posts {
		query := &dbi.Query{
			Conditions: []dbi.Field{
				{
					Name:      "data->>'postId'",
					Value:     post.ObjectId.String(),
					Operator:  "=",
					IsJSONB:   true,
					JSONBCast: "::uuid",
				},
				{
					Name:      "data->>'deleted'",
					Value:     false,
					Operator:  "=",
					IsJSONB:   true,
					JSONBCast: "::boolean",
				},
			},
		}
		one := int64(1)
		opts := &dbi.FindOptions{
			Limit: &one,
			Sort:  map[string]int{"created_date": -1},
		}
		cur := <-s.base.Repository.Find(ctx, "comment", query, opts)
		if err := cur.Error(); err != nil {
			continue
		}
		defer cur.Close()
		if cur.Next() {
			var c commentsModels.Comment
			if err := cur.Decode(&c); err == nil {
				postResponses[i].LatestComments = []models.CommentPreview{
					{
						ObjectId:         c.ObjectId.String(),
						OwnerUserId:      c.OwnerUserId.String(),
						OwnerDisplayName: c.OwnerDisplayName,
						OwnerAvatar:      c.OwnerAvatar,
						Text:             c.Text,
						CreatedDate:      c.CreatedDate,
					},
				}
			}
		}
	}

	return &models.PostsListResponse{
		Posts:      postResponses,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
		HasMore:    int64(filter.Page*filter.Limit) < totalCount,
	}, nil
}

// SearchPosts searches posts by query string
func (s *postService) SearchPosts(ctx context.Context, query string, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Attempt cache first for search queries
	cacheKey := ""
	if s.cacheService != nil && filter != nil {
		cacheKey = s.generateSearchCacheKey(query, filter)
		if cachedResult, err := s.getCachedPosts(ctx, cacheKey); err == nil && cachedResult != nil {
			return cachedResult, nil
		}
	}

	// Build search query using query builder
	qb := newPostQueryBuilder().
		WhereNotDeleted().
		WhereSearchText(query)

	// Add additional filters
	if filter != nil {
		if filter.PostTypeId != nil {
			qb.WherePostType(*filter.PostTypeId)
		}
		if len(filter.Tags) > 0 {
			qb.WhereTagsIn(filter.Tags)
		}
	}
	queryObj := qb.Build()

	// Build find options
	limit := int64(filter.Limit)
	skip := int64((filter.Page - 1) * filter.Limit)
	findOptions := &dbi.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  map[string]int{"created_date": -1}, // Default sort - use snake_case
	}

	// Query posts
	cursor := <-s.base.Repository.Find(ctx, postCollectionName, queryObj, findOptions)
	if err := cursor.Error(); err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}
	defer cursor.Close()

	var posts []models.Post
	for cursor.Next() {
		var post models.Post
		if err := cursor.Decode(&post); err != nil {
			return nil, fmt.Errorf("failed to decode post: %w", err)
		}
		posts = append(posts, post)
	}

	// Get total count for pagination
	totalCount, err := s.getPostCount(ctx, queryObj)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, &post)
	}

	result := &models.PostsListResponse{
		Posts:      postResponses,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
		HasMore:    int64(filter.Page*filter.Limit) < totalCount,
	}

	// Cache the search result
	if s.cacheService != nil && cacheKey != "" {
		_ = s.cachePosts(ctx, cacheKey, result)
	}

	return result, nil
}

// UpdatePost updates an existing post
func (s *postService) UpdatePost(ctx context.Context, postID uuid.UUID, req *models.UpdatePostRequest, user *types.UserContext) error {
	// Verify ownership
	if err := s.ValidatePostOwnership(ctx, postID, user.UserID); err != nil {
		return err
	}

	// Build update fields
	updateFields := make(map[string]interface{})

	if req.Body != nil {
		updateFields["body"] = *req.Body
	}
	if req.Image != nil {
		updateFields["image"] = *req.Image
	}
	if req.ImageFullPath != nil {
		updateFields["imageFullPath"] = *req.ImageFullPath
	}
	if req.Video != nil {
		updateFields["video"] = *req.Video
	}
	if req.Thumbnail != nil {
		updateFields["thumbnail"] = *req.Thumbnail
	}
	if req.Tags != nil {
		updateFields["tags"] = *req.Tags
	}
	if req.Album != nil {
		updateFields["album"] = *req.Album
	}
	if req.DisableComments != nil {
		updateFields["disableComments"] = *req.DisableComments
	}
	if req.DisableSharing != nil {
		updateFields["disableSharing"] = *req.DisableSharing
	}
	if req.AccessUserList != nil {
		updateFields["accessUserList"] = *req.AccessUserList
	}
	if req.Permission != nil {
		updateFields["permission"] = *req.Permission
	}
	if req.Version != nil {
		updateFields["version"] = *req.Version
	}

	// Use clean abstraction for updates
	if err := s.UpdateFields(ctx, postID, updateFields); err != nil {
		return err
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
	// Build query using query builder
	query := newPostQueryBuilder().WhereOwner(userID).Build()
	updates := map[string]interface{}{
		"ownerDisplayName": displayName,
		"ownerAvatar":      avatar,
		// lastUpdated is automatically handled by repository at database level
	}

	// Use the repository's UpdateMany method
	result := <-s.base.Repository.UpdateMany(ctx, postCollectionName, query, updates, nil)
	return result.Error
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
	// Build a query that includes the ownership check for atomic update
	query := newPostQueryBuilder().
		WhereObjectID(objectId).
		WhereOwner(owner).
		WhereNotDeleted().
		Build()

	result := <-s.base.Repository.UpdateFields(ctx, postCollectionName, query, fields)
	return result.Error
}

// UpdateProfileForOwner updates display and avatar across all posts for an owner (SECURITY: validates ownership)
func (s *postService) UpdateProfileForOwner(ctx context.Context, owner uuid.UUID, displayName string, avatar string) error {
	// Build query using query builder
	query := newPostQueryBuilder().WhereOwner(owner).Build()
	updates := map[string]interface{}{
		"ownerDisplayName": displayName,
		"ownerAvatar":      avatar,
		// lastUpdated is automatically handled by repository at database level
	}

	// Use the repository's UpdateMany method
	result := <-s.base.Repository.UpdateMany(ctx, postCollectionName, query, updates, nil)
	return result.Error
}

// IncrementScore increments the post score using native database operation (no ownership validation needed for voting)
func (s *postService) IncrementScore(ctx context.Context, postID uuid.UUID, delta int, user *types.UserContext) error {
	// Build a query that includes the ownership check for atomic increment
	query := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereOwner(user.UserID).
		WhereNotDeleted().
		Build()

	increments := map[string]interface{}{"score": delta}
	result := <-s.base.Repository.IncrementFields(ctx, postCollectionName, query, increments)
	return result.Error
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

// IncrementViewCount increments the view count using native database operation with ownership validation
func (s *postService) IncrementViewCount(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	// Build a query that includes the ownership check for atomic increment
	query := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereOwner(user.UserID).
		WhereNotDeleted().
		Build()

	increments := map[string]interface{}{
		"viewCount": 1,
	}
	result := <-s.base.Repository.IncrementFields(ctx, postCollectionName, query, increments)
	return result.Error
}

// findPostForOwnershipCheck finds a post for ownership validation, regardless of deleted status.
// This is useful for operations like idempotent deletes or permanent data purges.
// It does NOT filter by deleted status, allowing it to find already-deleted posts.
func (s *postService) findPostForOwnershipCheck(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (*models.Post, error) {
	// The ONLY change from ValidatePostOwnership is removing "deleted": false.
	query := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereOwner(userID).
		Build()

	single := <-s.base.Repository.FindOne(ctx, postCollectionName, query)

	// Use the robust "decode-then-check" pattern.
	var post models.Post
	if err := single.Decode(&post); err != nil {
		if errors.Is(err, dbi.ErrNoDocuments) {
			// Cleanly return our standard not found error.
			return nil, postsErrors.ErrPostNotFound
		}
		// Any other error is a real database problem.
		return nil, fmt.Errorf("database error during ownership check: %w", err)
	}

	if single.NoResult() {
		return nil, postsErrors.ErrPostNotFound
	}

	return &post, nil
}

// ValidatePostOwnership validates that a user owns a specific post
func (s *postService) ValidatePostOwnership(ctx context.Context, postID uuid.UUID, userID uuid.UUID) error {
	query := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereOwner(userID).
		WhereNotDeleted().
		Build()

	single := <-s.base.Repository.FindOne(ctx, postCollectionName, query)

	// Use the robust "decode-then-check" pattern
	var post models.Post
	if err := single.Decode(&post); err != nil {
		if errors.Is(err, dbi.ErrNoDocuments) {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("database error during ownership validation: %w", err)
	}

	// If Decode succeeds, the document exists and ownership is validated
	return nil
}

// validatePostExists validates that a post exists without checking ownership
func (s *postService) validatePostExists(ctx context.Context, postID uuid.UUID) error {
	query := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereNotDeleted().
		Build()

	single := <-s.base.Repository.FindOne(ctx, postCollectionName, query)

	// Use the robust "decode-then-check" pattern
	var post models.Post
	if err := single.Decode(&post); err != nil {
		if errors.Is(err, dbi.ErrNoDocuments) {
			return postsErrors.ErrPostNotFound
		}
		return fmt.Errorf("database error during post existence validation: %w", err)
	}

	// If Decode succeeds, the document exists
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

func (s *postService) getPostCount(ctx context.Context, query *dbi.Query) (int64, error) {
	countResult := <-s.base.Repository.Count(ctx, postCollectionName, query)
	if countResult.Error != nil {
		return 0, countResult.Error
	}
	return countResult.Count, nil
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
				query := newPostQueryBuilder().
					WhereObjectID(post.ObjectId).
					WhereNotDeleted().
					Build()
				
				updates := map[string]interface{}{
					"commentCounter": actualCount,
				}
				result := <-s.base.Repository.UpdateFields(updateCtx, postCollectionName, query, updates)
				if result.Error != nil {
					log.Warn("Failed to update commentCounter for post %s: %v", post.ObjectId.String(), result.Error)
				}
			}()
		}
	}
	
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
}

// QueryPosts queries posts based on filter criteria
func (s *postService) QueryPosts(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Normalize pagination defaults early to keep cache keys stable
	if filter != nil {
		if filter.Limit <= 0 {
			filter.Limit = 10
		}
		if filter.Page <= 0 {
			filter.Page = 1
		}
	}

	// Attempt cache first for offset-based pagination
	cacheKey := ""
	if s.cacheService != nil && filter != nil {
		cacheKey = s.generateQueryCacheKey(filter)
		if cachedResult, err := s.getCachedPosts(ctx, cacheKey); err == nil && cachedResult != nil {
			return cachedResult, nil
		}
	}

	// Build query using the new Query Object pattern
	qb := newPostQueryBuilder()

	// Default: exclude deleted posts
	if filter == nil || filter.Deleted == nil {
		qb.WhereNotDeleted()
	} else if filter.Deleted != nil {
		qb.WhereDeleted(*filter.Deleted)
	}

	// Add additional filters
	if filter != nil {
		if filter.OwnerUserId != nil {
			qb.WhereOwner(*filter.OwnerUserId)
		}
		if filter.PostTypeId != nil {
			qb.WherePostType(*filter.PostTypeId)
		}
		if len(filter.Tags) > 0 {
			qb.WhereTagsIn(filter.Tags)
		}
		if filter.CreatedAfter != nil {
			qb.WhereCreatedAfter(filter.CreatedAfter.Unix())
		}
	}

	queryObj := qb.Build()

	// Build find options with stable pagination defaults
	page := 1
	var limit int64 = 10
	if filter != nil {
		limit = int64(filter.Limit)
		page = filter.Page
	}
	skip := int64(page-1) * limit

	findOptions := &dbi.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort: map[string]int{
			"created_date": -1,
			"object_id":    -1,
		}, // Default sort with deterministic tie-breaker
	}

	// Query posts using the new Query object
	cursor := <-s.base.Repository.Find(ctx, postCollectionName, queryObj, findOptions)
	if err := cursor.Error(); err != nil {
		return nil, fmt.Errorf("failed to find posts: %w", err)
	}
	defer cursor.Close()

	var posts []models.Post
	for cursor.Next() {
		var post models.Post
		if err := cursor.Decode(&post); err != nil {
			return nil, fmt.Errorf("failed to decode post: %w", err)
		}
		posts = append(posts, post)
	}

	// Get total count for pagination
	totalCount, err := s.getPostCount(ctx, queryObj)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, &post)
	}

	result := &models.PostsListResponse{
		Posts:      postResponses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      int(limit),
		HasMore:    int64(page)*limit < totalCount,
	}

	// Cache the result for subsequent identical queries when cache is enabled
	if s.cacheService != nil && cacheKey != "" {
		_ = s.cachePosts(ctx, cacheKey, result)
	}

	return result, nil
}

// QueryPostsWithCursor retrieves posts with cursor-based pagination
func (s *postService) QueryPostsWithCursor(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Validate and set defaults
	if filter == nil {
		filter = &models.PostQueryFilter{}
	}

	// Parse and validate sort parameters
	filter.SortField = models.ParseSortField(filter.SortField)
	filter.SortDirection = models.ParseSortDirection(filter.SortDirection)
	filter.Limit = models.ValidateLimit(filter.Limit)

	// Check cache first
	cacheKey := ""
	if s.cacheService != nil {
		cacheKey = s.generateCursorCacheKey(filter)
		if cachedResult, err := s.getCachedPosts(ctx, cacheKey); err == nil && cachedResult != nil {
			return cachedResult, nil
		}
	}

	// Build base query using query builder
	qb := newPostQueryBuilder()

	// Add base filters
	if filter.Deleted != nil {
		qb.WhereDeleted(*filter.Deleted)
	} else {
		qb.WhereNotDeleted()
	}

	if filter.OwnerUserId != nil {
		qb.WhereOwner(*filter.OwnerUserId)
	}
	if filter.PostTypeId != nil {
		qb.WherePostType(*filter.PostTypeId)
	}
	if len(filter.Tags) > 0 {
		qb.WhereTagsIn(filter.Tags)
	}
	if filter.CreatedAfter != nil {
		qb.WhereCreatedAfter(filter.CreatedAfter.Unix())
	}

	// Parse cursor data
	var cursorData *models.CursorData
	var err error

	if filter.Cursor != "" {
		cursorData, err = models.DecodeCursor(filter.Cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to decode cursor: %w", err)
		}
	} else if filter.AfterCursor != "" {
		cursorData, err = models.DecodeCursor(filter.AfterCursor)
		if err != nil {
			return nil, fmt.Errorf("failed to decode after cursor: %w", err)
		}
	} else if filter.BeforeCursor != "" {
		cursorData, err = models.DecodeCursor(filter.BeforeCursor)
		if err != nil {
			return nil, fmt.Errorf("failed to decode before cursor: %w", err)
		}
	}

	// Apply cursor logic using the query builder
	if cursorData != nil {
		// Map API sort field to database column/path
		sortColumn := "created_date" // Default
		if filter.SortField == "objectId" {
			sortColumn = "object_id"
		} else if filter.SortField == "createdDate" {
			sortColumn = "created_date"
		} else if filter.SortField == "lastUpdated" {
			sortColumn = "last_updated"
		} else if filter.SortField == "score" {
			sortColumn = "data->>'score'"
		} else if filter.SortField == "viewCount" {
			sortColumn = "data->>'viewCount'"
		}

		tieBreakerID, err := uuid.FromString(cursorData.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor ID: %w", err)
		}

		isBefore := filter.BeforeCursor != ""
		qb.WhereCursor(sortColumn, cursorData.Value, tieBreakerID, filter.SortDirection, isBefore)
	}

	// Build the final query
	queryObj := qb.Build()

	// Map API sort field to database column/path for CursorFindOptions
	sortColumn := "created_date" // Default
	if filter.SortField == "objectId" {
		sortColumn = "object_id"
	} else if filter.SortField == "createdDate" {
		sortColumn = "created_date"
	} else if filter.SortField == "lastUpdated" {
		sortColumn = "last_updated"
	} else if filter.SortField == "score" {
		sortColumn = "data->>'score'"
	} else if filter.SortField == "viewCount" {
		sortColumn = "data->>'viewCount'"
	}

	// Build cursor find options
	limit := int64(filter.Limit + 1) // Request one extra to check if there are more results
	cursorOptions := &dbi.CursorFindOptions{
		Limit:         &limit,
		SortField:     sortColumn, // Use mapped snake_case column name
		SortDirection: filter.SortDirection,
	}

	// Query posts with cursor
	cursor := <-s.base.Repository.FindWithCursor(ctx, postCollectionName, queryObj, cursorOptions)
	if err := cursor.Error(); err != nil {
		log.Error("QueryPostsWithCursor: Failed to find posts with cursor: %v", err)
		return nil, fmt.Errorf("failed to find posts with cursor: %w", err)
	}
	defer cursor.Close()

	// Decode posts
	var posts []models.Post
	for cursor.Next() {
		var post models.Post
		if err := cursor.Decode(&post); err != nil {
			return nil, fmt.Errorf("failed to decode post: %w", err)
		}
		posts = append(posts, post)
	}

	// Determine pagination state
	hasNext := len(posts) > filter.Limit
	hasPrev := filter.AfterCursor != "" || filter.Cursor != ""

	// Remove the extra item if we have more than requested
	if hasNext {
		posts = posts[:filter.Limit]
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, &post)
	}

	// Generate cursor values
	var nextCursor, prevCursor string
	if hasNext && len(posts) > 0 {
		cursor, err := models.CreateCursorFromPost(&posts[len(posts)-1], filter.SortField, filter.SortDirection)
		if err != nil {
			return nil, fmt.Errorf("failed to create next cursor: %w", err)
		}
		nextCursor = cursor
	}
	if hasPrev && len(posts) > 0 {
		cursor, err := models.CreateCursorFromPost(&posts[0], filter.SortField, filter.SortDirection)
		if err != nil {
			return nil, fmt.Errorf("failed to create prev cursor: %w", err)
		}
		prevCursor = cursor
	}

	result := &models.PostsListResponse{
		Posts:      postResponses,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Limit:      filter.Limit,
	}

	// Cache the result
	if s.cacheService != nil && cacheKey != "" {
		if err := s.cachePosts(ctx, cacheKey, result); err != nil {
			// Log but don't fail the request if caching fails
			log.Warn("Failed to cache posts result: %v", err)
		}
	}

	return result, nil
}

// SearchPostsWithCursor retrieves posts matching search criteria with cursor-based pagination
func (s *postService) SearchPostsWithCursor(ctx context.Context, searchTerm string, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Validate and set defaults
	if filter == nil {
		filter = &models.PostQueryFilter{}
	}

	// Parse and validate sort parameters
	filter.SortField = models.ParseSortField(filter.SortField)
	filter.SortDirection = models.ParseSortDirection(filter.SortDirection)
	filter.Limit = models.ValidateLimit(filter.Limit)

	// Check cache first (include search term in cache key)
	cacheKey := ""
	if s.cacheService != nil {
		cacheKey = s.generateSearchCacheKey(searchTerm, filter)
		if cachedResult, err := s.getCachedPosts(ctx, cacheKey); err == nil && cachedResult != nil {
			return cachedResult, nil
		}
	}

	// Build search query using query builder
	qb := newPostQueryBuilder()

	// Add base filters
	qb.WhereNotDeleted()
	qb.WhereSearchText(searchTerm)

	// Add additional filters
	if filter != nil {
		if filter.OwnerUserId != nil {
			qb.WhereOwner(*filter.OwnerUserId)
		}
		if filter.PostTypeId != nil {
			qb.WherePostType(*filter.PostTypeId)
		}
		if len(filter.Tags) > 0 {
			// Combine search tags with filter tags (must have all specified tags)
			qb.WhereTagsAll(filter.Tags)
		}
	}

	// Create a modified filter for cursor-based search
	searchQueryFilter := &models.PostQueryFilter{
		SortField:     filter.SortField,
		SortDirection: filter.SortDirection,
		Limit:         filter.Limit,
		Cursor:        filter.Cursor,
		AfterCursor:   filter.AfterCursor,
		BeforeCursor:  filter.BeforeCursor,
	}

	// Validate and set defaults
	searchQueryFilter.SortField = models.ParseSortField(searchQueryFilter.SortField)
	searchQueryFilter.SortDirection = models.ParseSortDirection(searchQueryFilter.SortDirection)
	searchQueryFilter.Limit = models.ValidateLimit(searchQueryFilter.Limit)

	// Parse cursor data
	var cursorData *models.CursorData
	var err error

	if searchQueryFilter.Cursor != "" {
		cursorData, err = models.DecodeCursor(searchQueryFilter.Cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to decode cursor: %w", err)
		}
	} else if searchQueryFilter.AfterCursor != "" {
		cursorData, err = models.DecodeCursor(searchQueryFilter.AfterCursor)
		if err != nil {
			return nil, fmt.Errorf("failed to decode after cursor: %w", err)
		}
	} else if searchQueryFilter.BeforeCursor != "" {
		cursorData, err = models.DecodeCursor(searchQueryFilter.BeforeCursor)
		if err != nil {
			return nil, fmt.Errorf("failed to decode before cursor: %w", err)
		}
	}

	// Apply cursor logic using the query builder
	if cursorData != nil {
		// Map API sort field to database column/path
		sortColumn := "created_date" // Default
		if searchQueryFilter.SortField == "objectId" {
			sortColumn = "object_id"
		} else if searchQueryFilter.SortField == "createdDate" {
			sortColumn = "created_date"
		} else if searchQueryFilter.SortField == "lastUpdated" {
			sortColumn = "last_updated"
		} else if searchQueryFilter.SortField == "score" {
			sortColumn = "data->>'score'"
		} else if searchQueryFilter.SortField == "viewCount" {
			sortColumn = "data->>'viewCount'"
		}

		tieBreakerID, err := uuid.FromString(cursorData.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor ID: %w", err)
		}

		isBefore := filter.BeforeCursor != ""
		if searchQueryFilter.BeforeCursor != "" {
			isBefore = true
		}
		qb.WhereCursor(sortColumn, cursorData.Value, tieBreakerID, searchQueryFilter.SortDirection, isBefore)
	}

	// Build the final query
	queryObj := qb.Build()

	// Map API sort field to database column/path for CursorFindOptions
	// Note: sortColumn is already set from cursor condition mapping above, but we need to ensure it's set here too
	searchSortColumn := "created_date" // Default
	if searchQueryFilter.SortField == "objectId" {
		searchSortColumn = "object_id"
	} else if searchQueryFilter.SortField == "createdDate" {
		searchSortColumn = "created_date"
	} else if searchQueryFilter.SortField == "lastUpdated" {
		searchSortColumn = "last_updated"
	} else if searchQueryFilter.SortField == "score" {
		searchSortColumn = "data->>'score'"
	} else if searchQueryFilter.SortField == "viewCount" {
		searchSortColumn = "data->>'viewCount'"
	}

	// Build cursor find options
	limit := int64(searchQueryFilter.Limit + 1) // Request one extra to check if there are more results
	cursorOptions := &dbi.CursorFindOptions{
		Limit:         &limit,
		SortField:     searchSortColumn, // Use mapped snake_case column name
		SortDirection: searchQueryFilter.SortDirection,
	}

	// Query posts with cursor
	result := <-s.base.Repository.FindWithCursor(ctx, postCollectionName, queryObj, cursorOptions)
	if err := result.Error(); err != nil {
		return nil, fmt.Errorf("failed to search posts with cursor: %w", err)
	}
	defer result.Close()

	// Decode posts
	var posts []models.Post
	for result.Next() {
		var post models.Post
		if err := result.Decode(&post); err != nil {
			return nil, fmt.Errorf("failed to decode post: %w", err)
		}
		posts = append(posts, post)
	}

	// Determine pagination state
	hasNext := len(posts) > searchQueryFilter.Limit
	hasPrev := searchQueryFilter.AfterCursor != "" || searchQueryFilter.Cursor != ""

	// Remove the extra item if we have more than requested
	if hasNext {
		posts = posts[:searchQueryFilter.Limit]
	}

	// Convert to response format
	postResponses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		postResponses[i] = s.ConvertPostToResponse(ctx, &post)
	}

	// Generate cursor values
	var nextCursor, prevCursor string
	if hasNext && len(posts) > 0 {
		cursor, err := models.CreateCursorFromPost(&posts[len(posts)-1], searchQueryFilter.SortField, searchQueryFilter.SortDirection)
		if err != nil {
			return nil, fmt.Errorf("failed to create next cursor: %w", err)
		}
		nextCursor = cursor
	}
	if hasPrev && len(posts) > 0 {
		cursor, err := models.CreateCursorFromPost(&posts[0], searchQueryFilter.SortField, searchQueryFilter.SortDirection)
		if err != nil {
			return nil, fmt.Errorf("failed to create prev cursor: %w", err)
		}
		prevCursor = cursor
	}

	searchResult := &models.PostsListResponse{
		Posts:      postResponses,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Limit:      searchQueryFilter.Limit,
	}

	// Cache the search result
	if s.cacheService != nil && cacheKey != "" {
		if err := s.cachePosts(ctx, cacheKey, searchResult); err != nil {
			// Log but don't fail the request if caching fails
			log.Warn("Failed to cache search result: %v", err)
		}
	}

	return searchResult, nil
}

// UpdateFields updates post fields using field-based syntax (maps to native DB operations)
func (s *postService) UpdateFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error {
	// lastUpdated is automatically handled by repository at database level
	// Use the field-based abstraction method
	query := newPostQueryBuilder().WhereObjectID(postID).Build()
	result := <-s.base.Repository.UpdateFields(ctx, postCollectionName, query, updates)
	return result.Error
}

// IncrementFields increments numeric fields using field-based syntax (maps to native DB operations)
func (s *postService) IncrementFields(ctx context.Context, postID uuid.UUID, increments map[string]interface{}) error {
	// Use the field-based abstraction method
	query := newPostQueryBuilder().WhereObjectID(postID).Build()
	result := <-s.base.Repository.IncrementFields(ctx, postCollectionName, query, increments)
	return result.Error
}

// UpdateAndIncrementFields performs both update and increment operations
func (s *postService) UpdateAndIncrementFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error {
	// lastUpdated is automatically handled by repository at database level
	// Use the field-based abstraction method
	query := newPostQueryBuilder().WhereObjectID(postID).Build()
	result := <-s.base.Repository.UpdateAndIncrement(ctx, postCollectionName, query, updates, increments)
	return result.Error
}

// SetCommentDisabled sets comment disabled status with ownership validation
func (s *postService) SetCommentDisabled(ctx context.Context, postID uuid.UUID, disabled bool, user *types.UserContext) error {
	// Build a query that includes the ownership check for atomic update
	query := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereOwner(user.UserID).
		WhereNotDeleted().
		Build()

	updates := map[string]interface{}{"disableComments": disabled}
	result := <-s.base.Repository.UpdateFields(ctx, postCollectionName, query, updates)
	return result.Error
}

// SetSharingDisabled sets sharing disabled status with ownership validation
func (s *postService) SetSharingDisabled(ctx context.Context, postID uuid.UUID, disabled bool, user *types.UserContext) error {
	// Build a query that includes the ownership check for atomic update
	query := newPostQueryBuilder().
		WhereObjectID(postID).
		WhereOwner(user.UserID).
		WhereNotDeleted().
		Build()

	updates := map[string]interface{}{"disableSharing": disabled}
	result := <-s.base.Repository.UpdateFields(ctx, postCollectionName, query, updates)
	return result.Error
}

// DeletePost permanently deletes a post
func (s *postService) DeletePost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
	// Verify ownership
	if err := s.ValidatePostOwnership(ctx, postID, user.UserID); err != nil {
		return err
	}

	// Delete the post
	query := newPostQueryBuilder().WhereObjectID(postID).Build()
	res := <-s.base.Repository.Delete(ctx, postCollectionName, query)
	if err := res.Error; err != nil {
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

	// 3. If the post exists and is not yet deleted, perform the update.
	updates := map[string]interface{}{
		"deleted":     true,
		"deletedDate": time.Now().Unix(),
	}

	if err := s.UpdateFields(ctx, postID, updates); err != nil {
		return err // Return the update error if it occurs.
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
	// Build a query that includes the ownership check for atomic delete
	query := newPostQueryBuilder().
		WhereObjectID(objectId).
		WhereOwner(owner).
		WhereNotDeleted().
		Build()

	result := <-s.base.Repository.Delete(ctx, postCollectionName, query)
	return result.Error
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

	// Determine the comparison operator for items that come BEFORE this post in the sort order
	// For desc: items with greater value come before; For asc: items with smaller value come before
	primaryOp := "$lt"
	idOp := "$lt"
	if direction == "desc" {
		primaryOp = "$gt"
		idOp = "$gt"
	}

	// Build query using query builder
	qb := newPostQueryBuilder().WhereNotDeleted()

	// Compute the sort value of the current post
	var sortValue interface{}
	var sortFieldName string
	switch sortField {
	case "createdDate":
		sortValue = post.CreatedDate
		sortFieldName = "created_date"
	case "lastUpdated":
		sortValue = post.LastUpdated
		sortFieldName = "last_updated"
	case "score":
		sortValue = post.Score
		sortFieldName = "data->>'score'"
	case "viewCount":
		sortValue = post.ViewCount
		sortFieldName = "data->>'viewCount'"
	case "commentCounter":
		sortValue = post.CommentCounter
		sortFieldName = "data->>'commentCounter'"
	case "objectId":
		sortValue = post.ObjectId.String()
		sortFieldName = "object_id"
	default:
		return nil, fmt.Errorf("unsupported sort field: %s", sortField)
	}

	// Build a compound comparison for accurate ordering with ID tiebreaker
	if sortField == "objectId" {
		// Simple comparison for object_id
		operator := "<"
		if idOp == "$gt" {
			operator = ">"
		}
		qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
			Name:     "object_id",
			Value:    sortValue,
			Operator: operator,
			IsJSONB:  false,
		})
	} else {
		// Compound OR condition: (sortField < value) OR (sortField = value AND object_id < id)
		operator := "<"
		if primaryOp == "$gt" {
			operator = ">"
		}
		idOperator := "<"
		if idOp == "$gt" {
			idOperator = ">"
		}

		// First OR condition: sortField < value
		orFields1 := []dbi.Field{
			{
				Name:     sortFieldName,
				Value:    sortValue,
				Operator: operator,
				IsJSONB:  sortField != "createdDate" && sortField != "lastUpdated",
			},
		}

		// Second OR condition: sortField = value AND object_id < id
		orFields2 := []dbi.Field{
			{
				Name:     sortFieldName,
				Value:    sortValue,
				Operator: "=",
				IsJSONB:  sortField != "createdDate" && sortField != "lastUpdated",
			},
			{
				Name:     "object_id",
				Value:    post.ObjectId.String(),
				Operator: idOperator,
				IsJSONB:  false,
			},
		}

		qb.query.OrGroups = append(qb.query.OrGroups, orFields1, orFields2)
	}

	// Count how many posts come before this one in the chosen order
	countQuery := qb.Build()
	beforeCount, err := s.getPostCount(ctx, countQuery)
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
