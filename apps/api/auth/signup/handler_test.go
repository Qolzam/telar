package signup

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"strings"

	"github.com/qolzam/telar/apps/api/internal/types"
)

func TestSignup_Handle_Get_OK(t *testing.T) {
	app := fiber.New()
	h := &Handler{svc: &Service{}}
	app.Get("/signup", h.Handle)
	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestSignup_Handle_Post_MinimalBranches(t *testing.T) {
	app := fiber.New()
	h := &Handler{svc: &Service{}}
	app.Post("/signup", h.Handle)
	// Missing required fields -> BadRequest branch
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("fullName=&email=&newPassword="))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSignup_Handle_ValidationErrors(t *testing.T) {
	// Create handler with local test configuration
	priv := "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIF9p6oRkqKp7qkQGJJ4lmHn9qI7a1g7S0t7y2sYgHnQeoAoGCCqGSM49\nAwEHoUQDQgAEtq2jh2Qyq5gS5i8Eac1Q5E8p5i2vVh7mQmCw5HqB8w+f2h1O3F6C\n2wzJ6QJk0p8xgS6j4XGxqkF6J8nXGm+3vw==\n-----END EC PRIVATE KEY-----"
	rec := "dummy-recaptcha-key"

	app := fiber.New()
	h := NewHandler(&Service{}, rec, priv)
	app.Post("/signup", h.Handle)
	// missing fullname
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=a@b.c&newPassword=weak&verifyType=email"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSignup_Handle_Post_Email_And_Phone_SSR(t *testing.T) {
	app := fiber.New()

	// Create handler with local test configuration
	priv := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICyQKk8n0P3XqU6oGfR3R3H7bJZzr8u1ZqJ3UqgGj7CdoAoGCCqGSM49
AwEHoUQDQgAEcTq4YwJ1Hq9R7c3pXkqB2Yz9Cj6T02VJx3l7y4C2F2rBv3qT0p8e
f8bqD0tJgqz9qQ4rJcVxv5iL8k7Uq1f8cQ==
-----END EC PRIVATE KEY-----`
	key := "dummy-key"

	h := NewHandler(&Service{}, key, priv)
	app.Post("/signup", h.Handle)

	// Email verify type path
	formEmail := "fullName=User%20A&email=usera@example.com&newPassword=StrongPassw0rd!&verifyType=email&g-recaptcha-response=ok&responseType=ssr"
	req1 := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(formEmail))
	req1.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	_, _ = app.Test(req1)
	// Phone verify type path
	formPhone := "fullName=User%20B&email=userb@example.com&newPassword=StrongPassw0rd!&verifyType=phone&phoneNumber=+123&g-recaptcha-response=ok&responseType=ssr"
	req2 := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(formPhone))
	req2.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	_, _ = app.Test(req2)
}
