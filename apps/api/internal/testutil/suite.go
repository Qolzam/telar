package testutil

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// Suite manages shared, pooled database connections for high-performance testing.
type Suite struct {
	mu                 sync.RWMutex
	mongoConnection    dbi.Repository
	postgresConnection dbi.Repository
	initialized        bool
	config             *platformconfig.Config
}

var (
	globalSuite *Suite
	suiteOnce   sync.Once
)

// Setup initializes the global suite with shared connections. It's safe to call
func Setup(t *testing.T) *Suite {
	t.Helper()
	
	suiteOnce.Do(func() {
		globalSuite = &Suite{}

		cfg, err := platformconfig.LoadFromEnv()
		if err != nil {
			cfg = &platformconfig.Config{
				JWT: platformconfig.JWTConfig{
				},
				HMAC: platformconfig.HMACConfig{
					Secret: "test-secret",
				},
				Database: platformconfig.DatabaseConfig{
					Type: "mongodb",
					MongoDB: platformconfig.MongoDBConfig{
						Host:     "localhost",
						Port:     27017,
						Database: "telar_test",
					},
					Postgres: platformconfig.PostgreSQLConfig{
						Host:     "localhost",
						Port:     5432,
						Database: "telar_test",
					},
				},
				Server: platformconfig.ServerConfig{
					WebDomain: "http://localhost",
				},
				Email: platformconfig.EmailConfig{
					SMTPEmail:    "test@example.com",
					RefEmail:     "test@example.com",
					RefEmailPass: "test-password",
				},
			}
		}
		globalSuite.config = cfg

		if err := globalSuite.createSharedConnections(); err != nil {
			t.Logf("Warning: Not all database connections were available: %v", err)
		}

		globalSuite.initialized = true
	})
	
	if globalSuite.mongoConnection == nil && globalSuite.config.Database.MongoDB.Host != "" {
		t.Log("Mongo connection lost, attempting to reconnect...")
		_ = globalSuite.createSharedConnections()
	}
	if globalSuite.postgresConnection == nil && globalSuite.config.Database.Postgres.Host != "" {
		t.Log("Postgres connection lost, attempting to reconnect...")
		_ = globalSuite.createSharedConnections()
	}

	return globalSuite
}

// Config returns the canonical platform config for dependency injection.
func (s *Suite) Config() *platformconfig.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetTestJWTConfig provides direct access to the centralized JWT configuration for tests.
func (s *Suite) GetTestJWTConfig() platformconfig.JWTConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.JWT
}

// GenerateUniqueJWTKeys provides a new, unique ECDSA key pair for a single test.
func (s *Suite) GenerateUniqueJWTKeys(t *testing.T) (publicKeyPEM string, privateKeyPEM string) {
	t.Helper()
	return GenerateECDSAKeyPairPEM(t)
}

// createSharedConnections attempts to connect to both databases concurrently.
func (s *Suite) createSharedConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	healthChecker := NewHealthChecker(s.config)
	if err := healthChecker.ValidateTestEnvironment(ctx); err != nil {
		return fmt.Errorf("test environment validation failed: %w", err)
	}

	var wg sync.WaitGroup
	var mongoErr, pgErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		if base, err := platform.NewBaseService(ctx, s.config); err == nil {
			s.mu.Lock()
			s.mongoConnection = base.Repository
			s.mu.Unlock()
		} else {
			mongoErr = err
		}
	}()

	go func() {
		defer wg.Done()
		if base, err := platform.NewBaseService(ctx, s.config); err == nil {
			s.mu.Lock()
			s.postgresConnection = base.Repository
			s.mu.Unlock()
		} else {
			pgErr = err
		}
	}()

	wg.Wait()

	if mongoErr != nil || pgErr != nil {
		return fmt.Errorf("mongoErr: [%v], pgErr: [%v]", mongoErr, pgErr)
	}
	return nil
}

// GetMongoPool returns the shared MongoDB connection pool.
func (s *Suite) GetMongoPool() dbi.Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mongoConnection
}

// GetPostgresPool returns the shared PostgreSQL connection pool.
func (s *Suite) GetPostgresPool() dbi.Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.postgresConnection
}

// ShouldRunDatabaseTests checks if database tests should be executed.
func ShouldRunDatabaseTests() bool {
	return os.Getenv("RUN_DB_TESTS") == "1"
}

