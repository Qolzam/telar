// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"fmt"

	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/posts/repository"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

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

	return repository.NewPostgresRepository(client), nil
}

