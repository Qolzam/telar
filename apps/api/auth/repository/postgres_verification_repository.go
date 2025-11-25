// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qolzam/telar/apps/api/auth/models"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
)

// postgresVerificationRepository implements VerificationRepository using raw SQL queries
type postgresVerificationRepository struct {
	client *postgres.Client
}

// NewPostgresVerificationRepository creates a new PostgreSQL repository for verifications
func NewPostgresVerificationRepository(client *postgres.Client) VerificationRepository {
	return &postgresVerificationRepository{
		client: client,
	}
}

// getExecutor returns either the transaction from context or the DB connection
func (r *postgresVerificationRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	// Check for transaction in context (shared key for cross-package transactions)
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.client.DB()
}

// SaveVerification inserts a new verification record
func (r *postgresVerificationRepository) SaveVerification(ctx context.Context, verification *models.UserVerification) error {
	// Set timestamps if not set
	nowUnix := time.Now().Unix()
	if verification.CreatedDate == 0 {
		verification.CreatedDate = nowUnix
	}
	if verification.LastUpdated == 0 {
		verification.LastUpdated = nowUnix
	}

	query := `
		INSERT INTO verifications (
			id, user_id, code, target, target_type, counter,
			created_date, last_updated, remote_ip_address, is_verified,
			hashed_password, expires_at, used, full_name
		) VALUES (
			:id, :user_id, :code, :target, :target_type, :counter,
			:created_date, :last_updated, :remote_ip_address, :is_verified,
			:hashed_password, :expires_at, :used, :full_name
		)`

	insertData := struct {
		ID             uuid.UUID  `db:"id"`
		UserID         *uuid.UUID `db:"user_id"` // Nullable
		Code           string     `db:"code"`
		Target         string     `db:"target"`
		TargetType     string     `db:"target_type"`
		Counter        int64      `db:"counter"`
		CreatedDate    int64      `db:"created_date"`
		LastUpdated    int64      `db:"last_updated"`
		RemoteIPAddr   string     `db:"remote_ip_address"`
		IsVerified     bool       `db:"is_verified"`
		HashedPassword []byte     `db:"hashed_password"`
		ExpiresAt      int64      `db:"expires_at"`
		Used           bool       `db:"used"`
		FullName       string     `db:"full_name"`
	}{
		ID:             verification.ObjectId,
		Code:           verification.Code,
		Target:         verification.Target,
		TargetType:     verification.TargetType,
		Counter:        verification.Counter,
		CreatedDate:    verification.CreatedDate,
		LastUpdated:    verification.LastUpdated,
		RemoteIPAddr:   verification.RemoteIpAddress,
		IsVerified:     verification.IsVerified,
		HashedPassword: verification.HashedPassword,
		ExpiresAt:      verification.ExpiresAt,
		Used:           verification.Used,
		FullName:       verification.FullName,
	}

	// Handle nullable user_id
	if verification.UserId != uuid.Nil {
		insertData.UserID = &verification.UserId
	}

	_, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, insertData)
	if err != nil {
		return fmt.Errorf("failed to save verification (ID: %s): %w", verification.ObjectId.String(), err)
	}

	return nil
}

// FindByID retrieves a verification by its ID
func (r *postgresVerificationRepository) FindByID(ctx context.Context, verificationID uuid.UUID) (*models.UserVerification, error) {
	query := `
		SELECT 
			id, user_id, code, target, target_type, counter,
			created_date, last_updated, remote_ip_address, is_verified,
			hashed_password, expires_at, used, full_name
		FROM verifications 
		WHERE id = $1`

	var result struct {
		ID             uuid.UUID      `db:"id"`
		UserID         *uuid.UUID     `db:"user_id"`
		Code           string         `db:"code"`
		Target         string         `db:"target"`
		TargetType     string         `db:"target_type"`
		Counter        int64          `db:"counter"`
		CreatedDate    int64          `db:"created_date"`
		LastUpdated    int64          `db:"last_updated"`
		RemoteIPAddr   sql.NullString `db:"remote_ip_address"`
		IsVerified     bool           `db:"is_verified"`
		HashedPassword []byte         `db:"hashed_password"`
		ExpiresAt      int64          `db:"expires_at"`
		Used           bool           `db:"used"`
		FullName       sql.NullString `db:"full_name"`
	}

	executor := r.getExecutor(ctx)
	
	// Debug: Log the query and parameters
	log.Printf("[FindByID] Query: %s, ID: %s", query, verificationID.String())
	
	err := sqlx.GetContext(ctx, executor, &result, query, verificationID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Debug: Check if table exists and has any records, and if this specific ID exists
			var count int
			var specificCount int
			countQuery := `SELECT COUNT(*) FROM verifications`
			specificQuery := `SELECT COUNT(*) FROM verifications WHERE id = $1`
			countErr := sqlx.GetContext(ctx, executor, &count, countQuery)
			specificErr := sqlx.GetContext(ctx, executor, &specificCount, specificQuery, verificationID)
			log.Printf("[FindByID] No rows found. Total records: %d (err: %v), Records with ID: %d (err: %v)", count, countErr, specificCount, specificErr)
			return nil, fmt.Errorf("verification not found (ID: %s, total records: %d, records with this ID: %d)", verificationID.String(), count, specificCount)
		}
		log.Printf("[FindByID] Query error: %v", err)
		return nil, fmt.Errorf("failed to find verification by ID: %w", err)
	}
	log.Printf("[FindByID] Successfully found verification ID: %s", verificationID.String())

	verification := &models.UserVerification{
		ObjectId:        result.ID,
		Code:            result.Code,
		Target:          result.Target,
		TargetType:      result.TargetType,
		Counter:         result.Counter,
		CreatedDate:     result.CreatedDate,
		LastUpdated:     result.LastUpdated,
		IsVerified:      result.IsVerified,
		HashedPassword:  result.HashedPassword,
		ExpiresAt:       result.ExpiresAt,
		Used:            result.Used,
	}

	if result.UserID != nil {
		verification.UserId = *result.UserID
	}
	if result.RemoteIPAddr.Valid {
		verification.RemoteIpAddress = result.RemoteIPAddr.String
	}
	if result.FullName.Valid {
		verification.FullName = result.FullName.String
	}

	return verification, nil
}

// FindVerification retrieves a verification by code and type
func (r *postgresVerificationRepository) FindVerification(ctx context.Context, code string, verificationType string) (*models.UserVerification, error) {
	query := `
		SELECT 
			id, user_id, code, target, target_type, counter,
			created_date, last_updated, remote_ip_address, is_verified,
			hashed_password, expires_at, used, full_name
		FROM verifications 
		WHERE code = $1 AND target_type = $2 AND used = FALSE
		ORDER BY created_date DESC
		LIMIT 1`

	var result struct {
		ID             uuid.UUID   `db:"id"`
		UserID         *uuid.UUID  `db:"user_id"` // Nullable
		Code           string      `db:"code"`
		Target         string      `db:"target"`
		TargetType     string      `db:"target_type"`
		Counter        int64       `db:"counter"`
		CreatedDate    int64       `db:"created_date"`
		LastUpdated    int64       `db:"last_updated"`
		RemoteIPAddr   sql.NullString `db:"remote_ip_address"`
		IsVerified     bool        `db:"is_verified"`
		HashedPassword []byte      `db:"hashed_password"`
		ExpiresAt      int64       `db:"expires_at"`
		Used           bool        `db:"used"`
		FullName       sql.NullString `db:"full_name"`
	}

	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &result, query, code, verificationType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("verification not found")
		}
		return nil, fmt.Errorf("failed to find verification: %w", err)
	}

	verification := &models.UserVerification{
		ObjectId:        result.ID,
		Code:            result.Code,
		Target:          result.Target,
		TargetType:      result.TargetType,
		Counter:         result.Counter,
		CreatedDate:     result.CreatedDate,
		LastUpdated:     result.LastUpdated,
		IsVerified:      result.IsVerified,
		HashedPassword:  result.HashedPassword,
		ExpiresAt:       result.ExpiresAt,
		Used:            result.Used,
	}

	// Handle nullable fields
	if result.UserID != nil {
		verification.UserId = *result.UserID
	}
	if result.RemoteIPAddr.Valid {
		verification.RemoteIpAddress = result.RemoteIPAddr.String
	}
	if result.FullName.Valid {
		verification.FullName = result.FullName.String
	}

	return verification, nil
}

// FindVerificationByUser retrieves a verification by user ID and type
func (r *postgresVerificationRepository) FindVerificationByUser(ctx context.Context, userID uuid.UUID, verificationType string) (*models.UserVerification, error) {
	query := `
		SELECT 
			id, user_id, code, target, target_type, counter,
			created_date, last_updated, remote_ip_address, is_verified,
			hashed_password, expires_at, used, full_name
		FROM verifications 
		WHERE user_id = $1 AND target_type = $2 AND used = FALSE
		ORDER BY created_date DESC
		LIMIT 1`

	var result struct {
		ID             uuid.UUID   `db:"id"`
		UserID         *uuid.UUID  `db:"user_id"`
		Code           string      `db:"code"`
		Target         string      `db:"target"`
		TargetType     string      `db:"target_type"`
		Counter        int64       `db:"counter"`
		CreatedDate    int64       `db:"created_date"`
		LastUpdated    int64       `db:"last_updated"`
		RemoteIPAddr   sql.NullString `db:"remote_ip_address"`
		IsVerified     bool        `db:"is_verified"`
		HashedPassword []byte      `db:"hashed_password"`
		ExpiresAt      int64       `db:"expires_at"`
		Used           bool        `db:"used"`
		FullName       sql.NullString `db:"full_name"`
	}

	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &result, query, userID, verificationType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("verification not found")
		}
		return nil, fmt.Errorf("failed to find verification by user: %w", err)
	}

	verification := &models.UserVerification{
		ObjectId:        result.ID,
		Code:            result.Code,
		Target:          result.Target,
		TargetType:      result.TargetType,
		Counter:         result.Counter,
		CreatedDate:     result.CreatedDate,
		LastUpdated:     result.LastUpdated,
		IsVerified:      result.IsVerified,
		HashedPassword:  result.HashedPassword,
		ExpiresAt:       result.ExpiresAt,
		Used:            result.Used,
	}

	if result.UserID != nil {
		verification.UserId = *result.UserID
	}
	if result.RemoteIPAddr.Valid {
		verification.RemoteIpAddress = result.RemoteIPAddr.String
	}
	if result.FullName.Valid {
		verification.FullName = result.FullName.String
	}

	return verification, nil
}

// FindVerificationByTarget retrieves a verification by target (email/phone) and type
func (r *postgresVerificationRepository) FindVerificationByTarget(ctx context.Context, target string, verificationType string) (*models.UserVerification, error) {
	query := `
		SELECT 
			id, user_id, code, target, target_type, counter,
			created_date, last_updated, remote_ip_address, is_verified,
			hashed_password, expires_at, used, full_name
		FROM verifications 
		WHERE target = $1 AND target_type = $2 AND used = FALSE
		ORDER BY created_date DESC
		LIMIT 1`

	var result struct {
		ID             uuid.UUID   `db:"id"`
		UserID         *uuid.UUID  `db:"user_id"`
		Code           string      `db:"code"`
		Target         string      `db:"target"`
		TargetType     string      `db:"target_type"`
		Counter        int64       `db:"counter"`
		CreatedDate    int64       `db:"created_date"`
		LastUpdated    int64       `db:"last_updated"`
		RemoteIPAddr   sql.NullString `db:"remote_ip_address"`
		IsVerified     bool        `db:"is_verified"`
		HashedPassword []byte      `db:"hashed_password"`
		ExpiresAt      int64       `db:"expires_at"`
		Used           bool        `db:"used"`
		FullName       sql.NullString `db:"full_name"`
	}

	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &result, query, target, verificationType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("verification not found")
		}
		return nil, fmt.Errorf("failed to find verification by target: %w", err)
	}

	verification := &models.UserVerification{
		ObjectId:        result.ID,
		Code:            result.Code,
		Target:          result.Target,
		TargetType:      result.TargetType,
		Counter:         result.Counter,
		CreatedDate:     result.CreatedDate,
		LastUpdated:     result.LastUpdated,
		IsVerified:      result.IsVerified,
		HashedPassword:  result.HashedPassword,
		ExpiresAt:       result.ExpiresAt,
		Used:            result.Used,
	}

	if result.UserID != nil {
		verification.UserId = *result.UserID
	}
	if result.RemoteIPAddr.Valid {
		verification.RemoteIpAddress = result.RemoteIPAddr.String
	}
	if result.FullName.Valid {
		verification.FullName = result.FullName.String
	}

	return verification, nil
}

// FindByHashedPassword retrieves a verification by hashed password (for secure reset tokens)
func (r *postgresVerificationRepository) FindByHashedPassword(ctx context.Context, hashedPassword string) (*models.UserVerification, error) {
	// The hashedPassword is stored as []byte(hexString) in bytea column
	// Compare by converting stored bytea to hex string and comparing with input hex string
	log.Printf("[FindByHashedPassword] Looking for hash: %s (length: %d)", hashedPassword, len(hashedPassword))
	
	// The stored hashed_password is 32 bytes (decoded hash bytes)
	// Compare by encoding bytea to hex in SQL and matching with input hex string
	// This avoids N+1 queries and handles the comparison in the database
	executor := r.getExecutor(ctx)
	log.Printf("[FindByHashedPassword] Looking for hash: %s", hashedPassword)
	
	// Decode the hex string to bytes for bytea comparison
	hashedPasswordBytes, err := hex.DecodeString(hashedPassword)
	if err != nil {
		return nil, fmt.Errorf("invalid hashed password format: %w", err)
	}
	
	query := `
		SELECT 
			id, user_id, code, target, target_type, counter,
			created_date, last_updated, remote_ip_address, is_verified,
			hashed_password, expires_at, used, full_name
		FROM verifications 
		WHERE target_type = 'password_reset' 
			AND used = FALSE
			AND hashed_password = $1
		ORDER BY created_date DESC
		LIMIT 1`

	var result struct {
		ID             uuid.UUID      `db:"id"`
		UserID         *uuid.UUID     `db:"user_id"`
		Code           string         `db:"code"`
		Target         string         `db:"target"`
		TargetType     string         `db:"target_type"`
		Counter        int64          `db:"counter"`
		CreatedDate    int64          `db:"created_date"`
		LastUpdated    int64          `db:"last_updated"`
		RemoteIPAddr   sql.NullString `db:"remote_ip_address"`
		IsVerified     bool           `db:"is_verified"`
		HashedPassword []byte         `db:"hashed_password"`
		ExpiresAt      int64          `db:"expires_at"`
		Used           bool           `db:"used"`
		FullName       sql.NullString `db:"full_name"`
	}

	err = sqlx.GetContext(ctx, executor, &result, query, hashedPasswordBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[FindByHashedPassword] No verification found with hash: %s", hashedPassword)
			return nil, fmt.Errorf("verification not found")
		}
		log.Printf("[FindByHashedPassword] Query error: %v", err)
		return nil, fmt.Errorf("failed to find verification by hashed password: %w", err)
	}
	log.Printf("[FindByHashedPassword] Found verification ID: %s", result.ID.String())

	verification := &models.UserVerification{
		ObjectId:        result.ID,
		Code:            result.Code,
		Target:          result.Target,
		TargetType:      result.TargetType,
		Counter:         result.Counter,
		CreatedDate:     result.CreatedDate,
		LastUpdated:     result.LastUpdated,
		IsVerified:      result.IsVerified,
		HashedPassword:  result.HashedPassword,
		ExpiresAt:       result.ExpiresAt,
		Used:            result.Used,
	}

	if result.UserID != nil {
		verification.UserId = *result.UserID
	}
	if result.RemoteIPAddr.Valid {
		verification.RemoteIpAddress = result.RemoteIPAddr.String
	}
	if result.FullName.Valid {
		verification.FullName = result.FullName.String
	}

	return verification, nil
}

// MarkVerified marks a verification as verified
func (r *postgresVerificationRepository) MarkVerified(ctx context.Context, verificationID uuid.UUID) error {
	query := `
		UPDATE verifications 
		SET is_verified = TRUE,
		    last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $1`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, verificationID)
	if err != nil {
		return fmt.Errorf("failed to mark verification as verified: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("verification not found")
	}

	return nil
}

// MarkUsed marks a verification as used (for password reset tokens)
func (r *postgresVerificationRepository) MarkUsed(ctx context.Context, verificationID uuid.UUID) error {
	query := `
		UPDATE verifications 
		SET used = TRUE,
		    last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $1`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, verificationID)
	if err != nil {
		return fmt.Errorf("failed to mark verification as used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("verification not found")
	}

	return nil
}

// UpdateVerificationCode updates the code and expiration for a verification
// This is used for resending verification emails
func (r *postgresVerificationRepository) UpdateVerificationCode(ctx context.Context, verificationID uuid.UUID, newCode string, newExpiresAt int64) error {
	query := `
		UPDATE verifications 
		SET code = $1,
		    expires_at = $2,
		    last_updated = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $3`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, newCode, newExpiresAt, verificationID)
	if err != nil {
		return fmt.Errorf("failed to update verification code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("verification not found")
	}

	return nil
}

// DeleteExpired deletes expired verification records
func (r *postgresVerificationRepository) DeleteExpired(ctx context.Context, beforeTime int64) error {
	query := `DELETE FROM verifications WHERE expires_at < $1`

	_, err := r.getExecutor(ctx).ExecContext(ctx, query, beforeTime)
	if err != nil {
		return fmt.Errorf("failed to delete expired verifications: %w", err)
	}

	return nil
}

