package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/auth"
	adminUC "github.com/qolzam/telar/apps/api/auth/admin"
	jwksUC "github.com/qolzam/telar/apps/api/auth/jwks"
	loginUC "github.com/qolzam/telar/apps/api/auth/login"
	oauthUC "github.com/qolzam/telar/apps/api/auth/oauth"
	passwordUC "github.com/qolzam/telar/apps/api/auth/password"
	profileUC "github.com/qolzam/telar/apps/api/auth/profile"
	signupUC "github.com/qolzam/telar/apps/api/auth/signup"
	verifyUC "github.com/qolzam/telar/apps/api/auth/verification"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	platformemail "github.com/qolzam/telar/apps/api/internal/platform/email"
)

func main() {
	cfg, err := platformconfig.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load platform config: %v", err)
	}
	
	app := fiber.New()

	payloadSecret := cfg.HMAC.Secret
	publicKey := cfg.JWT.PublicKey
	privateKey := cfg.JWT.PrivateKey
	webDomain := cfg.App.WebDomain
	smtpEmail := cfg.Email.SMTPEmail
	refEmail := cfg.Email.RefEmail
	refEmailPass := cfg.Email.RefEmailPass

	baseService, err := platform.NewBaseService(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to create base service: %v", err)
	}

	adminService := adminUC.NewService(baseService, privateKey, cfg)
	adminHandler := adminUC.NewAdminHandler(adminService, platformconfig.JWTConfig{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, platformconfig.HMACConfig{
		Secret: payloadSecret,
	})

	signupServiceConfig := &signupUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: payloadSecret,
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: webDomain,
		},
	}
	signupService := signupUC.NewService(baseService, signupServiceConfig)
	if smtpHost := cfg.Email.SMTPHost; smtpHost != "" {
		smtpPort := fmt.Sprintf("%d", cfg.Email.SMTPPort)
		smtpUser := cfg.Email.SMTPUser
		smtpPass := cfg.Email.SMTPPass
		sender, err := platformemail.NewSMTPSender(smtpHost, smtpPort, smtpUser, smtpPass)
		if err == nil {
			signupService = signupService.WithEmailSender(sender)
		}
	}
	signupHandler := signupUC.NewHandler(signupService, "", privateKey)

	loginServiceConfig := &loginUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: payloadSecret,
		},
	}
	loginService := loginUC.NewService(baseService, loginServiceConfig)
	
	loginHandlerConfig := &loginUC.HandlerConfig{
		WebDomain:           webDomain,
		PrivateKey:          privateKey,
		HeaderCookieName:    "telar-header",
		PayloadCookieName:   "telar-payload",
		SignatureCookieName: "telar-signature",
	}
	loginHandler := loginUC.NewHandler(loginService, loginHandlerConfig)

	verifyServiceConfig := &verifyUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: payloadSecret,
		},
		AppConfig: platformconfig.AppConfig{
			OrgName:   "Telar",
			WebDomain: webDomain,
		},
	}
	verifyService := verifyUC.NewServiceWithKeys(
		baseService,
		verifyServiceConfig,
		privateKey,
		cfg.App.OrgName,
		cfg.App.WebDomain,
	)
	
	verifyHandlerConfig := &verifyUC.HandlerConfig{
		PublicKey: publicKey,
		OrgName:   "Telar",
		WebDomain: webDomain,
	}
	verifyHandler := verifyUC.NewHandler(verifyService, verifyHandlerConfig)

	passwordServiceConfig := &passwordUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: payloadSecret,
		},
		EmailConfig: platformconfig.EmailConfig{
			SMTPEmail:    smtpEmail,
			RefEmail:     refEmail,
			RefEmailPass: refEmailPass,
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: webDomain,
		},
	}
	passwordService := passwordUC.NewService(baseService, passwordServiceConfig)

	smtpHost := cfg.Email.SMTPHost
	smtpPort := fmt.Sprintf("%d", cfg.Email.SMTPPort)
	smtpUser := cfg.Email.SMTPUser
	smtpPass := cfg.Email.SMTPPass
	if smtpHost != "" {
		sender, err := platformemail.NewSMTPSender(smtpHost, smtpPort, smtpUser, smtpPass)
		if err != nil {
			log.Printf("WARN: failed to initialize SMTP sender: %v", err)
		} else {
			passwordService = passwordService.WithEmailSender(sender)
		}
	}
	
	passwordHandlerConfig := &passwordUC.HandlerConfig{
		RefEmail:     refEmail,
		RefEmailPass: refEmailPass,
		SMTPEmail:    smtpEmail,
		WebDomain:    webDomain,
	}
	passwordHandler, err := passwordUC.NewPasswordHandler(passwordService, passwordHandlerConfig)
	if err != nil {
		log.Fatalf("Failed to create password handler: %v", err)
	}

	oauthConfig := oauthUC.NewOAuthConfig(
		webDomain,
		"",
		"",
		"",
		"",
	)
	
	oauthServiceConfig := &oauthUC.ServiceConfig{
		OAuthConfig: oauthConfig,
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: payloadSecret,
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: webDomain,
		},
	}
	oauthService := oauthUC.NewService(baseService, oauthServiceConfig)
	
	stateStore := oauthUC.NewMemoryStateStore()
	oauthHandlerConfig := &oauthUC.HandlerConfig{
		WebDomain:  webDomain,
		PrivateKey: privateKey,
	}
	oauthHandler := oauthUC.NewHandler(oauthService, oauthHandlerConfig, stateStore)

	authProfileServiceConfig := &profileUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: payloadSecret,
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: webDomain,
		},
	}
	authProfileService := profileUC.NewService(baseService, authProfileServiceConfig)
	authProfileHandler := profileUC.NewProfileHandler(authProfileService, platformconfig.JWTConfig{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, platformconfig.HMACConfig{
		Secret: payloadSecret,
	})

	jwksHandler := jwksUC.NewHandler(publicKey, "telar-auth-key-1")

	authHandlers := &auth.AuthHandlers{
		AdminHandler:    adminHandler,
		SignupHandler:   signupHandler,
		LoginHandler:    loginHandler,
		VerifyHandler:   verifyHandler,
		PasswordHandler: passwordHandler,
		OAuthHandler:    oauthHandler,
		ProfileHandler:  authProfileHandler,
		JWKSHandler:     jwksHandler,
	}

	auth.RegisterRoutes(app, authHandlers, cfg)

	log.Printf("Starting Auth Service on port 8080")
	log.Fatal(app.Listen(":8080"))
}
