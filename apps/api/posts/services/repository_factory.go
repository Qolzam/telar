// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/posts/repository"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// extractSchemaFromDSN extracts the schema name from a PostgreSQL DSN string
// Returns empty string if no schema is specified (defaults to public schema)
func extractSchemaFromDSN(dsn string) string {
	if dsn == "" {
		return ""
	}

	// Parse DSN as URL
	parsedURL, err := url.Parse(dsn)
	if err != nil {
		// If DSN is not a URL, try to extract search_path from query string manually
		if strings.Contains(dsn, "search_path=") {
			parts := strings.Split(dsn, "search_path=")
			if len(parts) > 1 {
				schema := strings.Split(parts[1], "&")[0]
				schema = strings.Split(schema, "?")[0]
				return strings.TrimSpace(schema)
			}
		}
		return ""
	}

	// Extract search_path from query parameters
	query := parsedURL.Query()
	if schema := query.Get("search_path"); schema != "" {
		return schema
	}

	return ""
}

// NewPostRepositoryFromConfig creates a PostRepository from platform config
// This is a helper function for wiring up the new domain-specific repository
func NewPostRepositoryFromConfig(ctx context.Context, cfg *platformconfig.Config) (repository.PostRepository, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.Database.Type != "postgresql" {
		return nil, fmt.Errorf("only PostgreSQL is supported for posts repository")
	}

	pgConfig := &dbi.PostgreSQLConfig{
		Host:               cfg.Database.Postgres.Host,
		Port:               cfg.Database.Postgres.Port,
		Username:           cfg.Database.Postgres.Username,
		Password:           cfg.Database.Postgres.Password,
		SSLMode:            cfg.Database.Postgres.SSLMode,
		MaxOpenConnections: cfg.Database.Postgres.MaxOpenConns,
		MaxIdleConnections: cfg.Database.Postgres.MaxIdleConns,
		MaxLifetime:        int(cfg.Database.Postgres.ConnMaxLifetime.Seconds()),
		ConnectTimeout:     10,
		// Schema is extracted from DSN or defaults to empty (public schema)
		// For production, schema should be set via environment variable or DSN
		Schema: extractSchemaFromDSN(cfg.Database.Postgres.DSN),
	}

	client, err := postgres.NewClient(ctx, pgConfig, cfg.Database.Postgres.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client: %w", err)
	}

	// Apply migration if schema is specified
	if pgConfig.Schema != "" {
		if err := repository.ApplyPostsMigration(ctx, client, pgConfig.Schema); err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to apply posts migration: %w", err)
		}
	}

	// Use schema-aware constructor to ensure transactions set search_path correctly
	return repository.NewPostgresRepositoryWithSchema(client, pgConfig.Schema), nil
}

