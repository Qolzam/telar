package oauth

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	tokenutil "github.com/qolzam/telar/apps/api/internal/auth/tokens"
	"github.com/qolzam/telar/apps/api/internal/types"
)

type Handler struct {
	service    *Service
	webDomain  string
	privateKey string
	stateStore StateStore // For storing PKCE parameters
	config     *HandlerConfig
}

type HandlerConfig struct {
	WebDomain  string
	PrivateKey string
}

// StateStore interface for storing OAuth state and PKCE parameters
type StateStore interface {
	Store(state string, pkce *PKCEParams, ttl time.Duration) error
	Retrieve(state string) (*PKCEParams, error)
	Delete(state string) error
}

func NewHandler(service *Service, config *HandlerConfig, stateStore StateStore) *Handler {
	return &Handler{
		service:    service,
		webDomain:  config.WebDomain,
		privateKey: config.PrivateKey,
		stateStore: stateStore,
		config:     config,
	}
}

// Github initiates GitHub OAuth flow with PKCE
func (h *Handler) Github(c *fiber.Ctx) error {
	return h.initiateOAuth(c, "github")
}

// Google initiates Google OAuth flow with PKCE
func (h *Handler) Google(c *fiber.Ctx) error {
	return h.initiateOAuth(c, "google")
}

// initiateOAuth starts OAuth flow with proper state and PKCE
func (h *Handler) initiateOAuth(c *fiber.Ctx, provider string) error {
	// 1. Generate PKCE parameters and state
	pkce, err := GeneratePKCEParams()
	if err != nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to generate PKCE: %w", err))
	}

	// 2. Store PKCE parameters with state (5 minute TTL)
	if err := h.stateStore.Store(pkce.State, pkce, 5*time.Minute); err != nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to store OAuth state: %w", err))
	}

	// 3. Generate authorization URL
	authURL, err := h.service.config.OAuthConfig.GetAuthURL(provider, pkce)
	if err != nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to generate auth URL: %w", err))
	}

	// 4. Redirect to OAuth provider
	return c.Redirect(authURL)
}

// Authorized handles OAuth callback with full security validation
func (h *Handler) Authorized(c *fiber.Ctx) error {
	// 1. Extract parameters
	code := c.Query("code")
	state := c.Query("state")
	provider := c.Query("provider") // Add provider to callback URL

	if code == "" {
		return errors.HandleValidationError(c, "missing authorization code")
	}
	if state == "" {
		return errors.HandleValidationError(c, "missing state parameter")
	}

	// 2. Retrieve and validate PKCE parameters
	pkce, err := h.stateStore.Retrieve(state)
	if err != nil {
		return errors.HandleValidationError(c, "invalid or expired state")
	}
	defer h.stateStore.Delete(state) // Clean up state

	// 3. Exchange code for token
	token, err := h.service.ExchangeCodeForToken(c.Context(), provider, code, pkce.CodeVerifier)
	if err != nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to exchange code: %w", err))
	}

	// 4. Get user information from provider
	userInfo, err := h.service.GetUserInfo(c.Context(), provider, token)
	if err != nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to get user info: %w", err))
	}

	// 5. Find or create user account
	userAuth, userProfile, err := h.service.FindOrCreateUser(c.Context(), userInfo)
	if err != nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to process user: %w", err))
	}

	// 6. Generate session JWT using existing token utility
	profile := map[string]string{
		"id":       userAuth.ObjectId.String(),
		"login":    userAuth.Username,
		"name":     userProfile.FullName,
		"audience": "telar",
	}

	claimData := map[string]interface{}{
		"displayName":   userProfile.FullName,
		"socialName":    userProfile.SocialName,
		"email":         userProfile.Email,
		"avatar":        userProfile.Avatar,
		types.HeaderUID: userAuth.ObjectId.String(),
		"role":          userAuth.Role,
		"createdDate":   userProfile.CreatedDate,
		"provider":      provider,
		"jti":           uuid.Must(uuid.NewV4()).String(),
	}

	sessionToken, err := tokenutil.CreateTokenWithKey("telar", profile, "telar-org", claimData, h.privateKey)
	if err != nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to create session token: %w", err))
	}

	// 7. Return session token (JSON response for SPA)
	return c.JSON(fiber.Map{
		"accessToken": sessionToken,
		"tokenType":   "Bearer",
		"user":        claimData,
		"provider":    provider,
	})
}
