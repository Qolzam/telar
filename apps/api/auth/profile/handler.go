package profile

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/utils"
)

type ProfileUpdateModel struct {
	FullName   string `json:"fullName"`
	Avatar     string `json:"avatar"`
	Banner     string `json:"banner"`
	TagLine    string `json:"tagLine"`
	SocialName string `json:"socialName"`
}

// ProfileHandler handles profile-related HTTP requests
type ProfileHandler struct {
	service *Service
	config  *HandlerConfig
}

type HandlerConfig struct {
	JWTConfig  platformconfig.JWTConfig
	HMACConfig platformconfig.HMACConfig
}

// NewProfileHandler creates a new ProfileHandler with injected dependencies
func NewProfileHandler(service *Service, jwtConfig platformconfig.JWTConfig, hmacConfig platformconfig.HMACConfig) *ProfileHandler {
	return &ProfileHandler{
		service: service,
		config: &HandlerConfig{
			JWTConfig:  jwtConfig,
			HMACConfig: hmacConfig,
		},
	}
}

// Handle updates user profile
func (h *ProfileHandler) Handle(c *fiber.Ctx) error {
	// Parse body as JSON (legacy handler also used BodyParser)
	model := new(ProfileUpdateModel)
	if err := c.BodyParser(model); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.Error("internal/parseProfileUpdateModel", "Error while parsing body"))
	}
	// Use injected service instead of creating empty one
	if err := h.service.UpdateProfile(c.Context(), model.FullName, model.Avatar, model.Banner, model.TagLine, model.SocialName); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.Error("internal/updateUserProfile", "Can not update user profile!"))
	}
	return c.SendStatus(http.StatusOK)
}
