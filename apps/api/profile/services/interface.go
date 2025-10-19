package services

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/profile/models"
)

type ProfileService interface {
	CreateProfile(ctx context.Context, req *models.CreateProfileRequest, user *types.UserContext) (*models.Profile, error)
	CreateIndex(ctx context.Context, indexes map[string]interface{}) error

	GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error)
	GetProfileBySocialName(ctx context.Context, socialName string) (*models.Profile, error)
	GetProfilesBySearch(ctx context.Context, query string, filter *models.ProfileQueryFilter) (*models.ProfilesResponse, error)
	QueryProfiles(ctx context.Context, filter *models.ProfileQueryFilter) (*models.ProfilesResponse, error)
	
	UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest, user *types.UserContext) error
	UpdateLastSeen(ctx context.Context, userID uuid.UUID) error
	UpdateProfileFields(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error
	
	DeleteProfile(ctx context.Context, userID uuid.UUID, user *types.UserContext) error
	SoftDeleteProfile(ctx context.Context, userID uuid.UUID, user *types.UserContext) error

	ValidateProfileOwnership(ctx context.Context, userID uuid.UUID, currentUser *types.UserContext) error
	
	SetField(ctx context.Context, userID uuid.UUID, field string, value interface{}) error
	IncrementField(ctx context.Context, userID uuid.UUID, field string, delta int) error
	UpdateByOwner(ctx context.Context, userID uuid.UUID, owner uuid.UUID, fields map[string]interface{}) error

	UpdateFields(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error
	IncrementFields(ctx context.Context, userID uuid.UUID, increments map[string]interface{}) error
	UpdateAndIncrementFields(ctx context.Context, userID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error
	UpdateFieldsWithOwnership(ctx context.Context, userID uuid.UUID, ownerID uuid.UUID, updates map[string]interface{}) error
	DeleteWithOwnership(ctx context.Context, userID uuid.UUID, ownerID uuid.UUID) error
	IncrementFieldsWithOwnership(ctx context.Context, userID uuid.UUID, ownerID uuid.UUID, increments map[string]interface{}) error
}

// ProfileServiceClient is the public interface for profile operations from other services.
// This interface enables the Public Interface + Adapter pattern for service-to-service communication:
// - Other services (e.g., Auth) depend on this interface, not concrete implementations
// - Multiple adapters implement this interface for different deployment modes:
//   • DirectCallAdapter: In-process calls for serverless/monolith deployment
//   • GrpcAdapter: Network calls via gRPC for microservices deployment
// This pattern allows the same codebase to work in both serverless and Kubernetes environments.
type ProfileServiceClient interface {
	CreateProfileOnSignup(ctx context.Context, req *models.CreateProfileRequest) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest) error
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error)
	GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.Profile, error)
}

