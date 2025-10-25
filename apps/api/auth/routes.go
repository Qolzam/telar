package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/auth/admin"
	"github.com/qolzam/telar/apps/api/auth/jwks"
	"github.com/qolzam/telar/apps/api/auth/login"
	"github.com/qolzam/telar/apps/api/auth/oauth"
	"github.com/qolzam/telar/apps/api/auth/password"
	"github.com/qolzam/telar/apps/api/auth/signup"
	"github.com/qolzam/telar/apps/api/auth/verification"
	authhmac "github.com/qolzam/telar/apps/api/internal/middleware/authhmac"
	authjwt "github.com/qolzam/telar/apps/api/internal/middleware/authjwt"
	"github.com/qolzam/telar/apps/api/internal/middleware/ratelimit"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// AuthHandlers holds all the handlers this router needs.
type AuthHandlers struct {
	AdminHandler    *admin.AdminHandler
	SignupHandler   *signup.Handler
	LoginHandler    *login.Handler
	VerifyHandler   *verification.Handler
	PasswordHandler *password.PasswordHandler
	OAuthHandler    *oauth.Handler
	JWKSHandler     *jwks.Handler
}

// NewAuthHandlers creates a new AuthHandlers with injected dependencies
func NewAuthHandlers(
	adminHandler *admin.AdminHandler,
	signupHandler *signup.Handler,
	loginHandler *login.Handler,
	verifyHandler *verification.Handler,
	passwordHandler *password.PasswordHandler,
	oauthHandler *oauth.Handler,
	jwksHandler *jwks.Handler,
) *AuthHandlers {
	return &AuthHandlers{
		AdminHandler:    adminHandler,
		SignupHandler:   signupHandler,
		LoginHandler:    loginHandler,
		VerifyHandler:   verifyHandler,
		PasswordHandler: passwordHandler,
		OAuthHandler:    oauthHandler,
		JWKSHandler:     jwksHandler,
	}
}

// RouterConfig holds the configuration needed for the router's middleware.
type RouterConfig struct {
	PayloadSecret string
	PublicKey     string
}

func authHMACMiddleware(hmacWithCookie bool, config RouterConfig) fiber.Handler {
	var Next func(c *fiber.Ctx) bool
	if hmacWithCookie {
		Next = func(c *fiber.Ctx) bool {
			if c.Get(types.HeaderHMACAuthenticate) != "" {
				return false
			}
			return true
		}
	}
	return authhmac.New(authhmac.Config{
		Next:          Next,
		PayloadSecret: config.PayloadSecret,
	})
}

func authJWTMiddleware(config RouterConfig) fiber.Handler {
	return authjwt.New(authjwt.Config{
		PublicKey:   config.PublicKey,
		ClaimKey:    "claim",
		UserCtxName: types.UserCtxName,
	})
}

// RegisterRoutes is the single entry point for setting up auth routes.
// It accepts all its dependencies and creates nothing.
func RegisterRoutes(app *fiber.App, handlers *AuthHandlers, cfg *platformconfig.Config) {
	group := app.Group("/auth")

	// Create router config from platform config
	routerConfig := &RouterConfig{
		PayloadSecret: cfg.HMAC.Secret,
		PublicKey:     cfg.JWT.PublicKey,
	}

	// Admin (HMAC only with rate limiting)
	admin := group.Group("/admin",
		authHMACMiddleware(false, *routerConfig),
		ratelimit.NewWithConfig(
			cfg.RateLimits.Login.Enabled,
			cfg.RateLimits.Login.Max,
			cfg.RateLimits.Login.Duration,
			"login",
		),
	)
	admin.Post("/check", handlers.AdminHandler.Check)
	admin.Post("/signup", handlers.AdminHandler.Signup)
	admin.Post("/login", handlers.AdminHandler.Login)

	// Signup (public with rate limiting)
	group.Post("/signup/verify",
		ratelimit.NewWithConfig(
			cfg.RateLimits.Verification.Enabled,
			cfg.RateLimits.Verification.Max,
			cfg.RateLimits.Verification.Duration,
			"verification",
		),
		handlers.VerifyHandler.Handle,
	)
	group.Get("/verify", handlers.VerifyHandler.HandleVerificationLink)
	group.Post("/signup/resend",
		ratelimit.NewWithConfig(
			cfg.RateLimits.Verification.Enabled,
			cfg.RateLimits.Verification.Max,
			cfg.RateLimits.Verification.Duration,
			"resend verification",
		),
		handlers.SignupHandler.Resend,
	)
	group.Post("/signup",
		ratelimit.NewWithConfig(
			cfg.RateLimits.Signup.Enabled,
			cfg.RateLimits.Signup.Max,
			cfg.RateLimits.Signup.Duration,
			"signup",
		),
		handlers.SignupHandler.Handle,
	)
	group.Get("/signup", handlers.SignupHandler.Handle)

	// Password reset (with specific rate limits)
	group.Get("/password/reset/:verifyId", handlers.PasswordHandler.ResetPage)
	group.Post("/password/reset/:verifyId",
		ratelimit.NewWithConfig(
			cfg.RateLimits.PasswordReset.Enabled,
			cfg.RateLimits.PasswordReset.Max,
			cfg.RateLimits.PasswordReset.Duration,
			"password reset",
		),
		handlers.PasswordHandler.ResetForm,
	)
	group.Get("/password/forget", handlers.PasswordHandler.ForgetPage)
	group.Post("/password/forget",
		ratelimit.NewWithConfig(
			cfg.RateLimits.PasswordReset.Enabled,
			cfg.RateLimits.PasswordReset.Max,
			cfg.RateLimits.PasswordReset.Duration,
			"password reset",
		),
		handlers.PasswordHandler.ForgetForm,
	)
	group.Put("/password/change",
		authJWTMiddleware(*routerConfig),
		ratelimit.NewWithConfig(
			cfg.RateLimits.PasswordReset.Enabled,
			cfg.RateLimits.PasswordReset.Max,
			cfg.RateLimits.PasswordReset.Duration,
			"password change",
		),
		handlers.PasswordHandler.Change,
	)

	// Login (public group with rate limiting)
	login := group.Group("/login")
	login.Get("/", handlers.LoginHandler.Handle)
	login.Post("/",
		ratelimit.NewWithConfig(
			cfg.RateLimits.Login.Enabled,
			cfg.RateLimits.Login.Max,
			cfg.RateLimits.Login.Duration,
			"login",
		),
		handlers.LoginHandler.Handle,
	)
	login.Get("/github", handlers.LoginHandler.Github)
	login.Get("/google", handlers.LoginHandler.Google)

	group.Get("/oauth2/authorized", handlers.OAuthHandler.Authorized) // OAuth callbacks don't need rate limiting

	// JWKS endpoint (public, no authentication required)
	group.Get("/.well-known/jwks.json", handlers.JWKSHandler.Handle)

}
