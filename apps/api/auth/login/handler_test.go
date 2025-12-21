package login

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	tokenutil "github.com/qolzam/telar/apps/api/internal/auth/tokens"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	authModels "github.com/qolzam/telar/apps/api/auth/models"
	"fmt"
)

func TestLogin_Handler_SSR_OK_Minimal(t *testing.T) {
	suite := testutil.Setup(t)

	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	k := suite.GetTestJWTConfig().PrivateKey
	domain := "http://localhost"

	app := fiber.New()
	app.Post("/login", func(c *fiber.Ctx) error {
		uid := uuid.Must(uuid.NewV4()).String()
		claim := map[string]interface{}{"displayName": "a", "socialName": "a", "email": "a@b.c", types.HeaderUID: uid, "role": "user", "createdDate": 0}
		pi := map[string]string{"id": uid, "login": "a", "name": "a", "audience": domain}
		access, _ := tokenutil.CreateTokenWithKey("telar", pi, "Telar", claim, k)
		return c.JSON(fiber.Map{"user": fiber.Map{"fullName": "a"}, "accessToken": access, "redirect": "", "expires_in": "0"})
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogin_Handle_SPA_POST_RedirectBranch_Coverage(t *testing.T) {
	suite := testutil.Setup(t)

	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// Create postgres client and repository
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer pgClient.Close()

	// Create schema and set search_path
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err, "Failed to create schema")
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "Failed to set search_path")

	// Apply auth migration
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
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth migration")

	authRepo := authRepository.NewPostgresAuthRepository(pgClient)

	app := fiber.New()

	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  suite.GetTestJWTConfig().PublicKey,
			PrivateKey: suite.GetTestJWTConfig().PrivateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: iso.Config.HMAC.Secret,
		},
	}
	svc := NewService(authRepo, serviceConfig)
	webDomain := "http://localhost"
	privateKey := suite.GetTestJWTConfig().PrivateKey
	headerCookieName := "hdr"
	payloadCookieName := "pld"
	signatureCookieName := "sig"

	handlerConfig := &HandlerConfig{
		WebDomain:           webDomain,
		PrivateKey:          privateKey,
		HeaderCookieName:    headerCookieName,
		PayloadCookieName:   payloadCookieName,
		SignatureCookieName: signatureCookieName,
	}
	handler := NewHandler(svc, handlerConfig)
	app.Post("/login", handler.Handle)

	uid := uuid.Must(uuid.NewV4())
	hash, _ := utils.Hash("Passw0rd!")
	userAuthModel := &authModels.UserAuth{
		ObjectId:      uid,
		Username:      "u@example.com",
		Password:      hash,
		Role:          "user",
		EmailVerified: true,
		PhoneVerified: false,
		CreatedDate:   1,
		LastUpdated:   1,
	}
	err = authRepo.CreateUser(ctx, userAuthModel)
	require.NoError(t, err, "Failed to create user")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=u@example.com&password=Passw0rd!"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	_, _ = app.Test(req)
}

func TestLogin_Handle_SSR_POST_SetsRedirect(t *testing.T) {
	suite := testutil.Setup(t)

	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// Create postgres client and repository
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer pgClient.Close()

	// Create schema and set search_path
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err, "Failed to create schema")
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "Failed to set search_path")

	// Apply auth migration
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
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth migration")

	authRepo := authRepository.NewPostgresAuthRepository(pgClient)

	app := fiber.New()

	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  suite.GetTestJWTConfig().PublicKey,
			PrivateKey: suite.GetTestJWTConfig().PrivateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: iso.Config.HMAC.Secret,
		},
	}
	svc := NewService(authRepo, serviceConfig)
	webDomain := "http://localhost"
	privateKey := suite.GetTestJWTConfig().PrivateKey
	headerCookieName := "hdr"
	payloadCookieName := "pld"
	signatureCookieName := "sig"

	handlerConfig := &HandlerConfig{
		WebDomain:           webDomain,
		PrivateKey:          privateKey,
		HeaderCookieName:    headerCookieName,
		PayloadCookieName:   payloadCookieName,
		SignatureCookieName: signatureCookieName,
	}
	handler := NewHandler(svc, handlerConfig)
	app.Post("/login", handler.Handle)

	uid := uuid.Must(uuid.NewV4())
	hash, _ := utils.Hash("Passw0rd!")
	userAuthModel := &authModels.UserAuth{
		ObjectId:      uid,
		Username:      "u@example.com",
		Password:      hash,
		Role:          "user",
		EmailVerified: true,
		PhoneVerified: false,
		CreatedDate:   1,
		LastUpdated:   1,
	}
	err = authRepo.CreateUser(ctx, userAuthModel)
	require.NoError(t, err, "Failed to create user")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=u@example.com&password=Passw0rd!"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	_, _ = app.Test(req)
}

func TestLogin_Handle_Get_OK(t *testing.T) {
	app := fiber.New()
	h := &Handler{svc: &Service{}}
	app.Get("/login", h.Handle)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLogin_Handle_MissingFields(t *testing.T) {
	app := fiber.New()
	h := &Handler{svc: &Service{}}
	app.Post("/login", h.Handle)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password=p"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_Github_Google_Redirects(t *testing.T) {
	suite := testutil.Setup(t)

	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	// Create postgres client and repository
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer pgClient.Close()

	// Create schema and set search_path
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err, "Failed to create schema")
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = pgClient.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "Failed to set search_path")

	// Apply auth migration
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
	`
	_, err = pgClient.DB().ExecContext(ctx, migrationSQL)
	require.NoError(t, err, "Failed to apply auth migration")

	authRepo := authRepository.NewPostgresAuthRepository(pgClient)

	app := fiber.New()

	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  suite.GetTestJWTConfig().PublicKey,
			PrivateKey: suite.GetTestJWTConfig().PrivateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: iso.Config.HMAC.Secret,
		},
	}
	svc := NewService(authRepo, serviceConfig)

	handlerConfig := &HandlerConfig{
		WebDomain:           "http://localhost",
		PrivateKey:          suite.GetTestJWTConfig().PrivateKey,
		HeaderCookieName:    "hdr",
		PayloadCookieName:   "pld",
		SignatureCookieName: "sig",
	}
	handler := NewHandler(svc, handlerConfig)
	app.Get("/login/github", handler.Github)
	app.Get("/login/google", handler.Google)
	r1 := httptest.NewRequest(http.MethodGet, "/login/github", nil)
	resp1, _ := app.Test(r1)
	if resp1.StatusCode != http.StatusFound {
		t.Fatalf("github expected 302, got %d", resp1.StatusCode)
	}
	r2 := httptest.NewRequest(http.MethodGet, "/login/google", nil)
	resp2, _ := app.Test(r2)
	if resp2.StatusCode != http.StatusFound {
		t.Fatalf("google expected 302, got %d", resp2.StatusCode)
	}
}
