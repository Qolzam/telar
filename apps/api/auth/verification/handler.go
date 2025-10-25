package verification

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/errors"
	"github.com/qolzam/telar/apps/api/auth/models"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// Phase 1.4: Secure verification model - legacy JWT support removed
type VerifySignupRequestSecure struct {
	VerificationId string `json:"verificationId"`
	Code           string `json:"code"`
	ResponseType   string `json:"responseType"`
}

type VerifySignupParams struct {
	VerificationId  uuid.UUID
	Code            string
	RemoteIpAddress string
	UserAgent       string
	UserId          string // Added for audit logging support
	ResponseType    string
	// Phase 1.2: HMAC protection fields
	HMACSignature string `json:"hmacSignature,omitempty"`
	Timestamp     int64  `json:"timestamp,omitempty"`
}

type VerifySignupResult struct {
	AccessToken string      `json:"accessToken"`
	TokenType   string      `json:"tokenType"`
	User        interface{} `json:"user"`
	Success     bool        `json:"success"`
}

type ServiceConfig struct {
	JWTConfig  platformconfig.JWTConfig
	HMACConfig platformconfig.HMACConfig
	AppConfig  platformconfig.AppConfig
}

type Handler struct {
	svc       *Service
	publicKey string
	orgName   string
	webDomain string
	config    *HandlerConfig
}

type HandlerConfig struct {
	PublicKey string
	OrgName   string
	WebDomain string
}

func NewHandler(s *Service, config *HandlerConfig) *Handler {
	return &Handler{
		svc:       s,
		publicKey: config.PublicKey,
		orgName:   config.OrgName,
		webDomain: config.WebDomain,
		config:    config,
	}
}

// Handle - Phase 1.4: Only secure verification supported
func (h *Handler) Handle(c *fiber.Ctx) error {
	// Parse secure request format only
	secureModel := &VerifySignupRequestSecure{}

	// Try to parse as JSON first, then fall back to form values
	_ = c.BodyParser(secureModel)

	// Fill in missing values from form data
	if secureModel.Code == "" {
		secureModel.Code = c.FormValue("code")
	}
	if secureModel.VerificationId == "" {
		secureModel.VerificationId = c.FormValue("verificationId")
	}
	if secureModel.ResponseType == "" {
		secureModel.ResponseType = c.FormValue("responseType")
	}

	return h.handleSecureVerification(c, secureModel)
}

// HandleVerificationLink handles GET requests for link-based email verification
// This allows users to verify by clicking a link in their email
func (h *Handler) HandleVerificationLink(c *fiber.Ctx) error {
	verificationId := c.Query("verificationId")
	code := c.Query("code")
	
	if verificationId == "" || code == "" {
		return c.Redirect(h.webDomain + "/signup?error=invalid_verification_link")
	}
	
	verifyUUID, err := uuid.FromString(verificationId)
	if err != nil {
		return c.Redirect(h.webDomain + "/signup?error=invalid_verification_id")
	}
	
	result, err := h.svc.VerifySignup(c.Context(), VerifySignupParams{
		VerificationId:  verifyUUID,
		Code:            code,
		RemoteIpAddress: c.IP(),
		UserAgent:       c.Get("User-Agent"),
		UserId:          "",
		ResponseType:    "ssr",
	})
	
	if err != nil {
		errorMsg := "Verification failed. Please try entering the code manually."
		return c.Redirect(h.webDomain + "/signup?error=verification_failed&message=" + errorMsg)
	}
	
	if result.AccessToken != "" {
		c.Cookie(&fiber.Cookie{
			Name:     "telar_session",
			Value:    result.AccessToken,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Lax",
			Path:     "/",
		})
	}
	
	return c.Redirect(h.webDomain + "/dashboard?verified=true")
}

// handleSecureVerification
// Phase 1.2: Enhanced with HMAC validation for additional security
// Phase 1.4: Legacy JWT support removed
func (h *Handler) handleSecureVerification(c *fiber.Ctx, model *VerifySignupRequestSecure) error {
	// Validate inputs
	if model.VerificationId == "" {
		return errors.HandleMissingFieldError(c, "verificationId")
	}
	if model.Code == "" {
		return errors.HandleMissingFieldError(c, "code")
	}

	// Parse UUID
	verifyUUID, err := uuid.FromString(model.VerificationId)
	if err != nil {
		return errors.HandleValidationError(c, "Invalid verification ID format")
	}

	// Phase 1.2: Parse HMAC headers for enhanced security (optional)
	hmacSignature := c.Get(types.HeaderHMACAuthenticate)
	timestampHeader := c.Get(types.HeaderTimestamp)
	var timestamp int64 = 0

	if timestampHeader != "" {
		if ts, parseErr := ParseTimestampFromHeader(timestampHeader); parseErr == nil {
			timestamp = ts
		}
	}

	// Get UserId for audit logging by looking up verification record
	var userId string
	if verification, err := h.svc.FindUserVerification(c.Context(), &models.DatabaseFilter{
		ObjectId: &verifyUUID,
	}); err == nil && verification != nil {
		userId = verification.UserId.String()
	}

	// Verify using secure service method
	result, err := h.svc.VerifySignup(c.Context(), VerifySignupParams{
		VerificationId:  verifyUUID,
		Code:            model.Code,
		RemoteIpAddress: c.IP(),
		UserAgent:       c.Get("User-Agent"),
		UserId:          userId,
		ResponseType:    model.ResponseType,
		// Phase 1.2: Include HMAC data
		HMACSignature: hmacSignature,
		Timestamp:     timestamp,
	})

	if err != nil {
		return errors.HandleServiceError(c, err)
	}

	// Return appropriate response
	if model.ResponseType == "spa" {
		return c.JSON(result)
	}

	// SSR: Return access token
	return c.JSON(fiber.Map{
		"accessToken": result.AccessToken,
		"tokenType":   "Bearer",
		"user":        result.User,
	})
}
