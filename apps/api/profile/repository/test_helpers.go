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

// ApplyProfilesMigration applies the profiles table migration to the given client
// This is used in tests to set up the schema before running repository tests
func ApplyProfilesMigration(ctx context.Context, client *postgres.Client, schema string) error {
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
		CREATE TABLE IF NOT EXISTS profiles (
			user_id UUID PRIMARY KEY,
			full_name VARCHAR(255),
			social_name VARCHAR(255),
			email VARCHAR(255),
			avatar VARCHAR(512),
			banner VARCHAR(512),
			tagline VARCHAR(500),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_seen BIGINT DEFAULT 0,
			birthday BIGINT DEFAULT 0,
			web_url VARCHAR(512),
			company_name VARCHAR(255),
			country VARCHAR(100),
			address TEXT,
			phone VARCHAR(50),
			vote_count BIGINT DEFAULT 0,
			share_count BIGINT DEFAULT 0,
			follow_count BIGINT DEFAULT 0,
			follower_count BIGINT DEFAULT 0,
			post_count BIGINT DEFAULT 0,
			facebook_id VARCHAR(255),
			instagram_id VARCHAR(255),
			twitter_id VARCHAR(255),
			linkedin_id VARCHAR(255),
			access_user_list TEXT[],
			permission VARCHAR(50) DEFAULT 'Public'
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_social_name ON profiles(social_name) WHERE social_name IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles(email) WHERE email IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON profiles(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_profiles_created_date ON profiles(created_date DESC);
	`

	_, err := client.DB().ExecContext(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to apply profiles migration: %w", err)
	}

	return nil
}

// NewPostgresProfileRepositoryForTest creates a PostgresProfileRepository for testing
// It applies the migration and sets up the schema
func NewPostgresProfileRepositoryForTest(ctx context.Context, iso *testutil.IsolatedTest) (ProfileRepository, error) {
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
	if err := ApplyProfilesMigration(ctx, client, iso.LegacyConfig.PGSchema); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to apply migration: %w", err)
	}

	return NewPostgresProfileRepository(client), nil
}

