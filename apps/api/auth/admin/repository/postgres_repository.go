// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/qolzam/telar/apps/api/auth/admin/models"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
)

// postgresAdminRepository implements AdminRepository using PostgreSQL
type postgresAdminRepository struct {
	client *postgres.Client
}

// NewPostgresAdminRepository creates a new PostgreSQL admin repository
func NewPostgresAdminRepository(client *postgres.Client) AdminRepository {
	return &postgresAdminRepository{
		client: client,
	}
}

// getExecutor returns either the transaction from context or the DB connection
func (r *postgresAdminRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	// Check for transaction in context (shared key for cross-package transactions)
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.client.DB()
}

// LogAction records an admin action in the audit trail
func (r *postgresAdminRepository) LogAction(ctx context.Context, log *models.AdminLog) error {
	query := `
		INSERT INTO admin_logs (id, admin_id, action, target_type, target_id, details, created_at, created_date)
		VALUES (:id, :admin_id, :action, :target_type, :target_id, :details, :created_at, :created_date)
	`

	var detailsJSON []byte
	var err error
	if log.Details != nil {
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	args := map[string]interface{}{
		"id":          log.ID,
		"admin_id":    log.AdminID,
		"action":      log.Action,
		"target_type": log.TargetType,
		"target_id":   log.TargetID,
		"details":     detailsJSON,
		"created_at":  log.CreatedAt,
		"created_date": log.CreatedDate,
	}

	_, err = sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, args)
	if err != nil {
		return fmt.Errorf("failed to log admin action: %w", err)
	}

	return nil
}

// GetAdminLogs retrieves admin logs with filtering and pagination
func (r *postgresAdminRepository) GetAdminLogs(ctx context.Context, filter AdminLogFilter, limit, offset int) ([]*models.AdminLog, error) {
	query := `SELECT id, admin_id, action, target_type, target_id, details, created_at, created_date
		FROM admin_logs WHERE 1=1`
	args := map[string]interface{}{}

	if filter.AdminID != nil {
		query += ` AND admin_id = :admin_id`
		args["admin_id"] = *filter.AdminID
	}
	if filter.Action != nil {
		query += ` AND action = :action`
		args["action"] = *filter.Action
	}
	if filter.TargetType != nil {
		query += ` AND target_type = :target_type`
		args["target_type"] = *filter.TargetType
	}
	if filter.TargetID != nil {
		query += ` AND target_id = :target_id`
		args["target_id"] = *filter.TargetID
	}
	if filter.FromDate != nil {
		query += ` AND created_date >= :from_date`
		args["from_date"] = *filter.FromDate
	}
	if filter.ToDate != nil {
		query += ` AND created_date <= :to_date`
		args["to_date"] = *filter.ToDate
	}

	query += ` ORDER BY created_date DESC LIMIT :limit OFFSET :offset`
	args["limit"] = limit
	args["offset"] = offset

	rows, err := sqlx.NamedQueryContext(ctx, r.getExecutor(ctx), query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to query admin logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.AdminLog
	for rows.Next() {
		var log models.AdminLog
		var detailsJSON []byte
		var targetType sql.NullString
		var targetID sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.AdminID,
			&log.Action,
			&targetType,
			&targetID,
			&detailsJSON,
			&log.CreatedAt,
			&log.CreatedDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin log: %w", err)
		}

		if targetType.Valid {
			log.TargetType = &targetType.String
		}
		if targetID.Valid {
			if id, err := uuid.FromString(targetID.String); err == nil {
				log.TargetID = &id
			}
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

// CountAdminLogs counts admin logs matching the filter
func (r *postgresAdminRepository) CountAdminLogs(ctx context.Context, filter AdminLogFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM admin_logs WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.AdminID != nil {
		query += fmt.Sprintf(` AND admin_id = $%d`, argIndex)
		args = append(args, *filter.AdminID)
		argIndex++
	}
	if filter.Action != nil {
		query += fmt.Sprintf(` AND action = $%d`, argIndex)
		args = append(args, *filter.Action)
		argIndex++
	}
	if filter.TargetType != nil {
		query += fmt.Sprintf(` AND target_type = $%d`, argIndex)
		args = append(args, *filter.TargetType)
		argIndex++
	}
	if filter.TargetID != nil {
		query += fmt.Sprintf(` AND target_id = $%d`, argIndex)
		args = append(args, *filter.TargetID)
		argIndex++
	}
	if filter.FromDate != nil {
		query += fmt.Sprintf(` AND created_date >= $%d`, argIndex)
		args = append(args, *filter.FromDate)
		argIndex++
	}
	if filter.ToDate != nil {
		query += fmt.Sprintf(` AND created_date <= $%d`, argIndex)
		args = append(args, *filter.ToDate)
		argIndex++
	}

	var count int64
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count admin logs: %w", err)
	}

	return count, nil
}

// CreateInvitation creates a new user invitation
func (r *postgresAdminRepository) CreateInvitation(ctx context.Context, invitation *models.Invitation) error {
	query := `
		INSERT INTO invitations (id, email, invited_by, role, code, expires_at, used, created_at, created_date)
		VALUES (:id, :email, :invited_by, :role, :code, :expires_at, :used, :created_at, :created_date)
	`

	args := map[string]interface{}{
		"id":          invitation.ID,
		"email":       invitation.Email,
		"invited_by":  invitation.InvitedBy,
		"role":        invitation.Role,
		"code":        invitation.Code,
		"expires_at":  invitation.ExpiresAt,
		"used":        invitation.Used,
		"created_at":  invitation.CreatedAt,
		"created_date": invitation.CreatedDate,
	}

	_, err := sqlx.NamedExecContext(ctx, r.getExecutor(ctx), query, args)
	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				if pqErr.Constraint == "idx_invitations_email" || pqErr.Constraint == "invitations_email_key" {
					return fmt.Errorf("invitation already exists for email: %w", err)
				}
				if pqErr.Constraint == "idx_invitations_code" || pqErr.Constraint == "invitations_code_key" {
					return fmt.Errorf("invitation code already exists: %w", err)
				}
			}
		}
		return fmt.Errorf("failed to create invitation: %w", err)
	}

	return nil
}

// FindInvitationByCode retrieves an invitation by its code
func (r *postgresAdminRepository) FindInvitationByCode(ctx context.Context, code string) (*models.Invitation, error) {
	query := `SELECT id, email, invited_by, role, code, expires_at, used, created_at, created_date
		FROM invitations WHERE code = $1 AND used = FALSE`

	var invitation models.Invitation
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &invitation, query, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invitation not found")
		}
		return nil, fmt.Errorf("failed to find invitation: %w", err)
	}

	return &invitation, nil
}

// FindInvitationByEmail retrieves an invitation by email
func (r *postgresAdminRepository) FindInvitationByEmail(ctx context.Context, email string) (*models.Invitation, error) {
	query := `SELECT id, email, invited_by, role, code, expires_at, used, created_at, created_date
		FROM invitations WHERE email = $1 ORDER BY created_date DESC LIMIT 1`

	var invitation models.Invitation
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &invitation, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invitation not found")
		}
		return nil, fmt.Errorf("failed to find invitation: %w", err)
	}

	return &invitation, nil
}

// MarkInvitationUsed marks an invitation as used
func (r *postgresAdminRepository) MarkInvitationUsed(ctx context.Context, invitationID uuid.UUID) error {
	query := `UPDATE invitations SET used = TRUE WHERE id = $1`

	result, err := r.getExecutor(ctx).ExecContext(ctx, query, invitationID)
	if err != nil {
		return fmt.Errorf("failed to mark invitation as used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invitation not found")
	}

	return nil
}

// DeleteExpiredInvitations deletes expired invitations
func (r *postgresAdminRepository) DeleteExpiredInvitations(ctx context.Context, beforeTime int64) error {
	query := `DELETE FROM invitations WHERE expires_at < to_timestamp($1)`

	_, err := r.getExecutor(ctx).ExecContext(ctx, query, beforeTime)
	if err != nil {
		return fmt.Errorf("failed to delete expired invitations: %w", err)
	}

	return nil
}

// GetSystemStats retrieves system statistics using optimized SQL queries
// Note: This queries tables that may not exist yet (posts, comments) - will return 0 if tables don't exist
func (r *postgresAdminRepository) GetSystemStats(ctx context.Context) (*models.SystemStats, error) {
	// Use a single query with subqueries for efficiency
	// Use COALESCE to handle cases where tables don't exist yet
	query := `
		SELECT 
			COALESCE((SELECT COUNT(*) FROM user_auths), 0)::BIGINT as total_users,
			COALESCE((SELECT COUNT(*) FROM posts WHERE is_deleted = FALSE), 0)::BIGINT as total_posts,
			COALESCE((SELECT COUNT(*) FROM comments WHERE is_deleted = FALSE), 0)::BIGINT as total_comments,
			COALESCE((SELECT COUNT(*) FROM user_auths WHERE role = 'admin'), 0)::BIGINT as total_admins,
			COALESCE((SELECT COUNT(DISTINCT owner_user_id) FROM posts 
			 WHERE created_date > EXTRACT(EPOCH FROM NOW() - INTERVAL '30 days')::BIGINT 
			 AND is_deleted = FALSE), 0)::BIGINT as active_users,
			COALESCE((SELECT COUNT(*) FROM invitations WHERE used = FALSE AND expires_at > NOW()), 0)::BIGINT as pending_invitations
	`

	var stats models.SystemStats
	err := sqlx.GetContext(ctx, r.getExecutor(ctx), &stats, query)
	if err != nil {
		// If query fails due to missing tables, return zero stats
		// This allows the admin service to work even if posts/comments tables don't exist yet
		return &models.SystemStats{
			TotalUsers:           0,
			TotalPosts:           0,
			TotalComments:        0,
			TotalAdmins:          0,
			ActiveUsers:          0,
			PendingInvitations:   0,
		}, nil
	}

	return &stats, nil
}

