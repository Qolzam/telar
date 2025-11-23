package adapters

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/services"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
)

// Ensure DirectCallCounter implements CommentCounter interface
var _ sharedInterfaces.CommentCounter = (*DirectCallCounter)(nil)

// DirectCallCounter is an adapter that implements CommentCounter interface
// by directly calling the concrete service implementation.
// Used in serverless/monolith deployment mode.
type DirectCallCounter struct {
	service services.CommentService
}

// NewDirectCallCounter creates a new DirectCallCounter adapter.
func NewDirectCallCounter(svc services.CommentService) *DirectCallCounter {
	return &DirectCallCounter{service: svc}
}

// GetRootCommentCount delegates to the concrete service's GetRootCommentCount method.
func (a *DirectCallCounter) GetRootCommentCount(ctx context.Context, postID uuid.UUID) (int64, error) {
	return a.service.GetRootCommentCount(ctx, postID)
}

