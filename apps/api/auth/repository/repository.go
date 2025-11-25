// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/models"
)

// AuthRepository defines the interface for user authentication database operations
// This is a domain-specific repository that knows exactly what a "UserAuth" is
// and how to execute optimized SQL queries for that specific domain.
type AuthRepository interface {
	// CreateUser inserts a new user authentication record
	// This method should support transaction context for atomic User+Profile creation
	CreateUser(ctx context.Context, userAuth *models.UserAuth) error

	// FindByUsername retrieves a user by username (email)
	// This is the primary lookup method for login
	FindByUsername(ctx context.Context, username string) (*models.UserAuth, error)

	// FindByID retrieves a user by ID
	FindByID(ctx context.Context, userID uuid.UUID) (*models.UserAuth, error)

	// FindByRole retrieves the first user with the specified role
	// Used for checking if admins exist, etc.
	FindByRole(ctx context.Context, role string) (*models.UserAuth, error)

	// UpdatePassword updates the password hash for a user
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash []byte) error

	// UpdateEmailVerified updates the email verification status
	UpdateEmailVerified(ctx context.Context, userID uuid.UUID, verified bool) error

	// UpdatePhoneVerified updates the phone verification status
	UpdatePhoneVerified(ctx context.Context, userID uuid.UUID, verified bool) error

	// Delete deletes a user authentication record
	Delete(ctx context.Context, userID uuid.UUID) error

	// WithTransaction executes a function within a database transaction
	// This is critical for atomic User+Profile creation
	WithTransaction(ctx context.Context, fn func(context.Context) error) error
}

// VerificationRepository defines the interface for verification code database operations
// This handles email/phone verification codes and password reset tokens
type VerificationRepository interface {
	// SaveVerification inserts a new verification record
	SaveVerification(ctx context.Context, verification *models.UserVerification) error

	// FindByID retrieves a verification by its ID
	FindByID(ctx context.Context, verificationID uuid.UUID) (*models.UserVerification, error)

	// FindVerification retrieves a verification by code and type
	// Used for email/phone verification and password reset
	FindVerification(ctx context.Context, code string, verificationType string) (*models.UserVerification, error)

	// FindVerificationByUser retrieves a verification by user ID and type
	// Used to check if a user has an active verification
	FindVerificationByUser(ctx context.Context, userID uuid.UUID, verificationType string) (*models.UserVerification, error)

	// FindVerificationByTarget retrieves a verification by target (email/phone) and type
	// Used for password reset when user_id might be NULL
	FindVerificationByTarget(ctx context.Context, target string, verificationType string) (*models.UserVerification, error)

	// FindByHashedPassword retrieves a verification by hashed password (for secure reset tokens)
	// Used to look up password reset tokens by their hashed value
	FindByHashedPassword(ctx context.Context, hashedPassword string) (*models.UserVerification, error)

	// MarkVerified marks a verification as verified
	MarkVerified(ctx context.Context, verificationID uuid.UUID) error

	// MarkUsed marks a verification as used (for password reset tokens)
	MarkUsed(ctx context.Context, verificationID uuid.UUID) error

	// DeleteExpired deletes expired verification records
	// Used for cleanup of old verification codes
	DeleteExpired(ctx context.Context, beforeTime int64) error

	// UpdateVerificationCode updates the code and expiration for a verification
	// Used for resending verification emails
	UpdateVerificationCode(ctx context.Context, verificationID uuid.UUID, newCode string, newExpiresAt int64) error
}

