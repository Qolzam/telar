package verification

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"strings"

	"github.com/qolzam/telar/apps/api/internal/types"

	"github.com/gofiber/fiber/v2"
)

func TestVerification_Handle_Get_OK(t *testing.T) {
	// Create handler with local test configuration
	pub := "-----BEGIN PUBLIC KEY-----\nMFYwEAYHKoZIzj0CAQYFK4EEAAoDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7m\nQmCw5HqB8w+f2h1O3F6C2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END PUBLIC KEY-----"
	org := "Telar"
	webDomain := "http://localhost"

	app := fiber.New()
	handlerConfig := &HandlerConfig{
		PublicKey: pub,
		OrgName:   org,
		WebDomain: webDomain,
	}
	h := NewHandler(&Service{}, handlerConfig)
	app.Get("/verify", h.Handle)
	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
}

func TestVerification_SSR_InvalidToken(t *testing.T) {
	pub := "-----BEGIN PUBLIC KEY-----\nMFYwEAYHKoZIzj0CAQYFK4EEAAoDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7m\nQmCw5HqB8w+f2h1O3F6C2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END PUBLIC KEY-----"
	org := "Telar"
	webDomain := "http://localhost"

	app := fiber.New()
	handlerConfig := &HandlerConfig{
		PublicKey: pub,
		OrgName:   org,
		WebDomain: webDomain,
	}
	h := NewHandler(&Service{}, handlerConfig)
	app.Post("/verify", h.Handle)

	// Test legacy verification with invalid token
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader("verificaitonSecret=invalid"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	// Test secure verification with missing fields
	req2 := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader("verificationId=&code="))
	req2.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp2.StatusCode)
	}
}
