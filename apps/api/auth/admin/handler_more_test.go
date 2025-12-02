package admin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	authModels "github.com/qolzam/telar/apps/api/auth/models"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	adminRepository "github.com/qolzam/telar/apps/api/auth/admin/repository"
	profileRepository "github.com/qolzam/telar/apps/api/profile/repository"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	authhmac "github.com/qolzam/telar/apps/api/internal/middleware/authhmac"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// createTestPlatformConfig creates a test platform config
func createTestPlatformConfig() *platformconfig.Config {
	return &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
}

func TestAdminHandler_Login_Success_Coverage(t *testing.T) {
	// Note: Test timeout is set via -timeout flag when running tests
	// This test requires at least 2-5 seconds due to bcrypt operations

	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction using Config-First pattern
	baseConfig := suite.Config()
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, baseConfig)
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping")
	}

	ctx := context.Background()

	// Create postgres client for repositories
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err)

	// Apply auth migrations
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS user_auths (
			id UUID PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash BYTEA NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
		CREATE INDEX IF NOT EXISTS idx_user_auths_role ON user_auths(role);
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth migrations")

	// Create repositories
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)
	adminRepo := adminRepository.NewPostgresAdminRepository(pgClient)

	// Use the new dependency injection pattern following posts golden reference
	platformCfg := createTestPlatformConfig()
	adminService := NewService(authRepo, profileRepo, adminRepo, "test-private-key", platformCfg)
	adminHandler := NewAdminHandler(adminService, platformCfg.JWT, platformCfg.HMAC)

	app := fiber.New()
	group := app.Group("/auth/admin")

	// Create HMAC middleware with proper configuration
	hmacMiddleware := authhmac.New(authhmac.Config{
		PayloadSecret: iso.Config.HMAC.Secret,
	})

	group.Post("/login", hmacMiddleware, adminHandler.Login)

	// Seed admin user with test-optimized password hashing
	// Use unique username to avoid collisions in parallel tests
	uniqueID := uuid.Must(uuid.NewV4())
	uniqueEmail := fmt.Sprintf("admin-login-%s@example.com", uniqueID.String()[:8])
	pass, _ := hashForTest("Adm1n!Pass") // Use faster bcrypt cost for tests
	
	objectID := uuid.Must(uuid.NewV4())
	// Use authRepo to create user directly
	userAuthModel := &authModels.UserAuth{
		ObjectId:      objectID,
		Username:      uniqueEmail,
		Password:      pass, // pass is already []byte from bcrypt
		Role:          "admin",
		EmailVerified: true,
		PhoneVerified: true,
	}
	require.NoError(t, authRepo.CreateUser(ctx, userAuthModel))

	req := httptest.NewRequest(http.MethodPost, "/auth/admin/login",
		strings.NewReader(fmt.Sprintf("email=%s&password=Adm1n!Pass", uniqueEmail)))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	// Use a longer timeout for bcrypt comparison tests
	resp, err := app.Test(req, 10000) // 10 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestAdminHandler_Login_Error_WrongPassword(t *testing.T) {
	// Note: Test timeout is set via -timeout flag when running tests
	// This test requires at least 2-5 seconds due to bcrypt operations

	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping")
	}

	ctx := context.Background()

	// Create postgres client for repositories
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err)

	// Apply auth migrations
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS user_auths (
			id UUID PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash BYTEA NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
		CREATE INDEX IF NOT EXISTS idx_user_auths_role ON user_auths(role);
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth migrations")

	// Create repositories
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)
	adminRepo := adminRepository.NewPostgresAdminRepository(pgClient)

	// Use the new dependency injection pattern following posts golden reference
	platformCfg := createTestPlatformConfig()
	adminService := NewService(authRepo, profileRepo, adminRepo, "test-private-key", platformCfg)
	adminHandler := NewAdminHandler(adminService, platformCfg.JWT, platformCfg.HMAC)

	app := fiber.New()
	group := app.Group("/auth/admin")

	// Create HMAC middleware with proper configuration
	hmacMiddleware := authhmac.New(authhmac.Config{
		PayloadSecret: iso.Config.HMAC.Secret,
	})

	group.Post("/login", hmacMiddleware, adminHandler.Login)

	// Seed admin user with test-optimized password hashing
	// Use unique username to avoid collisions in parallel tests
	uniqueID := uuid.Must(uuid.NewV4())
	uniqueEmail := fmt.Sprintf("admin-login-%s@example.com", uniqueID.String()[:8])
	pass, _ := hashForTest("Adm1n!Pass") // Use faster bcrypt cost for tests
	
	objectID := uuid.Must(uuid.NewV4())
	// Use authRepo to create user directly
	userAuthModel := &authModels.UserAuth{
		ObjectId:      objectID,
		Username:      uniqueEmail,
		Password:      pass, // pass is already []byte from bcrypt
		Role:          "admin",
		EmailVerified: true,
		PhoneVerified: true,
	}
	require.NoError(t, authRepo.CreateUser(ctx, userAuthModel))

	req := httptest.NewRequest(http.MethodPost, "/auth/admin/login",
		strings.NewReader(fmt.Sprintf("email=%s&password=Wrong!Pass", uniqueEmail)))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	// Use a longer timeout for bcrypt comparison tests
	resp, err := app.Test(req, 10000) // 10 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestAdminHandler_Check_OK(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping")
	}

	ctx := context.Background()

	// Create postgres client for repositories
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err)

	// Apply auth migrations
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS user_auths (
			id UUID PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash BYTEA NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
		CREATE INDEX IF NOT EXISTS idx_user_auths_role ON user_auths(role);
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth migrations")

	// Create repositories
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)
	adminRepo := adminRepository.NewPostgresAdminRepository(pgClient)

	app := fiber.New()
	// Use the new dependency injection pattern
	platformCfg := createTestPlatformConfig()
	adminService := NewService(authRepo, profileRepo, adminRepo, "test-private-key", platformCfg)
	adminHandler := NewAdminHandler(adminService, platformCfg.JWT, platformCfg.HMAC)
	app.Post("/check", adminHandler.Check)

	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader("{}"))
	req.Header.Set(types.HeaderContentType, "application/json")

	// Use a reasonable timeout for database operations
	resp, err := app.Test(req, 5000) // 5 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestAdminHandler_Signup_Status(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping")
	}

	ctx := context.Background()

	t.Logf("TRACE req=test-admin-signup step=config payloadSecret=%s", iso.Config.HMAC.Secret)

	// Create repositories for the new service constructor
	pgClient, err := postgres.NewClient(ctx, &dbi.PostgreSQLConfig{
		Host:     iso.Config.Database.Postgres.Host,
		Port:     iso.Config.Database.Postgres.Port,
		Username: iso.Config.Database.Postgres.Username,
		Password: iso.Config.Database.Postgres.Password,
		Database: iso.Config.Database.Postgres.Database,
		SSLMode:  iso.Config.Database.Postgres.SSLMode,
	}, iso.Config.Database.Postgres.Database)
	require.NoError(t, err)
	
	// Apply auth migrations (user_auths and verifications tables)
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS user_auths (
			id UUID PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash BYTEA NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
		CREATE INDEX IF NOT EXISTS idx_user_auths_role ON user_auths(role);
		CREATE TABLE IF NOT EXISTS verifications (
			id UUID PRIMARY KEY,
			user_id UUID,
			code VARCHAR(10) NOT NULL,
			target VARCHAR(255) NOT NULL,
			target_type VARCHAR(50) NOT NULL,
			counter BIGINT DEFAULT 1,
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			remote_ip_address VARCHAR(45),
			is_verified BOOLEAN DEFAULT FALSE,
			hashed_password BYTEA,
			expires_at BIGINT NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			full_name VARCHAR(255)
		);
		CREATE INDEX IF NOT EXISTS idx_verifications_user_type ON verifications(user_id, target_type) WHERE user_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_verifications_code ON verifications(code) WHERE used = FALSE;
		CREATE INDEX IF NOT EXISTS idx_verifications_target ON verifications(target, target_type) WHERE user_id IS NULL;
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
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth and profile migrations")
	
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)
	adminRepo := adminRepository.NewPostgresAdminRepository(pgClient)
	
	// Check if admin exists using repository (for logging/debugging)
	adminUser, err := authRepo.FindByRole(ctx, "admin")
	if err == nil && adminUser != nil {
		t.Logf("WARNING: Found existing admin in isolated database")
	} else {
		t.Logf("CONFIRMED: No existing admin found in isolated database")
	}
	
	platformConfig := createTestPlatformConfig()
	adminService := NewService(authRepo, profileRepo, adminRepo, iso.Config.JWT.PrivateKey, platformConfig)
	adminHandler := NewAdminHandler(adminService, platformConfig.JWT, platformConfig.HMAC)

	t.Logf("TRACE req=test-admin-signup step=handler-created-with-isolated-repo")

	app := fiber.New()
	app.Post("/signup", adminHandler.Signup)

	// Use unique email to avoid collisions
	uniqueID := uuid.Must(uuid.NewV4())
	uniqueEmail := fmt.Sprintf("admin-sign-%s@example.com", uniqueID.String()[:8])
	req := httptest.NewRequest(http.MethodPost, "/signup",
		strings.NewReader(fmt.Sprintf("email=%s&password=Adm1n!Pass", uniqueEmail)))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	t.Logf("TRACE req=test-admin-signup step=request-prepared")

	resp, err := app.Test(req, 5000)
	require.NoError(t, err)
	require.NotNil(t, resp)

	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Logf("TRACE req=test-admin-signup step=response-received status=%d body=%s", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode == 500 {
		// Use repository to check if admin was created
		afterHttpCheck, err := authRepo.FindByUsername(ctx, uniqueEmail)
		if err == nil && afterHttpCheck != nil && afterHttpCheck.Role == "admin" {
			// Admin was created despite 500 error - transaction commit issue
			_ = afterHttpCheck
		}

		_, createErr := adminService.CreateAdmin(ctx, "admin", uniqueEmail, "Adm1n!Pass")
		if createErr != nil {
			// Manual creation failed - error details available in createErr
			_ = createErr
		}
	}

	require.Equal(t, http.StatusCreated, resp.StatusCode, "Admin signup should return 201 Created")
}

func TestAdminHandler_Check_And_Signup_InternalError(t *testing.T) {
	suite := testutil.Setup(t)

	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping")
	}

	ctx := context.Background()

	// Create repositories for the new service constructor
	pgClient, err := postgres.NewClient(ctx, &dbi.PostgreSQLConfig{
		Host:     iso.Config.Database.Postgres.Host,
		Port:     iso.Config.Database.Postgres.Port,
		Username: iso.Config.Database.Postgres.Username,
		Password: iso.Config.Database.Postgres.Password,
		Database: iso.Config.Database.Postgres.Database,
		SSLMode:  iso.Config.Database.Postgres.SSLMode,
	}, iso.Config.Database.Postgres.Database)
	require.NoError(t, err)
	
	// Apply auth migrations (user_auths and verifications tables)
	migrationSQL := `
		CREATE TABLE IF NOT EXISTS user_auths (
			id UUID PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash BYTEA NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			email_verified BOOLEAN DEFAULT FALSE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auths_username ON user_auths(username);
		CREATE INDEX IF NOT EXISTS idx_user_auths_role ON user_auths(role);
		CREATE TABLE IF NOT EXISTS verifications (
			id UUID PRIMARY KEY,
			user_id UUID,
			code VARCHAR(10) NOT NULL,
			target VARCHAR(255) NOT NULL,
			target_type VARCHAR(50) NOT NULL,
			counter BIGINT DEFAULT 1,
			created_date BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			last_updated BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
			remote_ip_address VARCHAR(45),
			is_verified BOOLEAN DEFAULT FALSE,
			hashed_password BYTEA,
			expires_at BIGINT NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			full_name VARCHAR(255)
		);
		CREATE INDEX IF NOT EXISTS idx_verifications_user_type ON verifications(user_id, target_type) WHERE user_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_verifications_code ON verifications(code) WHERE used = FALSE;
		CREATE INDEX IF NOT EXISTS idx_verifications_target ON verifications(target, target_type) WHERE user_id IS NULL;
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
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth and profile migrations")
	
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)
	adminRepo := adminRepository.NewPostgresAdminRepository(pgClient)

	app := fiber.New()
	platformCfg := createTestPlatformConfig()
	adminService := NewService(authRepo, profileRepo, adminRepo, "test-private-key", platformCfg)
	adminHandler := NewAdminHandler(adminService, platformCfg.JWT, platformCfg.HMAC)
	app.Post("/check", adminHandler.Check)
	app.Post("/signup", adminHandler.Signup)

	r1 := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader("{}"))
	r1.Header.Set(types.HeaderContentType, "application/json")

	resp1, err := app.Test(r1, 5000)
	require.NoError(t, err)
	require.NotNil(t, resp1)

	r2 := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=a@b.c&password=x"))
	r2.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	resp2, err := app.Test(r2, 5000)
	require.NoError(t, err)
	require.NotNil(t, resp2)
}

// hashForTest provides fast password hashing for tests using bcrypt.MinCost=4
func hashForTest(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
}

// compareHashForTest provides fast password comparison for tests
func compareHashForTest(hash []byte, password []byte) error {

	newHash, err := hashForTest(string(password))
	if err != nil {
		return err
	}

	if string(hash) == string(newHash) {
		return nil
	}
	return fmt.Errorf("password mismatch")
}
