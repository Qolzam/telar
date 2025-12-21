package adapters

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/profile/models"
	"github.com/qolzam/telar/apps/api/profile/services"
	pb "github.com/qolzam/telar/protos/gen/go/profilepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ services.ProfileServiceClient = (*GrpcCreator)(nil)

type GrpcCreator struct {
	client pb.ProfileServiceClient
	conn   *grpc.ClientConn
}

func NewGrpcCreator(targetAddress string) (*GrpcCreator, error) {
	conn, err := grpc.Dial(targetAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GrpcCreator{
		client: pb.NewProfileServiceClient(conn),
		conn:   conn,
	}, nil
}

func (a *GrpcCreator) Close() error {
	return a.conn.Close()
}

func (a *GrpcCreator) CreateProfileOnSignup(ctx context.Context, req *models.CreateProfileRequest) error {
	grpcReq := &pb.CreateProfileRequest{
		ObjectId:    req.ObjectId.String(),
		FullName:    *req.FullName,
		SocialName:  *req.SocialName,
		Email:       *req.Email,
		Avatar:      *req.Avatar,
		Banner:      *req.Banner,
		TagLine:     *req.TagLine,
		CreatedDate: *req.CreatedDate,
		LastUpdated: *req.LastUpdated,
	}

	_, err := a.client.CreateProfile(ctx, grpcReq)
	return err
}

func (a *GrpcCreator) UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest) error {
	grpcReq := &pb.UpdateProfileRequest{
		ObjectId:   userID.String(),
		FullName:   req.FullName,
		Avatar:     req.Avatar,
		Banner:     req.Banner,
		TagLine:    req.TagLine,
		SocialName: req.SocialName,
	}

	_, err := a.client.UpdateProfile(ctx, grpcReq)
	return err
}

func (a *GrpcCreator) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	grpcReq := &pb.GetProfileRequest{
		ObjectId: userID.String(),
	}

	resp, err := a.client.GetProfile(ctx, grpcReq)
	if err != nil {
		return nil, err
	}

	pbProfile := resp.Profile
	objectId, err := uuid.FromString(pbProfile.ObjectId)
	if err != nil {
		return nil, err
	}

	profile := &models.Profile{
		ObjectId:    objectId,
		FullName:    pbProfile.FullName,
		SocialName:  pbProfile.SocialName,
		Email:       pbProfile.Email,
		Avatar:      pbProfile.Avatar,
		Banner:      pbProfile.Banner,
		Tagline:     pbProfile.TagLine,
		CreatedDate: pbProfile.CreatedDate,
		LastUpdated: pbProfile.LastUpdated,
		LastSeen:    pbProfile.LastSeen,
	}

	return profile, nil
}

func (a *GrpcCreator) GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.Profile, error) {
	objectIds := make([]string, 0, len(userIds))
	for _, id := range userIds {
		objectIds = append(objectIds, id.String())
	}

	grpcReq := &pb.GetProfilesByIdsRequest{
		ObjectIds: objectIds,
	}

	resp, err := a.client.GetProfilesByIds(ctx, grpcReq)
	if err != nil {
		return nil, err
	}

	profiles := make([]*models.Profile, 0, len(resp.Profiles))
	for _, pbProfile := range resp.Profiles {
		objectId, err := uuid.FromString(pbProfile.ObjectId)
		if err != nil {
			continue
		}

		profile := &models.Profile{
			ObjectId:    objectId,
			FullName:    pbProfile.FullName,
			SocialName:  pbProfile.SocialName,
			Email:       pbProfile.Email,
			Avatar:      pbProfile.Avatar,
			Banner:      pbProfile.Banner,
			Tagline:     pbProfile.TagLine,
			CreatedDate: pbProfile.CreatedDate,
			LastUpdated: pbProfile.LastUpdated,
			LastSeen:    pbProfile.LastSeen,
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

