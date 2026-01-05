package bootstrap

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	authModels "github.com/qolzam/telar/apps/api/auth/models"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	profileModels "github.com/qolzam/telar/apps/api/profile/models"
	profileRepository "github.com/qolzam/telar/apps/api/profile/repository"
	"golang.org/x/crypto/bcrypt"
)

// EnsureAdminUser creates the initial admin user if it doesn't exist.
// This function is called on server startup to bootstrap the admin account.
func EnsureAdminUser(ctx context.Context, authRepo authRepository.AuthRepository, profileRepo profileRepository.ProfileRepository, cfg *platformconfig.Config) error {
	email := cfg.App.InitialAdminEmail
	password := cfg.App.InitialAdminPassword

	// Validate that credentials are provided
	if email == "" {
		return fmt.Errorf("INITIAL_ADMIN_EMAIL is required but not set")
	}
	if password == "" || password == "ChangeMe123!" {
		log.Printf("[WARN] Using default admin password. Please set INITIAL_ADMIN_PASSWORD in production!")
	}

	// Check if admin user already exists
	existingUser, err := authRepo.FindByUsername(ctx, email)
	if err == nil && existingUser != nil {
		// User exists - log and return (not an error)
		if existingUser.Role == "admin" {
			log.Printf("[INFO] Admin user already exists: %s", email)
			return nil
		}
		// User exists but not admin - this is unexpected, log warning
		log.Printf("[WARN] User %s exists but is not an admin. Skipping admin bootstrap.", email)
		return nil
	}

	// If error is not "user not found", return it
	if err != nil && err.Error() != "user not found" {
		return fmt.Errorf("failed to check for existing admin: %w", err)
	}

	// Admin doesn't exist - create it
	log.Printf("[INFO] Creating initial admin user: %s", email)

	// Extract full name from email (fallback if not provided)
	fullName := extractFullNameFromEmail(email)

	// Use transaction for atomic User+Profile creation
	err = authRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		// 1. Hash password (CPU-intensive, do this before transaction work)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// 2. Create UserAuth (within transaction)
		userID := uuid.Must(uuid.NewV4())
		now := time.Now().Unix()
		userAuth := &authModels.UserAuth{
			ObjectId:      userID,
			Username:      email,
			Password:      hashedPassword,
			Role:          "admin",
			EmailVerified: true,
			PhoneVerified: false,
			CreatedDate:   now,
			LastUpdated:   now,
		}

		if err := authRepo.CreateUser(txCtx, userAuth); err != nil {
			// Check for unique constraint violation
			if err.Error() == "username already exists" {
				return fmt.Errorf("admin user already exists (race condition)")
			}
			return fmt.Errorf("failed to create user auth: %w", err)
		}

		// 3. Create Profile (within transaction)
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

		if err := profileRepo.Create(txCtx, profile); err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}

		return nil // Returning nil commits the transaction
	})

	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Printf("[INFO] Admin user created successfully: %s", email)
	return nil
}

// extractFullNameFromEmail extracts a full name from an email address
func extractFullNameFromEmail(email string) string {
	// Extract local part (before @)
	if !strings.Contains(email, "@") {
		return "Admin User"
	}

	localPart := strings.Split(email, "@")[0]
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
	result := strings.Join(words, " ")
	if result == "" {
		return "Admin User"
	}
	return result
}

// generateSocialName generates a social name from full name and user ID
// The generated name must conform to: alphanumeric, underscore, hyphen, period
// Must start and end with alphanumeric (per profile validation rules)
func generateSocialName(fullName string, userId string) string {
	// Simple implementation: use first name + first 8 chars of user ID
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		if len(userId) >= 8 {
			return "user_" + userId[:8]
		}
		return "user_" + userId
	}

	firstName := parts[0]
	// Sanitize firstName: remove all non-alphanumeric characters except underscore, hyphen, period
	// This ensures the generated social_name passes validation
	sanitized := strings.Builder{}
	for _, r := range firstName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			sanitized.WriteRune(r)
		}
	}
	sanitizedStr := strings.ToLower(sanitized.String())

	// Ensure we have at least one alphanumeric character
	if len(sanitizedStr) == 0 || !((sanitizedStr[0] >= 'a' && sanitizedStr[0] <= 'z') || (sanitizedStr[0] >= '0' && sanitizedStr[0] <= '9')) {
		sanitizedStr = "user"
	}

	// Remove leading/trailing non-alphanumeric to ensure it starts/ends with alphanumeric
	sanitizedStr = strings.Trim(sanitizedStr, "_.-")
	if len(sanitizedStr) == 0 {
		sanitizedStr = "user"
	}

	if len(userId) >= 8 {
		return fmt.Sprintf("%s_%s", sanitizedStr, userId[:8])
	}
	return sanitizedStr
}
