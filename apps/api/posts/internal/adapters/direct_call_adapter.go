package adapters

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/posts/services"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
)

// Ensure DirectCallStatsUpdater implements PostStatsUpdater interface
var _ sharedInterfaces.PostStatsUpdater = (*DirectCallStatsUpdater)(nil)

// DirectCallStatsUpdater is an adapter that implements PostStatsUpdater interface
// by directly calling the concrete service implementation.
// Used in serverless/monolith Ddeployment mode.
type DirectCallStatsUpdater struct {
	service services.PostService
}

// NewDirectCallStatsUpdater creates a new DirectCallStatsUpdater adapter.
func NewDirectCallStatsUpdater(svc services.PostService) *DirectCallStatsUpdater {
	return &DirectCallStatsUpdater{service: svc}
}

// IncrementCommentCountForService delegates to the concrete service's IncrementCommentCountForService method.
// The service implements PostStatsUpdater interface directly, so we can call the method via type assertion.
func (a *DirectCallStatsUpdater) IncrementCommentCountForService(ctx context.Context, postID uuid.UUID, delta int) error {
	// postService implements both PostService and PostStatsUpdater interfaces
	// The PostStatsUpdater.IncrementCommentCountForService doesn't require UserContext
	// Use type assertion to access the PostStatsUpdater method (consistent with Profile pattern)
	if updater, ok := a.service.(sharedInterfaces.PostStatsUpdater); ok {
		return updater.IncrementCommentCountForService(ctx, postID, delta)
	}
	
	// This should never happen if postService properly implements PostStatsUpdater
	return fmt.Errorf("service does not implement PostStatsUpdater")
}

