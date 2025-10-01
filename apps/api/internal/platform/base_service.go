// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package platform

import (
	"context"
	"fmt"

	"github.com/qolzam/telar/apps/api/internal/database/factory"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// BaseService provides common functionality for all services
type BaseService struct {
	Repository interfaces.Repository
	config     *ServiceConfig
}

// ServiceConfig represents service configuration
type ServiceConfig struct {
	DatabaseType       string
	ConnectionString   string
	DatabaseName       string
	MongoConfig        *interfaces.MongoDBConfig
	PostgreSQLConfig   *interfaces.PostgreSQLConfig
	EnableTransactions bool
	MaxRetries         int
}

// NewBaseService creates a new base service instance from platform config
func NewBaseService(ctx context.Context, cfg *platformconfig.Config) (*BaseService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("platform configuration is required")
	}

	// Create repository factory from platform config
	repositoryFactory := factory.NewRepositoryFactoryFromPlatformConfig(cfg.Database)

	// Validate configuration
	if err := repositoryFactory.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid repository configuration: %w", err)
	}

	// Create repository
	repository, err := repositoryFactory.CreateRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Convert platform config to legacy ServiceConfig for backward compatibility
	serviceConfig := &ServiceConfig{
		DatabaseType:       cfg.Database.Type,
		DatabaseName:       getDatabaseNameFromConfig(cfg),
		EnableTransactions: false, // Default to false for now
		MaxRetries:         3,     // Default retry count
	}

	return &BaseService{
		Repository: repository,
		config:     serviceConfig,
	}, nil
}

// getDatabaseNameFromConfig extracts database name from platform config
func getDatabaseNameFromConfig(cfg *platformconfig.Config) string {
	switch cfg.Database.Type {
	case interfaces.DatabaseTypeMongoDB:
		return cfg.Database.MongoDB.Database
	case interfaces.DatabaseTypePostgreSQL:
		return cfg.Database.Postgres.Database
	default:
		return "telar_social" // Default database name
	}
}

// NewBaseServiceWithRepo creates a BaseService with an existing repository
// This is used for tests to inject isolated repositories
func NewBaseServiceWithRepo(repo interfaces.Repository, config *ServiceConfig) *BaseService {
	return &BaseService{
		Repository: repo,
		config:     config,
	}
}

 // CreateServiceConfigFromEnv creates service configuration from environment variables
func CreateServiceConfigFromEnv() (*ServiceConfig, error) {
	// This would read from environment variables
	// For now, return a default configuration

	config := &ServiceConfig{
		DatabaseType:       interfaces.DatabaseTypeMongoDB, // Default
		DatabaseName:       "telar_social",
		EnableTransactions: false,
		MaxRetries:         3,
	}

	// Read database type from environment
	// dbType := os.Getenv("DATABASE_TYPE")
	// if dbType != "" {
	//     config.DatabaseType = dbType
	// }

	return config, nil
}

// WithTransaction executes a function within a transaction if supported
func (s *BaseService) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if !s.config.EnableTransactions {
		// Execute without transaction
		return fn(ctx)
	}

	return s.Repository.WithTransaction(ctx, fn)
}

// ExecuteWithRetry executes a function with retry logic
func (s *BaseService) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for i := 0; i <= s.config.MaxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < s.config.MaxRetries {
				// Add exponential backoff logic here if needed
				continue
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", s.config.MaxRetries, lastErr)
}

// HealthCheck performs a health check on the repository
func (s *BaseService) HealthCheck(ctx context.Context) error {
	result := <-s.Repository.Ping(ctx)
	return result
}

// Close closes the service and its resources
func (s *BaseService) Close() error {
	if s.Repository != nil {
		return s.Repository.Close()
	}
	return nil
}

// GetDatabaseType returns the configured database type
func (s *BaseService) GetDatabaseType() string {
	return s.config.DatabaseType
}

// IsTransactionSupported returns whether transactions are enabled
func (s *BaseService) IsTransactionSupported() bool {
	return s.config.EnableTransactions
}

// GetConfig returns the service configuration
func (s *BaseService) GetConfig() *ServiceConfig {
	return s.config
}

// ServiceBuilder provides a fluent interface for building services
type ServiceBuilder struct {
	config *ServiceConfig
}

// NewServiceBuilder creates a new service builder
func NewServiceBuilder() *ServiceBuilder {
	return &ServiceBuilder{
		config: &ServiceConfig{
			EnableTransactions: false,
			MaxRetries:         3,
		},
	}
}

// WithMongoDB configures the service to use MongoDB
func (b *ServiceBuilder) WithMongoDB(config *interfaces.MongoDBConfig, databaseName string) *ServiceBuilder {
	b.config.DatabaseType = interfaces.DatabaseTypeMongoDB
	b.config.MongoConfig = config
	b.config.DatabaseName = databaseName
	return b
}

// WithPostgreSQL configures the service to use PostgreSQL
func (b *ServiceBuilder) WithPostgreSQL(config *interfaces.PostgreSQLConfig, databaseName string) *ServiceBuilder {
	b.config.DatabaseType = interfaces.DatabaseTypePostgreSQL
	b.config.PostgreSQLConfig = config
	b.config.DatabaseName = databaseName
	return b
}

// WithTransactions enables transaction support
func (b *ServiceBuilder) WithTransactions() *ServiceBuilder {
	b.config.EnableTransactions = true
	return b
}

// WithRetries sets the maximum number of retries
func (b *ServiceBuilder) WithRetries(maxRetries int) *ServiceBuilder {
	b.config.MaxRetries = maxRetries
	return b
}

// Build creates the service instance
func (b *ServiceBuilder) Build(ctx context.Context) (*BaseService, error) {
	// Create repository factory
	factoryConfig := &interfaces.RepositoryConfig{
		DatabaseType:     b.config.DatabaseType,
		ConnectionString: b.config.ConnectionString,
		DatabaseName:     b.config.DatabaseName,
		MongoConfig:      b.config.MongoConfig,
		PostgresConfig:   b.config.PostgreSQLConfig,
	}

	repositoryFactory := factory.NewRepositoryFactory(factoryConfig)

	if err := repositoryFactory.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid repository configuration: %w", err)
	}

	repository, err := repositoryFactory.CreateRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return &BaseService{
		Repository: repository,
		config:     b.config,
	}, nil
}

// Common utility functions for services

// ExtractPaginationFromContext extracts pagination parameters from context
func ExtractPaginationFromContext(ctx context.Context) (limit int64, skip int64) {
	if val := ctx.Value("limit"); val != nil {
		if l, ok := val.(int64); ok {
			limit = l
		}
	}
	if val := ctx.Value("skip"); val != nil {
		if s, ok := val.(int64); ok {
			skip = s
		}
	}

	// Set defaults
	if limit <= 0 {
		limit = 20
	}
	if skip < 0 {
		skip = 0
	}

	return limit, skip
}

// ExtractSortFromContext extracts sort parameters from context
func ExtractSortFromContext(ctx context.Context) map[string]int {
	if val := ctx.Value("sort"); val != nil {
		if sort, ok := val.(map[string]int); ok {
			return sort
		}
	}

	// Default sort by creation date, newest first
	return map[string]int{"created_date": -1}
}

// BuildFindOptions creates FindOptions from context parameters
func BuildFindOptions(ctx context.Context) *interfaces.FindOptions {
	limit, skip := ExtractPaginationFromContext(ctx)
	sort := ExtractSortFromContext(ctx)

	return &interfaces.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
}
