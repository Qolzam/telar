package comments

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/services"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
	pb "github.com/qolzam/telar/protos/gen/go/commentspb"
)

// grpcServer implements the CommentsServiceServer interface generated from proto.
type grpcServer struct {
	pb.UnimplementedCommentsServiceServer
	service services.CommentService
}

// NewGrpcServer creates a new gRPC server for Comments service.
func NewGrpcServer(svc services.CommentService) pb.CommentsServiceServer {
	return &grpcServer{service: svc}
}

// GetRootCommentCount is the implementation of the gRPC endpoint.
func (s *grpcServer) GetRootCommentCount(ctx context.Context, req *pb.GetRootCommentCountRequest) (*pb.GetRootCommentCountResponse, error) {
	postID, err := uuid.FromString(req.PostId)
	if err != nil {
		return nil, err
	}

	// The service's GetRootCommentCount method implements CommentCounter interface
	if counter, ok := s.service.(sharedInterfaces.CommentCounter); ok {
		count, err := counter.GetRootCommentCount(ctx, postID)
		if err != nil {
			return nil, err
		}

		return &pb.GetRootCommentCountResponse{Count: count}, nil
	}

	// Fallback: if service doesn't implement CommentCounter, this shouldn't happen
	return nil, fmt.Errorf("service does not implement CommentCounter")
}

