package repository

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/ai-engine/internal/models"
)

type TenantRepository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
}

type AppRepository interface {
	Create(ctx context.Context, app *models.App) error
	FindByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.App, error)
	// Helper to find by raw key for validation (usually requires fetching all active apps and comparing bcrypt, or using a prefix search optimization)
	FindAndValidateKey(ctx context.Context, rawKey string) (*models.App, error)
}
