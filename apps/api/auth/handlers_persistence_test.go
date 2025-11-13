package auth_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	auth "github.com/qolzam/telar/apps/api/auth"
	adminUC "github.com/qolzam/telar/apps/api/auth/admin"
	jwksUC "github.com/qolzam/telar/apps/api/auth/jwks"
	loginUC "github.com/qolzam/telar/apps/api/auth/login"
	oauthUC "github.com/qolzam/telar/apps/api/auth/oauth"
	passwordUC "github.com/qolzam/telar/apps/api/auth/password"
	signupUC "github.com/qolzam/telar/apps/api/auth/signup"
	verifyUC "github.com/qolzam/telar/apps/api/auth/verification"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/profile"
	profileServices "github.com/qolzam/telar/apps/api/profile/services"
	"github.com/stretchr/testify/require"
)

// newAuthAppForTest creates a test app using dependency injection

func newAuthAppForTest(t *testing.T, base *platform.BaseService, config *platformconfig.Config) *fiber.App {
	t.Helper()

	// Generate proper ECDSA key pair for testing
	pubPEM, privPEM := testutil.GenerateECDSAKeyPairPEM(t)

	// Create all auth handlers using the injected base service
	adminService := adminUC.NewService(base, privPEM, &platformconfig.Config{
		JWT:  platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
		HMAC: platformconfig.HMACConfig{Secret: config.HMAC.Secret},
	})
	adminHandler := adminUC.NewAdminHandler(adminService, platformconfig.JWTConfig{
		PublicKey:  pubPEM,
		PrivateKey: privPEM,
	}, platformconfig.HMACConfig{
		Secret: config.HMAC.Secret,
	})

	signupServiceConfig := &signupUC.ServiceConfig{
		JWTConfig:  platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
		HMACConfig: platformconfig.HMACConfig{Secret: config.HMAC.Secret},
		AppConfig:  platformconfig.AppConfig{WebDomain: "http://localhost"},
	}
	signupService := signupUC.NewService(base, signupServiceConfig)
	signupHandler := signupUC.NewHandler(signupService, "test-recaptcha-key", privPEM)

	loginServiceConfig := &loginUC.ServiceConfig{
		JWTConfig:  platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
		HMACConfig: platformconfig.HMACConfig{Secret: config.HMAC.Secret},
	}
	loginService := loginUC.NewService(base, loginServiceConfig)
	loginHandlerConfig := &loginUC.HandlerConfig{
		WebDomain:           "http://localhost",
		PrivateKey:          privPEM,
		HeaderCookieName:    "hdr",
		PayloadCookieName:   "pld",
		SignatureCookieName: "sig",
	}
	loginHandler := loginUC.NewHandler(loginService, loginHandlerConfig)

	verifyServiceConfig := &verifyUC.ServiceConfig{
		JWTConfig:  platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
		HMACConfig: platformconfig.HMACConfig{Secret: config.HMAC.Secret},
		AppConfig:  platformconfig.AppConfig{WebDomain: "http://localhost"},
	}
	verifyService := verifyUC.NewService(base, verifyServiceConfig)
	verifyHandlerConfig := &verifyUC.HandlerConfig{
		PublicKey: pubPEM,
		OrgName:   "Telar",
		WebDomain: "http://localhost",
	}
	var verifyHandler *verifyUC.Handler

	passwordServiceConfig := &passwordUC.ServiceConfig{
		JWTConfig:   platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
		HMACConfig:  platformconfig.HMACConfig{Secret: config.HMAC.Secret},
		EmailConfig: platformconfig.EmailConfig{SMTPEmail: "test@example.com", RefEmail: "smtp@example.com"},
		AppConfig:   platformconfig.AppConfig{WebDomain: "http://localhost"},
	}
	passwordService := passwordUC.NewService(base, passwordServiceConfig)
	passwordHandlerConfig := &passwordUC.HandlerConfig{
		RefEmail:     "smtp@example.com",
		RefEmailPass: "testpass",
		SMTPEmail:    "test@example.com",
		WebDomain:    "http://localhost",
	}
	passwordHandler, err := passwordUC.NewPasswordHandler(passwordService, passwordHandlerConfig)
	if err != nil {
		t.Fatalf("Failed to create password handler: %v", err)
	}

	// Create OAuth service and state store
	oauthConfig := oauthUC.NewOAuthConfig("http://localhost", "test_client", "test_secret", "test_google", "test_google_secret")
	oauthServiceConfig := &oauthUC.ServiceConfig{
		OAuthConfig: oauthConfig,
		JWTConfig:   platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
		HMACConfig:  platformconfig.HMACConfig{Secret: config.HMAC.Secret},
		AppConfig:   platformconfig.AppConfig{WebDomain: "http://localhost"},
	}
	oauthService := oauthUC.NewService(base, oauthServiceConfig)
	stateStore := oauthUC.NewMemoryStateStore()
	oauthHandlerConfig := &oauthUC.HandlerConfig{
		WebDomain:  "http://localhost",
		PrivateKey: privPEM,
	}
	oauthHandler := oauthUC.NewHandler(oauthService, oauthHandlerConfig, stateStore)

	// Create profile service adapter (using direct call adapter for tests)
	profileService := profileServices.NewService(base, &platformconfig.Config{
		JWT:      platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
		HMAC:     platformconfig.HMACConfig{Secret: config.HMAC.Secret},
		App:      platformconfig.AppConfig{WebDomain: "http://localhost"},
		Database: config.Database,
	})
	profileCreator := profile.NewDirectCallAdapter(profileService)

	// Update verification service to use profile creator
	verifyService = verifyUC.NewServiceWithKeys(
		base,
		verifyServiceConfig,
		pubPEM,
		"Telar",
		"http://localhost",
		profileCreator,
	)
	verifyHandler = verifyUC.NewHandler(verifyService, verifyHandlerConfig)

	// Create JWKS handler
	jwksHandler := jwksUC.NewHandler(pubPEM, "telar-auth-key-1")

	// Assemble auth handlers
	authHandlers := &auth.AuthHandlers{
		AdminHandler:    adminHandler,
		SignupHandler:   signupHandler,
		LoginHandler:    loginHandler,
		VerifyHandler:   verifyHandler,
		PasswordHandler: passwordHandler,
		OAuthHandler:    oauthHandler,
		JWKSHandler:     jwksHandler,
	}

	// Create router config using the generated public key
	platformCfg := &platformconfig.Config{
		HMAC: platformconfig.HMACConfig{Secret: config.HMAC.Secret},
		JWT:  platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
	}

	app := fiber.New()
	auth.RegisterRoutes(app, authHandlers, platformCfg)

	return app
}

// DELETED: Redundant signHMAC helper removed per g-sol10.md Step 2
// All HMAC signing now uses the centralized testutil.signHMAC with SHA256

func TestAuth_Admin_Check_SetsOK(t *testing.T) {
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Get the base config from the suite (CONFIG-FIRST pattern)
	baseConfig := suite.Config()

	// 3. Create isolated test environment using the base config
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, baseConfig)

	// 4. Create base service for the test using the platform config
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}

	// 5. Use base service for all database operations in your test.
	// Cleanup is automatic.
	app := newAuthAppForTest(t, base, iso.Config)
	httpHelper := testutil.NewHTTPHelper(t, app)

	// Use the same secret that was injected into the router config
	secret := iso.Config.HMAC.Secret
	uid := uuid.Must(uuid.NewV4()).String()

	// HMAC header required by admin group with canonical signing
	resp := httpHelper.NewRequest(http.MethodPost, "/auth/admin/check", []byte("{}")).
		WithHeader(types.HeaderContentType, "application/json").
		WithHMACAuth(secret, uid).Send()

	require.Equal(t, http.StatusOK, resp.StatusCode, "admin check failed")
}

func TestAuth_Admin_Signup_Persistence(t *testing.T) {
	// 1. Get the shared connection pool manager.
	suite := testutil.Setup(t)

	// 2. Make a local copy of the config for test-specific overrides.
	localConfig := *suite.Config()
	localConfig.Server.WebDomain = "http://localhost"

	// 3. Create a SINGLE isolated test environment. This creates a unique, temporary
	//    database and returns a repository connected to it. THIS IS OUR SOURCE OF TRUTH.
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, &localConfig)

	// 4. Create the application dependencies by INJECTING the ISOLATED repository.
	serviceCfg := &platform.ServiceConfig{
		DatabaseType:     dbi.DatabaseTypePostgreSQL,
		EnableTransactions: true,
		MaxRetries:         3,
	}
	baseService := platform.NewBaseServiceWithRepo(iso.Repo, serviceCfg)
	app := newAuthAppForTest(t, baseService, iso.Config)


	// 5. Prepare and execute the request.
	httpHelper := testutil.NewHTTPHelper(t, app)

	email := fmt.Sprintf("admin+persist-%d@example.com", time.Now().UnixNano())
	body := []byte("email=" + email + "&password=Secret123!@#")

	uid := uuid.Must(uuid.NewV4()).String()
	resp := httpHelper.NewRequest(http.MethodPost, "/auth/admin/signup", body).
		WithHeader("Content-Type", "application/x-www-form-urlencoded").
		WithHMACAuth(iso.Config.HMAC.Secret, uid). // Use the isolated config for the secret
		Send()

	require.Equal(t, http.StatusCreated, resp.StatusCode, "Admin signup failed")

	// 6. VERIFICATION: Use the SAME isolated repository to verify the result.
	t.Logf("Admin signup successful! Verifying persistence in isolated database")
	ctx := context.Background()
	actualUsername := strings.ReplaceAll(email, "+", " ")

	// Use `require.Eventually` for robustness. The verification is now checking the correct database.
	require.Eventually(t, func() bool {
		queryObj := &dbi.Query{
			Conditions: []dbi.Field{
				{Name: "data->>'username'", Value: actualUsername, Operator: "=", IsJSONB: true},
			},
		}
		countResult := <-iso.Repo.Count(ctx, "userAuth", queryObj)
		if countResult.Error != nil {
			t.Logf("Verification query failed: %v", countResult.Error)
			return false
		}
		return countResult.Count == 1
	}, 5*time.Second, 100*time.Millisecond, "Admin user did not appear in the isolated database after creation")

	t.Logf("Success: Admin user was correctly created and verified in its isolated database.")
}
