package testutil

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// TestConfig holds secure, environment-aware configuration for tests.
type TestConfig struct {
	MongoHost     string
	MongoDatabase string
	PGHost        string
	PGUser        string
	PGPassword    string
	PGDatabase    string
	PGSchema      string
	PayloadSecret string
	PublicKey     string
	PrivateKey    string
	RefEmail      string
	RefEmailPass  string
	SmtpEmail     string
	WebDomain     string

	// Additional fields for comprehensive testing
	AppName             string
	Debug               bool
	Gateway             string
	InternalGateway     string
	OrgName             string
	OrgAvatar           string
	Server              string
	RecaptchaKey        string
	RecaptchaSiteKey    string
	Origin              string
	HeaderCookieName    string
	PayloadCookieName   string
	SignatureCookieName string
	BaseRoute           string
	PhoneSourceNumber   string
	PhoneAuthToken      string
	PhoneAuthId         string
	DBType              string
}

// parsePostgresDSN parses a PostgreSQL DSN string and extracts connection parameters.
func parsePostgresDSN() (host, user, password, database, schema string) {
	// Default values
	host = "127.0.0.1"
	user = "postgres"
	password = "postgres"
	database = "telar_social_test"
	schema = "public"

	// Try to parse POSTGRES_DSN first
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		// Parse postgres://user:password@host:port/database?sslmode=disable&search_path=schema
		// Simple regex-based parsing for our specific format
		re := regexp.MustCompile(`postgres://([^:]+):([^@]+)@([^:]+):(\d+)/([^?]+)`)
		matches := re.FindStringSubmatch(dsn)
		if len(matches) == 6 {
			user = matches[1]
			password = matches[2]
			host = matches[3]
			database = matches[5]
		}
		
		// Extract search_path from query parameters
		if strings.Contains(dsn, "search_path=") {
			searchPathRe := regexp.MustCompile(`search_path=([^&]+)`)
			searchMatches := searchPathRe.FindStringSubmatch(dsn)
			if len(searchMatches) == 2 {
				schema = searchMatches[1]
			}
		}
	} else {
		// Fall back to individual environment variables
		host = getEnv("pg_host", host)
		user = getEnv("pg_user", user)
		password = getEnv("pg_pass", password)
		database = getEnv("pg_database", database)
		schema = getEnv("pg_schema", schema)
	}

	return host, user, password, database, schema
}

// LoadTestConfig loads configuration from environment
func LoadTestConfig() (*TestConfig, error) {
	// Generate RSA keys for better JWT library compatibility
	pubPEM, privPEM := GenerateECDSAKeyPairPEM(&testing.T{})

	pgHost, pgUser, pgPassword, pgDatabase, pgSchema := parsePostgresDSN()

	cfg := &TestConfig{
		MongoHost:     getEnv("mongo_host", "127.0.0.1"),
		MongoDatabase: getEnv("mongo_database", "telar_social_test"),
		PGHost:        pgHost,
		PGUser:        pgUser,
		PGPassword:    pgPassword,
		PGDatabase:    pgDatabase,
		PGSchema:      pgSchema,

		PayloadSecret: "test-secret",
		PublicKey:     pubPEM,
		PrivateKey:    privPEM,
		RefEmail:      getEnv("ref_email", "test@telar.dev"),
		RefEmailPass:  getEnv("ref_email_pass", "test-password"),
		SmtpEmail:     getEnv("smtp_email", "smtp@telar.dev"),
		WebDomain:     getEnv("web_domain", "https://test.telar.dev"),

		// Additional fields for comprehensive testing
		AppName:             getEnv("app_name", "Telar Social"),
		Debug:               getBoolEnv("debug", false),
		Gateway:             getEnv("gateway", "https://api.telar.dev"),
		InternalGateway:     getEnv("internal_gateway", "https://internal.telar.dev"),
		OrgName:             getEnv("org_name", "Telar"),
		OrgAvatar:           getEnv("org_avatar", "https://telar.dev/avatar.png"),
		Server:              getEnv("server", "https://telar.dev"),
		RecaptchaKey:        getEnv("recaptcha_key", "test-recaptcha-key"),
		RecaptchaSiteKey:    getEnv("recaptcha_site_key", "test-recaptcha-site-key"),
		Origin:              getEnv("origin", "https://telar.dev"),
		HeaderCookieName:    getEnv("header_cookie_name", "telar-header"),
		PayloadCookieName:   getEnv("payload_cookie_name", "telar-payload"),
		SignatureCookieName: getEnv("signature_cookie_name", "telar-signature"),
		BaseRoute:           getEnv("base_route", "/api"),
		PhoneSourceNumber:   getEnv("phone_source_number", "+1234567890"),
		PhoneAuthToken:      getEnv("phone_auth_token", "test-phone-token"),
		PhoneAuthId:         getEnv("phone_auth_id", "test-phone-id"),
		DBType:              getEnv("db_type", "mongo"),
	}


	if cfg.MongoHost == "" && cfg.PGHost == "" {
		return nil, fmt.Errorf("no database hosts configured: set MONGO_URI and/or POSTGRES_DSN")
	}
	return cfg, nil
}

// ToServiceConfig converts the test config to platform.ServiceConfig for a specific DB type.
func (c *TestConfig) ToServiceConfig(dbType string) *platform.ServiceConfig {
	sc := &platform.ServiceConfig{
		DatabaseType: dbType,
		DatabaseName: c.getDBName(dbType),
	}

	if dbType == dbi.DatabaseTypeMongoDB {
		sc.MongoConfig = &dbi.MongoDBConfig{
			Host: c.MongoHost,
			Port: 27017,
			MaxPoolSize:    200,
			MinPoolSize:    10,
			ConnectTimeout: 30,
			SocketTimeout:  60,
			MaxIdleTime:    300,
			ServerSelectionTimeout: 5,
		}
	} else if dbType == dbi.DatabaseTypePostgreSQL {
		sc.PostgreSQLConfig = &dbi.PostgreSQLConfig{
			Host:               c.PGHost,
			Port:               5432,
			Username:           c.PGUser,
			Password:           c.PGPassword,
			Database:           c.PGDatabase,
			SSLMode:            "disable",
			Schema:             c.PGSchema,
			ConnectTimeout:     30,
			MaxOpenConnections: 50,
			MaxIdleConnections: 10,
			MaxLifetime:        300,
		}
	}
	return sc
}

// ToPlatformConfig converts the test config to platformconfig.Config for a specific DB type.
func (c *TestConfig) ToPlatformConfig(dbType string) *platformconfig.Config {
	cfg := &platformconfig.Config{
		Server: platformconfig.ServerConfig{
			Host:            "localhost",
			BaseRoute:       "/api",
			Gateway:         "http://localhost:8080",
			InternalGateway: "http://localhost:8081",
			WebDomain:       "localhost",
			Debug:           true,
		},
		Database: platformconfig.DatabaseConfig{
			Type: dbType,
		},
		JWT: platformconfig.JWTConfig{
			PublicKey:  c.PublicKey,
			PrivateKey: c.PrivateKey,
		},
		HMAC: platformconfig.HMACConfig{
			Secret: c.PayloadSecret,
		},
		Email: platformconfig.EmailConfig{
			SMTPEmail:     "test@example.com",
			RefEmail:      "test@example.com",
			RefEmailPass:  "test-password",
		},
		Security: platformconfig.SecurityConfig{
			RecaptchaKey:    "test-recaptcha-key",
			RecaptchaSiteKey: "test-recaptcha-site-key",
			Origin:          "http://localhost:3000",
		},
		App: platformconfig.AppConfig{
			Name:            "test-app",
			OrgName:         "test-org",
			OrgAvatar:       "test-avatar",
			QueryPrettyURL:  true,
		},
		External: platformconfig.ExternalConfig{
			PhoneSourceNumber: "test-phone",
			PhoneAuthToken:    "test-token",
			PhoneAuthId:       "test-id",
		},
		Cache: platformconfig.CacheConfig{
			Enabled:         false,
			Backend:         "memory",
			TTL:             5 * time.Minute,
			Prefix:          "test:",
			MaxMemory:       100 * 1024 * 1024, // 100MB in bytes
			CleanupInterval: 1 * time.Minute,
		},
	}

	if dbType == dbi.DatabaseTypeMongoDB {
		cfg.Database.MongoDB = platformconfig.MongoDBConfig{
			Host:           c.MongoHost,
			Port:           27017,
			Database:       c.MongoDatabase,
			Username:       "",
			Password:       "",
			MaxPoolSize:    200,
			ConnectTimeout: 30 * time.Second,
			SocketTimeout:  60 * time.Second,
		}
	} else if dbType == dbi.DatabaseTypePostgreSQL {
		cfg.Database.Postgres = platformconfig.PostgreSQLConfig{
			Host:             c.PGHost,
			Port:             5432,
			Database:         c.PGDatabase,
			Username:         c.PGUser,
			Password:         c.PGPassword,
			SSLMode:          "disable",
			MaxOpenConns:     50,
			MaxIdleConns:     10,
			ConnMaxLifetime:  300 * time.Second,
		}
	}

	return cfg
}

func (c *TestConfig) getDBName(dbType string) string {
	if dbType == dbi.DatabaseTypeMongoDB {
		return c.MongoDatabase
	}
	return c.PGDatabase
}

func getEnv(key, defVal string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defVal
}

func getEnvOrEphemeral(key, prefix string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fmt.Sprintf("ephemeral-%s-%s", prefix, uuid.Must(uuid.NewV4()).String()[:8])
}

func getBoolEnv(key string, defVal bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		switch strings.ToLower(v) {
		case "true", "1":
			return true
		case "false", "0":
			return false
		}
	}
	return defVal
}

// SanitizeTestName cleans a test name to be used safely in DB/schema names.
// Enforces length limits to prevent MongoDB InvalidNamespace errors (63 char limit).
func SanitizeTestName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, " ", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	name = strings.ToLower(reg.ReplaceAllString(name, ""))

	// Enforce length limits for database naming compatibility
	// MongoDB has a 63-character limit, reserve 22 chars for "test_" prefix + "_" + 16-char UUID
	// This leaves 41 characters maximum for the sanitized test name
	const maxTestNameLength = 41
	if len(name) > maxTestNameLength {
		name = name[:maxTestNameLength]
	}

	return name
}
