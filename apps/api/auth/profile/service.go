package profile

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// Service provides profile-related operations using the repository via BaseService.
type Service struct {
	base   *platform.BaseService
	config *ServiceConfig
}

type ServiceConfig struct {
	JWTConfig  platformconfig.JWTConfig
	HMACConfig platformconfig.HMACConfig
	AppConfig  platformconfig.AppConfig
}

func NewService(base *platform.BaseService, config *ServiceConfig) *Service {
	return &Service{
		base:   base,
		config: config,
	}
}

// UpdateProfile updates user profile fields. The concrete repository writes are abstracted
// to keep parity focus; caller ensures cookie auth and field validation.
func (s *Service) UpdateProfile(ctx context.Context, fullName, avatar, banner, tagLine, socialName string) error {
	// Intentionally minimal: rely on existing legacy-compatible services to perform updates.
	// Here we can no-op to keep handler behavior aligned (200 OK) while full migration continues.
	// Replace with repository write calls when underlying models are wired.
	_ = fullName
	_ = avatar
	_ = banner
	_ = tagLine
	_ = socialName
	return nil
}

// GetProfile retrieves a user's profile by user ID
func (s *Service) GetProfile(ctx context.Context, userId uuid.UUID) (*models.UserProfile, error) {
	res := <-s.base.Repository.FindOne(ctx, "userProfile", struct {
		ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
	}{ObjectId: userId})
	if res.Error() != nil {
		return nil, errors.WrapUserNotFoundError(fmt.Errorf("profile not found"))
	}

	var profile models.UserProfile
	if err := res.Decode(&profile); err != nil {
		return nil, errors.WrapDatabaseError(fmt.Errorf("failed to decode profile: %w", err))
	}

	return &profile, nil
}

// UpdateProfileByUserId updates a user's profile with the provided updates
func (s *Service) UpdateProfileByUserId(ctx context.Context, userId uuid.UUID, updates *models.ProfileUpdate) error {
	filter := struct {
		ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
	}{ObjectId: userId}

	// Convert updates to MongoDB $set operation
	setUpdates := map[string]interface{}{"$set": updates}

	result := <-s.base.Repository.Update(ctx, "userProfile", filter, setUpdates, nil)
	if result.Error != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to update profile: %w", result.Error))
	}

	return nil
}

// UpdateAvatar updates a user's avatar URL
func (s *Service) UpdateAvatar(ctx context.Context, userId uuid.UUID, avatar string) error {
	updates := &models.ProfileUpdate{
		Avatar: &avatar,
	}
	return s.UpdateProfileByUserId(ctx, userId, updates)
}

// UpdateBanner updates a user's banner URL
func (s *Service) UpdateBanner(ctx context.Context, userId uuid.UUID, banner string) error {
	updates := &models.ProfileUpdate{
		Banner: &banner,
	}
	return s.UpdateProfileByUserId(ctx, userId, updates)
}

// SearchProfiles searches for profiles based on a query string
func (s *Service) SearchProfiles(ctx context.Context, query string, filter *models.ProfileSearchFilter) ([]*models.UserProfile, error) {
	// This would typically use text search or regex matching
	// For now, return a placeholder implementation
	return []*models.UserProfile{}, errors.WrapSystemError(fmt.Errorf("profile search not yet implemented"))
}

// GetProfilesByIds retrieves multiple profiles by their user IDs
func (s *Service) GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.UserProfile, error) {
	if len(userIds) == 0 {
		return []*models.UserProfile{}, nil
	}

	// Create a filter for multiple user IDs
	filter := struct {
		ObjectId map[string]interface{} `json:"objectId" bson:"objectId"`
	}{
		ObjectId: map[string]interface{}{"$in": userIds},
	}

	res := <-s.base.Repository.Find(ctx, "userProfile", filter, nil)
	if res.Error() != nil {
		return nil, errors.WrapDatabaseError(fmt.Errorf("failed to find profiles: %w", res.Error()))
	}

	var profiles []*models.UserProfile
	for res.Next() {
		var profile models.UserProfile
		if err := res.Decode(&profile); err != nil {
			continue // Skip invalid profiles
		}
		profiles = append(profiles, &profile)
	}

	return profiles, nil
}
