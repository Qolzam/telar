package services

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/gofrs/uuid"

    "github.com/qolzam/telar/apps/api/comments/common"
    commentsErrors "github.com/qolzam/telar/apps/api/comments/errors"
    "github.com/qolzam/telar/apps/api/comments/models"
    commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
    "github.com/qolzam/telar/apps/api/internal/cache"
    "github.com/qolzam/telar/apps/api/internal/pkg/log"
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
    if err := s.cacheService.InvalidatePattern(ctx, pattern); err != nil {
        log.Warn("Cache invalidation failed for user comments: %v", err)
    }
}

func (s *commentService) invalidatePostComments(ctx context.Context, postID uuid.UUID) {
    if s.cacheService == nil {
        return
    }
    pattern := "cursor:comments:" + postID.String() + "*"
    if err := s.cacheService.InvalidatePattern(ctx, pattern); err != nil {
        log.Warn("Cache invalidation failed for post comments: %v", err)
    }
}

func (s *commentService) invalidateAllComments(ctx context.Context) {
    if s.cacheService == nil {
        return
    }
    if err := s.cacheService.InvalidatePattern(ctx, "cursor:comments:*"); err != nil {
        log.Warn("Cache invalidation failed for all comments: %v", err)
    }
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
    
    // Two-Tier Architecture: Flatten nested replies to always point to root
    var rootParentID *uuid.UUID
    var replyToUserID *uuid.UUID
    var replyToDisplayName *string
    
    if req.ParentCommentId != nil && *req.ParentCommentId != uuid.Nil {
        // Fetch the target comment the user clicked "Reply" on
        targetComment, err := s.commentRepo.FindByID(ctx, *req.ParentCommentId)
        if err != nil {
            if err.Error() == "comment not found" {
                return nil, commentsErrors.ErrCommentNotFound
            }
            return nil, fmt.Errorf("failed to find parent comment: %w", err)
        }
        
        if targetComment.Deleted {
            return nil, commentsErrors.ErrCommentNotFound
        }
        
        // Verify the target comment belongs to the same post
        if targetComment.PostId != req.PostId {
            return nil, fmt.Errorf("parent comment does not belong to the specified post")
        }
        
        if targetComment.ParentCommentId == nil {
            // Case A: Replying to a Root Comment
            // The Root becomes the parent. Track who we're replying to.
            rootParentID = &targetComment.ObjectId
            replyToUserID = &targetComment.OwnerUserId
            replyToDisplayName = &targetComment.OwnerDisplayName
        } else {
            // Case B: Replying to a Reply (Nested)
            // FLATTEN IT: The parent's parent is the Root.
            rootParentID = targetComment.ParentCommentId
            // We explicitly track who we are replying to for the UI.
            replyToUserID = &targetComment.OwnerUserId
            replyToDisplayName = &targetComment.OwnerDisplayName
        }
    }
    
    comment := &models.Comment{
        ObjectId:         commentID,
        Score:            0,
        OwnerUserId:      user.UserID,
        OwnerDisplayName: user.DisplayName,
        OwnerAvatar:      user.Avatar,
        PostId:           req.PostId,
        ParentCommentId:  rootParentID,  // ALWAYS points to Root (or nil)
        ReplyToUserId:    replyToUserID, // Points to specific user being addressed
        ReplyToDisplayName: replyToDisplayName, // Display name of user being replied to
        Text:             req.Text,
        Deleted:          false,
        DeletedDate:      0,
        CreatedDate:      now,
        LastUpdated:      now,
    }

    // Determine if this is a root comment (affects comment_count update)
    isRootComment := rootParentID == nil

    // Use transaction for atomic comment creation + count increment
    if isRootComment {
        // Use PostRepository's WithTransaction to ensure atomicity
        err = s.postRepo.WithTransaction(ctx, func(txCtx context.Context) error {
            // Create comment within transaction
            if err := s.commentRepo.Create(txCtx, comment); err != nil {
                // Check for foreign key violations (user or post not found)
                if strings.Contains(err.Error(), "user does not exist") {
                    return commentsErrors.ErrUserNotFound
                }
                if strings.Contains(err.Error(), "post does not exist") {
                    return commentsErrors.ErrPostNotFound
                }
                return fmt.Errorf("failed to create comment: %w", err)
            }

            // Increment post comment_count within same transaction
            if err := s.postRepo.IncrementCommentCount(txCtx, req.PostId, 1); err != nil {
                return fmt.Errorf("failed to increment comment count: %w", err)
            }

            return nil
        })
        if err != nil {
            // Check if the error is already a domain error (ErrUserNotFound, ErrPostNotFound)
            if errors.Is(err, commentsErrors.ErrUserNotFound) || errors.Is(err, commentsErrors.ErrPostNotFound) {
                return nil, err
            }
            return nil, fmt.Errorf("failed to create comment atomically: %w", err)
        }
    } else {
        // For replies, no count update needed - just create the comment
        if err := s.commentRepo.Create(ctx, comment); err != nil {
            // Check for foreign key violations (user or post not found)
            if strings.Contains(err.Error(), "user does not exist") {
                return nil, commentsErrors.ErrUserNotFound
            }
            if strings.Contains(err.Error(), "post does not exist") {
                return nil, commentsErrors.ErrPostNotFound
            }
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

    // Filter out deleted comments (cascade soft-delete is handled at write-time)
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
    // Note: IsLiked will be set to false by default, caller should bulk-load votes if needed
    responses := make([]models.CommentResponse, len(comments))
    for i, comment := range comments {
        responses[i] = s.convertToCommentResponse(comment, false)
    }

    result := &models.CommentsListResponse{
        Comments: responses,
        Count:    int(totalCount),
        Page:     filter.Page,
        Limit:    filter.Limit,
    }

    s.cacheComments(ctx, cacheKey, result)
    return result, nil
}

// QueryCommentsWithCursor retrieves comments with cursor-based pagination
func (s *commentService) QueryCommentsWithCursor(ctx context.Context, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
    if filter == nil {
        return nil, fmt.Errorf("filter is required")
    }

    // Default limit
    limit := filter.Limit
    if limit <= 0 {
        limit = defaultCommentLimit
    } else if limit > maxCommentLimit {
        limit = maxCommentLimit
    }

    // Use cursor pagination for post comments (most common use case)
    if filter.PostId == nil {
        // Fall back to offset pagination if no post filter
        return s.QueryComments(ctx, filter)
    }

    // Use cursor pagination for post comments
    cursor := filter.Cursor

    comments, nextCursor, err := s.commentRepo.FindByPostIDWithCursor(ctx, *filter.PostId, cursor, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to query comments with cursor: %w", err)
    }

    // Convert to response format
    // Note: IsLiked will be set to false by default, caller should bulk-load votes if needed
    responses := make([]models.CommentResponse, len(comments))
    for i, comment := range comments {
        responses[i] = s.convertToCommentResponse(comment, false)
    }

    result := &models.CommentsListResponse{
        Comments:   responses,
        NextCursor: nextCursor,
        HasNext:    nextCursor != "",
        Limit:      limit,
    }

    return result, nil
}

// QueryRepliesWithCursor retrieves replies to a specific comment with cursor-based pagination
func (s *commentService) QueryRepliesWithCursor(ctx context.Context, parentID uuid.UUID, cursor string, limit int) (*models.CommentsListResponse, error) {
    // Default limit
    if limit <= 0 {
        limit = defaultCommentLimit
    } else if limit > maxCommentLimit {
        limit = maxCommentLimit
    }

    // Use cursor pagination for replies
    replies, nextCursor, err := s.commentRepo.FindRepliesWithCursor(ctx, parentID, cursor, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to query replies with cursor: %w", err)
    }

    // Convert to response format
    // Note: IsLiked will be set to false by default, caller should bulk-load votes if needed
    responses := make([]models.CommentResponse, len(replies))
    for i, reply := range replies {
        responses[i] = s.convertToCommentResponse(reply, false)
    }

    result := &models.CommentsListResponse{
        Comments:   responses,
        NextCursor: nextCursor,
        HasNext:    nextCursor != "",
        Limit:      limit,
    }

    return result, nil
}

// UpdateComment updates a comment's text for the owning user.
// Returns the updated comment to avoid Read-After-Write anti-pattern.
func (s *commentService) UpdateComment(ctx context.Context, commentID uuid.UUID, req *models.UpdateCommentRequest, user *types.UserContext) (*models.Comment, error) {
    if req == nil {
        return nil, fmt.Errorf("update comment request is required")
    }
    if user == nil {
        return nil, fmt.Errorf("user context is required")
    }

    // Fetch comment and verify ownership
    comment, err := s.commentRepo.FindByID(ctx, commentID)
    if err != nil {
        if err.Error() == "comment not found" {
            return nil, commentsErrors.ErrCommentNotFound
        }
        return nil, fmt.Errorf("failed to find comment: %w", err)
    }

    if comment.Deleted {
        return nil, commentsErrors.ErrCommentNotFound
    }

    if comment.OwnerUserId != user.UserID {
        return nil, commentsErrors.ErrCommentOwnershipRequired
    }

    // Update comment
    comment.Text = req.Text
    comment.LastUpdated = utils.UTCNowUnix()

    if err := s.commentRepo.Update(ctx, comment); err != nil {
        return nil, fmt.Errorf("failed to update comment: %w", err)
    }

    s.invalidateUserComments(ctx, user.UserID)
    s.invalidatePostComments(ctx, comment.PostId)
    s.invalidateAllComments(ctx)
    
    return comment, nil
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
        return commentsErrors.ErrCommentOwnershipRequired
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

            if err := s.commentRepo.DeleteRepliesByParentID(txCtx, commentID); err != nil {
                return fmt.Errorf("failed to cascade delete replies: %w", err)
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

            // Cascade: Soft delete all replies to this root comment
            if err := s.commentRepo.DeleteRepliesByParentID(txCtx, objectID); err != nil {
                return fmt.Errorf("failed to cascade delete replies: %w", err)
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

func (s *commentService) convertToCommentResponse(comment *models.Comment, isLiked bool) models.CommentResponse {
    return models.CommentResponse{
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
        IsLiked:          isLiked,
    }
}

// Legacy map-based update methods removed - use type-safe methods instead:
// - UpdateComment (for text updates)
// - IncrementScore (for score increments)
// - UpdateCommentProfile (for profile updates)

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

// GetReplyCountsBulk returns reply counts for multiple comments in a single query
// Returns a map of parentCommentID -> replyCount
// This avoids N+1 queries when loading comment lists
func (s *commentService) GetReplyCountsBulk(ctx context.Context, parentIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
    return s.commentRepo.CountRepliesBulk(ctx, parentIDs)
}

// ToggleLike toggles a user's like on a comment
// This is an atomic operation that updates both comment_votes and comments.score
// Returns (comment, newScore, isLiked, error) for efficient response without re-fetching
// The comment object is returned from the transaction to avoid a second database query
func (s *commentService) ToggleLike(ctx context.Context, commentID, userID uuid.UUID) (*models.Comment, int64, bool, error) {
    var comment *models.Comment
    var newScore int64
    var isLiked bool
    
    err := s.commentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
        // Get current comment before toggle (we'll return this to avoid re-fetching)
        var err error
        comment, err = s.commentRepo.FindByID(txCtx, commentID)
        if err != nil {
            return fmt.Errorf("failed to get comment: %w", err)
        }
        
        // 1. Try to insert a vote
        created, err := s.commentRepo.AddVote(txCtx, commentID, userID)
        if err != nil {
            return fmt.Errorf("failed to add vote: %w", err)
        }

        if created {
            // Vote was added -> Increment Score
            isLiked = true
            if err := s.commentRepo.IncrementScore(txCtx, commentID, 1); err != nil {
                return err
            }
            newScore = comment.Score + 1
            comment.Score = newScore // Update the comment object with new score
        } else {
            // Vote existed -> Remove it (Toggle logic) -> Decrement Score
            deleted, err := s.commentRepo.RemoveVote(txCtx, commentID, userID)
            if err != nil {
                return fmt.Errorf("failed to remove vote: %w", err)
            }
            if deleted {
                isLiked = false
                if err := s.commentRepo.IncrementScore(txCtx, commentID, -1); err != nil {
                    return err
                }
                newScore = comment.Score - 1
                comment.Score = newScore // Update the comment object with new score
            } else {
                // Edge case: vote didn't exist and wasn't created
                newScore = comment.Score
            }
        }

        return nil
    })
    
    return comment, newScore, isLiked, err
}

// GetUserVotesForComments bulk checks which comments the user has liked
func (s *commentService) GetUserVotesForComments(ctx context.Context, commentIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
    return s.commentRepo.GetUserVotesForComments(ctx, commentIDs, userID)
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
