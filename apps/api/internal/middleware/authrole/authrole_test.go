package authrole

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/internal/types"
)

func TestAuthRole_UnauthorizedWithoutUser(t *testing.T) {
    app := fiber.New()
    app.Get("/", New(Config{}), func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
    req := httptest.NewRequest("GET", "/", nil)
    resp, _ := app.Test(req)
    if resp.StatusCode != http.StatusUnauthorized { t.Fatalf("expected 401, got %d", resp.StatusCode) }
}

func TestAuthRole_AuthorizedWithMatchingRole(t *testing.T) {
    app := fiber.New()
    app.Use(func(c *fiber.Ctx) error {
        c.Locals(types.UserCtxName, types.UserContext{ SystemRole: "admin" })
        return c.Next()
    })
    app.Get("/", New(Config{ Role: "admin" }), func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
    req := httptest.NewRequest("GET", "/", nil)
    resp, _ := app.Test(req)
    if resp.StatusCode != http.StatusOK { t.Fatalf("expected 200, got %d", resp.StatusCode) }
}

func TestAuthRole_UnauthorizedWithMismatchedRole(t *testing.T) {
    app := fiber.New()
    app.Use(func(c *fiber.Ctx) error {
        c.Locals(types.UserCtxName, types.UserContext{ SystemRole: "user" })
        return c.Next()
    })
    app.Get("/", New(Config{ Role: "admin" }), func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
    req := httptest.NewRequest("GET", "/", nil)
    resp, _ := app.Test(req)
    if resp.StatusCode != http.StatusUnauthorized { t.Fatalf("expected 401, got %d", resp.StatusCode) }
}


