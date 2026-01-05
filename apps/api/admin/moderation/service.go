package moderation

import (
	"context"
	"fmt"

	uuid "github.com/gofrs/uuid"
	postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
)

type Service interface {
	GetQueue(ctx context.Context) (*ModerationList, error)
	ApprovePost(ctx context.Context, postID uuid.UUID) error
	RejectPost(ctx context.Context, postID uuid.UUID) error
}

type service struct {
	postRepo postsRepository.PostRepository
}

func NewService(postRepo postsRepository.PostRepository) Service {
	return &service{postRepo: postRepo}
}

func (s *service) GetQueue(ctx context.Context) (*ModerationList, error) {
	// Query posts where status IN ('needs_moderation', 'analysis_failed')
	// Includes both posts flagged by AI and posts that failed analysis (fail-safe)
	limit := 100
	offset := 0
	
	statuses := []string{"needs_moderation", "analysis_failed"}
	posts, err := s.postRepo.FindByStatuses(ctx, statuses, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query moderation queue: %w", err)
	}

	// Get total count
	totalCount, err := s.postRepo.CountByStatuses(ctx, statuses)
	if err != nil {
		return nil, fmt.Errorf("failed to count moderation queue: %w", err)
	}

	// Transform posts to FlaggedPost
	var items []FlaggedPost
	for _, post := range posts {
		// Convert ModerationDetails JSONB to interface{}
		var moderationDetails interface{}
		if post.ModerationDetails != nil {
			moderationDetails = post.ModerationDetails
		}

		items = append(items, FlaggedPost{
			ID:                post.ObjectId,
			Content:           post.Body,
			AuthorID:          post.OwnerUserId,
			AuthorName:        post.OwnerDisplayName,
			ModerationDetails: moderationDetails,
			CreatedAt:         post.CreatedDate,
		})
	}

	return &ModerationList{Items: items, Count: totalCount}, nil
}

func (s *service) ApprovePost(ctx context.Context, postID uuid.UUID) error {
	// Load the post first to get current state
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}

	// Update status to published
	post.Status = "published"
	
	if err := s.postRepo.Update(ctx, post); err != nil {
		return fmt.Errorf("failed to approve post: %w", err)
	}

	return nil
}

func (s *service) RejectPost(ctx context.Context, postID uuid.UUID) error {
	// Load the post first to get current state
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}

	// 'rejected' status hides it from the platform but keeps record
	post.Status = "rejected"
	
	if err := s.postRepo.Update(ctx, post); err != nil {
		return fmt.Errorf("failed to reject post: %w", err)
	}

	return nil
}

