package services

import (
    "context"
    "fmt"
    "time"

    "github.com/gofrs/uuid"

    "github.com/qolzam/telar/apps/api/comments/common"
    commentsErrors "github.com/qolzam/telar/apps/api/comments/errors"
    "github.com/qolzam/telar/apps/api/comments/models"
    commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
    "github.com/qolzam/telar/apps/api/internal/cache"
    platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
    "github.com/qolzam/telar/apps/api/internal/types"
    "github.com/qolzam/telar/apps/api/internal/utils"
    postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
    sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
)

const (
    defaultCommentLimit = 10
    maxCommentLimit     = 100
    defaultCommentPage  = 1
)

// commentService implements the CommentService interface using the new repository patterns.
type commentService struct {
    commentRepo      commentRepository.CommentRepository
    postRepo         postsRepository.PostRepository
    cacheService     *cache.GenericCacheService
    config           *platformconfig.Config
    postStatsUpdater sharedInterfaces.PostStatsUpdater
}

// Ensure commentService implements sharedInterfaces.CommentCounter interface
var _ sharedInterfaces.CommentCounter = (*commentService)(nil)

// Legacy query builder removed - all queries now use CommentRepository

// GetRootCommentCount counts root comments (non-reply comments) for a post
// This is a public method that implements the CommentCounter interface
func (s *commentService) GetRootCommentCount(ctx context.Context, postID uuid.UUID) (int64, error) {
    return s.commentRepo.CountByPostID(ctx, postID)
}

// NewCommentService wires the comment service with its dependencies.
func NewCommentService(commentRepo commentRepository.CommentRepository, postRepo postsRepository.PostRepository, cfg *platformconfig.Config, postStatsUpdater sharedInterfaces.PostStatsUpdater) CommentService {
    cacheService := cache.NewGenericCacheServiceFor("comments")
    return &commentService{
        commentRepo:      commentRepo,
        postRepo:         postRepo,
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
// Uses transaction to atomically create comment and increment post comment_count.
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

    // Determine if this is a root comment (affects comment_count update)
    isRootComment := req.ParentCommentId == nil || *req.ParentCommentId == uuid.Nil

    // Use transaction for atomic comment creation + count increment
    if isRootComment {
        // Use PostRepository's WithTransaction to ensure atomicity
        err = s.postRepo.WithTransaction(ctx, func(txCtx context.Context) error {
            // Create comment within transaction
            if err := s.commentRepo.Create(txCtx, comment); err != nil {
                return fmt.Errorf("failed to create comment: %w", err)
            }

            // Increment post comment_count within same transaction
            if err := s.postRepo.IncrementCommentCount(txCtx, req.PostId, 1); err != nil {
                return fmt.Errorf("failed to increment comment count: %w", err)
            }

            return nil
        })
        if err != nil {
            return nil, fmt.Errorf("failed to create comment atomically: %w", err)
        }
    } else {
        // For replies, no count update needed - just create the comment
        if err := s.commentRepo.Create(ctx, comment); err != nil {
            return nil, fmt.Errorf("failed to create comment: %w", err)
        }
    }

    s.invalidateUserComments(ctx, user.UserID)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)

    return comment, nil
}

// CreateIndex is a no-op for relational schema (indexes are defined in migration)
func (s *commentService) CreateIndex(ctx context.Context, indexes map[string]interface{}) error {
    return nil
}

// CreateIndexes is a no-op for relational schema (indexes are defined in migration)
func (s *commentService) CreateIndexes(ctx context.Context) error {
    return nil
}

// GetComment returns a single non-deleted comment by ID.
func (s *commentService) GetComment(ctx context.Context, commentID uuid.UUID) (*models.Comment, error) {
    comment, err := s.commentRepo.FindByID(ctx, commentID)
    if err != nil {
        if err.Error() == "comment not found" {
            return nil, commentsErrors.ErrCommentNotFound
        }
        return nil, fmt.Errorf("failed to find comment: %w", err)
    }

    // Filter out deleted comments
    if comment.Deleted {
        return nil, commentsErrors.ErrCommentNotFound
    }

    return comment, nil
}

// GetCommentsByPost lists root comments for a specific post.
func (s *commentService) GetCommentsByPost(ctx context.Context, postID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
    if filter == nil {
        filter = &models.CommentQueryFilter{}
    }
    filter.PostId = &postID
    filter.RootOnly = true // Only return root comments
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

    // Convert CommentQueryFilter to CommentFilter
    repoFilter := commentRepository.CommentFilter{
        PostID:         filter.PostId,
        OwnerUserID:    filter.OwnerUserId,
        ParentCommentID: filter.ParentCommentId,
        RootOnly:       filter.RootOnly,
        IncludeDeleted: filter.IncludeDeleted,
    }
    if filter.Deleted != nil {
        deleted := *filter.Deleted
        repoFilter.Deleted = &deleted
    }
    if filter.CreatedAfter != nil {
        createdAfter := filter.CreatedAfter.Unix()
        repoFilter.CreatedAfter = &createdAfter
    }
    if filter.CreatedBefore != nil {
        createdBefore := filter.CreatedBefore.Unix()
        repoFilter.CreatedBefore = &createdBefore
    }

    limit := filter.Limit
    offset := (filter.Page - 1) * filter.Limit

    // Query comments using repository
    comments, err := s.commentRepo.Find(ctx, repoFilter, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to query comments: %w", err)
    }

    // Count total matching comments
    totalCount, err := s.commentRepo.Count(ctx, repoFilter)
    if err != nil {
        return nil, fmt.Errorf("failed to count comments: %w", err)
    }

    // Convert to response format
    responses := make([]models.CommentResponse, len(comments))
    for i, comment := range comments {
        responses[i] = s.convertToCommentResponse(comment)
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

    // Fetch comment and verify ownership
    comment, err := s.commentRepo.FindByID(ctx, commentID)
    if err != nil {
        if err.Error() == "comment not found" {
            return commentsErrors.ErrCommentNotFound
        }
        return fmt.Errorf("failed to find comment: %w", err)
    }

    if comment.Deleted {
        return commentsErrors.ErrCommentNotFound
    }

    if comment.OwnerUserId != user.UserID {
        return commentsErrors.ErrCommentNotFound
    }

    // Update comment
    comment.Text = req.Text
    comment.LastUpdated = utils.UTCNowUnix()

    if err := s.commentRepo.Update(ctx, comment); err != nil {
        return fmt.Errorf("failed to update comment: %w", err)
    }

    s.invalidateUserComments(ctx, user.UserID)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)
    return nil
}

// UpdateCommentProfile updates profile information for all comments created by a user.
func (s *commentService) UpdateCommentProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error {
    if err := s.commentRepo.UpdateOwnerProfile(ctx, userID, displayName, avatar); err != nil {
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

    // Use repository's atomic increment
    if err := s.commentRepo.IncrementScore(ctx, commentID, delta); err != nil {
        return fmt.Errorf("failed to increment score: %w", err)
    }

    s.invalidateAllComments(ctx)
    return nil
}

// DeleteComment removes a comment permanently (soft delete).
// Uses transaction to atomically delete comment and decrement post comment_count.
func (s *commentService) DeleteComment(ctx context.Context, commentID uuid.UUID, postID uuid.UUID, user *types.UserContext) error {
    if user == nil {
        return fmt.Errorf("user context is required")
    }

    // Fetch comment and verify ownership
    comment, err := s.commentRepo.FindByID(ctx, commentID)
    if err != nil {
        if err.Error() == "comment not found" {
            return commentsErrors.ErrCommentNotFound
        }
        return fmt.Errorf("failed to find comment: %w", err)
    }

    if comment.Deleted {
        return commentsErrors.ErrCommentNotFound
    }

    if comment.OwnerUserId != user.UserID {
        return commentsErrors.ErrCommentNotFound
    }

    if postID != uuid.Nil && comment.PostId != postID {
        return fmt.Errorf("comment does not belong to the provided post")
    }

    // Check if this is a root comment (affects comment_count update)
    isRootComment := comment.ParentCommentId == nil || *comment.ParentCommentId == uuid.Nil

    // Use transaction for atomic comment deletion + count decrement
    if isRootComment {
        // Use PostRepository's WithTransaction to ensure atomicity
        err = s.postRepo.WithTransaction(ctx, func(txCtx context.Context) error {
            // Soft delete comment within transaction
            if err := s.commentRepo.Delete(txCtx, commentID); err != nil {
                return fmt.Errorf("failed to delete comment: %w", err)
            }

            // Decrement post comment_count within same transaction
            if err := s.postRepo.IncrementCommentCount(txCtx, comment.PostId, -1); err != nil {
                return fmt.Errorf("failed to decrement comment count: %w", err)
            }

            return nil
        })
        if err != nil {
            return fmt.Errorf("failed to delete comment atomically: %w", err)
        }
    } else {
        // For replies, no count update needed - just delete the comment
        if err := s.commentRepo.Delete(ctx, commentID); err != nil {
            return fmt.Errorf("failed to delete comment: %w", err)
        }
    }

    s.invalidateUserComments(ctx, user.UserID)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)
    return nil
}

// DeleteCommentsByPost removes all comments for a given post (soft delete).
func (s *commentService) DeleteCommentsByPost(ctx context.Context, postID uuid.UUID, user *types.UserContext) error {
    if user == nil {
        return fmt.Errorf("user context is required")
    }

    // Get count of root comments before deletion (for count update)
    rootCount, err := s.commentRepo.CountByPostID(ctx, postID)
    if err != nil {
        return fmt.Errorf("failed to count root comments: %w", err)
    }

    // Use transaction for atomic batch delete + count update
    if rootCount > 0 {
        err = s.postRepo.WithTransaction(ctx, func(txCtx context.Context) error {
            // Soft delete all comments for the post
            if err := s.commentRepo.DeleteByPostID(txCtx, postID); err != nil {
                return fmt.Errorf("failed to delete comments by post: %w", err)
            }

            // Decrement post comment_count by the number of root comments
            if err := s.postRepo.IncrementCommentCount(txCtx, postID, -int(rootCount)); err != nil {
                return fmt.Errorf("failed to decrement comment count: %w", err)
            }

            return nil
        })
        if err != nil {
            return fmt.Errorf("failed to delete comments atomically: %w", err)
        }
    } else {
        // No root comments, just delete
        if err := s.commentRepo.DeleteByPostID(ctx, postID); err != nil {
            return fmt.Errorf("failed to delete comments by post: %w", err)
        }
    }

    s.invalidatePostComments(ctx, postID)
    s.invalidateAllComments(ctx)
    return nil
}

// SoftDeleteComment marks a comment as deleted without removing it permanently.
// This is the same as DeleteComment (both use soft delete).
func (s *commentService) SoftDeleteComment(ctx context.Context, commentID uuid.UUID, user *types.UserContext) error {
    // SoftDeleteComment is the same as DeleteComment (both are soft deletes)
    return s.DeleteComment(ctx, commentID, uuid.Nil, user)
}

// DeleteByOwner removes a comment when ownership is known (admin utility).
func (s *commentService) DeleteByOwner(ctx context.Context, owner uuid.UUID, objectID uuid.UUID) error {
    comment, err := s.fetchOwnedComment(ctx, objectID, owner)
    if err != nil {
        return err
    }

    // Check if this is a root comment (affects comment_count update)
    isRootComment := comment.ParentCommentId == nil || *comment.ParentCommentId == uuid.Nil

    // Use transaction for atomic deletion + count decrement
    if isRootComment {
        err = s.postRepo.WithTransaction(ctx, func(txCtx context.Context) error {
            if err := s.commentRepo.Delete(txCtx, objectID); err != nil {
                return fmt.Errorf("failed to delete comment: %w", err)
            }
            if err := s.postRepo.IncrementCommentCount(txCtx, comment.PostId, -1); err != nil {
                return fmt.Errorf("failed to decrement comment count: %w", err)
            }
            return nil
        })
        if err != nil {
            return fmt.Errorf("failed to delete comment atomically: %w", err)
        }
    } else {
        if err := s.commentRepo.Delete(ctx, objectID); err != nil {
            return fmt.Errorf("failed to delete comment: %w", err)
        }
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

    // Load existing comment
    comment, err := s.commentRepo.FindByID(ctx, commentID)
    if err != nil {
        if err.Error() == "comment not found" {
            return commentsErrors.ErrCommentNotFound
        }
        return fmt.Errorf("failed to find comment: %w", err)
    }

    if comment.Deleted {
        return commentsErrors.ErrCommentNotFound
    }

    // Apply updates to comment struct
    if text, ok := updates["text"].(string); ok {
        comment.Text = text
    }
    if displayName, ok := updates["ownerDisplayName"].(string); ok {
        comment.OwnerDisplayName = displayName
    }
    if avatar, ok := updates["ownerAvatar"].(string); ok {
        comment.OwnerAvatar = avatar
    }

    comment.LastUpdated = utils.UTCNowUnix()

    if err := s.commentRepo.Update(ctx, comment); err != nil {
        return fmt.Errorf("failed to update fields: %w", err)
    }

    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) IncrementFields(ctx context.Context, commentID uuid.UUID, increments map[string]interface{}) error {
    if len(increments) == 0 {
        return nil
    }

    // Handle score increment (most common case)
    if scoreDelta, ok := increments["score"].(int); ok {
        if err := s.commentRepo.IncrementScore(ctx, commentID, scoreDelta); err != nil {
            return fmt.Errorf("failed to increment score: %w", err)
        }
        s.invalidateAllComments(ctx)
        return nil
    }

    // For other fields, load, update, and save
    comment, err := s.commentRepo.FindByID(ctx, commentID)
    if err != nil {
        if err.Error() == "comment not found" {
            return commentsErrors.ErrCommentNotFound
        }
        return fmt.Errorf("failed to find comment: %w", err)
    }

    if comment.Deleted {
        return commentsErrors.ErrCommentNotFound
    }

    // Apply increments (only score is supported for atomic increments)
    // Other fields would need to be handled differently
    comment.LastUpdated = utils.UTCNowUnix()

    if err := s.commentRepo.Update(ctx, comment); err != nil {
        return fmt.Errorf("failed to update comment: %w", err)
    }

    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) UpdateAndIncrementFields(ctx context.Context, commentID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error {
    if len(updates) == 0 && len(increments) == 0 {
        return nil
    }

    // Load existing comment
    comment, err := s.commentRepo.FindByID(ctx, commentID)
    if err != nil {
        if err.Error() == "comment not found" {
            return commentsErrors.ErrCommentNotFound
        }
        return fmt.Errorf("failed to find comment: %w", err)
    }

    if comment.Deleted {
        return commentsErrors.ErrCommentNotFound
    }

    // Apply updates
    if text, ok := updates["text"].(string); ok {
        comment.Text = text
    }
    if displayName, ok := updates["ownerDisplayName"].(string); ok {
        comment.OwnerDisplayName = displayName
    }
    if avatar, ok := updates["ownerAvatar"].(string); ok {
        comment.OwnerAvatar = avatar
    }

    // Apply increments (score is handled atomically, others need manual update)
    if scoreDelta, ok := increments["score"].(int); ok {
        comment.Score += int64(scoreDelta)
    }

    comment.LastUpdated = utils.UTCNowUnix()

    // Update comment
    if err := s.commentRepo.Update(ctx, comment); err != nil {
        return fmt.Errorf("failed to update and increment fields: %w", err)
    }

    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) UpdateFieldsWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID, updates map[string]interface{}) error {
    if len(updates) == 0 {
        return nil
    }

    // Verify ownership
    comment, err := s.fetchOwnedComment(ctx, commentID, ownerID)
    if err != nil {
        return err
    }

    // Apply updates
    if text, ok := updates["text"].(string); ok {
        comment.Text = text
    }
    if displayName, ok := updates["ownerDisplayName"].(string); ok {
        comment.OwnerDisplayName = displayName
    }
    if avatar, ok := updates["ownerAvatar"].(string); ok {
        comment.OwnerAvatar = avatar
    }

    comment.LastUpdated = utils.UTCNowUnix()

    if err := s.commentRepo.Update(ctx, comment); err != nil {
        return fmt.Errorf("failed to update fields with ownership: %w", err)
    }

    s.invalidateUserComments(ctx, ownerID)
    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) DeleteWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID) error {
    // Verify ownership and get comment info
    comment, err := s.fetchOwnedComment(ctx, commentID, ownerID)
    if err != nil {
        return err
    }

    // Check if root comment
    isRootComment := comment.ParentCommentId == nil || *comment.ParentCommentId == uuid.Nil

    // Use transaction for atomic deletion
    if isRootComment {
        err = s.postRepo.WithTransaction(ctx, func(txCtx context.Context) error {
            if err := s.commentRepo.Delete(txCtx, commentID); err != nil {
                return fmt.Errorf("failed to delete comment: %w", err)
            }
            if err := s.postRepo.IncrementCommentCount(txCtx, comment.PostId, -1); err != nil {
                return fmt.Errorf("failed to decrement comment count: %w", err)
            }
            return nil
        })
        if err != nil {
            return fmt.Errorf("failed to delete with ownership atomically: %w", err)
        }
    } else {
        if err := s.commentRepo.Delete(ctx, commentID); err != nil {
            return fmt.Errorf("failed to delete with ownership: %w", err)
        }
    }

    s.invalidateUserComments(ctx, ownerID)
    s.invalidateAllComments(ctx)
    return nil
}

func (s *commentService) IncrementFieldsWithOwnership(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID, increments map[string]interface{}) error {
    if len(increments) == 0 {
        return nil
    }

    // Verify ownership
    _, err := s.fetchOwnedComment(ctx, commentID, ownerID)
    if err != nil {
        return err
    }

    // Handle score increment (most common case)
    if scoreDelta, ok := increments["score"].(int); ok {
        if err := s.commentRepo.IncrementScore(ctx, commentID, scoreDelta); err != nil {
            return fmt.Errorf("failed to increment score: %w", err)
        }
        s.invalidateUserComments(ctx, ownerID)
        s.invalidateAllComments(ctx)
        return nil
    }

    // For other fields, would need to load, update, and save
    // This is a less common case, so we'll handle it generically
    return fmt.Errorf("only score increment is supported for atomic operations")
}

// getCommentCount is deprecated - use repository.Count directly
// This method is no longer used and kept for backward compatibility only

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

// GetReplyCount returns the number of replies for a parent comment
func (s *commentService) GetReplyCount(ctx context.Context, parentID uuid.UUID) (int64, error) {
    return s.commentRepo.CountReplies(ctx, parentID)
}

// updatePostCommentCounter is deprecated - count updates are now handled atomically in transactions
// This method is kept for backward compatibility but should not be used in new code
func (s *commentService) updatePostCommentCounter(ctx context.Context, postID uuid.UUID, delta int) error {
    // This method is no longer used - count updates are handled in transactions
    // Kept for backward compatibility only
    if s.postStatsUpdater != nil {
        return s.postStatsUpdater.IncrementCommentCountForService(ctx, postID, delta)
    }
    return nil
}

func (s *commentService) fetchOwnedComment(ctx context.Context, commentID uuid.UUID, ownerID uuid.UUID) (*models.Comment, error) {
    comment, err := s.commentRepo.FindByID(ctx, commentID)
    if err != nil {
        if err.Error() == "comment not found" {
            return nil, commentsErrors.ErrCommentNotFound
        }
        return nil, fmt.Errorf("failed to load comment: %w", err)
    }

    if comment.Deleted {
        return nil, commentsErrors.ErrCommentNotFound
    }

    if comment.OwnerUserId != ownerID {
        return nil, commentsErrors.ErrCommentNotFound
    }

    return comment, nil
}
