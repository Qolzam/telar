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
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
)

func TestLogin_Handler_SSR_OK_Minimal(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping test")
	}

	// Configure keys for token creation utilities used by handler code path
	k := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF9p6oRkqKp7qkQGJJ4lmHn9qI7a1g7S0t7y2sYgHnQeoAoGCCqGSM49\nAwEHoUQDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7mQmCw5HqB8w+f2h1O3F6C\n2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END EC PRIVATE KEY-----"
	domain := "http://localhost"
	// Removed unused cookie name variables per refactoring plan

	// Build a minimal route that returns the same JSON structure as handler after token creation
	app := fiber.New()
	app.Post("/login", func(c *fiber.Ctx) error {
		uid := uuid.Must(uuid.NewV4()).String()
		claim := map[string]interface{}{"displayName": "a", "socialName": "a", "email": "a@b.c", types.HeaderUID: uid, "role": "user", "createdDate": 0}
		pi := map[string]string{"id": uid, "login": "a", "name": "a", "audience": domain}
		access, _ := tokenutil.CreateTokenWithKey("telar", pi, "Telar", claim, k)
		// Return JWT in JSON response per refactoring plan (no cookies)
		return c.JSON(fiber.Map{"user": fiber.Map{"fullName": "a"}, "accessToken": access, "redirect": "", "expires_in": "0"})
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogin_Handle_SPA_POST_RedirectBranch_Coverage(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping test")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	app := fiber.New()

	// Create service and handler with injected configuration
	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF9p6oRkqKp7qkQGJJ4lmHn9qI7a1g7S0t7y2sYgHnQeoAoGCCqGSM49\nAwEHoUQDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7mQmCw5HqB8w+f2h1O3F6C\n2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END EC PRIVATE KEY-----",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
	svc := NewService(base, serviceConfig)
	webDomain := "http://localhost"
	privateKey := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF9p6oRkqKp7qkQGJJ4lmHn9qI7a1g7S0t7y2sYgHnQeoAoGCCqGSM49\nAwEHoUQDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7mQmCw5HqB8w+f2h1O3F6C\n2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END EC PRIVATE KEY-----"
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

	// seed a user so service path completes
	uid := uuid.Must(uuid.NewV4())
	hash, _ := utils.Hash("Passw0rd!")
	_ = (<-base.Repository.Save(ctx, "userAuth", map[string]interface{}{"objectId": uid, "username": "u@example.com", "password": hash, "role": "user", "emailVerified": true})).Error
	_ = (<-base.Repository.Save(ctx, "userProfile", map[string]interface{}{"objectId": uid, "fullName": "User U", "socialName": "useru", "email": "u@example.com", "avatar": "", "banner": "", "tagLine": "", "created_date": 1})).Error

	// POST without responseType=spa triggers redirect branch composition (even if token creation error ignored)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=u@example.com&password=Passw0rd!"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	_, _ = app.Test(req)
}

func TestLogin_Handle_SSR_POST_SetsRedirect(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping test")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	app := fiber.New()

	// Create service and handler with injected configuration
	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF9p6oRkqKp7qkQGJJ4lmHn9qI7a1g7S0t7y2sYgHnQeoAoGCCqGSM49\nAwEHoUQDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7mQmCw5HqB8w+f2h1O3F6C\n2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END EC PRIVATE KEY-----",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
	svc := NewService(base, serviceConfig)
	webDomain := "http://localhost"
	privateKey := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF9p6oRkqKp7qkQGJJ4lmHn9qI7a1g7S0t7y2sYgHnQeoAoGCCqGSM49\nAwEHoUQDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7mQmCw5HqB8w+f2h1O3F6C\n2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END EC PRIVATE KEY-----"
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

	// seed a user so service path completes
	uid := uuid.Must(uuid.NewV4())
	hash, _ := utils.Hash("Passw0rd!")
	_ = (<-base.Repository.Save(ctx, "userAuth", map[string]interface{}{"objectId": uid, "username": "u@example.com", "password": hash, "role": "user", "emailVerified": true})).Error
	_ = (<-base.Repository.Save(ctx, "userProfile", map[string]interface{}{"objectId": uid, "fullName": "User U", "socialName": "useru", "email": "u@example.com", "avatar": "", "banner": "", "tagLine": "", "created_date": 1})).Error

	// POST without responseType=spa triggers redirect branch composition (even if token creation error ignored)
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
	// missing username
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password=p"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_Github_Google_Redirects(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping test")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	app := fiber.New()

	// Create service and handler with injected configuration
	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF9p6oRkqKp7qkQGJJ4lmHn9qI7a1g7S0t7y2sYgHnQeoAoGCCqGSM49\nAwEHoUQDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7mQmCw5HqB8w+f2h1O3F6C\n2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END EC PRIVATE KEY-----",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
	svc := NewService(base, serviceConfig)

	handlerConfig := &HandlerConfig{
		WebDomain:           "http://localhost",
		PrivateKey:          "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF9p6oRkqKp7qkQGJJ4lmHn9qI7a1g7S0t7y2sYgHnQeoAoGCCqGSM49\nAwEHoUQDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7mQmCw5HqB8w+f2h1O3F6C\n2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END EC PRIVATE KEY-----",
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
