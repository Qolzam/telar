package signup

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"strings"

	"github.com/qolzam/telar/apps/api/internal/testutil"
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
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("fullName=&email=&newPassword="))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSignup_Handle_ValidationErrors(t *testing.T) {
	suite := testutil.Setup(t)
	priv := suite.GetTestJWTConfig().PrivateKey
	rec := "dummy-recaptcha-key"

	app := fiber.New()
	h := NewHandler(&Service{}, rec, priv)
	app.Post("/signup", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=a@b.c&newPassword=weak&verifyType=email"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSignup_Handle_Post_Email_And_Phone_SSR(t *testing.T) {
	app := fiber.New()

	suite := testutil.Setup(t)
	priv := suite.GetTestJWTConfig().PrivateKey
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
