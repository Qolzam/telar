package adapters

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
	pb "github.com/qolzam/telar/protos/gen/go/postspb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Ensure GrpcStatsUpdater implements PostStatsUpdater interface
var _ sharedInterfaces.PostStatsUpdater = (*GrpcStatsUpdater)(nil)

// GrpcStatsUpdater is an adapter that implements PostStatsUpdater interface
// by making gRPC calls to the Posts service.
// Used in microservices/Kubernetes deployment mode.
type GrpcStatsUpdater struct {
	client pb.PostsServiceClient
	conn   *grpc.ClientConn
}

// NewGrpcStatsUpdater creates a new GrpcStatsUpdater adapter.
func NewGrpcStatsUpdater(targetAddress string) (*GrpcStatsUpdater, error) {
	conn, err := grpc.Dial(targetAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GrpcStatsUpdater{
		client: pb.NewPostsServiceClient(conn),
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection.
func (a *GrpcStatsUpdater) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// IncrementCommentCountForService makes a gRPC call to increment the comment count.
func (a *GrpcStatsUpdater) IncrementCommentCountForService(ctx context.Context, postID uuid.UUID, delta int) error {
	grpcReq := &pb.IncrementCommentCountRequest{
		PostId: postID.String(),
		Delta:  int32(delta),
	}

	resp, err := a.client.IncrementCommentCount(ctx, grpcReq)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("increment comment count failed")
	}

	return nil
}

