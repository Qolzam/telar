package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/auth"
	adminUC "github.com/qolzam/telar/apps/api/auth/admin"
	jwksUC "github.com/qolzam/telar/apps/api/auth/jwks"
	loginUC "github.com/qolzam/telar/apps/api/auth/login"
	oauthUC "github.com/qolzam/telar/apps/api/auth/oauth"
	passwordUC "github.com/qolzam/telar/apps/api/auth/password"
	signupUC "github.com/qolzam/telar/apps/api/auth/signup"
	verifyUC "github.com/qolzam/telar/apps/api/auth/verification"
	platform 	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	platformemail "github.com/qolzam/telar/apps/api/internal/platform/email"
	"github.com/qolzam/telar/apps/api/internal/recaptcha"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/profile"
	profileServices "github.com/qolzam/telar/apps/api/profile/services"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	adminRepository "github.com/qolzam/telar/apps/api/auth/admin/repository"
	profileRepository "github.com/qolzam/telar/apps/api/profile/repository"
	signupOrchestrator "github.com/qolzam/telar/apps/api/orchestrator/signup"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
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

	// Create postgres client for repositories (used by orchestrator and signup service)
	ctx := context.Background()
	pgConfig := &dbi.PostgreSQLConfig{
		Host:               cfg.Database.Postgres.Host,
		Port:               cfg.Database.Postgres.Port,
		Username:           cfg.Database.Postgres.Username,
		Password:           cfg.Database.Postgres.Password,
		Database:           cfg.Database.Postgres.Database,
		SSLMode:            cfg.Database.Postgres.SSLMode,
		MaxOpenConnections: cfg.Database.Postgres.MaxOpenConns,
		MaxIdleConnections: cfg.Database.Postgres.MaxIdleConns,
		MaxLifetime:        int(cfg.Database.Postgres.ConnMaxLifetime.Seconds()),
		ConnectTimeout:     10,
	}
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	if err != nil {
		log.Fatalf("Failed to create postgres client for repositories: %v", err)
	}

	// Create repositories
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)
	verifRepo := authRepository.NewPostgresVerificationRepository(pgClient)
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)
	adminRepo := adminRepository.NewPostgresAdminRepository(pgClient)

	// Initialize Profile service with repository
	profileService := profileServices.NewProfileService(profileRepo, cfg)
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

	// Admin service will be created after repositories are set up (see below)
	var adminService *adminUC.Service
	var adminHandler *adminUC.AdminHandler

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
	// Create verification repository for signup service
	verifRepoForSignup := authRepository.NewPostgresVerificationRepository(pgClient)
	signupService := signupUC.NewService(verifRepoForSignup, signupServiceConfig)
	if smtpHost := cfg.Email.SMTPHost; smtpHost != "" {
		smtpPort := fmt.Sprintf("%d", cfg.Email.SMTPPort)
		smtpUser := cfg.Email.SMTPUser
		smtpPass := cfg.Email.SMTPPass
		sender, err := platformemail.NewSMTPSender(smtpHost, smtpPort, smtpUser, smtpPass)
		if err == nil {
			signupService = signupService.WithEmailSender(sender)
		}
	}
	
	// SECURITY: Fail Closed - Enforce Recaptcha configuration
	recaptchaKey := cfg.Security.RecaptchaKey
	recaptchaDisabled := cfg.Security.RecaptchaDisabled
	
	var recaptchaVerifier recaptcha.Verifier
	var errRecaptcha error
	
	if recaptchaKey == "" {
		if !recaptchaDisabled {
			log.Fatalf("SECURITY ERROR: RECAPTCHA_KEY is missing. Configure it or set RECAPTCHA_DISABLED=true in config.")
		}
		log.Printf("SECURITY WARNING: Recaptcha is explicitly disabled via configuration. Using FakeVerifier.")
		recaptchaVerifier = &testutil.FakeRecaptchaVerifier{ShouldSucceed: true}
	} else {
		recaptchaVerifier, errRecaptcha = recaptcha.NewGoogleVerifier(recaptchaKey)
		if errRecaptcha != nil {
			log.Fatalf("Failed to initialize Google Recaptcha: %v", errRecaptcha)
		}
	}
	
	signupHandler := signupUC.NewHandler(signupService, recaptchaVerifier, privateKey)

	loginServiceConfig := &loginUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: payloadSecret,
		},
	}
	// Create signup orchestrator
	signupOrchestrator := signupOrchestrator.NewService(authRepo, profileRepo, verifRepo)

	// Create admin service with repositories
	adminService = adminUC.NewService(authRepo, profileRepo, adminRepo, privateKey, cfg)
	adminHandler = adminUC.NewAdminHandler(adminService, platformconfig.JWTConfig{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, platformconfig.HMACConfig{
		Secret: payloadSecret,
	})

	// Create login service with AuthRepository
	loginService := loginUC.NewService(authRepo, loginServiceConfig)
	
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
	// Inject orchestrator into verification service
	verifyService.SetSignupOrchestrator(signupOrchestrator)
	
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
	// Create password service with repositories
	passwordService := passwordUC.NewServiceWithRepositories(authRepo, verifRepo, passwordServiceConfig)

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

	log.Printf("Starting Auth Service on port 9099")
	log.Fatal(app.Listen(":9099"))
}
