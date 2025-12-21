package services

import (
	"context"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/bookmarks/repository"
	"github.com/qolzam/telar/apps/api/posts/models"
)

// Service defines bookmark operations.
type Service interface {
	// ToggleBookmark flips bookmark state; returns true when bookmarked after the call.
	ToggleBookmark(ctx context.Context, userID, postID uuid.UUID) (bool, error)

	// ListBookmarks returns hydrated posts bookmarked by the user with cursor pagination.
	ListBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*models.PostsListResponse, error)
}

type service struct {
	repo        repository.Repository
	postService postProvider
}

// postProvider captures the subset of PostService we need to hydrate responses.
type postProvider interface {
	GetPostsByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Post, error)
	ConvertPostToResponse(ctx context.Context, post *models.Post) models.PostResponse
}

// NewService constructs a bookmark service.
func NewService(repo repository.Repository, postService postProvider) Service {
	return &service{repo: repo, postService: postService}
}

func (s *service) ToggleBookmark(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	if s.repo == nil {
		return false, fmt.Errorf("bookmark repository is not configured")
	}

	created, err := s.repo.AddBookmark(ctx, userID, postID)
	if err != nil {
		return false, fmt.Errorf("add bookmark: %w", err)
	}
	if created {
		return true, nil
	}

	// Already existed; remove to toggle off.
	_, err = s.repo.RemoveBookmark(ctx, userID, postID)
	if err != nil {
		return false, fmt.Errorf("remove bookmark: %w", err)
	}
	return false, nil
}

func (s *service) ListBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*models.PostsListResponse, error) {
	if s.repo == nil || s.postService == nil {
		return nil, fmt.Errorf("bookmark service dependencies are not configured")
	}

	entries, nextCursor, err := s.repo.FindMyBookmarks(ctx, userID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("find bookmarks: %w", err)
	}

	if len(entries) == 0 {
		return &models.PostsListResponse{Posts: []models.PostResponse{}, NextCursor: nextCursor, HasNext: nextCursor != ""}, nil
	}

	ids := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.PostID)
	}

	posts, err := s.postService.GetPostsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get posts: %w", err)
	}

	// Map for lookup
	postMap := make(map[uuid.UUID]*models.Post, len(posts))
	for _, p := range posts {
		postMap[p.ObjectId] = p
	}

	responses := make([]models.PostResponse, 0, len(entries))
	for _, e := range entries {
		p, ok := postMap[e.PostID]
		if !ok {
			continue // post may be deleted; skip
		}
		resp := s.postService.ConvertPostToResponse(ctx, p)
		resp.IsBookmarked = true
		responses = append(responses, resp)
	}

	return &models.PostsListResponse{
		Posts:      responses,
		NextCursor: nextCursor,
		HasNext:    nextCursor != "",
	}, nil
}
