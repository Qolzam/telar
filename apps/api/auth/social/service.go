package social

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/utils"
)

type Service struct{ base *platform.BaseService }

func NewService(base *platform.BaseService) *Service { return &Service{base: base} }

// SocialProfile represents a user's social media profile
type SocialProfile struct {
	ObjectId    uuid.UUID `json:"objectId" bson:"objectId"`
	UserId      uuid.UUID `json:"userId" bson:"userId"`
	Platform    string    `json:"platform" bson:"platform"`
	Username    string    `json:"username" bson:"username"`
	ProfileUrl  string    `json:"profileUrl" bson:"profileUrl"`
	Verified    bool      `json:"verified" bson:"verified"`
	CreatedDate int64     `json:"created_date" bson:"created_date"`
	LastUpdated int64     `json:"last_updated" bson:"last_updated"`
}

// CreateSocialProfile creates a new social profile for a user
func (s *Service) CreateSocialProfile(ctx context.Context, userId uuid.UUID, platform, username, profileUrl string) (*SocialProfile, error) {
	now := utils.UTCNowUnix()

	socialProfile := &SocialProfile{
		ObjectId:    uuid.Must(uuid.NewV4()),
		UserId:      userId,
		Platform:    platform,
		Username:    username,
		ProfileUrl:  profileUrl,
		Verified:    false,
		CreatedDate: now,
		LastUpdated: now,
	}

	result := <-s.base.Repository.Save(ctx, "socialProfiles", socialProfile)
	if result.Error != nil {
		return nil, errors.WrapDatabaseError(fmt.Errorf("failed to create social profile: %w", result.Error))
	}

	return socialProfile, nil
}

// FindSocialProfileByUserId finds social profiles for a specific user
func (s *Service) FindSocialProfileByUserId(ctx context.Context, userId uuid.UUID) ([]*SocialProfile, error) {
	filter := struct {
		UserId uuid.UUID `json:"userId" bson:"userId"`
	}{UserId: userId}

	res := <-s.base.Repository.Find(ctx, "socialProfiles", filter, nil)
	if res.Error() != nil {
		return nil, errors.WrapDatabaseError(fmt.Errorf("failed to find social profiles: %w", res.Error()))
	}

	var profiles []*SocialProfile
	for res.Next() {
		var profile SocialProfile
		if err := res.Decode(&profile); err != nil {
			continue // Skip invalid profiles
		}
		profiles = append(profiles, &profile)
	}

	return profiles, nil
}

// FindSocialProfileByPlatform finds social profiles by platform and username
func (s *Service) FindSocialProfileByPlatform(ctx context.Context, platform, username string) (*SocialProfile, error) {
	filter := struct {
		Platform string `json:"platform" bson:"platform"`
		Username string `json:"username" bson:"username"`
	}{Platform: platform, Username: username}

	res := <-s.base.Repository.FindOne(ctx, "socialProfiles", filter)
	if res.Error() != nil {
		return nil, errors.WrapUserNotFoundError(fmt.Errorf("social profile not found"))
	}

	var profile SocialProfile
	if err := res.Decode(&profile); err != nil {
		return nil, errors.WrapDatabaseError(fmt.Errorf("failed to decode social profile: %w", err))
	}

	return &profile, nil
}

// UpdateSocialProfile updates an existing social profile
func (s *Service) UpdateSocialProfile(ctx context.Context, profileId uuid.UUID, updates *models.DatabaseUpdate) error {
	filter := struct {
		ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
	}{ObjectId: profileId}

	// Ensure lastUpdated is set
	if updates.Set == nil {
		updates.Set = make(map[string]interface{})
	}
	updates.Set["lastUpdated"] = utils.UTCNowUnix()

	result := <-s.base.Repository.Update(ctx, "socialProfiles", filter, updates, nil)

	if result.Error != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to update social profile: %w", result.Error))
	}

	return nil
}

// DeleteSocialProfile deletes a social profile
func (s *Service) DeleteSocialProfile(ctx context.Context, profileId uuid.UUID) error {
	filter := struct {
		ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
	}{ObjectId: profileId}
	result := <-s.base.Repository.Delete(ctx, "socialProfiles", filter)

	if result.Error != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to delete social profile: %w", result.Error))
	}

	return nil
}

// VerifySocialProfile marks a social profile as verified
func (s *Service) VerifySocialProfile(ctx context.Context, profileId uuid.UUID) error {
	updates := &models.DatabaseUpdate{
		Set: map[string]interface{}{"verified": true},
	}
	return s.UpdateSocialProfile(ctx, profileId, updates)
}

// GetVerifiedSocialProfiles gets all verified social profiles for a user
func (s *Service) GetVerifiedSocialProfiles(ctx context.Context, userId uuid.UUID) ([]*SocialProfile, error) {
	filter := struct {
		UserId   uuid.UUID `json:"userId" bson:"userId"`
		Verified bool      `json:"verified" bson:"verified"`
	}{UserId: userId, Verified: true}

	res := <-s.base.Repository.Find(ctx, "socialProfiles", filter, nil)
	if res.Error() != nil {
		return nil, errors.WrapDatabaseError(fmt.Errorf("failed to find verified social profiles: %w", res.Error()))
	}

	var profiles []*SocialProfile
	for res.Next() {
		var profile SocialProfile
		if err := res.Decode(&profile); err != nil {
			continue // Skip invalid profiles
		}
		profiles = append(profiles, &profile)
	}

	return profiles, nil
}
