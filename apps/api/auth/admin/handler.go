package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/auth/errors"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// AdminHandler handles all admin-related HTTP requests
type AdminHandler struct {
	adminService *Service
	jwtConfig    platformconfig.JWTConfig
	hmacConfig   platformconfig.HMACConfig
}

// NewAdminHandler creates a new AdminHandler with injected dependencies
func NewAdminHandler(adminService *Service, jwtConfig platformconfig.JWTConfig, hmacConfig platformconfig.HMACConfig) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
		jwtConfig:    jwtConfig,
		hmacConfig:   hmacConfig,
	}
}

// Check handles POST /admin/check - check if admin exists
func (h *AdminHandler) Check(c *fiber.Ctx) error {
	ok, _ := h.adminService.CheckAdmin(c.Context())
	return c.JSON(fiber.Map{"admin": ok})
}

// Signup handles POST /admin/signup - create new admin
func (h *AdminHandler) Signup(c *fiber.Ctx) error {
	// Parse form data directly since test sends form-encoded data
	email := c.FormValue("email")
	password := c.FormValue("password")

	// If form values are empty, try JSON body parsing
	if email == "" || password == "" {
		var body struct{ Email, Password string }
		if err := c.BodyParser(&body); err == nil {
			if email == "" {
				email = body.Email
			}
			if password == "" {
				password = body.Password
			}
		}
	}

	token, err := h.adminService.CreateAdmin(c.Context(), "admin", email, password)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.Status(201).JSON(fiber.Map{"token": token})
}

// Login handles POST /admin/login - admin login
func (h *AdminHandler) Login(c *fiber.Ctx) error {
	var body struct{ Email, Password string }
	_ = c.BodyParser(&body)
	email := body.Email
	password := body.Password
	if email == "" {
		email = c.FormValue("email")
	}
	if password == "" {
		password = c.FormValue("password")
	}
	token, err := h.adminService.Login(c.Context(), email, password)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.JSON(fiber.Map{"token": token})
}
