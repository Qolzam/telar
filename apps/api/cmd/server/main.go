package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/qolzam/telar/apps/api/auth"
	adminUC "github.com/qolzam/telar/apps/api/auth/admin"
	adminRepository "github.com/qolzam/telar/apps/api/auth/admin/repository"
	jwksUC "github.com/qolzam/telar/apps/api/auth/jwks"
	loginUC "github.com/qolzam/telar/apps/api/auth/login"
	oauthUC "github.com/qolzam/telar/apps/api/auth/oauth"
	passwordUC "github.com/qolzam/telar/apps/api/auth/password"
	signupUC "github.com/qolzam/telar/apps/api/auth/signup"
	verifyUC "github.com/qolzam/telar/apps/api/auth/verification"
	"github.com/qolzam/telar/apps/api/comments"
	commentHandlers "github.com/qolzam/telar/apps/api/comments/handlers"
	commentServices "github.com/qolzam/telar/apps/api/comments/services"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	platformemail "github.com/qolzam/telar/apps/api/internal/platform/email"
	"github.com/qolzam/telar/apps/api/posts"
	"github.com/qolzam/telar/apps/api/posts/handlers"
	postsServices "github.com/qolzam/telar/apps/api/posts/services"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
	"github.com/qolzam/telar/apps/api/profile"
	profileServices "github.com/qolzam/telar/apps/api/profile/services"
	authRepository "github.com/qolzam/telar/apps/api/auth/repository"
	profileRepository "github.com/qolzam/telar/apps/api/profile/repository"
	postsRepository "github.com/qolzam/telar/apps/api/posts/repository"
	commentRepository "github.com/qolzam/telar/apps/api/comments/repository"
	signupOrchestrator "github.com/qolzam/telar/apps/api/orchestrator/signup"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
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

	// Profile service will be initialized after repositories are created (see below)
	var profileService profileServices.ProfileService
	var profileCreator profileServices.ProfileServiceClient
	var profileHandler *profile.ProfileHandler
	var profileHandlers *profile.ProfileHandlers

	// Admin service will be created after repositories are set up (see below)
	var adminService *adminUC.Service
	var adminHandler *adminUC.AdminHandler

	// Create postgres client for repositories (used by orchestrator, signup service, and admin service)
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
	// Create repositories early (will be reused)
	authRepo := authRepository.NewPostgresAuthRepository(pgClient)
	verifRepo := authRepository.NewPostgresVerificationRepository(pgClient)
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)
	
	// Create signup service with verification repository
	signupService := signupUC.NewService(verifRepo, signupServiceConfig)
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
	// Login service will be created after repositories are set up (see below)
	var loginService *loginUC.Service
	var loginHandler *loginUC.Handler

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

	// Create remaining repositories (using same DB connection pool)
	adminRepo := adminRepository.NewPostgresAdminRepository(pgClient)
	postRepo := postsRepository.NewPostgresRepository(pgClient)
	commentRepo := commentRepository.NewPostgresCommentRepository(pgClient)

	// Initialize Profile service with repository (now that repositories are available)
	profileService = profileServices.NewProfileService(profileRepo, cfg)

	// Initialize profile handler (now that profileService is available)
	profileHandler = profile.NewProfileHandler(profileService, platformconfig.JWTConfig{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, platformconfig.HMACConfig{
		Secret: payloadSecret,
	})
	profileHandlers = &profile.ProfileHandlers{
		ProfileHandler: profileHandler,
	}

	// Decide which adapter to use based on deployment mode (now that profileService is initialized)
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

	// Create admin service with repositories (now that repositories are available)
	adminService = adminUC.NewService(authRepo, profileRepo, adminRepo, privateKey, cfg)
	adminHandler = adminUC.NewAdminHandler(adminService, platformconfig.JWTConfig{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, platformconfig.HMACConfig{
		Secret: payloadSecret,
	})

	// Create login service with AuthRepository and ProfileCreator (now that authRepo and profileCreator are available)
	loginService = loginUC.NewServiceWithProfileCreator(authRepo, profileCreator, loginServiceConfig)
	
	// Create login handler now that loginService is initialized
	loginHandlerConfig := &loginUC.HandlerConfig{
		WebDomain:           webDomain,
		PrivateKey:          privateKey,
		HeaderCookieName:    "telar-header",
		PayloadCookieName:   "telar-payload",
		SignatureCookieName: "telar-signature",
	}
	loginHandler = loginUC.NewHandler(loginService, loginHandlerConfig)

	// Create signup orchestrator
	signupOrchestrator := signupOrchestrator.NewService(authRepo, profileRepo, verifRepo)

	verifyService := verifyUC.NewServiceWithRepositoriesAndKeys(
		verifRepo,
		authRepo,
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
	profile.RegisterRoutes(app, profileHandlers, cfg)

	// We'll re-initialize them after setting up the adapters
	var commentsService commentServices.CommentService
	var postsService postsServices.PostService

	var commentCounter sharedInterfaces.CommentCounter
	var postStatsUpdater sharedInterfaces.PostStatsUpdater

	// Decide which adapters to use based on deployment mode (reuse same env var)
	if deploymentMode == "microservices" {
		log.Println("🔌 Wiring Posts and Comments services using gRPC Adapters")
		
		commentsServiceAddr := os.Getenv("COMMENTS_SERVICE_GRPC_ADDR")
		if commentsServiceAddr == "" {
			commentsServiceAddr = "localhost:50052"
		}
		
		postsServiceAddr := os.Getenv("POSTS_SERVICE_GRPC_ADDR")
		if postsServiceAddr == "" {
			postsServiceAddr = "localhost:50053"
		}

		grpcCounter, err := comments.NewGrpcCounter(commentsServiceAddr)
		if err != nil {
			log.Fatalf("Failed to create gRPC comment counter: %v", err)
		}
		commentCounter = grpcCounter
		log.Printf("✅ Comments gRPC client connected to %s", commentsServiceAddr)

		grpcStatsUpdater, err := posts.NewGrpcStatsUpdater(postsServiceAddr)
		if err != nil {
			log.Fatalf("Failed to create gRPC post stats updater: %v", err)
		}
		postStatsUpdater = grpcStatsUpdater
		log.Printf("✅ Posts gRPC client connected to %s", postsServiceAddr)
	} else {
		log.Println("🔌 Wiring Posts and Comments services using Direct Call Adapters")
		
		// Create temporary service instances to get adapters
		tempCommentsService := commentServices.NewCommentService(commentRepo, postRepo, cfg, nil)
		tempPostsService := postsServices.NewPostService(postRepo, cfg, nil)
		
		// Create direct call adapters
		commentCounter = comments.NewDirectCallCounter(tempCommentsService)
		postStatsUpdater = posts.NewDirectCallStatsUpdater(tempPostsService)
		log.Println("✅ Direct call adapters initialized")
	}

	// Re-initialize services with cross-service dependencies
	commentsService = commentServices.NewCommentService(commentRepo, postRepo, cfg, postStatsUpdater)
	postsService = postsServices.NewPostService(postRepo, cfg, commentCounter)

	// Index creation is now handled by SQL migrations
	log.Println("✅ Posts service initialized (indexes managed via SQL migrations)")

	postsHandler := handlers.NewPostHandler(postsService, cfg.JWT, cfg.HMAC)

	postsHandlers := &posts.PostsHandlers{
		PostHandler: postsHandler,
	}

	posts.RegisterRoutes(app, postsHandlers, cfg)

	commentHandler := commentHandlers.NewCommentHandler(commentsService, cfg.JWT, cfg.HMAC)

	commentRoutes := &comments.CommentsHandlers{
		CommentHandler: commentHandler,
	}

	comments.RegisterRoutes(app, commentRoutes, cfg)

	log.Printf("Starting Telar API Server (Auth + Profile + Posts + Comments) on port 8080")
	log.Fatal(app.Listen(":8080"))
}
