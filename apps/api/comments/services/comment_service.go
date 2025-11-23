package services

import (
    "context"
    "errors"
    "fmt"
    "time"

    uuid "github.com/gofrs/uuid"

    "github.com/qolzam/telar/apps/api/comments/common"
    commentsErrors "github.com/qolzam/telar/apps/api/comments/errors"
    "github.com/qolzam/telar/apps/api/comments/models"
    "github.com/qolzam/telar/apps/api/internal/cache"
    dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
    "github.com/qolzam/telar/apps/api/internal/platform"
    platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
    "github.com/qolzam/telar/apps/api/internal/pkg/log"
    "github.com/qolzam/telar/apps/api/internal/types"
    "github.com/qolzam/telar/apps/api/internal/utils"
    sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
)

const (
    commentCollectionName = "comment"

    defaultCommentLimit = 10
    maxCommentLimit     = 100
    defaultCommentPage  = 1
)

// commentService implements the CommentService interface using the new repository patterns.
type commentService struct {
    base             *platform.BaseService
    cacheService     *cache.GenericCacheService
    config           *platformconfig.Config
    postStatsUpdater sharedInterfaces.PostStatsUpdater
}

// Ensure commentService implements sharedInterfaces.CommentCounter interface
var _ sharedInterfaces.CommentCounter = (*commentService)(nil)

// commentQueryBuilder provides fluent helpers for building comment queries.
type commentQueryBuilder struct {
    query *dbi.Query
}

func newCommentQueryBuilder() *commentQueryBuilder {
    return &commentQueryBuilder{
        query: &dbi.Query{
            Conditions: []dbi.Field{},
            OrGroups:   [][]dbi.Field{},
        },
    }
}

func (b *commentQueryBuilder) WhereObjectID(objectID uuid.UUID) *commentQueryBuilder {
    b.query.Conditions = append(b.query.Conditions, dbi.Field{
        Name:     "object_id",
        Value:    objectID,
        Operator: "=",
    })
    return b
}

func (b *commentQueryBuilder) WhereOwner(userID uuid.UUID) *commentQueryBuilder {
    b.query.Conditions = append(b.query.Conditions, dbi.Field{
        Name:     "owner_user_id",
        Value:    userID,
        Operator: "=",
    })
    return b
}

func (b *commentQueryBuilder) WherePostID(postID uuid.UUID) *commentQueryBuilder {
    b.query.Conditions = append(b.query.Conditions, dbi.Field{
        Name:      "data->>'postId'",
        Value:     postID.String(),
        Operator:  "=",
        IsJSONB:   true,
        JSONBCast: "::uuid",
    })
    return b
}

func (b *commentQueryBuilder) WhereDeleted(deleted bool) *commentQueryBuilder {
    b.query.Conditions = append(b.query.Conditions, dbi.Field{
        Name:      "data->>'deleted'",
        Value:     deleted,
        Operator:  "=",
        IsJSONB:   true,
        JSONBCast: "::boolean",
    })
    return b
}

func (b *commentQueryBuilder) WhereNotDeleted() *commentQueryBuilder {
    return b.WhereDeleted(false)
}

func (b *commentQueryBuilder) WhereCreatedAfter(t time.Time) *commentQueryBuilder {
    b.query.Conditions = append(b.query.Conditions, dbi.Field{
        Name:     "created_date",
        Value:    t.Unix(),
        Operator: ">=",
    })
    return b
}

func (b *commentQueryBuilder) WhereCreatedBefore(t time.Time) *commentQueryBuilder {
    b.query.Conditions = append(b.query.Conditions, dbi.Field{
        Name:     "created_date",
        Value:    t.Unix(),
        Operator: "<=",
    })
    return b
}

// WhereParentEquals filters comments by a specific parentCommentId
func (b *commentQueryBuilder) WhereParentEquals(parentID uuid.UUID) *commentQueryBuilder {
    b.query.Conditions = append(b.query.Conditions, dbi.Field{
        Name:      "data->>'parentCommentId'",
        Value:     parentID.String(),
        Operator:  "=",
        IsJSONB:   true,
        JSONBCast: "::uuid",
    })
    return b
}

// WhereParentIsNull filters to only root comments (no parent)
func (b *commentQueryBuilder) WhereParentIsNull() *commentQueryBuilder {
    b.query.Conditions = append(b.query.Conditions, dbi.Field{
        Name:      "data->>'parentCommentId'",
        Operator:  "IS NULL",
        IsJSONB:   true,
        JSONBCast: "",
    })
    return b
}

func (b *commentQueryBuilder) Build() *dbi.Query {
    return b.query
}

// GetRootCommentCount counts root comments (non-reply comments) for a post
// This is a public method that implements the CommentCounter interface
func (s *commentService) GetRootCommentCount(ctx context.Context, postID uuid.UUID) (int64, error) {
    query := newCommentQueryBuilder().
        WherePostID(postID).
        WhereParentIsNull().
        WhereNotDeleted().
        Build()

    return s.getCommentCount(ctx, query)
}

// NewCommentService wires the comment service with its dependencies.
func NewCommentService(base *platform.BaseService, cfg *platformconfig.Config, postStatsUpdater sharedInterfaces.PostStatsUpdater) CommentService {
    cacheService := cache.NewGenericCacheServiceFor("comments")
    return &commentService{
        base:             base,
        cacheService:     cacheService,
        config:           cfg,
        postStatsUpdater: postStatsUpdater,
    }
}

func (s *commentService) generateCursorCacheKey(filter *models.CommentQueryFilter) string {
    if s.cacheService == nil || filter == nil || filter.PostId == nil {
        return ""
    }

    limit := filter.Limit
    if limit <= 0 {
        limit = defaultCommentLimit
    }
    page := filter.Page
    if page <= 0 {
        page = defaultCommentPage
    }

    key := common.BuildCommentCacheKey(*filter.PostId, filter.OwnerUserId, page, limit)
    return "cursor:" + key
}

func (s *commentService) getCachedComments(ctx context.Context, cacheKey string) (*models.CommentsListResponse, error) {
    if s.cacheService == nil || cacheKey == "" {
        return nil, nil
    }

    var result models.CommentsListResponse
    if err := s.cacheService.GetCached(ctx, cacheKey, &result); err != nil {
        return nil, err
    }
    return &result, nil
}

func (s *commentService) cacheComments(ctx context.Context, cacheKey string, result *models.CommentsListResponse) {
    if s.cacheService == nil || cacheKey == "" || result == nil {
        return
    }
    _ = s.cacheService.CacheData(ctx, cacheKey, result, time.Hour)
}

func (s *commentService) invalidateUserComments(ctx context.Context, userID uuid.UUID) {
    if s.cacheService == nil {
        return
    }
    pattern := "cursor:comments:*:owner:" + userID.String() + "*"
    s.cacheService.InvalidatePattern(ctx, pattern)
}

func (s *commentService) invalidatePostComments(ctx context.Context, postID uuid.UUID) {
    if s.cacheService == nil {
        return
    }
    pattern := "cursor:comments:" + postID.String() + "*"
    s.cacheService.InvalidatePattern(ctx, pattern)
}

func (s *commentService) invalidateAllComments(ctx context.Context) {
    if s.cacheService == nil {
        return
    }
    s.cacheService.InvalidatePattern(ctx, "cursor:comments:*")
}

// CreateComment creates a new comment entity.
func (s *commentService) CreateComment(ctx context.Context, req *models.CreateCommentRequest, user *types.UserContext) (*models.Comment, error) {
    if req == nil {
        return nil, fmt.Errorf("create comment request is required")
    }
    if user == nil {
        return nil, fmt.Errorf("user context is required")
    }

    commentID, err := uuid.NewV4()
    if err != nil {
        return nil, fmt.Errorf("failed to generate comment ID: %w", err)
    }

    now := utils.UTCNowUnix()
    comment := &models.Comment{
        ObjectId:         commentID,
        Score:            0,
        OwnerUserId:      user.UserID,
        OwnerDisplayName: user.DisplayName,
        OwnerAvatar:      user.Avatar,
        PostId:           req.PostId,
		ParentCommentId:  req.ParentCommentId,
        Text:             req.Text,
        Deleted:          false,
        DeletedDate:      0,
        CreatedDate:      now,
        LastUpdated:      now,
    }

    result := <-s.base.Repository.Save(ctx, commentCollectionName, comment.ObjectId, comment.OwnerUserId, comment.CreatedDate, comment.LastUpdated, comment)
    if err := result.Error; err != nil {
        return nil, fmt.Errorf("failed to save comment: %w", err)
    }

    // Update post.commentCounter only for root comments (not replies)
    // This maintains the denormalized count for performance without fetching all comments
    if req.ParentCommentId == nil || *req.ParentCommentId == uuid.Nil {
        if err := s.updatePostCommentCounter(ctx, req.PostId, 1); err != nil {
            log.Warn("Failed to update post commentCounter for post %s: %v", req.PostId.String(), err)
        } else {
            log.Info("Successfully incremented commentCounter for post %s (delta: +1)", req.PostId.String())
        }
    }

    s.invalidateUserComments(ctx, user.UserID)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)

    return comment, nil
}

// CreateIndex proxies to the repository index creation.
func (s *commentService) CreateIndex(ctx context.Context, indexes map[string]interface{}) error {
    return <-s.base.Repository.CreateIndex(ctx, commentCollectionName, indexes)
}

// CreateIndexes creates default indexes for the comments collection.
func (s *commentService) CreateIndexes(ctx context.Context) error {
    indexes := map[string]interface{}{
        "objectId":    1,
        "ownerUserId": 1,
    }
    return s.CreateIndex(ctx, indexes)
}

// GetComment returns a single non-deleted comment by ID.
func (s *commentService) GetComment(ctx context.Context, commentID uuid.UUID) (*models.Comment, error) {
    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereNotDeleted().
        Build()

    single := <-s.base.Repository.FindOne(ctx, commentCollectionName, query)
    var comment models.Comment
    if err := single.Decode(&comment); err != nil {
        if errors.Is(err, dbi.ErrNoDocuments) {
            return nil, commentsErrors.ErrCommentNotFound
        }
        return nil, fmt.Errorf("failed to decode comment: %w", err)
    }

    return &comment, nil
}

// GetCommentsByPost lists comments for a specific post.
func (s *commentService) GetCommentsByPost(ctx context.Context, postID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
    if filter == nil {
        filter = &models.CommentQueryFilter{}
    }
    filter.PostId = &postID
    return s.QueryComments(ctx, filter)
}

// GetCommentsByUser lists comments created by a specific user.
func (s *commentService) GetCommentsByUser(ctx context.Context, userID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
    if filter == nil {
        filter = &models.CommentQueryFilter{}
    }
    filter.OwnerUserId = &userID
    return s.QueryComments(ctx, filter)
}

// QueryComments executes a filtered paginated query.
func (s *commentService) QueryComments(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
    if filter == nil {
        return nil, fmt.Errorf("filter is required")
    }

    sanitizePagination(filter)

    cacheKey := s.generateCursorCacheKey(filter)
    if cached, err := s.getCachedComments(ctx, cacheKey); err == nil && cached != nil {
        return cached, nil
    }

    qb := newCommentQueryBuilder()
    applyFilterToBuilder(qb, filter)
    query := qb.Build()

    limit := int64(filter.Limit)
    skip := int64((filter.Page - 1) * filter.Limit)
    findOptions := &dbi.FindOptions{
        Limit: &limit,
        Skip:  &skip,
        Sort:  map[string]int{"created_date": -1},
    }

    cursor := <-s.base.Repository.Find(ctx, commentCollectionName, query, findOptions)
    if err := cursor.Error(); err != nil {
        return nil, fmt.Errorf("failed to query comments: %w", err)
    }
    defer cursor.Close()

    var comments []models.Comment
    for cursor.Next() {
        var comment models.Comment
        if err := cursor.Decode(&comment); err != nil {
            return nil, fmt.Errorf("failed to decode comment: %w", err)
        }
        comments = append(comments, comment)
    }

    totalCount, err := s.getCommentCount(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to count comments: %w", err)
    }

    responses := make([]models.CommentResponse, len(comments))
    for i, comment := range comments {
        responses[i] = s.convertToCommentResponse(&comment)
    }

    result := &models.CommentsListResponse{
        Comments: responses,
        Count:    int(totalCount),
        Page:     filter.Page,
        Limit:    filter.Limit,
        HasMore:  int64(filter.Page*filter.Limit) < totalCount,
    }

    s.cacheComments(ctx, cacheKey, result)
    return result, nil
}

// QueryCommentsWithCursor currently reuses offset pagination.
func (s *commentService) QueryCommentsWithCursor(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
    return s.QueryComments(ctx, filter)
}

// UpdateComment updates a comment's text for the owning user.
func (s *commentService) UpdateComment(ctx context.Context, commentID uuid.UUID, req *models.UpdateCommentRequest, user *types.UserContext) error {
    if req == nil {
        return fmt.Errorf("update comment request is required")
    }
    if user == nil {
        return fmt.Errorf("user context is required")
    }

    comment, err := s.fetchOwnedComment(ctx, commentID, user.UserID)
    if err != nil {
        return err
    }

    updates := map[string]interface{}{
        "text":        req.Text,
        "lastUpdated": utils.UTCNowUnix(),
    }

    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereOwner(user.UserID).
        WhereNotDeleted().
        Build()

    result := <-s.base.Repository.UpdateFields(ctx, commentCollectionName, query, updates)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to update comment: %w", err)
    }

    s.invalidateUserComments(ctx, user.UserID)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)
    return nil
}

// UpdateCommentProfile updates profile information for all comments created by a user.
func (s *commentService) UpdateCommentProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
    updates := map[string]interface{}{}
    if displayName != "" {
        updates["ownerDisplayName"] = displayName
    }
    if avatar != "" {
        updates["ownerAvatar"] = avatar
    }
    if len(updates) == 0 {
        return nil
    }
    updates["lastUpdated"] = utils.UTCNowUnix()

    query := newCommentQueryBuilder().
        WhereOwner(userID).
        WhereNotDeleted().
        Build()

    result := <-s.base.Repository.UpdateFields(ctx, commentCollectionName, query, updates)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to update comment profile: %w", err)
    }

    s.invalidateUserComments(ctx, userID)
    s.invalidateAllComments(ctx)
    return nil
}

// IncrementScore adjusts the score field for a comment.
func (s *commentService) IncrementScore(ctx context.Context, commentID uuid.UUID, delta int, user *types.UserContext) error {
    if user == nil {
        return fmt.Errorf("user context is required")
    }

    increments := map[string]interface{}{"score": delta}
    return s.IncrementFields(ctx, commentID, increments)
}

// DeleteComment removes a comment permanently.
func (s *commentService) DeleteComment(ctx context.Context, commentID uuid.UUID, postID uuid.UUID, user *types.UserContext) error {
    if user == nil {
        return fmt.Errorf("user context is required")
    }

    comment, err := s.fetchOwnedComment(ctx, commentID, user.UserID)
    if err != nil {
        return err
    }

    if postID != uuid.Nil && comment.PostId != postID {
        return fmt.Errorf("comment does not belong to the provided post")
    }

    // Check if this is a root comment before deletion (for commentCounter update)
    isRootComment := comment.ParentCommentId == nil || *comment.ParentCommentId == uuid.Nil

    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereOwner(user.UserID).
        Build()

    result := <-s.base.Repository.Delete(ctx, commentCollectionName, query)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to delete comment: %w", err)
    }

    // Update post.commentCounter only for root comments (not replies)
    // This maintains the denormalized count for performance
    if isRootComment {
        if err := s.updatePostCommentCounter(ctx, comment.PostId, -1); err != nil {
            log.Warn("Failed to update post commentCounter for post %s: %v", comment.PostId.String(), err)
        } else {
            log.Info("Successfully decremented commentCounter for post %s (delta: -1)", comment.PostId.String())
        }
    }

    s.invalidateUserComments(ctx, user.UserID)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)
    return nil
}

// DeleteCommentsByPost removes all comments for a given post.
func (s *commentService) DeleteCommentsByPost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
    if user == nil {
        return fmt.Errorf("user context is required")
    }

    query := newCommentQueryBuilder().
        WherePostID(postID).
        Build()

    result := <-s.base.Repository.Delete(ctx, commentCollectionName, query)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to delete comments by post: %w", err)
    }

    s.invalidatePostComments(ctx, postID)
    s.invalidateAllComments(ctx)
    return nil
}

// SoftDeleteComment marks a comment as deleted without removing it permanently.
func (s *commentService) SoftDeleteComment(ctx context.Context, commentID uuid.UUID, user *types.UserContext) error {
    if user == nil {
        return fmt.Errorf("user context is required")
    }

    comment, err := s.fetchOwnedComment(ctx, commentID, user.UserID)
    if err != nil {
        return err
    }

    updates := map[string]interface{}{
        "deleted":     true,
        "deletedDate": utils.UTCNowUnix(),
        "lastUpdated": utils.UTCNowUnix(),
    }

    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereOwner(user.UserID).
        WhereNotDeleted().
        Build()

    result := <-s.base.Repository.UpdateFields(ctx, commentCollectionName, query, updates)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to soft delete comment: %w", err)
    }

    s.invalidateUserComments(ctx, user.UserID)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)
    return nil
}

// DeleteByOwner removes a comment when ownership is known (admin utility).
func (s *commentService) DeleteByOwner(ctx context.Context, owner uuid.UUID, objectID uuid.UUID) error {
    comment, err := s.fetchOwnedComment(ctx, objectID, owner)
    if err != nil {
        return err
    }

    query := newCommentQueryBuilder().
        WhereObjectID(objectID).
        WhereOwner(owner).
        Build()

    result := <-s.base.Repository.Delete(ctx, commentCollectionName, query)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to delete comment by owner: %w", err)
    }

    s.invalidateUserComments(ctx, owner)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)
    return nil
}

// ValidateCommentOwnership confirms that the comment belongs to the user.
func (s *commentService) ValidateCommentOwnership(ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error {
    _, err := s.fetchOwnedComment(ctx, commentID, userID)
    return err
}

func (s *commentService) convertToCommentResponse(comment *models.Comment) models.CommentResponse {
    return models.CommentResponse{
        ObjectId:         comment.ObjectId.String(),
        Score:            comment.Score,
        OwnerUserId:      comment.OwnerUserId.String(),
        OwnerDisplayName: comment.OwnerDisplayName,
        OwnerAvatar:      comment.OwnerAvatar,
        PostId:           comment.PostId.String(),
        Text:             comment.Text,
        Deleted:          comment.Deleted,
        DeletedDate:      comment.DeletedDate,
        CreatedDate:      comment.CreatedDate,
        LastUpdated:      comment.LastUpdated,
    }
}

// Backward compatibility helpers

func (s *commentService) SetField(ctx context.Context, objectID uuid.UUID, field string, value interface{}) error {
    updates := map[string]interface{}{field: value}
    return s.UpdateFields(ctx, objectID, updates)
}

func (s *commentService) IncrementField(ctx context.Context, objectID uuid.UUID, field string, delta int) error {
    increments := map[string]interface{}{field: delta}
    return s.IncrementFields(ctx, objectID, increments)
}

func (s *commentService) UpdateByOwner(ctx context.Context, objectID uuid.UUID, owner uuid.UUID, fields map[string]interface{}) error {
    return s.UpdateFieldsWithOwnership(ctx, objectID, owner, fields)
}

func (s *commentService) UpdateProfileForOwner(ctx context.Context, owner uuid.UUID, displayName, avatar string) error {
    return s.UpdateCommentProfile(ctx, owner, displayName, avatar)
}

func (s *commentService) UpdateFields(ctx context.Context, commentID uuid.UUID, updates map[string]interface{}) error {
    if len(updates) == 0 {
        return nil
    }

    updates["lastUpdated"] = utils.UTCNowUnix()
    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereNotDeleted().
        Build()

    result := <-s.base.Repository.UpdateFields(ctx, commentCollectionName, query, updates)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to update fields: %w", err)
    }

    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) IncrementFields(ctx context.Context, commentID uuid.UUID, increments map[string]interface{}) error {
    if len(increments) == 0 {
        return nil
    }

    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereNotDeleted().
        Build()

    result := <-s.base.Repository.IncrementFields(ctx, commentCollectionName, query, increments)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to increment fields: %w", err)
    }

    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) UpdateAndIncrementFields(ctx context.Context, commentID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error {
    if len(updates) == 0 && len(increments) == 0 {
        return nil
    }

    if len(updates) > 0 {
        updates["lastUpdated"] = utils.UTCNowUnix()
    }

    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereNotDeleted().
        Build()

    result := <-s.base.Repository.UpdateAndIncrement(ctx, commentCollectionName, query, updates, increments)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to update and increment fields: %w", err)
    }

    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) UpdateFieldsWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID, updates map[string]interface{}) error {
    if len(updates) == 0 {
        return nil
    }

    updates["lastUpdated"] = utils.UTCNowUnix()
    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereOwner(ownerID).
        WhereNotDeleted().
        Build()

    result := <-s.base.Repository.UpdateFields(ctx, commentCollectionName, query, updates)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to update fields with ownership: %w", err)
    }

    s.invalidateUserComments(ctx, ownerID)
    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) DeleteWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID) error {
    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereOwner(ownerID).
        Build()

    result := <-s.base.Repository.Delete(ctx, commentCollectionName, query)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to delete with ownership: %w", err)
    }

    s.invalidateUserComments(ctx, ownerID)
    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) IncrementFieldsWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID, increments map[string]interface{}) error {
    if len(increments) == 0 {
        return nil
    }

    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereOwner(ownerID).
        WhereNotDeleted().
        Build()

    result := <-s.base.Repository.IncrementFields(ctx, commentCollectionName, query, increments)
    if err := result.Error; err != nil {
        return fmt.Errorf("failed to increment fields with ownership: %w", err)
    }

    s.invalidateUserComments(ctx, ownerID)
    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) getCommentCount(ctx context.Context, query *dbi.Query) (int64, error) {
    countResult := <-s.base.Repository.Count(ctx, commentCollectionName, query)
    if countResult.Error != nil {
        return 0, countResult.Error
    }
    return countResult.Count, nil
}

func sanitizePagination(filter *models.CommentQueryFilter) {
    if filter.Limit <= 0 {
        filter.Limit = defaultCommentLimit
    } else if filter.Limit > maxCommentLimit {
        filter.Limit = maxCommentLimit
    }

    if filter.Page <= 0 {
        filter.Page = defaultCommentPage
    }
}

func applyFilterToBuilder(qb *commentQueryBuilder, filter *models.CommentQueryFilter) {
    includeDeleted := filter.IncludeDeleted
    if filter.Deleted != nil {
        qb.WhereDeleted(*filter.Deleted)
    } else if !includeDeleted {
        qb.WhereNotDeleted()
    }

    if filter.PostId != nil {
        qb.WherePostID(*filter.PostId)
    }
    if filter.OwnerUserId != nil {
        qb.WhereOwner(*filter.OwnerUserId)
    }
    if filter.ParentCommentId != nil {
        qb.WhereParentEquals(*filter.ParentCommentId)
    } else if filter.RootOnly {
        qb.WhereParentIsNull()
    }
    if filter.CreatedAfter != nil {
        qb.WhereCreatedAfter(*filter.CreatedAfter)
    }
    if filter.CreatedBefore != nil {
        qb.WhereCreatedBefore(*filter.CreatedBefore)
    }
}

// GetReplyCount returns the number of replies for a parent comment
func (s *commentService) GetReplyCount(ctx context.Context, parentID uuid.UUID) (int64, error) {
    qb := newCommentQueryBuilder().WhereParentEquals(parentID).WhereNotDeleted()
    query := qb.Build()
    return s.getCommentCount(ctx, query)
}

// updatePostCommentCounter updates the post's commentCounter field via the PostStatsUpdater interface
// This maintains the denormalized count for performance without fetching all comments
// delta: +1 to increment (when creating root comment), -1 to decrement (when deleting root comment)
// Only updates for root comments (replies don't affect post.commentCounter)
func (s *commentService) updatePostCommentCounter(ctx context.Context, postID uuid.UUID, delta int) error {
    if s.postStatsUpdater == nil {

        log.Warn("PostStatsUpdater is nil - skipping commentCounter update for post %s (delta: %d)", postID.String(), delta)
        return nil
    }

    log.Info("Calling PostStatsUpdater.IncrementCommentCountForService for post %s (delta: %d)", postID.String(), delta)
    err := s.postStatsUpdater.IncrementCommentCountForService(ctx, postID, delta)
    if err != nil {
        log.Error("PostStatsUpdater.IncrementCommentCountForService failed for post %s (delta: %d): %v", postID.String(), delta, err)
        return fmt.Errorf("failed to update post commentCounter: %w", err)
    }

    log.Info("PostStatsUpdater.IncrementCommentCountForService succeeded for post %s (delta: %d)", postID.String(), delta)
    return nil
}

func (s *commentService) fetchOwnedComment(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID) (*models.Comment, error) {
    query := newCommentQueryBuilder().
        WhereObjectID(commentID).
        WhereOwner(ownerID).
        WhereNotDeleted().
        Build()

    single := <-s.base.Repository.FindOne(ctx, commentCollectionName, query)
    var comment models.Comment
    if err := single.Decode(&comment); err != nil {
        if errors.Is(err, dbi.ErrNoDocuments) {
            return nil, commentsErrors.ErrCommentNotFound
        }
        return nil, fmt.Errorf("failed to load comment: %w", err)
    }
    return &comment, nil
}
