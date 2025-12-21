// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"errors"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	profileErrors "github.com/qolzam/telar/apps/api/profile/errors"
	"github.com/qolzam/telar/apps/api/profile/models"
)

// Test helper functions
func createTestUserContext() *types.UserContext {
	return &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "test@example.com",
		DisplayName: "Test User",
		SocialName:  "testuser",
		Avatar:      "https://example.com/avatar.jpg",
		SystemRole:  "user",
		CreatedDate: time.Now().Unix(),
	}
}

func createTestCreateProfileRequest() *models.CreateProfileRequest {
	userID := uuid.Must(uuid.NewV4())
	fullName := "Test User"
	socialName := "testuser"
	email := "test@example.com"
	avatar := "https://example.com/avatar.jpg"
	tagline := "Test tagline"
	
	return &models.CreateProfileRequest{
		ObjectId:   userID,
		FullName:   &fullName,
		SocialName: &socialName,
		Email:      &email,
		Avatar:     &avatar,
		TagLine:    &tagline,
	}
}

func createTestProfile() *models.Profile {
	userID := uuid.Must(uuid.NewV4())
	now := time.Now()
	
	return &models.Profile{
		ObjectId:      userID,
		FullName:      "Test User",
		SocialName:    "testuser",
		Email:         "test@example.com",
		Avatar:        "https://example.com/avatar.jpg",
		Banner:        "https://example.com/banner.jpg",
		Tagline:       "Test tagline",
		CreatedDate:   now.Unix(),
		LastUpdated:   now.Unix(),
		LastSeen:      now.Unix(),
		VoteCount:     0,
		ShareCount:    0,
		FollowCount:   0,
		FollowerCount: 0,
		PostCount:     0,
		Permission:    "Public",
		AccessUserList: pq.StringArray{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// setupTestService creates a profileService with a mocked ProfileRepository for testing
func setupTestService() (*profileService, *MockProfileRepository) {
	mockRepo := new(MockProfileRepository)
	
	// Create test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		Cache: platformconfig.CacheConfig{
			Enabled: false, // Disable cache for unit tests
		},
	}
	
	svc := &profileService{
		repo:   mockRepo,
		config: cfg,
	}
	
	return svc, mockRepo
}

// Test CreateProfile
func TestCreateProfile_ValidRequest_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreateProfileRequest()

	// Setup mock expectations - Create should be called with a profile that has the correct fields
	mockRepo.On("Create", ctx, mock.MatchedBy(func(profile *models.Profile) bool {
		return profile.ObjectId == req.ObjectId &&
			profile.FullName == *req.FullName &&
			profile.SocialName == *req.SocialName &&
			profile.Email == *req.Email
	})).Return(nil)

	// Execute
	result, err := service.CreateProfile(ctx, req, user)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.ObjectId, result.ObjectId)
	assert.Equal(t, *req.FullName, result.FullName)
	assert.Equal(t, *req.SocialName, result.SocialName)
	assert.Equal(t, *req.Email, result.Email)
	assert.Greater(t, result.CreatedDate, int64(0))

	mockRepo.AssertExpectations(t)
}

func TestCreateProfile_DatabaseError_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreateProfileRequest()

	mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("database connection failed"))

	result, err := service.CreateProfile(ctx, req, user)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create profile")
	mockRepo.AssertExpectations(t)
}

func TestCreateProfile_DuplicateSocialName_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreateProfileRequest()

	mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("social name already exists"))

	result, err := service.CreateProfile(ctx, req, user)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, profileErrors.ErrProfileAlreadyExists))
	mockRepo.AssertExpectations(t)
}

// Test GetProfile
func TestGetProfile_ValidID_ReturnsProfile(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	expectedProfile := createTestProfile()
	expectedProfile.ObjectId = userID

	mockRepo.On("FindByID", ctx, userID).Return(expectedProfile, nil)

	result, err := service.GetProfile(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.ObjectId)
	assert.Equal(t, expectedProfile.FullName, result.FullName)
	mockRepo.AssertExpectations(t)
}

func TestGetProfile_NotFound_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())

	mockRepo.On("FindByID", ctx, userID).Return(nil, errors.New("profile not found"))

	result, err := service.GetProfile(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, profileErrors.ErrProfileNotFound))
	mockRepo.AssertExpectations(t)
}

// Test GetProfileBySocialName
func TestGetProfileBySocialName_ValidName_ReturnsProfile(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	socialName := "testuser"
	expectedProfile := createTestProfile()
	expectedProfile.SocialName = socialName

	mockRepo.On("FindBySocialName", ctx, socialName).Return(expectedProfile, nil)

	result, err := service.GetProfileBySocialName(ctx, socialName)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, socialName, result.SocialName)
	mockRepo.AssertExpectations(t)
}

// Test UpdateProfile
func TestUpdateProfile_ValidRequest_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	userID := user.UserID
	
	fullName := "Updated Name"
	avatar := "https://example.com/new-avatar.jpg"
	req := &models.UpdateProfileRequest{
		FullName: &fullName,
		Avatar:   &avatar,
	}

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(profile *models.Profile) bool {
		return profile.ObjectId == userID &&
			profile.FullName == fullName &&
			profile.Avatar == avatar
	})).Return(nil)

	err := service.UpdateProfile(ctx, userID, req, user)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateProfile_NotFound_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	userID := user.UserID
	
	fullName := "Updated Name"
	req := &models.UpdateProfileRequest{
		FullName: &fullName,
	}

	mockRepo.On("FindByID", ctx, userID).Return(nil, errors.New("profile not found"))

	err := service.UpdateProfile(ctx, userID, req, user)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, profileErrors.ErrProfileNotFound))
	mockRepo.AssertExpectations(t)
}

func TestUpdateProfile_InvalidOwnership_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	userID := uuid.Must(uuid.NewV4()) // Different user ID
	
	fullName := "Updated Name"
	req := &models.UpdateProfileRequest{
		FullName: &fullName,
	}

	err := service.UpdateProfile(ctx, userID, req, user)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, profileErrors.ErrProfileOwnershipRequired))
	mockRepo.AssertNotCalled(t, "FindByID")
	mockRepo.AssertNotCalled(t, "Update")
}

// Test UpdateLastSeen
func TestUpdateLastSeen_ValidID_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())

	mockRepo.On("UpdateLastSeen", ctx, userID).Return(nil)

	err := service.UpdateLastSeen(ctx, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test IncrementField
func TestIncrementField_ValidField_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	delta := 5

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID
	existingProfile.FollowCount = 10

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(profile *models.Profile) bool {
		return profile.ObjectId == userID &&
			profile.FollowCount == int64(10+delta)
	})).Return(nil)

	err := service.IncrementField(ctx, userID, "followCount", delta)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test DeleteProfile
func TestDeleteProfile_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	userID := user.UserID

	// DeleteProfile validates ownership first, then calls Delete
	// Ownership validation doesn't call FindByID, it just checks user.UserID == userID
	mockRepo.On("Delete", ctx, userID).Return(nil)

	err := service.DeleteProfile(ctx, userID, user)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteProfile_InvalidOwnership_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	userID := uuid.Must(uuid.NewV4()) // Different user ID

	err := service.DeleteProfile(ctx, userID, user)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, profileErrors.ErrProfileOwnershipRequired))
	mockRepo.AssertNotCalled(t, "Delete")
}

// Test QueryProfiles
func TestQueryProfiles_ValidFilter_ReturnsProfiles(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	
	search := "test"
	filter := &models.ProfileQueryFilter{
		Search: search,
		Limit:  10,
		Page:   1,
	}

	expectedProfiles := []*models.Profile{
		createTestProfile(),
		createTestProfile(),
	}
	expectedCount := int64(2)

	mockRepo.On("Find", ctx, mock.Anything, 10, 0).Return(expectedProfiles, nil)
	mockRepo.On("Count", ctx, mock.Anything).Return(expectedCount, nil)

	result, err := service.QueryProfiles(ctx, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Profiles, 2)
	assert.Equal(t, expectedCount, result.Total)
	mockRepo.AssertExpectations(t)
}

// Test CreateIndex (deprecated - should be no-op)
func TestCreateIndex_NoOp_Success(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	err := service.CreateIndex(ctx, map[string]interface{}{"test": 1})

	assert.NoError(t, err)
}

// Test CreateProfileOnSignup
func TestCreateProfileOnSignup_NewProfile_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	req := createTestCreateProfileRequest()

	mockRepo.On("FindByID", ctx, req.ObjectId).Return(nil, errors.New("profile not found"))
	mockRepo.On("Create", ctx, mock.Anything).Return(nil)

	err := service.CreateProfileOnSignup(ctx, req)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCreateProfileOnSignup_ExistingProfile_Updates(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	req := createTestCreateProfileRequest()
	
	existingProfile := createTestProfile()
	existingProfile.ObjectId = req.ObjectId

	mockRepo.On("FindByID", ctx, req.ObjectId).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.Anything).Return(nil)

	err := service.CreateProfileOnSignup(ctx, req)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test GetProfilesByIds
func TestGetProfilesByIds_ValidIDs_ReturnsProfiles(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	
	userID1 := uuid.Must(uuid.NewV4())
	userID2 := uuid.Must(uuid.NewV4())
	userIDs := []uuid.UUID{userID1, userID2}

	expectedProfiles := []*models.Profile{
		createTestProfile(),
		createTestProfile(),
	}

	mockRepo.On("FindByIDs", ctx, userIDs).Return(expectedProfiles, nil)

	result, err := service.GetProfilesByIds(ctx, userIDs)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

// Test IncrementFields
func TestIncrementFields_ValidFields_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID
	existingProfile.FollowCount = 10
	existingProfile.FollowerCount = 5

	increments := map[string]interface{}{
		"followCount":   3,
		"followerCount": 2,
	}

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(profile *models.Profile) bool {
		return profile.FollowCount == int64(13) &&
			profile.FollowerCount == int64(7)
	})).Return(nil)

	err := service.IncrementFields(ctx, userID, increments)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test UpdateProfileFields
func TestUpdateProfileFields_ValidFields_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID

	updates := map[string]interface{}{
		"fullName": "New Name",
		"avatar":   "https://example.com/new-avatar.jpg",
	}

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(profile *models.Profile) bool {
		return profile.FullName == "New Name" &&
			profile.Avatar == "https://example.com/new-avatar.jpg"
	})).Return(nil)

	err := service.UpdateProfileFields(ctx, userID, updates)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test ValidateProfileOwnership
func TestValidateProfileOwnership_ValidOwnership_Success(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	userID := user.UserID

	err := service.ValidateProfileOwnership(ctx, userID, user)

	assert.NoError(t, err)
}

func TestValidateProfileOwnership_InvalidOwnership_ReturnsError(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	userID := uuid.Must(uuid.NewV4()) // Different user ID

	err := service.ValidateProfileOwnership(ctx, userID, user)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, profileErrors.ErrProfileOwnershipRequired))
}

// Test UpdateAndIncrementFields
func TestUpdateAndIncrementFields_ValidInput_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID
	existingProfile.FollowCount = 10

	updates := map[string]interface{}{
		"fullName": "New Name",
	}
	increments := map[string]interface{}{
		"followCount": 5,
	}

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(profile *models.Profile) bool {
		return profile.FullName == "New Name" &&
			profile.FollowCount == int64(15)
	})).Return(nil)

	err := service.UpdateAndIncrementFields(ctx, userID, updates, increments)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test UpdateFieldsWithOwnership
func TestUpdateFieldsWithOwnership_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	ownerID := uuid.Must(uuid.NewV4())
	userID := ownerID

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID

	updates := map[string]interface{}{
		"fullName": "New Name",
	}

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.Anything).Return(nil)

	err := service.UpdateFieldsWithOwnership(ctx, userID, ownerID, updates)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateFieldsWithOwnership_InvalidOwnership_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	ownerID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4()) // Different user ID

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID

	updates := map[string]interface{}{
		"fullName": "New Name",
	}

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)

	err := service.UpdateFieldsWithOwnership(ctx, userID, ownerID, updates)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, profileErrors.ErrProfileOwnershipRequired))
	mockRepo.AssertNotCalled(t, "Update")
}

// Test DeleteWithOwnership
func TestDeleteWithOwnership_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	ownerID := uuid.Must(uuid.NewV4())
	userID := ownerID

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Delete", ctx, userID).Return(nil)

	err := service.DeleteWithOwnership(ctx, userID, ownerID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test IncrementFieldsWithOwnership
func TestIncrementFieldsWithOwnership_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	ownerID := uuid.Must(uuid.NewV4())
	userID := ownerID

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID
	existingProfile.FollowCount = 10

	increments := map[string]interface{}{
		"followCount": 5,
	}

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(profile *models.Profile) bool {
		return profile.FollowCount == int64(15)
	})).Return(nil)

	err := service.IncrementFieldsWithOwnership(ctx, userID, ownerID, increments)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test GetProfilesBySearch
func TestGetProfilesBySearch_ValidQuery_ReturnsProfiles(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	
	query := "test"
	filter := &models.ProfileQueryFilter{
		Search: query,
		Limit:  10,
		Page:   1,
	}

	expectedProfiles := []*models.Profile{
		createTestProfile(),
	}
	expectedCount := int64(1)

	mockRepo.On("Find", ctx, mock.Anything, 10, 0).Return(expectedProfiles, nil)
	mockRepo.On("Count", ctx, mock.Anything).Return(expectedCount, nil)

	result, err := service.GetProfilesBySearch(ctx, query, filter)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Profiles, 1)
	assert.Equal(t, expectedCount, result.Total)
	mockRepo.AssertExpectations(t)
}

// Test SetField
func TestSetField_ValidField_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(profile *models.Profile) bool {
		return profile.FullName == "New Name"
	})).Return(nil)

	err := service.SetField(ctx, userID, "fullName", "New Name")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Test UpdateByOwner
func TestUpdateByOwner_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	ownerID := uuid.Must(uuid.NewV4())
	userID := ownerID

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID

	fields := map[string]interface{}{
		"fullName": "New Name",
	}

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)
	mockRepo.On("Update", ctx, mock.Anything).Return(nil)

	err := service.UpdateByOwner(ctx, userID, ownerID, fields)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateByOwner_InvalidOwnership_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	ownerID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4()) // Different user ID

	existingProfile := createTestProfile()
	existingProfile.ObjectId = userID

	fields := map[string]interface{}{
		"fullName": "New Name",
	}

	mockRepo.On("FindByID", ctx, userID).Return(existingProfile, nil)

	err := service.UpdateByOwner(ctx, userID, ownerID, fields)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, profileErrors.ErrProfileOwnershipRequired))
	mockRepo.AssertNotCalled(t, "Update")
}

