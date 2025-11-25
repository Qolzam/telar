// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"
	"fmt"

	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/testutil"
)

// ApplyPostsMigration applies the posts table migration to the given client
// This is used in tests to set up the schema before running repository tests
func ApplyPostsMigration(ctx context.Context, client *postgres.Client, schema string) error {
	// Set search_path to the schema
	if schema != "" {
		setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, schema)
		_, err := client.DB().ExecContext(ctx, setSearchPathSQL)
		if err != nil {
			return fmt.Errorf("failed to set search_path: %w", err)
		}
	}

	// Apply the migration SQL
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS posts (
			id UUID PRIMARY KEY,
			owner_user_id UUID NOT NULL,
			post_type_id INT NOT NULL,
			body TEXT,
			score BIGINT DEFAULT 0,
			view_count BIGINT DEFAULT 0,
			comment_count BIGINT DEFAULT 0,
			is_deleted BOOLEAN DEFAULT FALSE,
			deleted_date BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			tags TEXT[],
			url_key VARCHAR(255),
			owner_display_name VARCHAR(255),
			owner_avatar VARCHAR(512),
			image VARCHAR(512),
			image_full_path VARCHAR(512),
			video VARCHAR(512),
			thumbnail VARCHAR(512),
			disable_comments BOOLEAN DEFAULT FALSE,
			disable_sharing BOOLEAN DEFAULT FALSE,
			permission VARCHAR(50) DEFAULT 'Public',
			version VARCHAR(50),
			metadata JSONB DEFAULT '{}'::jsonb
		);
		CREATE INDEX IF NOT EXISTS idx_posts_owner ON posts(owner_user_id);
		CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_posts_created_date ON posts(created_date DESC);
		CREATE INDEX IF NOT EXISTS idx_posts_tags ON posts USING GIN(tags);
		CREATE INDEX IF NOT EXISTS idx_posts_post_type ON posts(post_type_id);
		CREATE INDEX IF NOT EXISTS idx_posts_deleted ON posts(is_deleted) WHERE is_deleted = FALSE;
		CREATE INDEX IF NOT EXISTS idx_posts_url_key ON posts(url_key) WHERE url_key IS NOT NULL;
	`

	_, err := client.DB().ExecContext(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to apply posts migration: %w", err)
	}

	return nil
}

// NewPostgresRepositoryForTest creates a PostgresRepository for testing
// It applies the migration and sets up the schema
func NewPostgresRepositoryForTest(ctx context.Context, iso *testutil.IsolatedTest) (PostRepository, error) {
	// Create postgres client
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	
	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client: %w", err)
	}

	// Create schema if it doesn't exist
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, schemaSQL)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Apply migration
	if err := ApplyPostsMigration(ctx, client, iso.LegacyConfig.PGSchema); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to apply migration: %w", err)
	}

	return NewPostgresRepository(client), nil
}

