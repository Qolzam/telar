package adapters

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/profile/models"
	"github.com/qolzam/telar/apps/api/profile/services"
)

var _ services.ProfileServiceClient = (*DirectCallCreator)(nil)

type DirectCallCreator struct {
	service *services.Service
}

func NewDirectCallCreator(svc *services.Service) *DirectCallCreator {
	return &DirectCallCreator{service: svc}
}

func (a *DirectCallCreator) CreateProfileOnSignup(ctx context.Context, req *models.CreateProfileRequest) error {
	return a.service.CreateProfileOnSignup(ctx, req)
}

func (a *DirectCallCreator) UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest) error {
	return a.service.UpdateProfile(ctx, userID, req)
}

func (a *DirectCallCreator) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	return a.service.GetProfile(ctx, userID)
}

func (a *DirectCallCreator) GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.Profile, error) {
	return a.service.GetProfilesByIds(ctx, userIds)
}

