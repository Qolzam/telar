package services

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/models"
	"github.com/qolzam/telar/apps/api/auth/signup"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	// User Authentication
	CreateUserAuth(ctx context.Context, userAuth *models.UserAuth) error
	FindUserAuthByUsername(ctx context.Context, username string) (*models.UserAuth, error)
	FindUserAuthByUserId(ctx context.Context, userId uuid.UUID) (*models.UserAuth, error)
	UpdateUserAuth(ctx context.Context, filter *models.DatabaseFilter, data *models.DatabaseUpdate) error
	UpdatePassword(ctx context.Context, userId uuid.UUID, newPassword []byte) error
	DeleteUserAuth(ctx context.Context, filter *models.DatabaseFilter) error
	CheckAdmin(ctx context.Context) (*models.UserAuth, error)
}

// UserVerificationService defines the interface for user verification operations
type UserVerificationService interface {
	// Secure Email Verification (Phase 1 refactoring)
	InitiateEmailVerification(ctx context.Context, input signup.EmailVerificationRequest) (*signup.EmailVerificationResponse, error)
	VerifyEmailCode(ctx context.Context, verificationId string, code string) error

	// Secure Phone Verification (Phase 1 refactoring)
	InitiatePhoneVerification(ctx context.Context, input signup.PhoneVerificationRequest) (*signup.PhoneVerificationResponse, error)
	VerifyPhoneCode(ctx context.Context, verificationId string, code string) error

	// General Verification
	SaveUserVerification(ctx context.Context, userVerification *models.UserVerification) error
	FindUserVerification(ctx context.Context, filter *models.DatabaseFilter) (*models.UserVerification, error)
	UpdateUserVerification(ctx context.Context, filter *models.DatabaseFilter, data *models.DatabaseUpdate) error
	DeleteUserVerification(ctx context.Context, filter *models.DatabaseFilter) error
}

// LoginService defines the interface for login operations
type LoginService interface {
	// Authentication
	AuthenticateUser(ctx context.Context, username, password string) (*models.AuthenticationResult, error)
	ValidateToken(ctx context.Context, token string) (*models.TokenClaim, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, error)

	// OAuth
	HandleGithubLogin(ctx context.Context, code string) (string, error)
	HandleGoogleLogin(ctx context.Context, code string) (string, error)
	ProcessOAuthCallback(ctx context.Context, provider, code string) (string, error)
}

// PasswordService defines the interface for password operations
type PasswordService interface {
	// Password Management
	ChangePassword(ctx context.Context, userId uuid.UUID, oldPassword, newPassword string) error
	ResetPassword(ctx context.Context, resetToken, newPassword string) error
	ForgetPassword(ctx context.Context, email string) error
	ValidateResetToken(ctx context.Context, resetToken string) (bool, error)
}

// ProfileService defines the interface for profile operations
type ProfileService interface {
	// Profile Management
	GetProfile(ctx context.Context, userId uuid.UUID) (*models.UserProfile, error)
	UpdateProfile(ctx context.Context, userId uuid.UUID, updates *models.ProfileUpdate) error
	UpdateAvatar(ctx context.Context, userId uuid.UUID, avatar string) error
	UpdateBanner(ctx context.Context, userId uuid.UUID, banner string) error

	// Profile Search
	SearchProfiles(ctx context.Context, query string, filter *models.ProfileSearchFilter) ([]*models.UserProfile, error)
	GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.UserProfile, error)
}

// AdminService defines the interface for admin operations
type AdminService interface {
	// Admin Management
	CheckAdmin(ctx context.Context) (bool, error)
	CreateAdmin(ctx context.Context, role, email, password string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	ValidateAdminToken(ctx context.Context, token string) (bool, error)
}
