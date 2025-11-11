// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package factory

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgresql"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// RepositoryFactory creates repository instances based on configuration
type RepositoryFactory struct {
	config *interfaces.RepositoryConfig
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(config *interfaces.RepositoryConfig) *RepositoryFactory {
	return &RepositoryFactory{
		config: config,
	}
}

// NewRepositoryFactoryFromPlatformConfig creates a new repository factory from platform config
func NewRepositoryFactoryFromPlatformConfig(dbConfig platformconfig.DatabaseConfig) *RepositoryFactory {
	config := &interfaces.RepositoryConfig{
		DatabaseType: dbConfig.Type,
		DatabaseName: getDatabaseName(dbConfig),
	}

	// Set database-specific configuration
	switch dbConfig.Type {
	case interfaces.DatabaseTypePostgreSQL:
		config.PostgresConfig = &interfaces.PostgreSQLConfig{
			Host:               dbConfig.Postgres.Host,
			Port:               dbConfig.Postgres.Port,
			Username:           dbConfig.Postgres.Username,
			Password:           dbConfig.Postgres.Password,
			Database:           dbConfig.Postgres.Database,
			SSLMode:            dbConfig.Postgres.SSLMode,
			MaxOpenConnections: dbConfig.Postgres.MaxOpenConns,
			MaxIdleConnections: dbConfig.Postgres.MaxIdleConns,
			MaxLifetime:        int(dbConfig.Postgres.ConnMaxLifetime.Seconds()),
			ConnectTimeout:     10, // Default 10 seconds
		}

		if dsn := dbConfig.Postgres.DSN; dsn != "" {
			if parsed, err := url.Parse(dsn); err == nil {
				if host := parsed.Hostname(); host != "" {
					config.PostgresConfig.Host = host
				}
				if portStr := parsed.Port(); portStr != "" {
					if port, err := strconv.Atoi(portStr); err == nil {
						config.PostgresConfig.Port = port
					}
				}
				if user := parsed.User; user != nil {
					config.PostgresConfig.Username = user.Username()
					if pass, ok := user.Password(); ok {
						config.PostgresConfig.Password = pass
					}
				}
				if dbName := strings.TrimPrefix(parsed.Path, "/"); dbName != "" {
					config.PostgresConfig.Database = dbName
					config.DatabaseName = dbName
				}
				query := parsed.Query()
				if searchPath := query.Get("search_path"); searchPath != "" {
					config.PostgresConfig.Schema = searchPath
				}
				if sslMode := query.Get("sslmode"); sslMode != "" {
					config.PostgresConfig.SSLMode = sslMode
				}
			}
		}

		if config.PostgresConfig.Port == 0 {
			config.PostgresConfig.Port = 5432
		}

		if config.PostgresConfig.Schema == "" {
			config.PostgresConfig.Schema = "public"
		}
	}

	return &RepositoryFactory{
		config: config,
	}
}

// getDatabaseName extracts the database name from platform config
func getDatabaseName(dbConfig platformconfig.DatabaseConfig) string {
	switch dbConfig.Type {
	case interfaces.DatabaseTypePostgreSQL:
		return dbConfig.Postgres.Database
	default:
		return "telar_social" // Default database name
	}
}

// CreateRepository creates a repository instance based on the configured database type
func (f *RepositoryFactory) CreateRepository(ctx context.Context) (interfaces.Repository, error) {
	switch f.config.DatabaseType {
	case interfaces.DatabaseTypePostgreSQL:
		return f.createPostgreSQLRepository(ctx)
	default:
		return nil, fmt.Errorf("unsupported database type: %s (only PostgreSQL is supported)", f.config.DatabaseType)
	}
}

// createPostgreSQLRepository creates a PostgreSQL repository instance
func (f *RepositoryFactory) createPostgreSQLRepository(ctx context.Context) (interfaces.Repository, error) {
	if f.config.PostgresConfig == nil {
		return nil, fmt.Errorf("PostgreSQL configuration is missing")
	}

	pgRepo, err := postgresql.NewPostgreSQLRepository(ctx, f.config.PostgresConfig, f.config.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL repository: %w", err)
	}

	return pgRepo, nil
}

// CreateRepositoryFromConnectionString creates a repository from a connection string
func CreateRepositoryFromConnectionString(ctx context.Context, databaseType, connectionString, databaseName string) (interfaces.Repository, error) {
	config := &interfaces.RepositoryConfig{
		DatabaseType:     databaseType,
		ConnectionString: connectionString,
		DatabaseName:     databaseName,
	}

	switch databaseType {
	case interfaces.DatabaseTypePostgreSQL:
		config.PostgresConfig = &interfaces.PostgreSQLConfig{
			// Parse connection string to extract PostgreSQL config
			Host:    "localhost",
			Port:    5432,
			SSLMode: "disable",
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", databaseType)
	}

	factory := NewRepositoryFactory(config)
	return factory.CreateRepository(ctx)
}

// CreateRepositoryFromConfig creates a repository from environment/config
func CreateRepositoryFromConfig(ctx context.Context, databaseType string, config interface{}) (interfaces.Repository, error) {
	switch databaseType {
	case interfaces.DatabaseTypePostgreSQL:
		pgConfig, ok := config.(*interfaces.PostgreSQLConfig)
		if !ok {
			return nil, fmt.Errorf("invalid PostgreSQL configuration type")
		}

		factoryConfig := &interfaces.RepositoryConfig{
			DatabaseType:   databaseType,
			PostgresConfig: pgConfig,
		}

		factory := NewRepositoryFactory(factoryConfig)
		return factory.CreateRepository(ctx)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", databaseType)
	}
}

// ValidateConfig validates the repository configuration
func (f *RepositoryFactory) ValidateConfig() error {
	if f.config == nil {
		return fmt.Errorf("repository configuration is nil")
	}

	if f.config.DatabaseType == "" {
		return fmt.Errorf("database type is required")
	}

	switch f.config.DatabaseType {
	case interfaces.DatabaseTypePostgreSQL:
		if f.config.PostgresConfig == nil {
			return fmt.Errorf("PostgreSQL configuration is required")
		}
		return f.validatePostgreSQLConfig()

	default:
		return fmt.Errorf("unsupported database type: %s", f.config.DatabaseType)
	}
}

// validatePostgreSQLConfig validates PostgreSQL configuration
func (f *RepositoryFactory) validatePostgreSQLConfig() error {
	config := f.config.PostgresConfig

	if config.Host == "" {
		return fmt.Errorf("PostgreSQL host is required")
	}

	if config.Port <= 0 {
		config.Port = 5432 // Default PostgreSQL port
	}

	if config.SSLMode == "" {
		config.SSLMode = "disable" // Default SSL mode
	}

	if config.MaxOpenConnections <= 0 {
		config.MaxOpenConnections = 50 // Increased default max open connections
	}

	if config.MaxIdleConnections <= 0 {
		config.MaxIdleConnections = 10 // Increased default max idle connections
	}

	if config.MaxLifetime <= 0 {
		config.MaxLifetime = 300 // Default connection lifetime in seconds
	}

	if config.ConnectTimeout <= 0 {
		config.ConnectTimeout = 10 // Default 10 seconds
	}

	return nil
}
