package management

import "context"

// UserManagement defines management operations for admin actions
type UserManagement interface {
	UpdateUserRole(ctx context.Context, userID string, newRole string) error
	UpdateUserStatus(ctx context.Context, userID string, newStatus string) error // e.g., "active", "banned"
}






