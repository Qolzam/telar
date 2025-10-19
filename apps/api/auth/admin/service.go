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
	tokens "github.com/qolzam/telar/apps/api/internal/auth/tokens"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	base       *platform.BaseService
	privateKey string
	config     *platformconfig.Config
}

func NewService(base *platform.BaseService, privateKey string, cfg *platformconfig.Config) *Service {
	return &Service{
		base:       base,
		privateKey: privateKey,
		config:     cfg,
	}
}

type userAuth struct {
	ObjectId      uuid.UUID `json:"objectId" bson:"objectId"`
	Username      string    `json:"username" bson:"username"`
	Password      []byte    `json:"password" bson:"password"`
	Role          string    `json:"role" bson:"role"`
	EmailVerified bool      `json:"emailVerified" bson:"emailVerified"`
	PhoneVerified bool      `json:"phoneVerified" bson:"phoneVerified"`
	CreatedDate   int64     `json:"createdDate" bson:"createdDate"`
	LastUpdated   int64     `json:"lastUpdated" bson:"lastUpdated"`
}

// CheckAdmin checks if any admin exists in the system
// This method is used for system status checks, not for preventing multiple admin creation
func (s *Service) CheckAdmin(ctx context.Context) (bool, error) {
	res := <-s.base.Repository.FindOne(ctx, "userAuth", struct {
		Role string `json:"role" bson:"role"`
	}{Role: "admin"})
	if res.Error() != nil {
		return false, nil
	}
	var ua userAuth
	if err := res.Decode(&ua); err != nil {
		return false, authErrors.WrapDatabaseError(fmt.Errorf("failed to decode user auth: %w", err))
	}
	return ua.ObjectId != uuid.Nil, nil
}

func (s *Service) CreateAdmin(ctx context.Context, fullName, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", authErrors.WrapValidationError(fmt.Errorf("email and password required"), "email,password")
	}

	var createdUserAuth models.UserAuth
	var token string

	// --- THE FIX: Use a single transaction for all database operations ---
	err := s.base.Repository.WithTransaction(ctx, func(txCtx context.Context) error {
		// 1. Check if admin exists (within transaction)
		// We can use a more specific filter here for clarity.
		findFilter := map[string]interface{}{"username": email, "role": "admin"}
		existingAdminCheck := <-s.base.Repository.FindOne(txCtx, "userAuth", findFilter)

		// Use the robust existence check pattern
		var dummy models.UserAuth
		decodeErr := existingAdminCheck.Decode(&dummy)
		if decodeErr == nil {
			// Found a user, so it already exists.
			return authErrors.ErrUserAlreadyExists
		}
		if !errors.Is(decodeErr, dbi.ErrNoDocuments) {
			// A real database error occurred during the check.
			return fmt.Errorf("failed to check for existing admin: %w", decodeErr)
		}

		// 2. Hash password (CPU-intensive, do this *before* saving)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return authErrors.NewAuthError(authErrors.CodeSystemError, "failed to hash password", err)
		}

		// 3. Create userAuth (within transaction)
		userID := uuid.Must(uuid.NewV4())
		now := time.Now().Unix()
		ua := models.UserAuth{
			ObjectId:      userID,
			Username:      email,
			Password:      hashedPassword,
			Role:          "admin",
			EmailVerified: true,
			PhoneVerified: true,
			CreatedDate:   now,
			LastUpdated:   now,
		}
		if err := (<-s.base.Repository.Save(txCtx, "userAuth", ua)).Error; err != nil {
			return fmt.Errorf("failed to save user auth: %w", err)
		}

		// 4. Create userProfile (within transaction)
		socialName := generateSocialName(fullName, userID.String())
		up := models.UserProfile{
			ObjectId:    userID,
			FullName:    fullName,
			SocialName:  socialName,
			Email:       email,
			Avatar:      "https://util.telar.dev/api/avatars/" + userID.String(),
			Banner:      "https://picsum.photos/id/1/900/300/?blur",
			CreatedDate: now,
			LastUpdated: now,
		}
		if err := (<-s.base.Repository.Save(txCtx, "userProfile", &up)).Error; err != nil {
			return fmt.Errorf("failed to save user profile: %w", err)
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
		// Otherwise, wrap as database error
		return "", authErrors.WrapDatabaseError(err)
	}
	// --- END OF FIX ---

	// 5. Token creation (CPU intensive, happens *after* the transaction is committed)
	// This ensures we are not holding a database connection while doing CPU work.
	profileInfo := map[string]string{
		"id":       createdUserAuth.ObjectId.String(),
		"login":    createdUserAuth.Username,
		"name":     fullName, // Assuming fullName from profile
		"audience": "",       // Will be set by config if needed
	}
	claim := map[string]interface{}{
		"displayName":   fullName,
		"socialName":    generateSocialName(fullName, createdUserAuth.ObjectId.String()),
		"email":         email,
		types.HeaderUID: createdUserAuth.ObjectId.String(),
		"role":          createdUserAuth.Role,
		"createdDate":   createdUserAuth.CreatedDate,
	}
	token, err = s.createTelarToken(profileInfo, claim)
	if err != nil {
		return "", authErrors.WrapAuthenticationError(fmt.Errorf("failed to create token: %w", err))
	}

	return token, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	// Find admin by username and role
	res := <-s.base.Repository.FindOne(ctx, "userAuth", struct {
		Username string `json:"username" bson:"username"`
		Role     string `json:"role" bson:"role"`
	}{Username: email, Role: "admin"})
	if res.Error() != nil {
		return "", authErrors.WrapUserNotFoundError(fmt.Errorf("admin not found"))
	}
	var ua userAuth
	if err := res.Decode(&ua); err != nil {
		return "", authErrors.WrapDatabaseError(fmt.Errorf("failed to decode user auth: %w", err))
	}
	if utils.CompareHash(ua.Password, []byte(password)) != nil {
		return "", authErrors.WrapAuthenticationError(fmt.Errorf("password does not match"))
	}
	claim := map[string]interface{}{"displayName": email, "socialName": generateSocialName(email, ua.ObjectId.String()), "email": email, types.HeaderUID: ua.ObjectId.String(), "role": "admin", "createdDate": ua.CreatedDate}
	profileInfo := map[string]string{"id": ua.ObjectId.String(), "login": email, "name": email, "audience": ""}
	return s.createTelarToken(profileInfo, claim)
}

// helpers
func generateSocialName(name, uid string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "") + strings.Split(uid, "-")[0])
}

func (s *Service) createTelarToken(profile map[string]string, claim map[string]interface{}) (string, error) {
	return tokens.CreateTokenWithKey("telar", profile, "Telar", claim, s.privateKey)
}
