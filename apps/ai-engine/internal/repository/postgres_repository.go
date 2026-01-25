package repository

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/qolzam/telar/apps/ai-engine/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type PostgresTenantRepository struct {
	db *sqlx.DB
}

func NewPostgresTenantRepository(db *sqlx.DB) *PostgresTenantRepository {
	return &PostgresTenantRepository{db: db}
}

func (r *PostgresTenantRepository) Create(ctx context.Context, t *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, owner_email, plan_tier, monthly_quota, encrypted_settings)
		VALUES (:id, :name, :owner_email, :plan_tier, :monthly_quota, :encrypted_settings)
	`
	_, err := r.db.NamedExecContext(ctx, query, t)
	return err
}

func (r *PostgresTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	query := `SELECT * FROM tenants WHERE id = $1`
	err := r.db.GetContext(ctx, &tenant, query, id)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

type PostgresAppRepository struct {
	db *sqlx.DB
}

func NewPostgresAppRepository(db *sqlx.DB) *PostgresAppRepository {
	return &PostgresAppRepository{db: db}
}

func (r *PostgresAppRepository) Create(ctx context.Context, app *models.App) error {
	query := `
		INSERT INTO apps (id, tenant_id, name, source_type, api_key_hash, webhook_url, webhook_secret, metadata)
		VALUES (:id, :tenant_id, :name, :source_type, :api_key_hash, :webhook_url, :webhook_secret, :metadata)
	`
	_, err := r.db.NamedExecContext(ctx, query, app)
	return err
}

func (r *PostgresAppRepository) FindByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.App, error) {
	var app models.App
	query := `SELECT * FROM apps WHERE api_key_hash = $1 AND is_active = true`
	err := r.db.GetContext(ctx, &app, query, apiKeyHash)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// FindAndValidateKey finds an app by validating the raw API key against stored bcrypt hashes
// Optimization: In a real system, you'd store a Key ID or Prefix to avoid scanning the whole table.
// For this phase, we will fetch all active apps and compare (or assume the ID is sent with the key).
// Better approach: API Key format `gk_UUID_RANDOM`. Use UUID to lookup, then compare hash.
func (r *PostgresAppRepository) FindAndValidateKey(ctx context.Context, rawKey string) (*models.App, error) {
	// For MVP: Fetch all active apps (CACHE THIS IN MEMORY in production!)
	var apps []models.App
	query := `SELECT * FROM apps WHERE is_active = true`
	if err := r.db.SelectContext(ctx, &apps, query); err != nil {
		return nil, err
	}

	for _, app := range apps {
		if bcrypt.CompareHashAndPassword([]byte(app.APIKeyHash), []byte(rawKey)) == nil {
			return &app, nil
		}
	}
	return nil, fmt.Errorf("invalid api key")
}
