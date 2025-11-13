package social

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/utils"
)

type Service struct{ base *platform.BaseService }

func NewService(base *platform.BaseService) *Service { return &Service{base: base} }

// socialQueryBuilder is a private helper for building social service queries
type socialQueryBuilder struct {
	query *dbi.Query
}

func newSocialQueryBuilder() *socialQueryBuilder {
	return &socialQueryBuilder{
		query: &dbi.Query{
			Conditions: []dbi.Field{},
		},
	}
}

func (qb *socialQueryBuilder) WhereObjectID(objectID uuid.UUID) *socialQueryBuilder {
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:     "object_id", // Indexed column
		Value:    objectID,
		Operator: "=",
		IsJSONB:  false,
	})
	return qb
}

func (qb *socialQueryBuilder) WhereUserId(userId uuid.UUID) *socialQueryBuilder {
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:     "data->>'userId'", // JSONB field
		Value:    userId.String(),
		Operator: "=",
		IsJSONB:  true,
	})
	return qb
}

func (qb *socialQueryBuilder) WherePlatform(platform string) *socialQueryBuilder {
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:     "data->>'platform'", // JSONB field
		Value:    platform,
		Operator: "=",
		IsJSONB:  true,
	})
	return qb
}

func (qb *socialQueryBuilder) WhereUsername(username string) *socialQueryBuilder {
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:     "data->>'username'", // JSONB field
		Value:    username,
		Operator: "=",
		IsJSONB:  true,
	})
	return qb
}

func (qb *socialQueryBuilder) WhereVerified(verified bool) *socialQueryBuilder {
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:       "data->>'verified'", // JSONB field
		Value:      verified,
		Operator:   "=",
		IsJSONB:    true,
		JSONBCast:  "::boolean",
	})
	return qb
}

func (qb *socialQueryBuilder) Build() *dbi.Query {
	return qb.query
}

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

	result := <-s.base.Repository.Save(
		ctx,
		"socialProfiles",
		socialProfile.ObjectId,
		socialProfile.UserId,
		socialProfile.CreatedDate,
		socialProfile.LastUpdated,
		socialProfile,
	)
	if result.Error != nil {
		return nil, errors.WrapDatabaseError(fmt.Errorf("failed to create social profile: %w", result.Error))
	}

	return socialProfile, nil
}

// FindSocialProfileByUserId finds social profiles for a specific user
func (s *Service) FindSocialProfileByUserId(ctx context.Context, userId uuid.UUID) ([]*SocialProfile, error) {
	query := newSocialQueryBuilder().WhereUserId(userId).Build()
	res := <-s.base.Repository.Find(ctx, "socialProfiles", query, nil)
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
	query := newSocialQueryBuilder().WherePlatform(platform).WhereUsername(username).Build()
	res := <-s.base.Repository.FindOne(ctx, "socialProfiles", query)
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
	query := newSocialQueryBuilder().WhereObjectID(profileId).Build()

	// Ensure lastUpdated is set
	updateMap := make(map[string]interface{})
	if updates != nil && updates.Set != nil {
		for k, v := range updates.Set {
			updateMap[k] = v
		}
	}
	updateMap["lastUpdated"] = utils.UTCNowUnix()

	result := <-s.base.Repository.Update(ctx, "socialProfiles", query, updateMap, nil)

	if result.Error != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to update social profile: %w", result.Error))
	}

	return nil
}

// DeleteSocialProfile deletes a social profile
func (s *Service) DeleteSocialProfile(ctx context.Context, profileId uuid.UUID) error {
	query := newSocialQueryBuilder().WhereObjectID(profileId).Build()
	result := <-s.base.Repository.Delete(ctx, "socialProfiles", query)

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
	query := newSocialQueryBuilder().WhereUserId(userId).WhereVerified(true).Build()
	res := <-s.base.Repository.Find(ctx, "socialProfiles", query, nil)
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
