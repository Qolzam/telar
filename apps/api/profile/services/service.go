// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/lib/pq"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
	profileErrors "github.com/qolzam/telar/apps/api/profile/errors"
	"github.com/qolzam/telar/apps/api/profile/models"
	"github.com/qolzam/telar/apps/api/profile/repository"
)

// profileService implements the ProfileService interface
type profileService struct {
	repo   repository.ProfileRepository
	config *platformconfig.Config
}

// Ensure profileService implements ProfileService interface
var _ ProfileService = (*profileService)(nil)

// NewProfileService creates a new ProfileService with the given repository
func NewProfileService(repo repository.ProfileRepository, cfg *platformconfig.Config) ProfileService {
	return &profileService{
		repo:   repo,
		config: cfg,
	}
}

// CreateProfile creates a new profile
func (s *profileService) CreateProfile(ctx context.Context, req *models.CreateProfileRequest, user *types.UserContext) (*models.Profile, error) {
	if req == nil {
		return nil, fmt.Errorf("create profile request is required")
	}
	if user == nil {
		return nil, fmt.Errorf("user context is required")
	}

	now := time.Now()
	createdDate := utils.UTCNowUnix()
	if req.CreatedDate != nil {
		createdDate = *req.CreatedDate
	}

	// Map request to Profile model
	profile := &models.Profile{
		ObjectId:      req.ObjectId,
		FullName:      getStringValue(req.FullName),
		SocialName:    getStringValue(req.SocialName),
		Email:         getStringValue(req.Email),
		Avatar:        getStringValue(req.Avatar),
		Banner:        getStringValue(req.Banner),
		Tagline:       getStringValue(req.TagLine),
		CreatedDate:   createdDate,
		LastUpdated:   getInt64Value(req.LastUpdated, createdDate),
		LastSeen:      getInt64Value(req.LastSeen, 0),
		Birthday:      getInt64Value(req.Birthday, 0),
		WebUrl:        getStringValue(req.WebUrl),
		CompanyName:   getStringValue(req.CompanyName),
		Country:       getStringValue(req.Country),
		Address:       getStringValue(req.Address),
		Phone:         getStringValue(req.Phone),
		VoteCount:     getInt64Value(req.VoteCount, 0),
		ShareCount:    getInt64Value(req.ShareCount, 0),
		FollowCount:   getInt64Value(req.FollowCount, 0),
		FollowerCount: getInt64Value(req.FollowerCount, 0),
		PostCount:     getInt64Value(req.PostCount, 0),
		FacebookId:    getStringValue(req.FacebookId),
		InstagramId:   getStringValue(req.InstagramId),
		TwitterId:     getStringValue(req.TwitterId),
		LinkedInId:    getStringValue(req.LinkedInId),
		Permission:    getStringValue(req.Permission, "Public"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if req.AccessUserList != nil {
		profile.AccessUserList = pq.StringArray(req.AccessUserList)
	}

	// Save to database using new repository
	err := s.repo.Create(ctx, profile)
	if err != nil {
		// Handle unique constraint violation on social_name
		if strings.Contains(err.Error(), "social name already exists") {
			return nil, profileErrors.ErrProfileAlreadyExists
		}
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	return profile, nil
}

// CreateIndex is deprecated - indexes are now created via SQL migrations
func (s *profileService) CreateIndex(ctx context.Context, indexes map[string]interface{}) error {
	// No-op: Index creation is handled by SQL migrations
	return nil
}

// GetProfile retrieves a profile by user ID
func (s *profileService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return nil, profileErrors.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return profile, nil
}

// GetProfileBySocialName retrieves a profile by social name
func (s *profileService) GetProfileBySocialName(ctx context.Context, socialName string) (*models.Profile, error) {
	profile, err := s.repo.FindBySocialName(ctx, socialName)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return nil, profileErrors.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile by social name: %w", err)
	}
	return profile, nil
}

// GetProfilesBySearch searches profiles by query string
func (s *profileService) GetProfilesBySearch(ctx context.Context, query string, filter *models.ProfileQueryFilter) (*models.ProfilesResponse, error) {
	if filter == nil {
		filter = &models.ProfileQueryFilter{
			Limit: 10,
			Page:  1,
		}
	}

	limit := int(filter.Limit)
	if limit <= 0 {
		limit = 10
	}
	offset := int((filter.Page - 1) * filter.Limit)

	repoFilter := repository.ProfileFilter{}
	if query != "" {
		repoFilter.SearchText = &query
	}

	profiles, err := s.repo.Find(ctx, repoFilter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search profiles: %w", err)
	}

	total, err := s.repo.Count(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count profiles: %w", err)
	}

	// Convert []*models.Profile to []models.Profile
	profileList := make([]models.Profile, len(profiles))
	for i, p := range profiles {
		profileList[i] = *p
	}

	return &models.ProfilesResponse{
		Profiles: profileList,
		Total:    total,
	}, nil
}

// SearchProfiles returns a limited list of profiles for autocomplete
func (s *profileService) SearchProfiles(ctx context.Context, query string, limit int) ([]*models.Profile, error) {
	trimmed := strings.TrimSpace(query)
	if len(trimmed) < 3 {
		return []*models.Profile{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	profiles, err := s.repo.Search(ctx, trimmed, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search profiles: %w", err)
	}

	return profiles, nil
}

// QueryProfiles queries profiles with filter
func (s *profileService) QueryProfiles(ctx context.Context, filter *models.ProfileQueryFilter) (*models.ProfilesResponse, error) {
	if filter == nil {
		filter = &models.ProfileQueryFilter{
			Limit: 10,
			Page:  1,
		}
	}

	limit := int(filter.Limit)
	if limit <= 0 {
		limit = 10
	}
	offset := int((filter.Page - 1) * filter.Limit)

	repoFilter := repository.ProfileFilter{}
	if filter.Search != "" {
		repoFilter.SearchText = &filter.Search
	}

	profiles, err := s.repo.Find(ctx, repoFilter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}

	total, err := s.repo.Count(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count profiles: %w", err)
	}

	// Convert []*models.Profile to []models.Profile
	profileList := make([]models.Profile, len(profiles))
	for i, p := range profiles {
		profileList[i] = *p
	}

	return &models.ProfilesResponse{
		Profiles: profileList,
		Total:    total,
	}, nil
}

// UpdateProfile updates a profile (ProfileService interface - with user context for ownership validation)
func (s *profileService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest, user *types.UserContext) error {
	if req == nil {
		return fmt.Errorf("update profile request is required")
	}
	if user == nil {
		return fmt.Errorf("user context is required")
	}

	// Validate ownership
	if err := s.ValidateProfileOwnership(ctx, userID, user); err != nil {
		return err
	}

	// Delegate to the ProfileServiceClient version (no ownership check needed)
	return s.updateProfileInternal(ctx, userID, req)
}

// updateProfileInternal is the internal implementation for updating a profile
func (s *profileService) updateProfileInternal(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest) error {
	// Load existing profile
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Apply updates
	if req.FullName != nil {
		profile.FullName = *req.FullName
	}
	if req.Avatar != nil {
		profile.Avatar = *req.Avatar
	}
	if req.Banner != nil {
		profile.Banner = *req.Banner
	}
	if req.TagLine != nil {
		profile.Tagline = *req.TagLine
	}
	if req.SocialName != nil {
		profile.SocialName = *req.SocialName
	}
	if req.WebUrl != nil {
		profile.WebUrl = *req.WebUrl
	}
	if req.CompanyName != nil {
		profile.CompanyName = *req.CompanyName
	}
	if req.FacebookId != nil {
		profile.FacebookId = *req.FacebookId
	}
	if req.InstagramId != nil {
		profile.InstagramId = *req.InstagramId
	}
	if req.TwitterId != nil {
		profile.TwitterId = *req.TwitterId
	}

	// Save updated profile
	err = s.repo.Update(ctx, profile)
	if err != nil {
		// Handle unique constraint violation on social_name
		if strings.Contains(err.Error(), "social name already exists") {
			return profileErrors.ErrProfileAlreadyExists
		}
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// UpdateLastSeen updates the last seen timestamp
func (s *profileService) UpdateLastSeen(ctx context.Context, userID uuid.UUID) error {
	return s.repo.UpdateLastSeen(ctx, userID)
}

// UpdateProfileFields updates profile fields using a map
func (s *profileService) UpdateProfileFields(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	// Load existing profile
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "fullName", "full_name":
			if str, ok := value.(string); ok {
				profile.FullName = str
			}
		case "socialName", "social_name":
			if str, ok := value.(string); ok {
				profile.SocialName = str
			}
		case "email":
			if str, ok := value.(string); ok {
				profile.Email = str
			}
		case "avatar":
			if str, ok := value.(string); ok {
				profile.Avatar = str
			}
		case "banner":
			if str, ok := value.(string); ok {
				profile.Banner = str
			}
		case "tagline", "tagLine":
			if str, ok := value.(string); ok {
				profile.Tagline = str
			}
		case "webUrl", "web_url":
			if str, ok := value.(string); ok {
				profile.WebUrl = str
			}
		case "companyName", "company_name":
			if str, ok := value.(string); ok {
				profile.CompanyName = str
			}
		case "country":
			if str, ok := value.(string); ok {
				profile.Country = str
			}
		case "address":
			if str, ok := value.(string); ok {
				profile.Address = str
			}
		case "phone":
			if str, ok := value.(string); ok {
				profile.Phone = str
			}
		case "voteCount", "vote_count":
			if num, ok := value.(int64); ok {
				profile.VoteCount = num
			} else if num, ok := value.(int); ok {
				profile.VoteCount = int64(num)
			}
		case "shareCount", "share_count":
			if num, ok := value.(int64); ok {
				profile.ShareCount = num
			} else if num, ok := value.(int); ok {
				profile.ShareCount = int64(num)
			}
		case "followCount", "follow_count":
			if num, ok := value.(int64); ok {
				profile.FollowCount = num
			} else if num, ok := value.(int); ok {
				profile.FollowCount = int64(num)
			}
		case "followerCount", "follower_count":
			if num, ok := value.(int64); ok {
				profile.FollowerCount = num
			} else if num, ok := value.(int); ok {
				profile.FollowerCount = int64(num)
			}
		case "postCount", "post_count":
			if num, ok := value.(int64); ok {
				profile.PostCount = num
			} else if num, ok := value.(int); ok {
				profile.PostCount = int64(num)
			}
		case "permission":
			if str, ok := value.(string); ok {
				profile.Permission = str
			}
		}
	}

	// Save updated profile
	err = s.repo.Update(ctx, profile)
	if err != nil {
		if strings.Contains(err.Error(), "social name already exists") {
			return profileErrors.ErrProfileAlreadyExists
		}
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// DeleteProfile deletes a profile (hard delete)
func (s *profileService) DeleteProfile(ctx context.Context, userID uuid.UUID, user *types.UserContext) error {
	if user == nil {
		return fmt.Errorf("user context is required")
	}

	// Validate ownership
	if err := s.ValidateProfileOwnership(ctx, userID, user); err != nil {
		return err
	}

	return s.repo.Delete(ctx, userID)
}

// SoftDeleteProfile is not applicable for profiles (no soft delete field)
// This method is kept for interface compatibility but performs hard delete
func (s *profileService) SoftDeleteProfile(ctx context.Context, userID uuid.UUID, user *types.UserContext) error {
	return s.DeleteProfile(ctx, userID, user)
}

// ValidateProfileOwnership validates that the current user owns the profile
func (s *profileService) ValidateProfileOwnership(ctx context.Context, userID uuid.UUID, currentUser *types.UserContext) error {
	if currentUser == nil {
		return profileErrors.ErrInvalidUserContext
	}

	if currentUser.UserID != userID {
		return profileErrors.ErrProfileOwnershipRequired
	}

	return nil
}

// SetField sets a single field on a profile
func (s *profileService) SetField(ctx context.Context, userID uuid.UUID, field string, value interface{}) error {
	updates := map[string]interface{}{field: value}
	return s.UpdateProfileFields(ctx, userID, updates)
}

// IncrementField increments a numeric field on a profile
func (s *profileService) IncrementField(ctx context.Context, userID uuid.UUID, field string, delta int) error {
	// Load existing profile
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Apply increment
	switch field {
	case "voteCount", "vote_count":
		profile.VoteCount += int64(delta)
	case "shareCount", "share_count":
		profile.ShareCount += int64(delta)
	case "followCount", "follow_count":
		profile.FollowCount += int64(delta)
	case "followerCount", "follower_count":
		profile.FollowerCount += int64(delta)
	case "postCount", "post_count":
		profile.PostCount += int64(delta)
	default:
		return fmt.Errorf("invalid field for increment: %s", field)
	}

	// Save updated profile
	err = s.repo.Update(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// UpdateByOwner updates profile fields with ownership validation
func (s *profileService) UpdateByOwner(ctx context.Context, userID uuid.UUID, owner uuid.UUID, fields map[string]interface{}) error {
	// Validate ownership
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile.ObjectId != owner {
		return profileErrors.ErrProfileOwnershipRequired
	}

	return s.UpdateProfileFields(ctx, userID, fields)
}

// UpdateFields is an alias for UpdateProfileFields
func (s *profileService) UpdateFields(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	return s.UpdateProfileFields(ctx, userID, updates)
}

// IncrementFields increments multiple numeric fields on a profile
func (s *profileService) IncrementFields(ctx context.Context, userID uuid.UUID, increments map[string]interface{}) error {
	// Load existing profile
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Apply increments
	for field, delta := range increments {
		var deltaInt int
		if num, ok := delta.(int); ok {
			deltaInt = num
		} else if num, ok := delta.(int64); ok {
			deltaInt = int(num)
		} else {
			continue
		}

		switch field {
		case "voteCount", "vote_count":
			profile.VoteCount += int64(deltaInt)
		case "shareCount", "share_count":
			profile.ShareCount += int64(deltaInt)
		case "followCount", "follow_count":
			profile.FollowCount += int64(deltaInt)
		case "followerCount", "follower_count":
			profile.FollowerCount += int64(deltaInt)
		case "postCount", "post_count":
			profile.PostCount += int64(deltaInt)
		}
	}

	// Save updated profile
	err = s.repo.Update(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// UpdateAndIncrementFields updates and increments fields in a single operation
func (s *profileService) UpdateAndIncrementFields(ctx context.Context, userID uuid.UUID, updates map[string]interface{}, increments map[string]interface{}) error {
	// Load existing profile
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "fullName", "full_name":
			if str, ok := value.(string); ok {
				profile.FullName = str
			}
		case "socialName", "social_name":
			if str, ok := value.(string); ok {
				profile.SocialName = str
			}
		case "avatar":
			if str, ok := value.(string); ok {
				profile.Avatar = str
			}
		case "banner":
			if str, ok := value.(string); ok {
				profile.Banner = str
			}
		case "tagline", "tagLine":
			if str, ok := value.(string); ok {
				profile.Tagline = str
			}
		}
	}

	// Apply increments
	for field, delta := range increments {
		var deltaInt int
		if num, ok := delta.(int); ok {
			deltaInt = num
		} else if num, ok := delta.(int64); ok {
			deltaInt = int(num)
		} else {
			continue
		}

		switch field {
		case "voteCount", "vote_count":
			profile.VoteCount += int64(deltaInt)
		case "shareCount", "share_count":
			profile.ShareCount += int64(deltaInt)
		case "followCount", "follow_count":
			profile.FollowCount += int64(deltaInt)
		case "followerCount", "follower_count":
			profile.FollowerCount += int64(deltaInt)
		case "postCount", "post_count":
			profile.PostCount += int64(deltaInt)
		}
	}

	// Save updated profile
	err = s.repo.Update(ctx, profile)
	if err != nil {
		if strings.Contains(err.Error(), "social name already exists") {
			return profileErrors.ErrProfileAlreadyExists
		}
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// UpdateFieldsWithOwnership updates profile fields with ownership validation
func (s *profileService) UpdateFieldsWithOwnership(ctx context.Context, userID uuid.UUID, ownerID uuid.UUID, updates map[string]interface{}) error {
	// Validate ownership
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile.ObjectId != ownerID {
		return profileErrors.ErrProfileOwnershipRequired
	}

	return s.UpdateProfileFields(ctx, userID, updates)
}

// DeleteWithOwnership deletes a profile with ownership validation
func (s *profileService) DeleteWithOwnership(ctx context.Context, userID uuid.UUID, ownerID uuid.UUID) error {
	// Validate ownership
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile.ObjectId != ownerID {
		return profileErrors.ErrProfileOwnershipRequired
	}

	return s.repo.Delete(ctx, userID)
}

// IncrementFieldsWithOwnership increments fields with ownership validation
func (s *profileService) IncrementFieldsWithOwnership(ctx context.Context, userID uuid.UUID, ownerID uuid.UUID, increments map[string]interface{}) error {
	// Validate ownership
	profile, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "profile not found") {
			return profileErrors.ErrProfileNotFound
		}
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile.ObjectId != ownerID {
		return profileErrors.ErrProfileOwnershipRequired
	}

	return s.IncrementFields(ctx, userID, increments)
}

// ProfileServiceClient interface methods

// CreateProfileOnSignup creates a profile during signup flow
func (s *profileService) CreateProfileOnSignup(ctx context.Context, req *models.CreateProfileRequest) error {
	if req == nil {
		return fmt.Errorf("create profile request is required")
	}

	// Use CreateOrUpdateDTO logic: try to create, if exists then update
	profile, err := s.repo.FindByID(ctx, req.ObjectId)
	if err != nil {
		// If profile not found, that's OK - we'll create it
		if !strings.Contains(err.Error(), "profile not found") {
			return fmt.Errorf("failed to check profile existence: %w", err)
		}
		// Profile not found - set profile to nil so we create it
		profile = nil
	}

	if profile != nil {
		// Profile exists, update it
		if req.FullName != nil {
			profile.FullName = *req.FullName
		}
		if req.SocialName != nil {
			profile.SocialName = *req.SocialName
		}
		if req.Email != nil {
			profile.Email = *req.Email
		}
		if req.Avatar != nil {
			profile.Avatar = *req.Avatar
		}
		if req.Banner != nil {
			profile.Banner = *req.Banner
		}
		if req.TagLine != nil {
			profile.Tagline = *req.TagLine
		}
		if req.AccessUserList != nil {
			profile.AccessUserList = pq.StringArray(req.AccessUserList)
		}

		return s.repo.Update(ctx, profile)
	}

	// Profile doesn't exist, create it
	now := time.Now()
	createdDate := utils.UTCNowUnix()
	if req.CreatedDate != nil {
		createdDate = *req.CreatedDate
	}

	newProfile := &models.Profile{
		ObjectId:      req.ObjectId,
		FullName:      getStringValue(req.FullName),
		SocialName:    getStringValue(req.SocialName),
		Email:         getStringValue(req.Email),
		Avatar:        getStringValue(req.Avatar),
		Banner:        getStringValue(req.Banner),
		Tagline:       getStringValue(req.TagLine),
		CreatedDate:   createdDate,
		LastUpdated:   getInt64Value(req.LastUpdated, createdDate),
		LastSeen:      getInt64Value(req.LastSeen, 0),
		Birthday:      getInt64Value(req.Birthday, 0),
		WebUrl:        getStringValue(req.WebUrl),
		CompanyName:   getStringValue(req.CompanyName),
		Country:       getStringValue(req.Country),
		Address:       getStringValue(req.Address),
		Phone:         getStringValue(req.Phone),
		VoteCount:     getInt64Value(req.VoteCount, 0),
		ShareCount:    getInt64Value(req.ShareCount, 0),
		FollowCount:   getInt64Value(req.FollowCount, 0),
		FollowerCount: getInt64Value(req.FollowerCount, 0),
		PostCount:     getInt64Value(req.PostCount, 0),
		FacebookId:    getStringValue(req.FacebookId),
		InstagramId:   getStringValue(req.InstagramId),
		TwitterId:     getStringValue(req.TwitterId),
		LinkedInId:    getStringValue(req.LinkedInId),
		Permission:    getStringValue(req.Permission, "Public"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if req.AccessUserList != nil {
		newProfile.AccessUserList = pq.StringArray(req.AccessUserList)
	}

	err = s.repo.Create(ctx, newProfile)
	if err != nil {
		if strings.Contains(err.Error(), "social name already exists") {
			return profileErrors.ErrProfileAlreadyExists
		}
		return fmt.Errorf("failed to create profile: %w", err)
	}

	return nil
}

// ProfileServiceClient methods - these are used by adapters
// Note: These methods have the same names as ProfileService methods but different signatures
// The adapter will need to be updated to use these methods

// UpdateProfileClient updates a profile without user context (for ProfileServiceClient interface)
// This is a separate method to avoid conflict with ProfileService.UpdateProfile
func (s *profileService) UpdateProfileClient(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest) error {
	return s.updateProfileInternal(ctx, userID, req)
}

// GetProfileClient retrieves a profile by user ID (for ProfileServiceClient interface)
func (s *profileService) GetProfileClient(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	return s.GetProfile(ctx, userID)
}

// GetProfilesByIds retrieves multiple profiles by user IDs
// This method is used by both ProfileService interface (via backward compatibility) and ProfileServiceClient
func (s *profileService) GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.Profile, error) {
	return s.repo.FindByIDs(ctx, userIds)
}

// GetProfilesByIdsClient retrieves multiple profiles by user IDs (for ProfileServiceClient interface)
func (s *profileService) GetProfilesByIdsClient(ctx context.Context, userIds []uuid.UUID) ([]*models.Profile, error) {
	return s.GetProfilesByIds(ctx, userIds)
}

// Helper methods for backward compatibility

// FindByID is kept for backward compatibility
func (s *profileService) FindByID(ctx context.Context, id uuid.UUID) (*models.Profile, error) {
	return s.GetProfile(ctx, id)
}

// FindBySocialName is kept for backward compatibility
func (s *profileService) FindBySocialName(ctx context.Context, name string) (*models.Profile, error) {
	return s.GetProfileBySocialName(ctx, name)
}

// FindManyByIDs is kept for backward compatibility
func (s *profileService) FindManyByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Profile, error) {
	return s.GetProfilesByIds(ctx, ids)
}

// FindMyProfile is kept for backward compatibility
func (s *profileService) FindMyProfile(ctx context.Context, userId uuid.UUID) (*models.Profile, error) {
	return s.GetProfile(ctx, userId)
}

// Query is kept for backward compatibility
func (s *profileService) Query(ctx context.Context, search string, limit int64, skip int64) ([]*models.Profile, error) {
	repoFilter := repository.ProfileFilter{}
	if search != "" {
		repoFilter.SearchText = &search
	}

	profiles, err := s.repo.Find(ctx, repoFilter, int(limit), int(skip))
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}

	return profiles, nil
}

// Increase is kept for backward compatibility
func (s *profileService) Increase(ctx context.Context, field string, inc int, userId uuid.UUID) error {
	return s.IncrementField(ctx, userId, field, inc)
}

// CreateOrUpdateDTO is kept for backward compatibility
func (s *profileService) CreateOrUpdateDTO(ctx context.Context, req *models.CreateProfileRequest) error {
	return s.CreateProfileOnSignup(ctx, req)
}

// Helper functions

func getStringValue(ptr *string, defaults ...string) string {
	if ptr != nil {
		return *ptr
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
}

func getInt64Value(ptr *int64, defaults ...int64) int64 {
	if ptr != nil {
		return *ptr
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return 0
}
