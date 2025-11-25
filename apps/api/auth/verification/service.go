package verification

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	authErrors "github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	"github.com/qolzam/telar/apps/api/internal/auth/tokens"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
	profileServices "github.com/qolzam/telar/apps/api/profile/services"
)

const (
    userVerificationCollectionName = "userVerification"
	userAuthCollectionName         = "userAuth"
	userProfileCollectionName      = "userProfile"
)

// verificationQueryBuilder is a helper struct for building verification-specific queries.
// It knows the schema of the verification tables and provides fluent methods for query construction.
type verificationQueryBuilder struct {
	query *dbi.Query
}

// newVerificationQueryBuilder creates a new verificationQueryBuilder instance.
func newVerificationQueryBuilder() *verificationQueryBuilder {
	return &verificationQueryBuilder{
		query: &dbi.Query{
			Conditions: []dbi.Field{},
			OrGroups:   [][]dbi.Field{},
		},
	}
}

// WhereObjectID adds a filter for the object_id (indexed column).
func (b *verificationQueryBuilder) WhereObjectID(objectID uuid.UUID) *verificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "object_id", // Indexed column - direct access
		Value:    objectID,
		Operator: "=",
		IsJSONB:  false,
	})
	return b
}

// WhereCode adds a filter for the code (JSONB field).
func (b *verificationQueryBuilder) WhereCode(code string) *verificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'code'", // JSONB field
		Value:    code,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereUserId adds a filter for the user_id (indexed column).
func (b *verificationQueryBuilder) WhereUserId(userID uuid.UUID) *verificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "owner_user_id", // Indexed column - owner_user_id maps to userId
		Value:    userID,
		Operator: "=",
		IsJSONB:  false,
	})
	return b
}

// Build returns the constructed Query object.
func (b *verificationQueryBuilder) Build() *dbi.Query {
	return b.query
}

// buildQueryFromFilter converts a DatabaseFilter to a Query object.
func buildQueryFromFilter(filter *models.DatabaseFilter) *dbi.Query {
	qb := newVerificationQueryBuilder()
	if filter.ObjectId != nil {
		qb.WhereObjectID(*filter.ObjectId)
	}
	if filter.UserId != nil {
		qb.WhereUserId(*filter.UserId)
	}
	return qb.Build()
}

type Service struct {
	base              *platform.BaseService // Deprecated: use repositories instead
	verifRepo         VerificationRepository
	authRepo          AuthRepository
	config            *ServiceConfig
	hmacUtil          *HMACUtil
	securityValidator *SecurityValidator
	privateKey        string
	orgName           string
	webDomain         string
	profileCreator    profileServices.ProfileServiceClient
	signupOrchestrator signupOrchestrator // Orchestrator for atomic user+profile creation
}

// VerificationRepository interface to avoid circular dependency
type VerificationRepository interface {
	FindByID(ctx context.Context, verificationID uuid.UUID) (*models.UserVerification, error)
	MarkUsed(ctx context.Context, verificationID uuid.UUID) error
	UpdateVerificationCode(ctx context.Context, verificationID uuid.UUID, newCode string, newExpiresAt int64) error
	SaveVerification(ctx context.Context, verification *models.UserVerification) error
	DeleteExpired(ctx context.Context, beforeTime int64) error
}

// AuthRepository interface to avoid circular dependency
type AuthRepository interface {
	CreateUser(ctx context.Context, userAuth *models.UserAuth) error
}

// signupOrchestrator interface to avoid circular dependency
type signupOrchestrator interface {
	CompleteSignup(ctx context.Context, verification *models.UserVerification) error
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

// NewServiceWithRepositories creates a service using repositories instead of base service
func NewServiceWithRepositories(verifRepo VerificationRepository, authRepo AuthRepository, config *ServiceConfig) *Service {
	service := &Service{
		verifRepo: verifRepo,
		authRepo:  authRepo,
		config:    config,
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
func NewServiceWithKeys(base *platform.BaseService, config *ServiceConfig, privateKey, orgName, webDomain string, profileCreator profileServices.ProfileServiceClient) *Service {
	service := NewService(base, config)
	service.privateKey = privateKey
	service.orgName = orgName
	service.webDomain = webDomain
	service.profileCreator = profileCreator
	return service
}

// NewServiceWithRepositoriesAndKeys creates a service with repositories and JWT token generation capability
func NewServiceWithRepositoriesAndKeys(verifRepo VerificationRepository, authRepo AuthRepository, config *ServiceConfig, privateKey, orgName, webDomain string, profileCreator profileServices.ProfileServiceClient) *Service {
	service := NewServiceWithRepositories(verifRepo, authRepo, config)
	service.privateKey = privateKey
	service.orgName = orgName
	service.webDomain = webDomain
	service.profileCreator = profileCreator
	return service
}

// SetSignupOrchestrator sets the signup orchestrator for atomic user+profile creation
func (s *Service) SetSignupOrchestrator(orchestrator signupOrchestrator) {
	s.signupOrchestrator = orchestrator
}

func (s *Service) verifyUserByCode(ctx context.Context, userId uuid.UUID, verifyId uuid.UUID, remoteIp string, code string, target string) (bool, error) {
	// Use repository if available, otherwise fall back to base
	var uv *models.UserVerification
	if s.verifRepo != nil {
		var err error
		uv, err = s.verifRepo.FindByID(ctx, verifyId)
		if err != nil {
			return false, fmt.Errorf("verification record not found: %w", err)
		}
	} else {
		// Fallback to base service
		if s.base == nil {
			return false, fmt.Errorf("verification service not properly initialized")
		}
		query := newVerificationQueryBuilder().WhereObjectID(verifyId).Build()
		res := <-s.base.Repository.FindOne(ctx, userVerificationCollectionName, query)
		var uvStruct struct {
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
		if err := res.Decode(&uvStruct); err != nil {
			if errors.Is(err, dbi.ErrNoDocuments) {
				return false, fmt.Errorf("verification record not found")
			}
			return false, fmt.Errorf("decode userVerification: %w", err)
		}
		// Convert to UserVerification model
		uv = &models.UserVerification{
			ObjectId:        uvStruct.ObjectId,
			Code:            uvStruct.Code,
			Target:          uvStruct.Target,
			Counter:         uvStruct.Counter,
			CreatedDate:     uvStruct.CreatedDate,
			RemoteIpAddress: uvStruct.RemoteIpAddress,
			IsVerified:      uvStruct.IsVerified,
			LastUpdated:     uvStruct.LastUpdated,
			HashedPassword:  uvStruct.HashedPassword,
			ExpiresAt:       uvStruct.ExpiresAt,
			Used:            uvStruct.Used,
		}
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
		// Use repository if available, otherwise fall back to base
		if s.verifRepo != nil {
			// Update counter and last_updated using UpdateVerificationCode (reuse same code)
			err := s.verifRepo.UpdateVerificationCode(ctx, verifyId, uv.Code, uv.ExpiresAt)
			if err != nil {
				return false, fmt.Errorf("createCodeVerification/updateVerificationCode: %w", err)
			}
		} else if s.base != nil {
			update := map[string]interface{}{"last_updated": uv.LastUpdated, "counter": newCounter}
			updateQuery := newVerificationQueryBuilder().WhereObjectID(verifyId).Build()
			err := (<-s.base.Repository.UpdateFields(ctx, userVerificationCollectionName, updateQuery, update)).Error
			if err != nil {
				return false, fmt.Errorf("createCodeVerification/updateVerificationCode")
			}
		} else {
			return false, fmt.Errorf("verification service not properly initialized")
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
	// Mark verification as used and verified
	if s.verifRepo != nil {
		err := s.verifRepo.MarkUsed(ctx, verifyId)
		if err != nil {
			return false, fmt.Errorf("createCodeVerification/updateVerificationCode: %w", err)
		}
	} else if s.base != nil {
		update := map[string]interface{}{"last_updated": nowMillis, "counter": newCounter, "isVerified": true, "used": true}
		updateQuery := newVerificationQueryBuilder().WhereObjectID(verifyId).Build()
		err := (<-s.base.Repository.UpdateFields(ctx, userVerificationCollectionName, updateQuery, update)).Error
		if err != nil {
			return false, fmt.Errorf("createCodeVerification/updateVerificationCode")
		}
	} else {
		return false, fmt.Errorf("verification service not properly initialized")
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
	// Use repository if available, otherwise fall back to base
	if s.authRepo != nil {
		userAuth := &models.UserAuth{
			ObjectId:      ua.ObjectId,
			Username:      ua.Username,
			Password:      ua.Password,
			EmailVerified: ua.EmailVerified,
			PhoneVerified: ua.PhoneVerified,
			Role:          ua.Role,
			CreatedDate:   ua.CreatedDate,
			LastUpdated:   ua.LastUpdated,
		}
		return s.authRepo.CreateUser(ctx, userAuth)
	}
	// Fallback to base service
	if s.base == nil {
		return fmt.Errorf("verification service not properly initialized")
	}
	saveResult := <-s.base.Repository.Save(ctx, userAuthCollectionName, ua.ObjectId, ua.ObjectId, ua.CreatedDate, ua.LastUpdated, ua)
	return saveResult.Error
}

func (s *Service) createUserProfile(ctx context.Context, up interface{}) error {
	// For user profile, we need to extract objectId, userId, createdDate, lastUpdated from the struct
	// Since we're passing interface{}, we'll need to handle this differently
	// For now, generate UUIDs and timestamps
	objectID := uuid.Must(uuid.NewV4())
	now := time.Now().Unix()
	return (<-s.base.Repository.Save(ctx, userProfileCollectionName, objectID, objectID, now, now, up)).Error
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
	// Use repository if available, otherwise fall back to base
	if s.verifRepo != nil {
		return s.verifRepo.SaveVerification(ctx, userVerification)
	}
	// Fallback to base service
	if s.base == nil {
		return authErrors.WrapDatabaseError(fmt.Errorf("verification service not properly initialized"))
	}
	result := <-s.base.Repository.Save(ctx, userVerificationCollectionName, userVerification.ObjectId, userVerification.UserId, userVerification.CreatedDate, userVerification.LastUpdated, userVerification)
	if result.Error != nil {
		return authErrors.WrapDatabaseError(fmt.Errorf("failed to save user verification: %w", result.Error))
	}
	return nil
}

// FindUserVerification finds a user verification record
func (s *Service) FindUserVerification(ctx context.Context, filter *models.DatabaseFilter) (*models.UserVerification, error) {
	// Use repository if available, otherwise fall back to base
	if s.verifRepo != nil && filter.ObjectId != nil {
		return s.verifRepo.FindByID(ctx, *filter.ObjectId)
	}
	
	// Fallback to base service for backward compatibility
	if s.base == nil {
		return nil, authErrors.WrapUserNotFoundError(fmt.Errorf("user verification not found"))
	}
	
	query := buildQueryFromFilter(filter)
	res := <-s.base.Repository.FindOne(ctx, userVerificationCollectionName, query)
	
	var verification models.UserVerification
	if err := res.Decode(&verification); err != nil {
		if errors.Is(err, dbi.ErrNoDocuments) {
			return nil, authErrors.WrapUserNotFoundError(fmt.Errorf("user verification not found"))
		}
		return nil, authErrors.WrapDatabaseError(fmt.Errorf("failed to decode verification: %w", err))
	}

	return &verification, nil
}

// UpdateUserVerification updates a user verification record
func (s *Service) UpdateUserVerification(ctx context.Context, filter *models.DatabaseFilter, data *models.DatabaseUpdate) error {
	query := buildQueryFromFilter(filter)
	// Convert DatabaseUpdate to map[string]interface{}
	updates := data.Set
	result := <-s.base.Repository.UpdateFields(ctx, userVerificationCollectionName, query, updates)
	if result.Error != nil {
		return authErrors.WrapDatabaseError(fmt.Errorf("failed to update user verification: %w", result.Error))
	}
	return nil
}

// DeleteUserVerification deletes a user verification record
func (s *Service) DeleteUserVerification(ctx context.Context, filter *models.DatabaseFilter) error {
	query := buildQueryFromFilter(filter)
	result := <-s.base.Repository.Delete(ctx, userVerificationCollectionName, query)
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
		// Log the full error for debugging (including schema/connection issues)
		log.Printf("[VerifySignup] Failed to find verification ID %s: %v", params.VerificationId.String(), err)
		// Preserve original error message for debugging
		return nil, authErrors.NewUserNotFoundError(fmt.Sprintf("verification record not found: %v", err))
	}

	// 2. Validate the verification code
	if verification.Code != params.Code {
		return nil, authErrors.NewValidationError("invalid verification code")
	}

	// 3. Check if verification has expired
	if time.Now().Unix() > verification.ExpiresAt {
		return nil, authErrors.NewValidationError("verification code has expired")
	}

	// 4. Check if verification already used
	if verification.Used {
		return nil, authErrors.NewValidationError("verification code already used")
	}

	// 5. Verify the user by code (handles verification state update)
	success, err := s.verifyUserByCode(ctx, verification.UserId, verification.ObjectId, params.RemoteIpAddress, params.Code, verification.Target)
	if err != nil {
		return nil, fmt.Errorf("user verification failed: %w", err)
	}

	if !success {
		return nil, fmt.Errorf("verification unsuccessful")
	}
	
	// 6. Create user account and profile atomically using orchestrator
	if s.signupOrchestrator == nil {
		return nil, authErrors.NewSystemError("signup orchestrator not available")
	}

	if err := s.signupOrchestrator.CompleteSignup(ctx, verification); err != nil {
		return nil, fmt.Errorf("failed to complete signup: %w", err)
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
				"jti":         uuid.Must(uuid.NewV4()).String(),
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
	// Use profileCreator if available (preferred method)
	if s.profileCreator != nil {
		profile, err := s.profileCreator.GetProfile(ctx, userId)
		if err != nil {
			return nil, fmt.Errorf("user profile not found: %w", err)
		}
		return &userProfileData{
			ObjectId:    profile.ObjectId,
			FullName:    profile.FullName,
			Email:       profile.Email,
			SocialName:  profile.SocialName,
			Avatar:      profile.Avatar,
			Banner:      profile.Banner,
			TagLine:     profile.Tagline,
			CreatedDate: profile.CreatedDate,
		}, nil
	}
	
	// Fallback to base service for backward compatibility
	if s.base == nil {
		return nil, fmt.Errorf("verification service not properly initialized")
	}
	
	query := &dbi.Query{
		Conditions: []dbi.Field{
			{
				Name:     "object_id",
				Value:    userId,
				Operator: "=",
				IsJSONB:  false,
			},
		},
	}
	res := <-s.base.Repository.FindOne(ctx, userProfileCollectionName, query)

	var profile userProfileData
	if err := res.Decode(&profile); err != nil {
		if errors.Is(err, dbi.ErrNoDocuments) {
			return nil, fmt.Errorf("user profile not found")
		}
		return nil, fmt.Errorf("failed to decode user profile: %w", err)
	}

	return &profile, nil
}

type userProfileData struct {
	ObjectId    uuid.UUID `json:"objectId" bson:"objectId"`
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
