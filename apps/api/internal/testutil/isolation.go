package testutil

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
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
	default:
		t.Fatalf("Unsupported database type for isolation: %s (only PostgreSQL is supported)", dbType)
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

	if isoTest.Config != nil {
		pgPort := isoTest.Config.Database.Postgres.Port
		if pgPort == 0 {
			pgPort = 5432
		}

		dsnURL := &url.URL{
			Scheme: "postgres",
			Host:   fmt.Sprintf("%s:%d", legacyCfg.PGHost, pgPort),
			Path:   "/" + legacyCfg.PGDatabase,
		}
		if legacyCfg.PGUser != "" {
			if legacyCfg.PGPassword != "" {
				dsnURL.User = url.UserPassword(legacyCfg.PGUser, legacyCfg.PGPassword)
			} else {
				dsnURL.User = url.User(legacyCfg.PGUser)
			}
		}

		query := dsnURL.Query()
		query.Set("sslmode", "disable")
		query.Set("search_path", uniqueSchema)
		dsnURL.RawQuery = query.Encode()

		isoTest.Config.Database.Type = dbi.DatabaseTypePostgreSQL
		isoTest.Config.Database.Postgres.Host = legacyCfg.PGHost
		isoTest.Config.Database.Postgres.Port = pgPort
		isoTest.Config.Database.Postgres.Username = legacyCfg.PGUser
		isoTest.Config.Database.Postgres.Password = legacyCfg.PGPassword
		isoTest.Config.Database.Postgres.Database = legacyCfg.PGDatabase
		isoTest.Config.Database.Postgres.DSN = dsnURL.String()
	}

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
