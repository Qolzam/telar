package signup

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	gopass "github.com/nbutton23/zxcvbn-go"
	"github.com/qolzam/telar/apps/api/auth/errors"

	recap "github.com/qolzam/telar/apps/api/internal/recaptcha"
)

type Handler struct {
	svc               *Service
	recaptchaVerifier recap.Verifier
	config            *HandlerConfig
}

type HandlerConfig struct {
	RecaptchaKey string
	PrivateKey   string
}

func NewHandler(s *Service, recaptchaKey, privateKey string) *Handler {
	// Backward compatibility: construct a Google verifier from key
	verifier, _ := recap.NewGoogleVerifier(recaptchaKey)
	return &Handler{
		svc:               s,
		recaptchaVerifier: verifier,
		config: &HandlerConfig{
			RecaptchaKey: recaptchaKey,
			PrivateKey:   privateKey,
		},
	}
}

// WithRecaptcha allows injecting a custom verifier (e.g., fake in tests)
func (h *Handler) WithRecaptcha(v recap.Verifier) *Handler {
	h.recaptchaVerifier = v
	return h
}

// SignupTokenHandle: mirror legacy form parsing and validation (structure only)
func (h *Handler) Handle(c *fiber.Ctx) error {
	if c.Method() == http.MethodGet {
		// Render simple HTML page for SSR signup
		html := "<!doctype html><html><head><title>Signup</title></head><body><h1>Signup</h1><form method='post'><input type='text' name='fullName' placeholder='Full name'/><input type='email' name='email' placeholder='Email'/><input type='password' name='newPassword' placeholder='Password'/><input type='hidden' name='responseType' value='ssr'/><input type='hidden' name='verifyType' value='email'/><button type='submit'>Signup</button></form></body></html>"
		c.Type("html")
		return c.SendString(html)
	}
	fullName := c.FormValue("fullName")
	email := c.FormValue("email")
	password := c.FormValue("newPassword")
	verifyType := c.FormValue("verifyType")
	recaptcha := c.FormValue("g-recaptcha-response")
	responseType := c.FormValue("responseType")

	model := &SignupTokenModel{
		User: UserSignupTokenModel{
			Fullname: fullName,
			Email:    email,
			Password: password,
		},
		VerifyType:   verifyType,
		Recaptcha:    recaptcha,
		ResponseType: responseType,
	}
	if model.User.Fullname == "" {
		return errors.HandleMissingFieldError(c, "fullName")
	}
	if model.User.Email == "" {
		return errors.HandleMissingFieldError(c, "email")
	}
	if model.User.Password == "" {
		return errors.HandleMissingFieldError(c, "password")
	}
	passStrength := gopass.PasswordStrength(model.User.Password, nil)

	if passStrength.Score < 3 || passStrength.Entropy < 37 {
		return errors.HandleValidationError(c, "Password is not strong enough!")
	}
	// Recaptcha validation via injected verifier
	remoteIP := c.IP()
	_ = remoteIP // kept for potential IP-based policies
	if h.recaptchaVerifier != nil {
		success, err := h.recaptchaVerifier.Verify(c.Context(), model.Recaptcha)
		if err != nil {
			return errors.HandleSystemError(c, "Error happened in verifying captcha!")
		}
		if !success {
			return errors.HandleValidationError(c, "Recaptcha is not valid!")
		}
	}
	newUserId := uuid.Must(uuid.NewV4())

	// Use new secure verification flow (Phase 1 refactoring)
	if model.VerifyType == "email" {
		response, err := h.svc.InitiateEmailVerification(c.Context(), EmailVerificationRequest{
			UserId:          newUserId,
			EmailTo:         model.User.Email,
			FullName:        model.User.Fullname,
			UserPassword:    model.User.Password,
			RemoteIpAddress: remoteIP,
			UserAgent:       c.Get("User-Agent"),
		})
		if err != nil {
			return errors.HandleServiceError(c, err)
		}

		// Return secure response format
		if model.ResponseType == "spa" {
			return c.JSON(response)
		}
		// For SSR, return the same secure format (HTML rendering can be added later)
		return c.JSON(response)
	}

	// Use new secure phone verification flow (Phase 1 refactoring)
	if model.VerifyType == "phone" {
		response, err := h.svc.InitiatePhoneVerification(c.Context(), PhoneVerificationRequest{
			UserId:          newUserId,
			PhoneNumber:     c.FormValue("phoneNumber"),
			FullName:        model.User.Fullname,
			UserPassword:    model.User.Password,
			RemoteIpAddress: remoteIP,
			UserAgent:       c.Get("User-Agent"),
		})
		if err != nil {
			return errors.HandleServiceError(c, err)
		}

		// Return secure response format
		if model.ResponseType == "spa" {
			return c.JSON(response)
		}
		// For SSR, return the same secure format (HTML rendering can be added later)
		return c.JSON(response)
	}

	return errors.HandleValidationError(c, "Invalid verification type")
}

// Resend handles POST /auth/signup/resend - resend verification email
func (h *Handler) Resend(c *fiber.Ctx) error {
	verificationId := c.FormValue("verificationId")
	if verificationId == "" {
		verificationId = c.Query("verificationId")
	}
	
	if verificationId == "" {
		return errors.HandleMissingFieldError(c, "verificationId")
	}
	
	verifyUUID, err := uuid.FromString(verificationId)
	if err != nil {
		return errors.HandleValidationError(c, "Invalid verification ID format")
	}
	
	if err := h.svc.ResendVerificationEmail(c.Context(), verifyUUID); err != nil {
		return errors.HandleServiceError(c, err)
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Verification email resent successfully",
	})
}
