package profile

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
)

func TestProfile_Handle_OK(t *testing.T) {
	// Skip if not running database tests
	if testing.Short() {
		t.Skip("short")
	}

	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping test")
	}

	// Build app using a BaseService bound to MongoDB
	base, err := platform.NewBaseService(context.Background(), iso.Config)
	if err != nil {
		t.Fatalf("failed to build mongodb base service: %v", err)
	}

	// Create service and handler using dependency injection
	serviceConfig := &ServiceConfig{
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
	service := NewService(base, serviceConfig)
	handler := NewProfileHandler(service, platformconfig.JWTConfig{
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}, platformconfig.HMACConfig{
		Secret: "test-secret",
	})

	app := fiber.New()
	app.Put("/auth/profile", handler.Handle)
	body := bytes.NewBufferString(`{"fullName":"A","avatar":"","banner":"","tagLine":"","socialName":"a"}`)
	req := httptest.NewRequest(http.MethodPut, "/auth/profile", body)
	req.Header.Set(types.HeaderContentType, "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
