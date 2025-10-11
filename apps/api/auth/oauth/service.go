package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/models"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"golang.org/x/oauth2"
)

type Service struct {
	base   *platform.BaseService
	config *ServiceConfig
}

type ServiceConfig struct {
	OAuthConfig *OAuthConfig
	JWTConfig   platformconfig.JWTConfig
	HMACConfig  platformconfig.HMACConfig
	AppConfig   platformconfig.AppConfig
}

type OAuthUserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Provider  string
}

func NewService(base *platform.BaseService, config *ServiceConfig) *Service {
	return &Service{
		base:   base,
		config: config,
	}
}

// ExchangeCodeForToken exchanges authorization code for access token
func (s *Service) ExchangeCodeForToken(ctx context.Context, provider, code, codeVerifier string) (*oauth2.Token, error) {
	var config *oauth2.Config

	switch provider {
	case "github":
		config = s.config.OAuthConfig.GitHub
	case "google":
		config = s.config.OAuthConfig.Google
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// Exchange code for token with PKCE
	token, err := config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// GetUserInfo fetches user information from OAuth provider
func (s *Service) GetUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*OAuthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	var userInfoURL string
	switch provider {
	case "github":
		userInfoURL = "https://api.github.com/user"
	case "google":
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info: %w", err)
	}

	var userInfo OAuthUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	userInfo.Provider = provider

	return &userInfo, nil
}

// FindOrCreateUser finds existing user by email or creates new user
func (s *Service) FindOrCreateUser(ctx context.Context, userInfo *OAuthUserInfo) (*models.UserAuth, *models.UserProfile, error) {
	// 1. Check if user exists by email
	userRes := <-s.base.Repository.FindOne(ctx, "userAuth", struct {
		Username string `json:"username" bson:"username"`
	}{Username: userInfo.Email})
	if userRes.Error() == nil {
		// User exists - return existing user
		var userAuth models.UserAuth
		if err := userRes.Decode(&userAuth); err != nil {
			return nil, nil, fmt.Errorf("failed to decode user auth: %w", err)
		}

		// Get user profile
		profileRes := <-s.base.Repository.FindOne(ctx, "userProfile", struct {
			ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
		}{ObjectId: userAuth.ObjectId})
		if profileRes.Error() != nil {
			return nil, nil, fmt.Errorf("failed to get user profile: %w", profileRes.Error())
		}

		var userProfile models.UserProfile
		if err := profileRes.Decode(&userProfile); err != nil {
			return nil, nil, fmt.Errorf("failed to decode user profile: %w", err)
		}

		return &userAuth, &userProfile, nil
	}

	// 2. Create new user account
	return s.createOAuthUser(ctx, userInfo)
}

// createOAuthUser creates new user account from OAuth information
func (s *Service) createOAuthUser(ctx context.Context, userInfo *OAuthUserInfo) (*models.UserAuth, *models.UserProfile, error) {
	userId := uuid.Must(uuid.NewV4())
	now := time.Now().Unix()

	// Create user auth record
	userAuth := models.UserAuth{
		ObjectId:      userId,
		Username:      userInfo.Email,
		Password:      []byte{}, // No password for OAuth-only users
		Role:          "user",
		EmailVerified: true, // OAuth email is pre-verified
		PhoneVerified: false,
		CreatedDate:   now,
		LastUpdated:   now,
	}

	// Create user profile
	userProfile := models.UserProfile{
		ObjectId:    userId,
		FullName:    userInfo.Name,
		SocialName:  generateSocialName(userInfo.Name),
		Email:       userInfo.Email,
		Avatar:      userInfo.AvatarURL,
		Banner:      "https://picsum.photos/id/1/900/300/?blur",
		TagLine:     "",
		CreatedDate: now,
		LastUpdated: now,
	}

	// Save both records
	if err := (<-s.base.Repository.Save(ctx, "userAuth", &userAuth)).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create user auth: %w", err)
	}

	if err := (<-s.base.Repository.Save(ctx, "userProfile", &userProfile)).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create user profile: %w", err)
	}

	return &userAuth, &userProfile, nil
}

// Helper function to generate social name from full name
func generateSocialName(fullName string) string {
	// Implementation to create URL-friendly social name
	// Remove spaces, convert to lowercase, handle special characters
	return strings.ToLower(strings.ReplaceAll(fullName, " ", ""))
}
