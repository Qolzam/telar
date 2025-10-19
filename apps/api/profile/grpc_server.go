package profile

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/profile/models"
	"github.com/qolzam/telar/apps/api/profile/services"
	pb "github.com/qolzam/telar/protos/gen/go/profilepb"
)

type grpcServer struct {
	pb.UnimplementedProfileServiceServer
	service services.ProfileServiceClient
}

func NewGrpcServer(svc services.ProfileServiceClient) pb.ProfileServiceServer {
	return &grpcServer{service: svc}
}

func (s *grpcServer) CreateProfile(ctx context.Context, req *pb.CreateProfileRequest) (*pb.CreateProfileResponse, error) {
	objectId, err := uuid.FromString(req.ObjectId)
	if err != nil {
		return nil, err
	}

	createReq := &models.CreateProfileRequest{
		ObjectId:    objectId,
		FullName:    &req.FullName,
		SocialName:  &req.SocialName,
		Email:       &req.Email,
		Avatar:      &req.Avatar,
		Banner:      &req.Banner,
		TagLine:     &req.TagLine,
		CreatedDate: &req.CreatedDate,
		LastUpdated: &req.LastUpdated,
	}

	if err := s.service.CreateProfileOnSignup(ctx, createReq); err != nil {
		return nil, err
	}

	return &pb.CreateProfileResponse{ObjectId: req.ObjectId}, nil
}

func (s *grpcServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	objectId, err := uuid.FromString(req.ObjectId)
	if err != nil {
		return nil, err
	}

	updateReq := &models.UpdateProfileRequest{}
	if req.FullName != nil {
		updateReq.FullName = req.FullName
	}
	if req.Avatar != nil {
		updateReq.Avatar = req.Avatar
	}
	if req.Banner != nil {
		updateReq.Banner = req.Banner
	}
	if req.TagLine != nil {
		updateReq.TagLine = req.TagLine
	}
	if req.SocialName != nil {
		updateReq.SocialName = req.SocialName
	}

	if err := s.service.UpdateProfile(ctx, objectId, updateReq); err != nil {
		return nil, err
	}

	return &pb.UpdateProfileResponse{}, nil
}

func (s *grpcServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	objectId, err := uuid.FromString(req.ObjectId)
	if err != nil {
		return nil, err
	}

	profile, err := s.service.GetProfile(ctx, objectId)
	if err != nil {
		return nil, err
	}

	pbProfile := &pb.Profile{
		ObjectId:    profile.ObjectId.String(),
		FullName:    profile.FullName,
		SocialName:  profile.SocialName,
		Email:       profile.Email,
		Avatar:      profile.Avatar,
		Banner:      profile.Banner,
		TagLine:     profile.TagLine,
		CreatedDate: profile.CreatedDate,
		LastUpdated: profile.LastUpdated,
		LastSeen:    profile.LastSeen,
	}

	return &pb.GetProfileResponse{Profile: pbProfile}, nil
}

func (s *grpcServer) GetProfilesByIds(ctx context.Context, req *pb.GetProfilesByIdsRequest) (*pb.GetProfilesByIdsResponse, error) {
	userIds := make([]uuid.UUID, 0, len(req.ObjectIds))
	for _, idStr := range req.ObjectIds {
		id, err := uuid.FromString(idStr)
		if err != nil {
			continue
		}
		userIds = append(userIds, id)
	}

	profiles, err := s.service.GetProfilesByIds(ctx, userIds)
	if err != nil {
		return nil, err
	}

	pbProfiles := make([]*pb.Profile, 0, len(profiles))
	for _, profile := range profiles {
		pbProfile := &pb.Profile{
			ObjectId:    profile.ObjectId.String(),
			FullName:    profile.FullName,
			SocialName:  profile.SocialName,
			Email:       profile.Email,
			Avatar:      profile.Avatar,
			Banner:      profile.Banner,
			TagLine:     profile.TagLine,
			CreatedDate: profile.CreatedDate,
			LastUpdated: profile.LastUpdated,
			LastSeen:    profile.LastSeen,
		}
		pbProfiles = append(pbProfiles, pbProfile)
	}

	return &pb.GetProfilesByIdsResponse{Profiles: pbProfiles}, nil
}

