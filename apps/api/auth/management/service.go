package management

import (
	"context"

	uuid "github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/cache"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/utils"
)

// UserManagementService updates auth data (roles/status)
type UserManagementService struct {
	base         *platform.BaseService
	cacheService *cache.GenericCacheService
}

func NewUserManagementService(base *platform.BaseService, cacheSvc *cache.GenericCacheService) *UserManagementService {
	return &UserManagementService{base: base, cacheService: cacheSvc}
}

func (s *UserManagementService) UpdateUserRole(ctx context.Context, userID string, newRole string) error {
	id, err := uuid.FromString(userID)
	if err != nil {
		return err
	}
	query := &dbi.Query{
		Conditions: []dbi.Field{
			{Name: "object_id", Value: id, Operator: "=", IsJSONB: false},
		},
	}
	updates := map[string]interface{}{
		"role":        newRole,
		"lastUpdated": utils.UTCNowUnix(),
	}
	return (<-s.base.Repository.UpdateFields(ctx, "userAuth", query, updates)).Error
}

func (s *UserManagementService) UpdateUserStatus(ctx context.Context, userID string, newStatus string) error {
	id, err := uuid.FromString(userID)
	if err != nil {
		return err
	}
	query := &dbi.Query{
		Conditions: []dbi.Field{
			{Name: "object_id", Value: id, Operator: "=", IsJSONB: false},
		},
	}
	updates := map[string]interface{}{
		"status":      newStatus,
		"lastUpdated": utils.UTCNowUnix(),
	}
	err = (<-s.base.Repository.UpdateFields(ctx, "userAuth", query, updates)).Error
	if err != nil {
		return err
	}

	// Invalidate all sessions if user is banned and cache service supports it
	if newStatus == "banned" && s.cacheService != nil && s.cacheService.IsEnabled() {
		sessionKey := s.cacheService.GenerateHashKey("sessions", map[string]interface{}{"uid": userID})
		_ = s.cacheService.InvalidateKey(ctx, sessionKey)
	}
	return nil
}


