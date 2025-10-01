package verification

import (
    "context"
    "fmt"
	"strings"
	"time"

    "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	"github.com/qolzam/telar/apps/api/internal/auth/tokens"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
)

const (
    userVerificationCollectionName = "userVerification"
	userAuthCollectionName         = "userAuth"
	userProfileCollectionName      = "userProfile"
)

type Service struct {
	base              *platform.BaseService
	config            *ServiceConfig
	hmacUtil          *HMACUtil
	securityValidator *SecurityValidator
	privateKey        string
	orgName           string
	webDomain         string
}

func NewService(base *platform.BaseService, config *ServiceConfig) *Service {
	service := &Service{
		base:   base,
		config: config,
	}

	// Initialize HMAC utility
	if config != nil && config.HMACConfig.Secret != "" {
		service.hmacUtil = NewHMACUtil(config.HMACConfig.Secret)
	}

	// Initialize security validator (after service is created)
	service.securityValidator = NewSecurityValidator(service)

	return service
}

// NewServiceWithKeys creates a service with JWT token generation capability
func NewServiceWithKeys(base *platform.BaseService, config *ServiceConfig, privateKey, orgName, webDomain string) *Service {
	service := NewService(base, config)
	service.privateKey = privateKey
	service.orgName = orgName
	service.webDomain = webDomain
	return service
}

func (s *Service) verifyUserByCode(ctx context.Context, userId uuid.UUID, verifyId uuid.UUID, remoteIp string, code string, target string) (bool, error) {
    // Load verification
	res := <-s.base.Repository.FindOne(ctx, userVerificationCollectionName, struct {
        ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
	}{ObjectId: verifyId})
	if res.Error() != nil {
		return false, res.Error()
	}
	var uv struct {
		ObjectId        uuid.UUID `json:"objectId" bson:"objectId"`
		Code            string    `json:"code" bson:"code"`
		Target          string    `json:"target" bson:"target"`
		Counter         int64     `json:"counter" bson:"counter"`
		CreatedDate     int64     `json:"created_date" bson:"created_date"`
		RemoteIpAddress string    `json:"remoteIpAddress" bson:"remoteIpAddress"`
		IsVerified      bool      `json:"isVerified" bson:"isVerified"`
		LastUpdated     int64     `json:"last_updated" bson:"last_updated"`
		HashedPassword  []byte    `json:"hashedPassword" bson:"hashedPassword"`
		ExpiresAt       int64     `json:"expiresAt" bson:"expiresAt"`
		Used            bool      `json:"used" bson:"used"`
	}
	if err := res.Decode(&uv); err != nil {
		return false, fmt.Errorf("decode userVerification")
	}
	if uv.RemoteIpAddress != remoteIp {
		return false, fmt.Errorf("verifyUserByCode/differentRemoteAddress")
	}
    newCounter := uv.Counter + 1
	if uv.IsVerified {
		return false, fmt.Errorf("verifyUserByCode/alreadyVerified")
	}
	if uv.Used {
		return false, fmt.Errorf("verifyUserByCode/alreadyUsed")
	}
    if uv.Code != code {
        uv.LastUpdated = utils.UTCNowUnix()
		update := map[string]interface{}{"$set": map[string]interface{}{"last_updated": uv.LastUpdated, "counter": newCounter}}
		err := (<-s.base.Repository.Update(ctx, userVerificationCollectionName, struct {
			ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
		}{ObjectId: verifyId}, update, &interfaces.UpdateOptions{})).Error
		if err != nil {
			return false, fmt.Errorf("createCodeVerification/updateVerificationCode")
		}
        return false, fmt.Errorf("createCodeVerification/wrongPinCod")
    }
	// Expiry: prefer ExpiresAt if present; fallback to CreatedDate TTL
	nowMillis := utils.UTCNowUnix()
	now := nowMillis / 1000 // Convert milliseconds to seconds for comparison with ExpiresAt
	if uv.ExpiresAt > 0 {
		if now > uv.ExpiresAt {
			return false, fmt.Errorf("verifyUserByCode/codeExpired")
		}
	} else {
		if utils.IsTimeExpired(uv.CreatedDate, 3600) {
			return false, fmt.Errorf("verifyUserByCode/codeExpired")
		}
	}
	update := map[string]interface{}{"$set": map[string]interface{}{"last_updated": nowMillis, "counter": newCounter, "isVerified": true, "used": true}}
	err := (<-s.base.Repository.Update(ctx, userVerificationCollectionName, struct {
		ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
	}{ObjectId: verifyId}, update, &interfaces.UpdateOptions{})).Error
	if err != nil {
		return false, fmt.Errorf("createCodeVerification/updateVerificationCode")
	}
    return true, nil
}

type userAuthDoc struct {
	ObjectId      uuid.UUID `json:"objectId" bson:"objectId" db:"objectId"`
	Username      string    `json:"username" bson:"username" db:"username"`
	Password      []byte    `json:"password" bson:"password" db:"password"`
	EmailVerified bool      `json:"emailVerified" bson:"emailVerified" db:"emailVerified"`
	PhoneVerified bool      `json:"phoneVerified" bson:"phoneVerified" db:"phoneVerified"`
	Role          string    `json:"role" bson:"role" db:"role"`
	CreatedDate   int64     `json:"createdDate" bson:"createdDate" db:"createdDate"`
	LastUpdated   int64     `json:"lastUpdated" bson:"lastUpdated" db:"lastUpdated"`
}

func (s *Service) createUserAuth(ctx context.Context, ua userAuthDoc) error {
	// Use the struct directly instead of map to ensure proper field mapping
	// Save user authentication record
	saveResult := <-s.base.Repository.Save(ctx, userAuthCollectionName, ua)
	return saveResult.Error
}

func (s *Service) createUserProfile(ctx context.Context, up interface{}) error {
    return (<-s.base.Repository.Save(ctx, userProfileCollectionName, up)).Error
}

// Extract full name from email target
func extractFullNameFromTarget(target string) string {
	// For email targets, extract local part and capitalize
	if strings.Contains(target, "@") {
		localPart := strings.Split(target, "@")[0]
		// Remove dots and underscores, capitalize
		name := strings.ReplaceAll(localPart, ".", " ")
		name = strings.ReplaceAll(name, "_", " ")
		// Simple capitalization
		words := strings.Fields(name)
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
			}
		}
		return strings.Join(words, " ")
	}
	// Fallback for non-email targets
	// Use simple capitalization instead of deprecated strings.Title
	if len(target) == 0 {
		return target
	}
	return strings.ToUpper(string(target[0])) + strings.ToLower(target[1:])
}

// VerifyEmailCode verifies an email verification code using verificationId
func (s *Service) VerifyEmailCode(ctx context.Context, verificationId string, code string) error {
	// This would validate the verification code against the stored verification record
	// For now, return a placeholder implementation
	return fmt.Errorf("email verification not yet implemented")
}

// VerifyPhoneCode verifies a phone verification code using verificationId
func (s *Service) VerifyPhoneCode(ctx context.Context, verificationId string, code string) error {
	// This would validate the verification code against the stored verification record
	// For now, return a placeholder implementation
	return fmt.Errorf("phone verification not yet implemented")
}

// SaveUserVerification saves a user verification record
func (s *Service) SaveUserVerification(ctx context.Context, userVerification *models.UserVerification) error {
	result := <-s.base.Repository.Save(ctx, userVerificationCollectionName, userVerification)
	if result.Error != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to save user verification: %w", result.Error))
	}
	return nil
}

// FindUserVerification finds a user verification record
func (s *Service) FindUserVerification(ctx context.Context, filter *models.DatabaseFilter) (*models.UserVerification, error) {
	res := <-s.base.Repository.FindOne(ctx, userVerificationCollectionName, filter)
	if res.Error() != nil {
		return nil, errors.WrapUserNotFoundError(fmt.Errorf("user verification not found"))
	}

	var verification models.UserVerification
	if err := res.Decode(&verification); err != nil {
		return nil, errors.WrapDatabaseError(fmt.Errorf("failed to decode verification: %w", err))
	}

	return &verification, nil
}

// UpdateUserVerification updates a user verification record
func (s *Service) UpdateUserVerification(ctx context.Context, filter *models.DatabaseFilter, data *models.DatabaseUpdate) error {
	result := <-s.base.Repository.Update(ctx, userVerificationCollectionName, filter, data, nil)
	if result.Error != nil {
		return errors.WrapDatabaseError(fmt.Errorf("failed to update user verification: %w", result.Error))
	}
	return nil
}

// DeleteUserVerification deletes a user verification record
func (s *Service) DeleteUserVerification(ctx context.Context, filter *models.DatabaseFilter) error {
	result := <-s.base.Repository.Delete(ctx, userVerificationCollectionName, filter)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user verification: %w", result.Error)
	}
	return nil
}

// VerifySignup - Secure verification method for signup completion
// This method handles both public user verification and service-to-service verification
func (s *Service) VerifySignup(ctx context.Context, params VerifySignupParams) (*VerifySignupResult, error) {
	// 1. Find the verification record
	verification, err := s.FindUserVerification(ctx, &models.DatabaseFilter{
		ObjectId: &params.VerificationId,
	})
	if err != nil {
		return nil, fmt.Errorf("verification record not found: %w", err)
	}

	// 2. Validate the verification code
	if verification.Code != params.Code {
		return nil, fmt.Errorf("invalid verification code")
	}

	// 3. Check if verification has expired
	if time.Now().Unix() > verification.ExpiresAt {
		return nil, fmt.Errorf("verification code has expired")
	}

	// 4. Check if verification already used
	if verification.Used {
		return nil, fmt.Errorf("verification code already used")
	}

	// 5. Verify the user by code (handles verification state update)
	success, err := s.verifyUserByCode(ctx, verification.UserId, verification.ObjectId, params.RemoteIpAddress, params.Code, verification.Target)
	if err != nil {
		return nil, fmt.Errorf("user verification failed: %w", err)
	}

	if !success {
		return nil, fmt.Errorf("verification unsuccessful")
	}
	// 6. Create user account and profile
	// Create user authentication record
	userAuth := userAuthDoc{
		ObjectId:      verification.UserId,
		Username:      verification.Target,
		Password:      verification.HashedPassword,
		EmailVerified: verification.TargetType == "email",
		PhoneVerified: verification.TargetType == "phone",
		Role:          "user",
		CreatedDate:   time.Now().Unix(),
		LastUpdated:   time.Now().Unix(),
	}

	if err := s.createUserAuth(ctx, userAuth); err != nil {
		return nil, fmt.Errorf("failed to create user auth: %w", err)
	}

	// Create user profile - Use stored full name from signup form
	fullName := verification.FullName
	if fullName == "" {
		// Fallback for legacy verification records without stored full name
		fullName = extractFullNameFromTarget(verification.Target)
	}

	newUserProfile := userProfileData{
		ObjectId:    verification.UserId,
		FullName:    fullName,
		Email:       verification.Target,
		SocialName:  generateSocialName(fullName, verification.UserId.String()),
		Avatar:      "",
		Banner:      "",
		TagLine:     "",
		CreatedDate: time.Now().Unix(),
	}

	if err := s.createUserProfile(ctx, newUserProfile); err != nil {
		return nil, fmt.Errorf("failed to create user profile: %w", err)
	}

	// 7. Generate access token for successful verification
	// Note: verifyUserByCode already marked verification as used
	accessToken := ""
	userProfile := map[string]interface{}{"userId": verification.UserId.String()}

	// If we have JWT token generation capability, create an access token
	if s.privateKey != "" && s.webDomain != "" {
		// Fetch user profile for token generation
		userProfileData, profileErr := s.findUserProfile(ctx, verification.UserId)
		if profileErr == nil && userProfileData != nil {
			// Create token claim
			tokenClaim := map[string]interface{}{
				"displayName": userProfileData.FullName,
				"email":       userProfileData.Email,
				types.HeaderUID:         verification.UserId.String(),
				"role":        "user", // Default role for verified users
				"createdDate": userProfileData.CreatedDate,
			}

			// Create profile info for token
			profileInfo := map[string]string{
				"id":       verification.UserId.String(),
				"login":    userProfileData.Email,
				"name":     userProfileData.FullName,
				"audience": s.webDomain,
			}

			// Generate JWT token
			if token, tokenErr := tokens.CreateTokenWithKey("telar", profileInfo, s.orgName, tokenClaim, s.privateKey); tokenErr == nil {
				accessToken = token
			}

			// Update user profile for response
			userProfile = map[string]interface{}{
				"objectId":    userProfileData.ObjectId.String(),
				"fullName":    userProfileData.FullName,
				"email":       userProfileData.Email,
				"socialName":  userProfileData.SocialName,
				"avatar":      userProfileData.Avatar,
				"banner":      userProfileData.Banner,
				"tagLine":     userProfileData.TagLine,
				"createdDate": userProfileData.CreatedDate,
			}
		}
	}

	return &VerifySignupResult{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		User:        userProfile,
		Success:     true,
	}, nil
}

// incrementVerificationAttempts increments the verification attempt counter
func (s *Service) incrementVerificationAttempts(ctx context.Context, verificationId uuid.UUID) error {
	return s.UpdateUserVerification(ctx, &models.DatabaseFilter{
		ObjectId: &verificationId,
	}, &models.DatabaseUpdate{
		Set: map[string]interface{}{
			"counter":     map[string]interface{}{"$inc": map[string]interface{}{"counter": 1}},
			"lastUpdated": time.Now().Unix(),
		},
	})
}

// validateHMACSignature validates HMAC signature for verification requests
func (s *Service) validateHMACSignature(ctx context.Context, params *VerifySignupParams) error {
	if s.hmacUtil == nil {
		return fmt.Errorf("HMAC utility not initialized")
	}

	// Convert UserId string to UUID
	userId, err := uuid.FromString(params.UserId)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	// Build HMAC data structure
	hmacData := VerificationHMACData{
		VerificationId:  params.VerificationId,
		Code:            params.Code,
		RemoteIpAddress: params.RemoteIpAddress,
		Timestamp:       params.Timestamp,
		UserId:          userId,
	}

	return s.hmacUtil.ValidateVerificationHMAC(hmacData, params.HMACSignature)
}

// findUserProfile finds a user profile by user ID
func (s *Service) findUserProfile(ctx context.Context, userId uuid.UUID) (*userProfileData, error) {
	res := <-s.base.Repository.FindOne(ctx, userProfileCollectionName, struct {
		ObjectId uuid.UUID `json:"objectId" bson:"_id"`
	}{ObjectId: userId})

	if res.Error() != nil {
		return nil, fmt.Errorf("user profile not found: %w", res.Error())
	}

	var profile userProfileData
	if err := res.Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode user profile: %w", err)
	}

	return &profile, nil
}

type userProfileData struct {
	ObjectId    uuid.UUID `json:"objectId" bson:"_id"`
	FullName    string    `json:"fullName" bson:"fullName"`
	Email       string    `json:"email" bson:"email"`
	SocialName  string    `json:"socialName" bson:"socialName"`
	Avatar      string    `json:"avatar" bson:"avatar"`
	Banner      string    `json:"banner" bson:"banner"`
	TagLine     string    `json:"tagLine" bson:"tagLine"`
	CreatedDate int64     `json:"createdDate" bson:"createdDate"`
}

// generateSocialName generates a social name from full name and user ID
func generateSocialName(fullName string, userId string) string {
	// Simple implementation: use first name + first 8 chars of user ID
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "user_" + userId[:8]
	}

	firstName := parts[0]
	if len(userId) >= 8 {
		return fmt.Sprintf("%s_%s", strings.ToLower(firstName), userId[:8])
	}

	return fmt.Sprintf("%s_%s", strings.ToLower(firstName), userId)
}
