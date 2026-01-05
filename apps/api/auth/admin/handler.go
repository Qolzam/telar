package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/auth/errors"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
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

// CreateUserRequest represents the request payload for creating a user
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
	Role     string `json:"role"`
}

// CreateUser handles POST /admin/users - create new user directly (admin only)
func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	// Security check: Ensure user is admin (double-check even if middleware does it)
	user, ok := c.Locals(types.UserCtxName).(types.UserContext)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "UNAUTHORIZED",
			"message": "missing user context",
		})
	}
	if user.SystemRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"code":    "FORBIDDEN",
			"message": "admin access required",
		})
	}

	// Parse and validate request
	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return errors.HandleValidationError(c, "invalid request body")
	}

	// Basic validation
	if req.Email == "" {
		return errors.HandleValidationError(c, "email is required")
	}
	if req.Password == "" {
		return errors.HandleValidationError(c, "password is required")
	}
	if len(req.Password) < 8 {
		return errors.HandleValidationError(c, "password must be at least 8 characters")
	}
	if req.FullName == "" {
		return errors.HandleValidationError(c, "fullName is required")
	}
	if req.Role == "" {
		return errors.HandleValidationError(c, "role is required")
	}
	if req.Role != "user" && req.Role != "admin" {
		return errors.HandleValidationError(c, "role must be 'user' or 'admin'")
	}

	// Call service
	userID, err := h.adminService.CreateUserDirectly(c.Context(), req.Email, req.Password, req.FullName, req.Role)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":       userID.String(),
		"email":    req.Email,
		"fullName": req.FullName,
		"role":     req.Role,
	})
}
