package adapters

import (
	"context"

	"github.com/gofrs/uuid"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
	pb "github.com/qolzam/telar/protos/gen/go/commentspb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Ensure GrpcCounter implements CommentCounter interface
var _ sharedInterfaces.CommentCounter = (*GrpcCounter)(nil)

// GrpcCounter is an adapter that implements CommentCounter interface
// by making gRPC calls to the Comments service.
// Used in microservices/Kubernetes deployment mode.
type GrpcCounter struct {
	client pb.CommentsServiceClient
	conn   *grpc.ClientConn
}

// NewGrpcCounter creates a new GrpcCounter adapter.
func NewGrpcCounter(targetAddress string) (*GrpcCounter, error) {
	conn, err := grpc.Dial(targetAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GrpcCounter{
		client: pb.NewCommentsServiceClient(conn),
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection.
func (a *GrpcCounter) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// GetRootCommentCount makes a gRPC call to get the root comment count.
func (a *GrpcCounter) GetRootCommentCount(ctx context.Context, postID uuid.UUID) (int64, error) {
	grpcReq := &pb.GetRootCommentCountRequest{
		PostId: postID.String(),
	}

	resp, err := a.client.GetRootCommentCount(ctx, grpcReq)
	if err != nil {
		return 0, err
	}

	return resp.Count, nil
}

