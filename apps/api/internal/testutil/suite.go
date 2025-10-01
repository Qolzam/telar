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
// It is a singleton designed solely to minimize connection overhead across test packages.
// CONFIG-FIRST: Suite now holds the canonical platform config instead of legacy TestConfig.
type Suite struct {
	mu                 sync.RWMutex
	mongoConnection    dbi.Repository
	postgresConnection dbi.Repository
	initialized        bool
	config             *platformconfig.Config // CONFIG-FIRST: Use platform config as source of truth
}

var (
	globalSuite *Suite
	suiteOnce   sync.Once
)

// Setup initializes the global suite with shared connections. It's safe to call
// from multiple tests and packages; it will only run its logic once.
// CONFIG-FIRST: Load platform config once and use it as the source of truth.
func Setup(t *testing.T) *Suite {
	t.Helper()
	
	suiteOnce.Do(func() {
		globalSuite = &Suite{}
		
		// CONFIG-FIRST: Load the canonical platform config ONCE inside suiteOnce.Do
		// This eliminates all os.Setenv calls and makes the suite parallel-safe
		cfg, err := platformconfig.LoadFromEnv()
		if err != nil {
			// If loading from env fails, create a default config for testing
			cfg = &platformconfig.Config{
				JWT: platformconfig.JWTConfig{
					PublicKey:  "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE1ISIQzdrnaTaiyqpQRWgK/pXLGyi\nUq5ssFd6Ay55mGWyqb9X0NrDjwc5kziI74j+nhgRxXFQCHeGCBIIDSR+Jg==\n-----END PUBLIC KEY-----",
					PrivateKey: "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIEXoWfqfRGBwIinSGPae2+2/FbHj5J6m5qufrLM+mEjboAoGCCqGSM49\nAwEHoUQDQgAE1ISIQzdrnaTaiyqpQRWgK/pXLGyiUq5ssFd6Ay55mGWyqb9X0NrD\njwc5kziI74j+nhgRxXFQCHeGCBIIDSR+Jg==\n-----END EC PRIVATE KEY-----",
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

		// Create shared connections with connection pooling.
		if err := globalSuite.createSharedConnections(); err != nil {
			t.Logf("Warning: Not all database connections were available: %v", err)
		}

		globalSuite.initialized = true
	})
	
	// Re-check connections if they are nil, in case a previous package's
	// non-standard cleanup closed them. This makes the suite resilient.
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
// CONFIG-FIRST: This is the primary method for accessing configuration in tests.
func (s *Suite) Config() *platformconfig.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// createSharedConnections attempts to connect to both databases concurrently.
func (s *Suite) createSharedConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perform health checks before creating connections
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
// This replaces direct os.Getenv("RUN_DB_TESTS") checks with a centralized approach.
func ShouldRunDatabaseTests() bool {
	return os.Getenv("RUN_DB_TESTS") == "1"
}

