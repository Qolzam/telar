package interfaces

import (
	"context"

	"github.com/gofrs/uuid"
)

// PostStatsUpdater is the public interface for updating post statistics.
// Any service that needs to update post stats will depend on this.
// This interface enables the Public Interface + Adapter pattern for service-to-service communication:
// - Other services (e.g., Comments) depend on this interface, not concrete implementations
// - Multiple adapters implement this interface for different deployment modes:
//   • DirectCallAdapter: In-process calls for serverless/monolith deployment
//   • GrpcAdapter: Network calls via gRPC for microservices deployment
// This pattern allows the same codebase to work in both serverless and Kubernetes environments.
type PostStatsUpdater interface {
	IncrementCommentCountForService(ctx context.Context, postID uuid.UUID, delta int) error
}

