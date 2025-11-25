package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	authErrors "github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	adminRepository "github.com/qolzam/telar/apps/api/auth/admin/repository"
	profileRepository "github.com/qolzam/telar/apps/api/profile/repository"
	profileModels "github.com/qolzam/telar/apps/api/profile/models"
	tokens "github.com/qolzam/telar/apps/api/internal/auth/tokens"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	authRepo    authRepository.AuthRepository
	profileRepo profileRepository.ProfileRepository
	adminRepo   adminRepository.AdminRepository
	privateKey  string
	config      *platformconfig.Config
}

// NewService creates a service with repositories injected
func NewService(authRepo authRepository.AuthRepository, profileRepo profileRepository.ProfileRepository, adminRepo adminRepository.AdminRepository, privateKey string, cfg *platformconfig.Config) *Service {
	return &Service{
		authRepo:    authRepo,
		profileRepo: profileRepo,
		adminRepo:   adminRepo,
		privateKey:  privateKey,
		config:      cfg,
	}
}

// Legacy query builder removed - all queries now use repository interfaces

// CheckAdmin checks if any admin exists in the system
// This method is used for system status checks, not for preventing multiple admin creation
func (s *Service) CheckAdmin(ctx context.Context) (bool, error) {
	if s.authRepo == nil {
		return false, fmt.Errorf("auth repository not available")
	}
	
	// Find any user with admin role
	_, err := s.authRepo.FindByRole(ctx, "admin")
	if err != nil {
		// If not found, return false (no error)
		if err.Error() == "user with role admin not found" {
			return false, nil
		}
		return false, authErrors.WrapDatabaseError(fmt.Errorf("failed to check for admin: %w", err))
	}
	
	return true, nil
}

func (s *Service) CreateAdmin(ctx context.Context, fullName, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", authErrors.WrapValidationError(fmt.Errorf("email and password required"), "email,password")
	}

	if s.authRepo == nil || s.profileRepo == nil {
		return "", fmt.Errorf("repositories not available")
	}

	var createdUserAuth *models.UserAuth
	var token string

	// Use AuthRepository's WithTransaction for atomic User+Profile creation
	err := s.authRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		// 1. Check if admin with this email already exists
		existingUser, err := s.authRepo.FindByUsername(txCtx, email)
		if err == nil && existingUser != nil {
			// User exists - check if it's an admin
			if existingUser.Role == "admin" {
				return authErrors.ErrUserAlreadyExists
			}
			// User exists but not admin - still return error
			return authErrors.ErrUserAlreadyExists
		}
		// If error is "user not found", that's fine - continue
		if err != nil && err.Error() != "user not found" {
			return fmt.Errorf("failed to check for existing admin: %w", err)
		}

		// 2. Hash password (CPU-intensive, do this *before* saving)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return authErrors.NewAuthError(authErrors.CodeSystemError, "failed to hash password", err)
		}

		// 3. Create userAuth (within transaction)
		userID := uuid.Must(uuid.NewV4())
		now := time.Now().Unix()
		ua := &models.UserAuth{
			ObjectId:      userID,
			Username:      email,
			Password:      hashedPassword,
			Role:          "admin",
			EmailVerified: true,
			PhoneVerified: true,
			CreatedDate:   now,
			LastUpdated:   now,
		}
		if err := s.authRepo.CreateUser(txCtx, ua); err != nil {
			// Check for unique constraint violation
			if err.Error() == "username already exists" {
				return authErrors.ErrUserAlreadyExists
			}
			return fmt.Errorf("failed to create user auth: %w", err)
		}

		// 4. Create userProfile (within transaction)
		socialName := generateSocialName(fullName, userID.String())
		profile := &profileModels.Profile{
			ObjectId:    userID,
			FullName:    fullName,
			SocialName:  socialName,
			Email:       email,
			Avatar:      "https://util.telar.dev/api/avatars/" + userID.String(),
			Banner:      "https://picsum.photos/id/1/900/300/?blur",
			Tagline:     "",
			CreatedDate: now,
			LastUpdated: now,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Permission:  "Public",
		}
		if err := s.profileRepo.Create(txCtx, profile); err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}

		// Store the created user for token generation after commit
		createdUserAuth = ua
		return nil // Returning nil commits the transaction
	})

	if err != nil {
		// If the transaction failed, check if it's already a properly typed error
		if authErr, ok := err.(*authErrors.AuthError); ok {
			// Already a typed AuthError, return it directly
			return "", authErr
		}
		// Check if it's the "user already exists" error
		if err == authErrors.ErrUserAlreadyExists || errors.Is(err, authErrors.ErrUserAlreadyExists) {
			return "", authErrors.ErrUserAlreadyExists
		}
		// Otherwise, wrap as database error but preserve the underlying error message
		return "", authErrors.NewAuthError(authErrors.CodeDatabaseError, fmt.Sprintf("Database operation failed: %v", err), err)
	}

	// 5. Token creation (CPU intensive, happens *after* the transaction is committed)
	// This ensures we are not holding a database connection while doing CPU work.
	profileInfo := map[string]string{
		"id":       createdUserAuth.ObjectId.String(),
		"login":    createdUserAuth.Username,
		"name":     fullName,
		"audience": "",
	}
	claim := map[string]interface{}{
		"displayName":   fullName,
		"socialName":    generateSocialName(fullName, createdUserAuth.ObjectId.String()),
		"email":         email,
		types.HeaderUID: createdUserAuth.ObjectId.String(),
		"role":          createdUserAuth.Role,
		"createdDate":   createdUserAuth.CreatedDate,
		"jti":           uuid.Must(uuid.NewV4()).String(),
	}
	token, err = s.createTelarToken(profileInfo, claim)
	if err != nil {
		return "", authErrors.WrapAuthenticationError(fmt.Errorf("failed to create token: %w", err))
	}

	return token, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	if s.authRepo == nil {
		return "", fmt.Errorf("auth repository not available")
	}
	
	// Find user by username
	user, err := s.authRepo.FindByUsername(ctx, email)
	if err != nil {
		return "", authErrors.WrapUserNotFoundError(fmt.Errorf("admin not found: %w", err))
	}
	
	// Verify it's an admin
	if user.Role != "admin" {
		return "", authErrors.WrapUserNotFoundError(fmt.Errorf("admin not found"))
	}
	
	// Verify password
	if utils.CompareHash(user.Password, []byte(password)) != nil {
		return "", authErrors.WrapAuthenticationError(fmt.Errorf("password does not match"))
	}
	
	claim := map[string]interface{}{
		"displayName":   email,
		"socialName":    generateSocialName(email, user.ObjectId.String()),
		"email":         email,
		types.HeaderUID: user.ObjectId.String(),
		"role":          "admin",
		"createdDate":   user.CreatedDate,
	}
	profileInfo := map[string]string{
		"id":       user.ObjectId.String(),
		"login":    email,
		"name":     email,
		"audience": "",
	}
	return s.createTelarToken(profileInfo, claim)
}

// helpers
func generateSocialName(name, uid string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "") + strings.Split(uid, "-")[0])
}

func (s *Service) createTelarToken(profile map[string]string, claim map[string]interface{}) (string, error) {
	return tokens.CreateTokenWithKey("telar", profile, "Telar", claim, s.privateKey)
}

