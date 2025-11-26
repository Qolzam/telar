// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package signup

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	authModels "github.com/qolzam/telar/apps/api/auth/models"
	authRepo "github.com/qolzam/telar/apps/api/auth/repository"
	profileModels "github.com/qolzam/telar/apps/api/profile/models"
	profileRepo "github.com/qolzam/telar/apps/api/profile/repository"
)

// Service defines the orchestration logic for signup completion
// This orchestrator coordinates atomic creation of User Auth and Profile
// across service boundaries using transactions
type Service interface {
	CompleteSignup(ctx context.Context, verification *authModels.UserVerification) error
}

type service struct {
	authRepo    authRepo.AuthRepository
	profileRepo profileRepo.ProfileRepository
	verifRepo   authRepo.VerificationRepository
}

// NewService creates a new signup orchestrator service
func NewService(
	authRepo authRepo.AuthRepository,
	profileRepo profileRepo.ProfileRepository,
	verifRepo authRepo.VerificationRepository,
) Service {
	return &service{
		authRepo:    authRepo,
		profileRepo: profileRepo,
		verifRepo:   verifRepo,
	}
}

// CompleteSignup orchestrates the atomic creation of User Auth and Profile
// This method ensures both entities are created atomically within a single transaction
func (s *service) CompleteSignup(ctx context.Context, verification *authModels.UserVerification) error {
	if verification == nil {
		return fmt.Errorf("verification record is required")
	}

	// Validate verification is not already used
	if verification.Used {
		return fmt.Errorf("verification code already used")
	}

	// Validate verification has not expired
	if time.Now().Unix() > verification.ExpiresAt {
		return fmt.Errorf("verification code has expired")
	}

	// Debug: Log verification details
	log.Printf("[CompleteSignup] Verification ID: %s, UserId: %s, Target: %s", 
		verification.ObjectId.String(), verification.UserId.String(), verification.Target)

	// Validate UserId is not Nil (should be populated from future_user_id by repository)
	if verification.UserId == uuid.Nil {
		return fmt.Errorf("verification UserId is Nil - cannot create user")
	}

	// Use authRepo.WithTransaction to start the transaction scope
	// Crucially, we pass the transaction context `txCtx` to ALL repositories
	// Note: Verification is already marked as used by verifyUserByCode before this is called
		return s.authRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		// A. Create Auth User (within transaction)
		log.Printf("[CompleteSignup] Creating user auth with ObjectId: %s, Username: %s", 
			verification.UserId.String(), verification.Target)
		userAuth := &authModels.UserAuth{
			ObjectId:      verification.UserId,
			Username:      verification.Target,
			Password:      verification.HashedPassword,
			EmailVerified: verification.TargetType == "email",
			PhoneVerified: verification.TargetType == "phone",
			Role:          "user",
			CreatedDate:   time.Now().Unix(),
			LastUpdated:   time.Now().Unix(),
		}

		if err := s.authRepo.CreateUser(txCtx, userAuth); err != nil {
			log.Printf("[CompleteSignup] Failed to create user auth: %v", err)
			return fmt.Errorf("failed to create user auth: %w", err)
		}
		log.Printf("[CompleteSignup] User auth created successfully")

		// C. Create Profile (within transaction)
		// This works because `profileRepo` methods accept a context.
		// If `txCtx` contains the *sqlx.Tx, and `profileRepo` knows how to extract it
		// (which it should, via shared transaction key "tx"), this will be atomic.
		fullName := verification.FullName
		if fullName == "" {
			// Fallback for legacy verification records without stored full name
			fullName = extractFullNameFromTarget(verification.Target)
		}

		socialName := generateSocialName(fullName, verification.UserId.String())
		createdDate := time.Now().Unix()

		profile := &profileModels.Profile{
			ObjectId:      verification.UserId,
			FullName:      fullName,
			SocialName:    socialName,
			Email:         verification.Target,
			Avatar:        "",
			Banner:        "",
			Tagline:       "",
			CreatedDate:   createdDate,
			LastUpdated:   createdDate,
			LastSeen:      0,
			Birthday:      0,
			WebUrl:        "",
			CompanyName:   "",
			Country:       "",
			Address:       "",
			Phone:         "",
			VoteCount:     0,
			ShareCount:    0,
			FollowCount:   0,
			FollowerCount: 0,
			PostCount:     0,
			FacebookId:    "",
			InstagramId:   "",
			TwitterId:     "",
			LinkedInId:    "",
			Permission:    "Public",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		log.Printf("[CompleteSignup] Creating profile with ObjectId: %s, FullName: %s, SocialName: %s", 
			verification.UserId.String(), fullName, socialName)
		if err := s.profileRepo.Create(txCtx, profile); err != nil {
			log.Printf("[CompleteSignup] Failed to create user profile: %v", err)
			return fmt.Errorf("failed to create user profile: %w", err)
		}
		log.Printf("[CompleteSignup] Profile created successfully")

		// Update verification record to set user_id now that user exists (within same transaction)
		// This satisfies the FK constraint and allows future lookups
		log.Printf("[CompleteSignup] Updating verification user_id: %s", verification.UserId.String())
		if err := s.verifRepo.UpdateUserID(txCtx, verification.ObjectId, verification.UserId); err != nil {
			// Log but don't fail - user and profile are already created
			// The verification record can be updated later if needed
			log.Printf("[CompleteSignup] Warning: Failed to update verification user_id: %v", err)
		} else {
			log.Printf("[CompleteSignup] Verification user_id updated successfully")
		}

		log.Printf("[CompleteSignup] Signup completed successfully")
		return nil
	})
}

// extractFullNameFromTarget extracts a full name from an email target
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
	return target
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
	return strings.ToLower(firstName)
}

