package login

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/auth/errors"
	tokenutil "github.com/qolzam/telar/apps/api/internal/auth/tokens"
	"github.com/qolzam/telar/apps/api/internal/types"
)

type Handler struct {
	svc                 *Service
	webDomain           string
	privateKey          string
	headerCookieName    string
	payloadCookieName   string
	signatureCookieName string
	config              *HandlerConfig
}

type HandlerConfig struct {
	WebDomain           string
	PrivateKey          string
	HeaderCookieName    string
	PayloadCookieName   string
	SignatureCookieName string
}

func NewHandler(s *Service, config *HandlerConfig) *Handler {
	return &Handler{
		svc:                 s,
		webDomain:           config.WebDomain,
		privateKey:          config.PrivateKey,
		headerCookieName:    config.HeaderCookieName,
		payloadCookieName:   config.PayloadCookieName,
		signatureCookieName: config.SignatureCookieName,
		config:              config,
	}
}

func (h *Handler) Handle(c *fiber.Ctx) error {
	// SSR GET: return 200 OK placeholder for login page
	if c.Method() == http.MethodGet {
		return c.SendStatus(http.StatusOK)
	}

	// SPA/SSR POST: accept both JSON and form
	model := &LoginModel{}
	if c.Is("json") {
		_ = c.BodyParser(model)
	}

	if model.Username == "" {
		model.Username = c.FormValue("username")
	}
	if model.Password == "" {
		model.Password = c.FormValue("password")
	}
	if model.ResponseType == "" {
		model.ResponseType = c.FormValue("responseType")
	}
	if model.State == "" {
		model.State = c.FormValue("state")
	}

	if model.Username == "" {
		return errors.HandleMissingFieldError(c, "username")
	}
	if model.Password == "" {
		return errors.HandleMissingFieldError(c, "password")
	}

	foundUser, err := h.svc.FindUserByUsername(c.Context(), model.Username)
	if err != nil {
		// Log error for debugging but continue
	}
	if foundUser == nil {
		return errors.HandleUserNotFoundError(c, "User not found!")
	}

	if !foundUser.EmailVerified && !foundUser.PhoneVerified {
		return errors.HandleValidationError(c, "User is not verified!")
	}

	if h.svc.ComparePassword(foundUser.Password, model.Password) != nil {
		return errors.HandleAuthenticationError(c, "Password doesn't match!")
	}

	profile, _, err := h.svc.ReadProfileAndLanguage(c.Context(), *foundUser)
	if err != nil || profile == nil {
		return errors.HandleSystemError(c, "Can not find user profile!")
	}

	// Create token session using existing token util (in legacy it writes cookie + returns accessToken)
	tokenModel := map[string]interface{}{
		"claim": map[string]interface{}{
			"displayName":   profile.FullName,
			"socialName":    profile.SocialName,
			"email":         profile.Email,
			"avatar":        profile.Avatar,
			"banner":        profile.Banner,
			"tagLine":       profile.TagLine,
			types.HeaderUID: foundUser.ObjectId.String(),
			"role":          foundUser.Role,
			"createdDate":   profile.CreatedDate,
		},
	}

	// Create ES256 token (no cookies, no URL redirects)
	profileInfo := map[string]string{"id": foundUser.ObjectId.String(), "login": foundUser.Username, "name": profile.FullName, "audience": h.webDomain}
	accessToken, _ := tokenutil.CreateTokenWithKey("telar", profileInfo, "Telar", tokenModel["claim"].(map[string]interface{}), h.privateKey)

	return c.JSON(fiber.Map{
		"user":        profile,
		"accessToken": accessToken,
		"tokenType":   "Bearer",
		"expires_in":  strconv.Itoa(0),
	})
}

// Github redirects user to GitHub OAuth consent (placeholder minimal)
func (h *Handler) Github(c *fiber.Ctx) error {
	return c.Redirect("https://github.com/login/oauth/authorize", http.StatusFound)
}

// Google redirects user to Google OAuth consent (placeholder minimal)
func (h *Handler) Google(c *fiber.Ctx) error {
	return c.Redirect("https://accounts.google.com/o/oauth2/v2/auth", http.StatusFound)
}

// Note: OAuth provider flows are placeholders for redirects; callback issues ES256 token and cookies
