package interfaces

import (
	"context"

	"github.com/gofrs/uuid"
)

// CommentCounter is the public interface for counting comments.
// Any service that needs comment counts will depend on this.
// This interface enables the Public Interface + Adapter pattern for service-to-service communication:
// - Other services (e.g., Posts) depend on this interface, not concrete implementations
// - Multiple adapters implement this interface for different deployment modes:
//   • DirectCallAdapter: In-process calls for serverless/monolith deployment
//   • GrpcAdapter: Network calls via gRPC for microservices deployment
// This pattern allows the same codebase to work in both serverless and Kubernetes environments.
type CommentCounter interface {
	GetRootCommentCount(ctx context.Context, postID uuid.UUID) (int64, error)
}

