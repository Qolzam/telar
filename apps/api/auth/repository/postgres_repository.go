// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qolzam/telar/apps/api/auth/models"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
)

// getExecutor returns either the transaction from context or the DB connection
func (r *postgresAuthRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	// Check for transaction in context (shared key for cross-package transactions)
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.client.DB()
}

// postgresAuthRepository implements AuthRepository using raw SQL queries
type postgresAuthRepository struct {
	client *postgres.Client
}

// NewPostgresAuthRepository creates a new PostgreSQL repository for user authentication
func NewPostgresAuthRepository(client *postgres.Client) AuthRepository {
	return &postgresAuthRepository{
		client: client,
	}
}

// CreateUser inserts a new user authentication record
func (r *postgresAuthRepository) CreateUser(ctx context.Context, userAuth *models.UserAuth) error {
	// Set timestamps if not set
	now := time.Now()
	nowUnix := now.Unix()
	if userAuth.CreatedDate == 0 {
		userAuth.CreatedDate = nowUnix
	}
	if userAuth.LastUpdated == 0 {
		userAuth.LastUpdated = nowUnix
	}

	query := `
		INSERT INTO user_auths (
			id, username, password_hash, role, email_verified, phone_verified,
			created_at, updated_at, created_date, last_updated
		) VALUES (
			:id, :username, :password_hash, :role, :email_verified, :phone_verified,
			:created_at, :updated_at, :created_date, :last_updated
		)`

	insertData := struct {
		ID            uuid.UUID `db:"id"`
		Username      string    `db:"username"`
		PasswordHash  []byte    `db:"password_hash"`
		Role          string    `db:"role"`
		EmailVerified bool      `db:"email_verified"`
		PhoneVerified bool      `db:"phone_verified"`
		CreatedAt     time.Time `db:"created_at"`
		UpdatedAt     time.Time `db:"updated_at"`
		CreatedDate   int64     `db:"created_date"`
		LastUpdated   int64     `db:"last_updated"`
	}{
		ID:            userAuth.ObjectId,
		Username:      userAuth.Username,
		PasswordHash:  userAuth.Password,
		Role:          userAuth.Role,
		EmailVerified: userAuth.EmailVerified,
		PhoneVerified: userAuth.PhoneVerified,
		CreatedAt:     now,
		UpdatedAt:     now,
		CreatedDate:   userAuth.CreatedDate,
		LastUpdated:   userAuth.LastUpdated,
	}

	_, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, insertData)
	if err != nil {
		// Check for unique constraint violation on username
		if isUniqueConstraintError(err, "username") {
			return fmt.Errorf("username already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// FindByUsername retrieves a user by username (email)
func (r *postgresAuthRepository) FindByUsername(ctx context.Context, username string) (*models.UserAuth, error) {
	query := `
		SELECT 
			id, username, password_hash, role, email_verified, phone_verified,
			created_at, updated_at, created_date, last_updated
		FROM user_auths 
		WHERE username = $1`

	var result struct {
		ID            uuid.UUID  `db:"id"`
		Username      string     `db:"username"`
		PasswordHash  []byte     `db:"password_hash"`
		Role          string     `db:"role"`
		EmailVerified bool       `db:"email_verified"`
		PhoneVerified bool       `db:"phone_verified"`
		CreatedAt     time.Time  `db:"created_at"`
		UpdatedAt     time.Time  `db:"updated_at"`
		CreatedDate   int64      `db:"created_date"`
		LastUpdated   int64      `db:"last_updated"`
	}

	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &result, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}

	return &models.UserAuth{
		ObjectId:      result.ID,
		Username:      result.Username,
		Password:      result.PasswordHash,
		Role:          result.Role,
		EmailVerified: result.EmailVerified,
		PhoneVerified: result.PhoneVerified,
		CreatedDate:   result.CreatedDate,
		LastUpdated:   result.LastUpdated,
	}, nil
}

// FindByID retrieves a user by ID
func (r *postgresAuthRepository) FindByID(ctx context.Context, userID uuid.UUID) (*models.UserAuth, error) {
	query := `
		SELECT 
			id, username, password_hash, role, email_verified, phone_verified,
			created_at, updated_at, created_date, last_updated
		FROM user_auths 
		WHERE id = $1`

	var result struct {
		ID            uuid.UUID  `db:"id"`
		Username      string     `db:"username"`
		PasswordHash  []byte     `db:"password_hash"`
		Role          string     `db:"role"`
		EmailVerified bool       `db:"email_verified"`
		PhoneVerified bool       `db:"phone_verified"`
		CreatedAt     time.Time  `db:"created_at"`
		UpdatedAt     time.Time  `db:"updated_at"`
		CreatedDate   int64      `db:"created_date"`
		LastUpdated   int64      `db:"last_updated"`
	}

	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &result, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}

	return &models.UserAuth{
		ObjectId:      result.ID,
		Username:      result.Username,
		Password:      result.PasswordHash,
		Role:          result.Role,
		EmailVerified: result.EmailVerified,
		PhoneVerified: result.PhoneVerified,
		CreatedDate:   result.CreatedDate,
		LastUpdated:   result.LastUpdated,
	}, nil
}

// FindByRole retrieves the first user with the specified role
func (r *postgresAuthRepository) FindByRole(ctx context.Context, role string) (*models.UserAuth, error) {
	query := `
		SELECT 
			id, username, password_hash, role, email_verified, phone_verified,
			created_at, updated_at, created_date, last_updated
		FROM user_auths 
		WHERE role = $1
		LIMIT 1`

	var result struct {
		ID            uuid.UUID  `db:"id"`
		Username      string     `db:"username"`
		PasswordHash  []byte     `db:"password_hash"`
		Role          string     `db:"role"`
		EmailVerified bool       `db:"email_verified"`
		PhoneVerified bool       `db:"phone_verified"`
		CreatedAt     time.Time  `db:"created_at"`
		UpdatedAt     time.Time  `db:"updated_at"`
		CreatedDate   int64      `db:"created_date"`
		LastUpdated   int64      `db:"last_updated"`
	}

	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &result, query, role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with role %s not found", role)
		}
		return nil, fmt.Errorf("failed to find user by role: %w", err)
	}

	return &models.UserAuth{
		ObjectId:      result.ID,
		Username:      result.Username,
		Password:      result.PasswordHash,
		Role:          result.Role,
		EmailVerified: result.EmailVerified,
		PhoneVerified: result.PhoneVerified,
		CreatedDate:   result.CreatedDate,
		LastUpdated:   result.LastUpdated,
	}, nil
}

// UpdatePassword updates the password hash for a user
func (r *postgresAuthRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash []byte) error {
	query := `
		UPDATE user_auths 
		SET password_hash = $1, 
		    updated_at = NOW(),
		    last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateEmailVerified updates the email verification status
func (r *postgresAuthRepository) UpdateEmailVerified(ctx context.Context, userID uuid.UUID, verified bool) error {
	query := `
		UPDATE user_auths 
		SET email_verified = $1,
		    updated_at = NOW(),
		    last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, verified, userID)
	if err != nil {
		return fmt.Errorf("failed to update email verified status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdatePhoneVerified updates the phone verification status
func (r *postgresAuthRepository) UpdatePhoneVerified(ctx context.Context, userID uuid.UUID, verified bool) error {
	query := `
		UPDATE user_auths 
		SET phone_verified = $1,
		    updated_at = NOW(),
		    last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, verified, userID)
	if err != nil {
		return fmt.Errorf("failed to update phone verified status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Delete deletes a user authentication record
func (r *postgresAuthRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_auths WHERE id = $1`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// WithTransaction executes a function within a database transaction
// This is critical for atomic User+Profile creation
func (r *postgresAuthRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := r.client.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a new context with the transaction (using string key for cross-package compatibility)
	txCtx := context.WithValue(ctx, "tx", tx)

	// Execute the function
	if err := fn(txCtx); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return fmt.Errorf("transaction failed and rollback failed: %w (original error: %v)", rollbackErr, err)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Helper function to check for unique constraint violations
func isUniqueConstraintError(err error, column string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL unique constraint error format: "pq: duplicate key value violates unique constraint..."
	return contains(errStr, "unique constraint") || contains(errStr, "duplicate key")
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

