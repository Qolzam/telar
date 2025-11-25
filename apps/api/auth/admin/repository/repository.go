// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/admin/models"
)

// AdminRepository defines the interface for admin-specific database operations
// This handles admin logs, invitations, and system statistics
type AdminRepository interface {
	// LogAction records an admin action in the audit trail
	LogAction(ctx context.Context, log *models.AdminLog) error

	// GetAdminLogs retrieves admin logs with filtering and pagination
	GetAdminLogs(ctx context.Context, filter AdminLogFilter, limit, offset int) ([]*models.AdminLog, error)

	// CountAdminLogs counts admin logs matching the filter
	CountAdminLogs(ctx context.Context, filter AdminLogFilter) (int64, error)

	// CreateInvitation creates a new user invitation
	CreateInvitation(ctx context.Context, invitation *models.Invitation) error

	// FindInvitationByCode retrieves an invitation by its code
	FindInvitationByCode(ctx context.Context, code string) (*models.Invitation, error)

	// FindInvitationByEmail retrieves an invitation by email
	FindInvitationByEmail(ctx context.Context, email string) (*models.Invitation, error)

	// MarkInvitationUsed marks an invitation as used
	MarkInvitationUsed(ctx context.Context, invitationID uuid.UUID) error

	// DeleteExpiredInvitations deletes expired invitations
	DeleteExpiredInvitations(ctx context.Context, beforeTime int64) error

	// GetSystemStats retrieves system statistics (user counts, post counts, etc.)
	// This uses optimized SQL queries with JOINs and aggregations
	GetSystemStats(ctx context.Context) (*models.SystemStats, error)
}

// AdminLogFilter defines filtering criteria for admin logs
type AdminLogFilter struct {
	AdminID    *uuid.UUID
	Action     *string
	TargetType *string
	TargetID   *uuid.UUID
	FromDate   *int64
	ToDate     *int64
}

