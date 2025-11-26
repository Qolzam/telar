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

// ApplyCommentsMigration applies the comments table migration to the test database
// Dependencies: Requires posts and user_auths tables to exist first
func ApplyCommentsMigration(ctx context.Context, iso *testutil.IsolatedTest) error {
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema

	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	if err != nil {
		return fmt.Errorf("failed to create postgres client: %w", err)
	}
	defer client.Close()

	// Create schema if it doesn't exist
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Set search_path to the isolated schema
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, setSearchPathSQL)
	if err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// Apply auth migration first (required for user_auths FK)
	authMigrationSQL := `
		CREATE TABLE IF NOT EXISTS user_auths (
			id UUID PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash BYTEA NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
	`
	_, err = client.DB().ExecContext(ctx, authMigrationSQL)
	if err != nil {
		return fmt.Errorf("failed to apply auth migration: %w", err)
	}

	// Apply comments migration (depends on posts and user_auths)
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY,
			post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			owner_user_id UUID NOT NULL REFERENCES user_auths(id) ON DELETE CASCADE,
			parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
			text TEXT NOT NULL,
			score BIGINT DEFAULT 0,
			owner_display_name VARCHAR(255),
			owner_avatar VARCHAR(512),
			is_deleted BOOLEAN DEFAULT FALSE,
			deleted_date BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);

		CREATE INDEX IF NOT EXISTS idx_comments_post ON comments(post_id);
		CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_comment_id) WHERE parent_comment_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_comments_owner ON comments(owner_user_id);
		CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_comments_created_date ON comments(created_date DESC);
		CREATE INDEX IF NOT EXISTS idx_comments_deleted ON comments(is_deleted) WHERE is_deleted = FALSE;
		CREATE INDEX IF NOT EXISTS idx_comments_post_active ON comments(post_id, created_date DESC) WHERE is_deleted = FALSE;
	`

	_, err = client.DB().ExecContext(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to apply comments migration: %w", err)
	}

	return nil
}

// NewPostgresCommentRepositoryForTest creates a CommentRepository for testing
// It applies the necessary migrations (posts, auth, comments) and returns a configured repository
// NOTE: This function assumes posts migration has already been applied (posts table must exist)
func NewPostgresCommentRepositoryForTest(ctx context.Context, iso *testutil.IsolatedTest) (CommentRepository, error) {
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema

	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client: %w", err)
	}

	// Apply comments migration (includes auth migration, assumes posts already exists)
	if err := ApplyCommentsMigration(ctx, iso); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to apply comments migration: %w", err)
	}

	return NewPostgresCommentRepositoryWithSchema(client, iso.LegacyConfig.PGSchema), nil
}

