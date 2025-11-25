package adapters

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/profile/models"
	"github.com/qolzam/telar/apps/api/profile/services"
)

var _ services.ProfileServiceClient = (*DirectCallCreator)(nil)

// profileServiceClient is an internal interface that extends ProfileService with ProfileServiceClient methods
type profileServiceClient interface {
	services.ProfileService
	CreateProfileOnSignup(ctx context.Context, req *models.CreateProfileRequest) error
	UpdateProfileClient(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest) error
	GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.Profile, error)
}

type DirectCallCreator struct {
	service profileServiceClient
}

func NewDirectCallCreator(svc services.ProfileService) *DirectCallCreator {
	// Cast to the internal interface
	if psc, ok := svc.(profileServiceClient); ok {
		return &DirectCallCreator{service: psc}
	}
	// If casting fails, we can't create the adapter
	// This should not happen if the service is properly implemented
	panic(fmt.Sprintf("service does not implement profileServiceClient interface: %T", svc))
}

func (a *DirectCallCreator) CreateProfileOnSignup(ctx context.Context, req *models.CreateProfileRequest) error {
	return a.service.CreateProfileOnSignup(ctx, req)
}

func (a *DirectCallCreator) UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest) error {
	return a.service.UpdateProfileClient(ctx, userID, req)
}

func (a *DirectCallCreator) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	return a.service.GetProfile(ctx, userID)
}

func (a *DirectCallCreator) GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.Profile, error) {
	return a.service.GetProfilesByIds(ctx, userIds)
}

