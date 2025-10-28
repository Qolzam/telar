package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/qolzam/telar/apps/api/auth"
	adminUC "github.com/qolzam/telar/apps/api/auth/admin"
	jwksUC "github.com/qolzam/telar/apps/api/auth/jwks"
	loginUC "github.com/qolzam/telar/apps/api/auth/login"
	oauthUC "github.com/qolzam/telar/apps/api/auth/oauth"
	passwordUC "github.com/qolzam/telar/apps/api/auth/password"
	signupUC "github.com/qolzam/telar/apps/api/auth/signup"
	verifyUC "github.com/qolzam/telar/apps/api/auth/verification"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	platformemail "github.com/qolzam/telar/apps/api/internal/platform/email"
	"github.com/qolzam/telar/apps/api/profile"
	profileServices "github.com/qolzam/telar/apps/api/profile/services"
)

func main() {
	cfg, err := platformconfig.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load platform config: %v", err)
	}
	
	app := fiber.New(fiber.Config{
		// Disable default error handler that might interfere with custom responses
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			log.Printf("[ErrorHandler] Path: %s, Error: %v, Code: %d, ResponseSet: %d bytes", 
				c.Path(), err, code, len(c.Response().Body()))
			
			// If response already set by handler, don't override it
			if len(c.Response().Body()) > 0 {
				log.Printf("[ErrorHandler] Response already set, passing through")
				return nil
			}
			
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})


	payloadSecret := cfg.HMAC.Secret
	publicKey := cfg.JWT.PublicKey
	privateKey := cfg.JWT.PrivateKey
	webDomain := cfg.App.WebDomain
	smtpEmail := cfg.Email.SMTPEmail
	refEmail := cfg.Email.RefEmail
	refEmailPass := cfg.Email.RefEmailPass

	// CORS Configuration for Browser Direct Access
	app.Use(cors.New(cors.Config{
		AllowOrigins:     webDomain,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
	}))

	baseService, err := platform.NewBaseService(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to create base service: %v", err)
	}

	// Initialize Profile service (concrete implementation)
	profileService := profileServices.NewService(baseService, cfg)

	// Create database indexes on startup
	log.Println("🔧 Creating database indexes for Profile service...")
	indexCtx, indexCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := profileService.CreateIndexes(indexCtx); err != nil {
		indexCancel()
		log.Printf("⚠️  Warning: Failed to create indexes (may already exist): %v", err)
	} else {
		indexCancel()
		log.Println("✅ Profile database indexes created successfully")
	}

	// Decide which adapter to use based on deployment mode
	var profileCreator profileServices.ProfileServiceClient
	deploymentMode := os.Getenv("DEPLOYMENT_MODE")

	if deploymentMode == "microservices" {
		log.Println("🔌 Wiring Profile service using gRPC Adapter")
		profileServiceAddr := os.Getenv("PROFILE_SERVICE_GRPC_ADDR")
		if profileServiceAddr == "" {
			profileServiceAddr = "localhost:50051"
		}

		grpcCreator, err := profile.NewGrpcAdapter(profileServiceAddr)
		if err != nil {
			log.Fatalf("Failed to create gRPC profile creator: %v", err)
		}
		profileCreator = grpcCreator
		log.Printf("✅ Profile gRPC client connected to %s", profileServiceAddr)
	} else {
		log.Println("🔌 Wiring Profile service using Direct Call Adapter")
		profileCreator = profile.NewDirectCallAdapter(profileService)
		log.Println("✅ Profile direct call adapter initialized")
	}

	profileHandler := profile.NewProfileHandler(profileService, platformconfig.JWTConfig{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, platformconfig.HMACConfig{
		Secret: payloadSecret,
	})
	profileHandlers := &profile.ProfileHandlers{
		ProfileHandler: profileHandler,
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
		profileCreator,
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

	jwksHandler := jwksUC.NewHandler(publicKey, "telar-auth-key-1")

	authHandlers := &auth.AuthHandlers{
		AdminHandler:    adminHandler,
		SignupHandler:   signupHandler,
		LoginHandler:    loginHandler,
		VerifyHandler:   verifyHandler,
		PasswordHandler: passwordHandler,
		OAuthHandler:    oauthHandler,
		JWKSHandler:     jwksHandler,
	}

	auth.RegisterRoutes(app, authHandlers, cfg)
	profile.RegisterRoutes(app, profileHandlers, cfg)

	log.Printf("Starting Telar API Server (Auth + Profile) on port 8080")
	log.Fatal(app.Listen(":8080"))
}
