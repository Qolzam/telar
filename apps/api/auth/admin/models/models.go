// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package models

import (
	"time"

	"github.com/gofrs/uuid"
)

// AdminLog represents an admin action audit log entry
type AdminLog struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	AdminID    uuid.UUID              `json:"adminId" db:"admin_id"`
	Action     string                 `json:"action" db:"action"`
	TargetType *string                `json:"targetType,omitempty" db:"target_type"`
	TargetID   *uuid.UUID             `json:"targetId,omitempty" db:"target_id"`
	Details    map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedAt  time.Time              `json:"createdAt" db:"created_at"`
	CreatedDate int64                 `json:"createdDate" db:"created_date"`
}

// Invitation represents a user invitation sent by an admin
type Invitation struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Email       string    `json:"email" db:"email"`
	InvitedBy   uuid.UUID `json:"invitedBy" db:"invited_by"`
	Role        string    `json:"role" db:"role"`
	Code        string    `json:"code" db:"code"`
	ExpiresAt   time.Time `json:"expiresAt" db:"expires_at"`
	Used        bool      `json:"used" db:"used"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	CreatedDate int64     `json:"createdDate" db:"created_date"`
}

// SystemStats represents aggregated system statistics
type SystemStats struct {
	TotalUsers      int64 `json:"totalUsers"`
	TotalPosts      int64 `json:"totalPosts"`
	TotalComments   int64 `json:"totalComments"`
	TotalAdmins     int64 `json:"totalAdmins"`
	ActiveUsers     int64 `json:"activeUsers"` // Users active in last 30 days
	PendingInvitations int64 `json:"pendingInvitations"`
}

