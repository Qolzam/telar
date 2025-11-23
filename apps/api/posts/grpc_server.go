package posts

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/posts/services"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
	pb "github.com/qolzam/telar/protos/gen/go/postspb"
)

// grpcServer implements the PostsServiceServer interface generated from proto.
type grpcServer struct {
	pb.UnimplementedPostsServiceServer
	service services.PostService
}

// NewGrpcServer creates a new gRPC server for Posts service.
func NewGrpcServer(svc services.PostService) pb.PostsServiceServer {
	return &grpcServer{service: svc}
}

// IncrementCommentCount is the implementation of the gRPC endpoint.
func (s *grpcServer) IncrementCommentCount(ctx context.Context, req *pb.IncrementCommentCountRequest) (*pb.IncrementCommentCountResponse, error) {
	postID, err := uuid.FromString(req.PostId)
	if err != nil {
		return nil, err
	}

	// The service's IncrementCommentCountForService method implements PostStatsUpdater interface
	if updater, ok := s.service.(sharedInterfaces.PostStatsUpdater); ok {
		err := updater.IncrementCommentCountForService(ctx, postID, int(req.Delta))
		if err != nil {
			return &pb.IncrementCommentCountResponse{Success: false}, err
		}

		return &pb.IncrementCommentCountResponse{Success: true}, nil
	}

	// Fallback: if service doesn't implement PostStatsUpdater, this shouldn't happen
	return &pb.IncrementCommentCountResponse{Success: false}, fmt.Errorf("service does not implement PostStatsUpdater")
}

