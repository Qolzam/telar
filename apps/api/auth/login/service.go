package login

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	"github.com/qolzam/telar/apps/api/internal/auth/tokens"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
)

type Service struct {
	base   *platform.BaseService
	config *ServiceConfig
}

type ServiceConfig struct {
	JWTConfig  platformconfig.JWTConfig
	HMACConfig platformconfig.HMACConfig
}

func NewService(base *platform.BaseService, config *ServiceConfig) *Service {
	return &Service{
		base:   base,
		config: config,
	}
}

type userAuth struct {
	ObjectId      uuid.UUID `json:"objectId" bson:"objectId" db:"objectId"`
	Username      string    `json:"username" bson:"username" db:"username"`
	Password      []byte    `json:"password" bson:"password" db:"password"`
	EmailVerified bool      `json:"emailVerified" bson:"emailVerified" db:"emailVerified"`
	PhoneVerified bool      `json:"phoneVerified" bson:"phoneVerified" db:"phoneVerified"`
	Role          string    `json:"role" bson:"role" db:"role"`
}

func (s *Service) FindUserByUsername(ctx context.Context, username string) (*userAuth, error) {
	res := <-s.base.Repository.FindOne(ctx, "userAuth", struct {
		Username string `json:"username" bson:"username"`
	}{Username: username})
	if res.Error() != nil {
		return nil, res.Error()
	}
	var ua userAuth
	if err := res.Decode(&ua); err != nil {
		return nil, err
	}
	return &ua, nil
}

type userProfile struct {
	ObjectId    uuid.UUID `json:"objectId" bson:"_id" db:"objectId"`
	FullName    string    `json:"fullName" bson:"fullName" db:"fullName"`
	SocialName  string    `json:"socialName" bson:"socialName" db:"socialName"`
	Email       string    `json:"email" bson:"email" db:"email"`
	Avatar      string    `json:"avatar" bson:"avatar" db:"avatar"`
	Banner      string    `json:"banner" bson:"banner" db:"banner"`
	TagLine     string    `json:"tagLine" bson:"tagLine" db:"tagLine"`
	CreatedDate int64     `json:"createdDate" bson:"createdDate" db:"createdDate"`
}

func (s *Service) ReadProfileAndLanguage(ctx context.Context, user userAuth) (*userProfile, string, error) {
	// Read profile
	profRes := <-s.base.Repository.FindOne(ctx, "userProfile", struct {
		ObjectId uuid.UUID `json:"objectId" bson:"_id" db:"objectId"`
	}{ObjectId: user.ObjectId})
	if profRes.Error() != nil {
		return nil, "", profRes.Error()
	}
	var profile userProfile
	if err := profRes.Decode(&profile); err != nil {
		return nil, "", err
	}
	// Language: get setting key path, fallback to en
	langPath := "lang:current"
	_ = langPath
	current := "en"
	return &profile, current, nil
}

func (s *Service) ComparePassword(hashed []byte, plain string) error {
	return utils.CompareHash(hashed, []byte(plain))
}

// AuthenticateUser authenticates a user with username and password
func (s *Service) AuthenticateUser(ctx context.Context, username, password string) (string, error) {
	user, err := s.FindUserByUsername(ctx, username)
	if err != nil {
		return "", errors.WrapUserNotFoundError(fmt.Errorf("user not found"))
	}

	if err := s.ComparePassword(user.Password, password); err != nil {
		return "", errors.WrapAuthenticationError(fmt.Errorf("invalid password"))
	}

	// Generate authentication token
	claim := map[string]interface{}{
		"displayName":   user.Username,
		"email":         user.Username,
		types.HeaderUID: user.ObjectId.String(),
		"role":          user.Role,
		"createdDate":   utils.UTCNowUnix(),
	}

	profileInfo := map[string]string{
		"id":       user.ObjectId.String(),
		"login":    user.Username,
		"name":     user.Username,
		"audience": "",
	}

	return tokens.CreateTokenWithKey("telar", profileInfo, "Telar", claim, s.config.JWTConfig.PrivateKey)
}

// ValidateToken validates a JWT token and returns user information
func (s *Service) ValidateToken(ctx context.Context, token string) (*models.TokenClaim, error) {
	// This would typically use JWT validation
	// For now, return a placeholder implementation
	return &models.TokenClaim{
		DisplayName: "placeholder",
		SocialName:  "placeholder",
		Email:       "placeholder@example.com",
		UID:         "placeholder",
		Role:        "user",
		CreatedDate: 0,
	}, nil
}

// RefreshToken refreshes an expired JWT token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// This would typically validate the refresh token and generate a new access token
	// For now, return a placeholder implementation
	return "", fmt.Errorf("token refresh not yet implemented")
}

// HandleGithubLogin processes GitHub OAuth login
func (s *Service) HandleGithubLogin(ctx context.Context, code string) (string, error) {
	// This would typically exchange the code for an access token and get user info
	// For now, return a placeholder implementation
	return "", fmt.Errorf("GitHub OAuth not yet implemented")
}

// HandleGoogleLogin processes Google OAuth login
func (s *Service) HandleGoogleLogin(ctx context.Context, code string) (string, error) {
	// This would typically exchange the code for an access token and get user info
	// For now, return a placeholder implementation
	return "", fmt.Errorf("Google OAuth not yet implemented")
}

// ProcessOAuthCallback processes OAuth callback for any provider
func (s *Service) ProcessOAuthCallback(ctx context.Context, provider, code string) (string, error) {
	switch provider {
	case "github":
		return s.HandleGithubLogin(ctx, code)
	case "google":
		return s.HandleGoogleLogin(ctx, code)
	default:
		return "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}
