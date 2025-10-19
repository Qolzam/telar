package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/mongodb"
	"github.com/qolzam/telar/apps/api/internal/database/postgresql"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// IsolatedTest provides a truly isolated environment for a single test.
type IsolatedTest struct {
	t            *testing.T
	Repo         dbi.Repository
	Config       *platformconfig.Config 
	LegacyConfig *TestConfig           
}

// NewIsolatedTest creates a new isolated test environment.
func NewIsolatedTest(t *testing.T, dbType string, cfg *platformconfig.Config) *IsolatedTest {
	t.Helper()

	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("RUN_DB_TESTS not set, skipping database test")
	}

	// Create a deep copy of the config to avoid race conditions in parallel tests
	configCopy := *cfg
	isoTest := &IsolatedTest{
		t:      t,
		Config: &configCopy, 
	}

	// Dispatch to the correct isolation strategy.
	switch dbType {
	case dbi.DatabaseTypePostgreSQL:
		setupPostgresIsolatedTest(t, isoTest)
	case dbi.DatabaseTypeMongoDB:
		setupMongoDatabasePerTest(t, isoTest)
	default:
		t.Fatalf("Unsupported database type for isolation: %s", dbType)
	}
	
	return isoTest
}

// setupPostgresIsolatedTest implements schema-per-test isolation.
func setupPostgresIsolatedTest(t *testing.T, isoTest *IsolatedTest) {
	sanitizedName := SanitizeTestName(t.Name())
	uniqueSuffix := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")[:16]
	uniqueSchema := fmt.Sprintf("test_%s_%s", sanitizedName, uniqueSuffix)
	
	legacyCfg, err := LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load legacy config for isolated test: %v", err)
	}
	isoTest.LegacyConfig = legacyCfg
	
	isoTest.LegacyConfig.PGSchema = uniqueSchema

	dbCfg := legacyCfg.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	postgresRepo, err := postgresql.NewPostgreSQLRepository(context.Background(), dbCfg, dbCfg.Database)
	if err != nil {
		t.Fatalf("Failed to create isolated PostgreSQL repository for schema %s: %v", uniqueSchema, err)
	}
	isoTest.Repo = postgresRepo

	t.Cleanup(func() {
		if postgresRepo != nil {
			postgresRepo.Close()
			t.Logf("Closed PostgreSQL repository for test schema: %s", uniqueSchema)
		}
	})
}

// setupMongoDatabasePerTest implements database-per-test isolation for MongoDB.
func setupMongoDatabasePerTest(t *testing.T, isoTest *IsolatedTest) {
	t.Helper()

	// 1. Generate a unique database name and update the local config.
	uniqueName := fmt.Sprintf("test_%s_%s", SanitizeTestName(t.Name()), uuid.Must(uuid.NewV4()).String()[:8])
	isoTest.Config.Database.MongoDB.URI = updateURIWithDatabase(isoTest.Config.Database.MongoDB.URI, uniqueName)


	legacyCfg, err := LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load legacy config for isolated test: %v", err)
	}
	isoTest.LegacyConfig = legacyCfg
	
	dbCfg := legacyCfg.ToServiceConfig(dbi.DatabaseTypeMongoDB).MongoConfig
	mongoRepo, err := mongodb.NewMongoRepository(context.Background(), dbCfg, uniqueName)
	if err != nil {
		t.Fatalf("Failed to create isolated MongoDB repository for database %s: %v", uniqueName, err)
	}
	isoTest.Repo = mongoRepo

	t.Cleanup(func() {
		if mongoRepo != nil {
			mongoRepo.Close()
			t.Logf("Closed MongoDB repository for test database: %s", uniqueName)
		}
	})
}

// updateURIWithDatabase updates a MongoDB URI to use a specific database name
func updateURIWithDatabase(uri, dbName string) string {
	if strings.Contains(uri, "/") {
		parts := strings.Split(uri, "/")
		if len(parts) >= 2 {
			parts[len(parts)-1] = dbName
			return strings.Join(parts, "/")
		}
	}
	return uri + "/" + dbName
}

