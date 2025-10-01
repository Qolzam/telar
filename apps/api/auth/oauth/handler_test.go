package oauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/stretchr/testify/require"
)

func TestAuthorized_Security_Validation(t *testing.T) {
	// Create minimal test setup
	app := fiber.New()

	// Create mock state store
	stateStore := NewMemoryStateStore()

	// Create OAuth config (minimal for test)
	config := NewOAuthConfig(
		"http://localhost:3000",
		"test_client",
		"test_secret",
		"test_google_client",
		"test_google_secret",
	)

	// Create service and handler
	serviceConfig := &ServiceConfig{
		OAuthConfig: config,
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}
	service := NewService(nil, serviceConfig) // nil BaseService for this test

	handlerConfig := &HandlerConfig{
		WebDomain:  "http://localhost",
		PrivateKey: "test-key",
	}
	handler := NewHandler(service, handlerConfig, stateStore)

	app.Get("/auth/oauth2/authorized", handler.Authorized)

	t.Run("Missing_Code_Parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/oauth2/authorized?state=test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Missing_State_Parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/oauth2/authorized?code=test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Invalid_State_Parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/oauth2/authorized?code=test&state=invalid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
