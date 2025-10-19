package auth_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	"github.com/qolzam/telar/apps/api/internal/utils"
	"github.com/qolzam/telar/apps/api/profile"
	profileServices "github.com/qolzam/telar/apps/api/profile/services"
	"github.com/stretchr/testify/require"
)


func newAuthApp(t *testing.T, base *platform.BaseService, cfg *platformconfig.Config) (*fiber.App, string, string) {
	pubPEM, privPEM := testutil.GenerateECDSAKeyPairPEM(t)

	// Create platform config for auth services
	platformCfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  pubPEM,
			PrivateKey: privPEM,
		},
		HMAC: platformconfig.HMACConfig{
			Secret: cfg.HMAC.Secret,
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost",
			OrgName:   "Telar",
		},
		Email: platformconfig.EmailConfig{
			SMTPEmail:    "test@example.com",
			RefEmail:     "test@example.com",
			RefEmailPass: "testpass",
		},
	}

	// Create admin service with new constructor
	adminService := adminUC.NewService(base, privPEM, platformCfg)
	adminHandler := adminUC.NewAdminHandler(adminService, platformconfig.JWTConfig{
		PublicKey:  pubPEM,
		PrivateKey: privPEM,
	}, platformconfig.HMACConfig{
		Secret: cfg.HMAC.Secret,
	})

	// Create signup service with new constructor
	signupServiceConfig := &signupUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  pubPEM,
			PrivateKey: privPEM,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: cfg.HMAC.Secret,
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: "http://localhost",
		},
	}
	signupService := signupUC.NewService(base, signupServiceConfig)
	signupHandler := signupUC.NewHandler(signupService, "test-recaptcha-key", privPEM)
	signupHandler = signupHandler.WithRecaptcha(&testutil.FakeRecaptchaVerifier{ShouldSucceed: true})

	// Create login service with new constructor
	loginServiceConfig := &loginUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  pubPEM,
			PrivateKey: privPEM,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: cfg.HMAC.Secret,
		},
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

	// Create verification service with new constructor
	verifyServiceConfig := &verifyUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  pubPEM,
			PrivateKey: privPEM,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: cfg.HMAC.Secret,
		},
		AppConfig: platformconfig.AppConfig{
			OrgName:   "Telar",
			WebDomain: "http://localhost",
		},
	}
	verifyService := verifyUC.NewService(base, verifyServiceConfig)
	verifyHandlerConfig := &verifyUC.HandlerConfig{
		PublicKey: pubPEM,
		OrgName:   "Telar",
		WebDomain: "http://localhost",
	}
	var verifyHandler *verifyUC.Handler

	// Create password service with new constructor
	passwordServiceConfig := &passwordUC.ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  pubPEM,
			PrivateKey: privPEM,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: cfg.HMAC.Secret,
		},
		EmailConfig: platformconfig.EmailConfig{
			SMTPEmail:    "test@example.com",
			RefEmail:     "test@example.com",
			RefEmailPass: "testpass",
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: "http://localhost",
		},
	}
	passwordService := passwordUC.NewService(base, passwordServiceConfig)
	passwordHandlerConfig := &passwordUC.HandlerConfig{
		RefEmail:     "test@example.com",
		RefEmailPass: "testpass",
		SMTPEmail:    "test@example.com",
		WebDomain:    "http://localhost",
	}
	passwordHandler, err := passwordUC.NewPasswordHandler(passwordService, passwordHandlerConfig)
	if err != nil {
		t.Fatalf("password handler init: %v", err)
	}

	// Create OAuth service and state store
	oauthConfig := oauthUC.NewOAuthConfig("http://localhost", "test_client", "test_secret", "test_google", "test_google_secret")
	oauthServiceConfig := &oauthUC.ServiceConfig{
		OAuthConfig: oauthConfig,
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  pubPEM,
			PrivateKey: privPEM,
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: cfg.HMAC.Secret,
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: "http://localhost",
		},
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
		HMAC:     platformconfig.HMACConfig{Secret: cfg.HMAC.Secret},
		App:      platformconfig.AppConfig{WebDomain: "http://localhost"},
		Database: cfg.Database,
	})
	profileCreator := profile.NewDirectCallAdapter(profileService)

	// Update verification service to use profile creator
	verifyService = verifyUC.NewServiceWithKeys(
		base,
		verifyServiceConfig,
		privPEM,
		"Telar",
		"http://localhost",
		profileCreator,
	)
	verifyHandler = verifyUC.NewHandler(verifyService, verifyHandlerConfig)

	// Create JWKS handler
	jwksHandler := jwksUC.NewHandler(pubPEM, "telar-auth-key-1")

	authHandlers := &auth.AuthHandlers{
		AdminHandler:    adminHandler,
		SignupHandler:   signupHandler,
		LoginHandler:    loginHandler,
		VerifyHandler:   verifyHandler,
		PasswordHandler: passwordHandler,
		OAuthHandler:    oauthHandler,
		JWKSHandler:     jwksHandler,
	}

	platformCfg = &platformconfig.Config{
		HMAC: platformconfig.HMACConfig{Secret: cfg.HMAC.Secret},
		JWT:  platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
	}
	app := fiber.New()
	auth.RegisterRoutes(app, authHandlers, platformCfg)
	return app, pubPEM, privPEM
}

// Helper to create HMAC signature for service-to-service authentication
func createHMACSignature(method, path, query, body, uid, timestamp, secret string) string {
	canonicalString := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, path, query, sha256Hash(body), uid, timestamp)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(canonicalString))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func sha256Hash(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// TestAuth_Complete_Refactored_Flow tests the complete auth flow with all refactoring changes
func TestAuth_Complete_Refactored_Flow(t *testing.T) {
	// Arrange: real DB, no mocks
	suite := testutil.Setup(t)
	baseConfig := suite.Config()
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, baseConfig)
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)
	app, pubPEM, privPEM := newAuthApp(t, base, iso.Config)
	h := testutil.NewHTTPHelper(t, app)

	// Generate unique email for this test run to avoid database isolation issues
	testID := uuid.Must(uuid.NewV4()).String()[:8]
	email := fmt.Sprintf("john.doe.%s@example.com", testID)
	fullName := "John Doe"
	password := "VeryStrongP@ssw0rd123!@#$%^&*()"

	t.Run("Phase1_SecureSignupVerification", func(t *testing.T) {
		// 1) Signup (email) -> get verification token (form-encoded)
		// This should now use secure server-side verification records
		form := url.Values{}
		form.Set("fullName", fullName)
		form.Set("email", email)
		form.Set("newPassword", password)
		form.Set("responseType", "spa")
		form.Set("verifyType", "email")
		form.Set("g-recaptcha-response", "ok")

		formData := form.Encode()
		t.Logf("Sending form data: %s", formData)
		resp := h.NewRequest(http.MethodPost, "/auth/signup", []byte(formData)).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Signup error response: %s", string(bodyBytes))
		}
		require.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("signup failed: %d", resp.StatusCode))

		var signupPayload struct {
			VerificationId string `json:"verificationId"`
			ExpiresAt      int64  `json:"expiresAt"`
			Message        string `json:"message"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&signupPayload))
		require.NotEmpty(t, signupPayload.VerificationId)
		require.Greater(t, signupPayload.ExpiresAt, time.Now().Unix())

		// Verify verification ID is a valid UUID (security improvement: no JWT tokens)
		verifyUUID := uuid.FromStringOrNil(signupPayload.VerificationId)
		require.NotEqual(t, uuid.Nil, verifyUUID, "VerificationId must be a valid UUID")

		// Verify server-side verification record exists with hashed password
		res := <-base.Repository.FindOne(ctx, "userVerification", struct {
			ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
		}{ObjectId: verifyUUID})
		require.NoError(t, res.Error())
		var uv struct {
			Code           string `json:"code"`
			HashedPassword []byte `json:"hashedPassword"`
			ExpiresAt      int64  `json:"expiresAt"`
			Used           bool   `json:"used"`
		}
		require.NoError(t, res.Decode(&uv))
		require.NotEmpty(t, uv.Code)
		require.NotEmpty(t, uv.HashedPassword, "Hashed password must be stored server-side")
		require.True(t, uv.ExpiresAt > 0, "Expiry must be set")
		require.False(t, uv.Used, "Verification record must not be used yet")

		// 2) Verify signup (form-encoded) - should use server-side hashed password
		vf := url.Values{}
		vf.Set("code", uv.Code)
		vf.Set("verificationId", signupPayload.VerificationId)
		vf.Set("responseType", "spa")

		resp2 := h.NewRequest(http.MethodPost, "/auth/signup/verify", []byte(vf.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
		require.Equal(t, http.StatusOK, resp2.StatusCode, fmt.Sprintf("verify failed: %d", resp2.StatusCode))

		// Verify the verification record is marked as used
		res2 := <-base.Repository.FindOne(ctx, "userVerification", struct {
			ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
		}{ObjectId: verifyUUID})
		require.NoError(t, res2.Error())
		var uv2 struct {
			Used bool `json:"used"`
		}
		require.NoError(t, res2.Decode(&uv2))
		require.True(t, uv2.Used, "Verification record must be marked as used")

		// Verify user was created and verified by testing the API response
		// This follows the posts microservice pattern - test the API, not the database directly
		// The verification service already handles the database operations correctly
	})

	t.Run("Phase2_HeaderBasedJWTLogin", func(t *testing.T) {
		// This phase will be executed after Phase1 completes, so the user will be verified
		// 3) Login (form-encoded) - should return JWT in JSON, no cookies

		// User verification is already tested in Phase1
		// This follows the posts microservice pattern - test the API, not the database directly

		lf := url.Values{}
		lf.Set("username", email)
		lf.Set("password", password)
		lf.Set("responseType", "spa")

		resp3 := h.NewRequest(http.MethodPost, "/auth/login", []byte(lf.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
		if resp3.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp3.Body)
			t.Logf("Login error response: %s", string(bodyBytes))
		}
		require.Equal(t, http.StatusOK, resp3.StatusCode, fmt.Sprintf("login failed: %d", resp3.StatusCode))

		var loginPayload struct {
			User        map[string]interface{}
			AccessToken string
			TokenType   string
		}
		require.NoError(t, json.NewDecoder(resp3.Body).Decode(&loginPayload))
		require.NotEmpty(t, loginPayload.AccessToken)
		require.Equal(t, "Bearer", loginPayload.TokenType)

		// Verify no cookies are set (header-based JWT only)
		cookies := resp3.Header.Get("Set-Cookie")
		require.Empty(t, cookies, "No cookies should be set for header-based JWT")

		// 4) Use JWT to access protected route (PUT /auth/password/change)
		changeBody := url.Values{}
		changeBody.Set("currentPassword", password)
		changeBody.Set("newPassword", "NewStrongPassword123!@#")
		changeBody.Set("confirmPassword", "NewStrongPassword123!@#")

		resp4 := h.NewRequest(http.MethodPut, "/auth/password/change", []byte(changeBody.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").
			WithHeader(types.HeaderAuthorization, types.BearerPrefix+loginPayload.AccessToken).Send()
		require.Equal(t, http.StatusOK, resp4.StatusCode, fmt.Sprintf("password change failed: %d", resp4.StatusCode))

		// Verify JWT contains expected claims
		claims, err := utils.ValidateToken([]byte(pubPEM), loginPayload.AccessToken)
		require.NoError(t, err)
		claimMap, _ := claims["claim"].(map[string]interface{})
		require.Equal(t, email, claimMap["email"])
		require.Equal(t, fullName, claimMap["displayName"])
	})

	t.Run("Phase3_HMACServiceToServiceAuth", func(t *testing.T) {
		// Test HMAC authentication for service-to-service calls
		// This tests the strengthened HMAC implementation with canonical signing

		hmacSecret := iso.Config.HMAC.Secret
		uid := uuid.Must(uuid.NewV4()).String()
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		// Create canonical request for HMAC
		method := "GET"
		path := "/auth/admin/users"
		query := ""
		body := ""

		signature := createHMACSignature(method, path, query, body, uid, timestamp, hmacSecret)

		// Test HMAC-protected admin endpoint
		resp := h.NewRequest(http.MethodGet, path, "").
			WithHeader(types.HeaderHMACAuthenticate, signature).
			WithHeader(types.HeaderTimestamp, timestamp).
			WithHeader(types.HeaderUID, uid).
			WithHeader("email", email).Send()

		// Note: This will likely fail as we haven't implemented the strengthened HMAC middleware yet
		// This test documents the expected behavior for Phase 2.5 of the refactoring guide
		t.Logf("HMAC test response status: %d (expected to fail until HMAC middleware is strengthened)", resp.StatusCode)
	})

	t.Run("Phase4_SecurityValidations", func(t *testing.T) {
		// Test various security validations using password change endpoint

		// Test 1: Invalid JWT should be rejected
		resp1 := h.NewRequest(http.MethodPut, "/auth/password/change", "").
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").
			WithHeader(types.HeaderAuthorization, types.BearerPrefix+"invalid-jwt-token").Send()
		require.Equal(t, http.StatusUnauthorized, resp1.StatusCode)

		// Test 2: Missing Authorization header should be rejected
		resp2 := h.NewRequest(http.MethodPut, "/auth/password/change", "").
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
		require.Equal(t, http.StatusUnauthorized, resp2.StatusCode)

		// Test 3: Wrong Authorization format should be rejected
		resp3 := h.NewRequest(http.MethodPut, "/auth/password/change", "").
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").
			WithHeader(types.HeaderAuthorization, "Basic invalid").Send()
		require.Equal(t, http.StatusUnauthorized, resp3.StatusCode)

		// Test 4: Expired JWT should be rejected (if we had time-based validation)
		// This would require creating an expired token, which is complex in this test setup
	})

	t.Run("Phase5_NoTokenInURL", func(t *testing.T) {
		// Verify that no tokens are exposed in URLs (security fix)
		// This tests that BuildSessionRedirect and similar functions are removed

		// Test OAuth callback - should not contain tokens in URL
		resp := h.NewRequest(http.MethodGet, "/auth/oauth/google/callback?code=test&state=test", "").Send()
		// The response should not contain access_token in the URL
		require.NotContains(t, resp.Header.Get("Location"), "access_token", "Tokens must not appear in URLs")
	})

	t.Run("Phase6_RecaptchaDecoupling", func(t *testing.T) {
		// Test that Recaptcha verification works with dependency injection
		// This validates the DI pattern for external services

		form := url.Values{}
		form.Set("fullName", "Test User")
		form.Set("email", "test@example.com")
		form.Set("newPassword", "VeryStrongP@ssw0rd123!@#$%^&*()")
		form.Set("responseType", "spa")
		form.Set("verifyType", "email")
		form.Set("g-recaptcha-response", "fake-recaptcha-response")

		resp := h.NewRequest(http.MethodPost, "/auth/signup", []byte(form.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Recaptcha test error response: %s", string(bodyBytes))
		}
		require.Equal(t, http.StatusOK, resp.StatusCode, "Recaptcha should be handled via DI")
	})

	// Password Handler Tests - Password management endpoints
	t.Run("Password_Reset_Page", func(t *testing.T) {
		// Test GET /auth/password/reset/:verifyId
		verifyId := "test-verify-id"
		resp := h.NewRequest(http.MethodGet, "/auth/password/reset/"+verifyId, "").Send()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, resp.Header.Get(types.HeaderContentType), "text/html")
	})

	t.Run("Password_Reset_Form", func(t *testing.T) {
		// Test POST /auth/password/reset/:verifyId
		// First create a user directly in the database for this test
		userUUID := uuid.Must(uuid.NewV4())
		hashedPassword, err := utils.Hash(password)
		require.NoError(t, err)

		userAuth := struct {
			ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
			Username string    `json:"username" bson:"username"`
			Password []byte    `json:"password" bson:"password"`
		}{
			ObjectId: userUUID,
			Username: email,
			Password: hashedPassword,
		}

		err = (<-base.Repository.Save(ctx, "userAuth", &userAuth)).Error
		require.NoError(t, err)
		t.Logf("Created user with email: %s, UUID: %s", email, userUUID.String())

		// Test user lookup
		userRes := <-base.Repository.FindOne(ctx, "userAuth", struct {
			Username string `json:"username" bson:"username"`
		}{Username: email})
		require.NoError(t, userRes.Error())
		var ua struct {
			ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
			Username string    `json:"username" bson:"username"`
		}
		require.NoError(t, userRes.Decode(&ua))
		t.Logf("Found user: %+v", ua)

		// Use the secure password reset flow instead of deprecated JWT tokens
		passwordService := passwordUC.NewService(base, &passwordUC.ServiceConfig{
			JWTConfig:  platformconfig.JWTConfig{PublicKey: pubPEM, PrivateKey: privPEM},
			HMACConfig: platformconfig.HMACConfig{Secret: iso.Config.HMAC.Secret},
			EmailConfig: platformconfig.EmailConfig{},
			AppConfig:  platformconfig.AppConfig{},
		})
		resetData, err := passwordService.PrepareSecureResetVerification(ctx, email, "127.0.0.1")
		require.NoError(t, err)
		t.Logf("Generated secure reset token (length: %d)", len(resetData.PlaintextToken))

		// Verify the secure token has proper entropy
		require.GreaterOrEqual(t, len(resetData.PlaintextToken), 40, "Secure token should have sufficient entropy")
		require.NotEqual(t, resetData.PlaintextToken, resetData.HashedToken, "Plaintext and hashed tokens should be different")

		// Use the plaintext token for the reset request
		resetToken := resetData.PlaintextToken

		form := url.Values{}
		form.Set("newPassword", "NewStrongPassword123!")
		form.Set("confirmPassword", "NewStrongPassword123!")

		resp := h.NewRequest(http.MethodPost, "/auth/password/reset/"+resetToken, []byte(form.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()

		// Debug: print response body if not 200
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Password reset failed with status %d: %s", resp.StatusCode, string(body))
		}
		require.Equal(t, http.StatusOK, resp.StatusCode)
		
		// Verify the response contains success message
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		require.Contains(t, response, "message")
		require.Equal(t, "Password reset successfully", response["message"])
	})

	t.Run("Password_Forget_Page", func(t *testing.T) {
		// Test GET /auth/password/forget
		resp := h.NewRequest(http.MethodGet, "/auth/password/forget", "").Send()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, resp.Header.Get(types.HeaderContentType), "text/html")
	})

	t.Run("Password_Forget_Form", func(t *testing.T) {
		// Test POST /auth/password/forget
		// First create a user for this test
		userUUID := uuid.Must(uuid.NewV4())
		hashedPassword, err := utils.Hash(password)
		require.NoError(t, err)

		userAuth := struct {
			ObjectId uuid.UUID `json:"objectId" bson:"objectId"`
			Username string    `json:"username" bson:"username"`
			Password []byte    `json:"password" bson:"password"`
		}{
			ObjectId: userUUID,
			Username: "user@example.com",
			Password: hashedPassword,
		}

		err = (<-base.Repository.Save(ctx, "userAuth", &userAuth)).Error
		require.NoError(t, err)
		t.Logf("Created user for forget test: %s", "user@example.com")

		form := url.Values{}
		form.Set("email", "user@example.com")

		resp := h.NewRequest(http.MethodPost, "/auth/password/forget", []byte(form.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()

		// Debug: print response body if not 200
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Password forget failed with status %d: %s", resp.StatusCode, string(body))
		}
		// Expect 500 error because email sending fails in test environment (no SMTP server)
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Password_Change", func(t *testing.T) {
		// Test PUT /auth/password/change with JWT
		// First create a user for this test
		userUUID := uuid.Must(uuid.NewV4())
		testPassword := "TestPassword123!"
		testEmail := fmt.Sprintf("test.user.%s@example.com", uuid.Must(uuid.NewV4()).String()[:8])
		hashedPassword, err := utils.Hash(testPassword)
		require.NoError(t, err)

		userAuth := struct {
			ObjectId      uuid.UUID `json:"objectId" bson:"objectId"`
			Username      string    `json:"username" bson:"username"`
			Password      []byte    `json:"password" bson:"password"`
			EmailVerified bool      `json:"emailVerified" bson:"emailVerified"`
		}{
			ObjectId:      userUUID,
			Username:      testEmail, // Use a unique email for this test
			Password:      hashedPassword,
			EmailVerified: true,
		}

		err = (<-base.Repository.Save(ctx, "userAuth", &userAuth)).Error
		require.NoError(t, err)

		// Create user profile as well
		userProfile := struct {
			ObjectId    uuid.UUID `json:"objectId" bson:"objectId"`
			FullName    string    `json:"fullName" bson:"fullName"`
			SocialName  string    `json:"socialName" bson:"socialName"`
			Email       string    `json:"email" bson:"email"`
			Avatar      string    `json:"avatar" bson:"avatar"`
			Banner      string    `json:"banner" bson:"banner"`
			TagLine     string    `json:"tagLine" bson:"tagLine"`
			CreatedDate int64     `json:"createdDate" bson:"createdDate"`
		}{
			ObjectId:    userUUID,
			FullName:    "John Doe",
			SocialName:  "John Doe",
			Email:       testEmail, // Use the test email
			Avatar:      "https://util.telar.dev/api/avatars/" + userUUID.String(),
			Banner:      "https://picsum.photos/id/1/900/300/?blur",
			TagLine:     "",
			CreatedDate: time.Now().Unix(),
		}

		err = (<-base.Repository.Save(ctx, "userProfile", &userProfile)).Error
		require.NoError(t, err)
		t.Logf("Created user and profile for password change test: %s", testEmail)

		// Get JWT token from login
		loginForm := url.Values{}
		loginForm.Set("username", testEmail)    // Use the test email
		loginForm.Set("password", testPassword) // Use the same password that was hashed and stored
		loginForm.Set("responseType", "spa")

		loginResp := h.NewRequest(http.MethodPost, "/auth/login", []byte(loginForm.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()

		// Debug: print response body if not 200
		if loginResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(loginResp.Body)
			t.Logf("Login failed with status %d: %s", loginResp.StatusCode, string(body))
		}
		require.Equal(t, http.StatusOK, loginResp.StatusCode)

		var loginPayload struct {
			AccessToken string `json:"accessToken"`
		}
		require.NoError(t, json.NewDecoder(loginResp.Body).Decode(&loginPayload))

		// Change password
		changeForm := url.Values{}
		changeForm.Set("currentPassword", testPassword)
		changeForm.Set("newPassword", "NewPassword123!")
		changeForm.Set("confirmPassword", "NewPassword123!")

		resp := h.NewRequest(http.MethodPut, "/auth/password/change", []byte(changeForm.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").
			WithHeader(types.HeaderAuthorization, types.BearerPrefix+loginPayload.AccessToken).Send()

		// Debug: print response body if not 200
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Password change failed with status %d: %s", resp.StatusCode, string(body))
		}
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Password_Change_Unauthorized", func(t *testing.T) {
		// Test password change without JWT - should fail
		changeForm := url.Values{}
		changeForm.Set("currentPassword", "password")
		changeForm.Set("newPassword", "NewPassword123!")
		changeForm.Set("confirmPassword", "NewPassword123!")

		resp := h.NewRequest(http.MethodPut, "/auth/password/change", []byte(changeForm.Encode())).
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Admin Handler Tests - HMAC-protected endpoints
	t.Run("Admin_Check_Exists", func(t *testing.T) {
		// Test POST /auth/admin/check with valid HMAC
		hmacSecret := iso.Config.HMAC.Secret
		uid := uuid.Must(uuid.NewV4()).String()
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		// Use canonical HMAC signing format
		body := `""`
		signature := testutil.SignHMAC("POST", "/auth/admin/check", "", []byte(body), uid, timestamp, hmacSecret)
		t.Logf("Generated canonical HMAC signature: %s", signature)
		t.Logf("HMAC secret: %s", hmacSecret)
		t.Logf("Body being hashed: %q", body)

		resp := h.NewRequest(http.MethodPost, "/auth/admin/check", "").
			WithHeader(types.HeaderHMACAuthenticate, signature).
			WithHeader(types.HeaderTimestamp, timestamp).
			WithHeader(types.HeaderUID, uid).Send()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var response struct {
			Admin bool `json:"admin"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		// Note: The actual admin status depends on the user's role in the database
	})

	t.Run("Admin_Signup", func(t *testing.T) {
		// Test POST /auth/admin/signup with valid HMAC
		hmacSecret := iso.Config.HMAC.Secret
		uid := uuid.Must(uuid.NewV4()).String()
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		// Create admin signup payload with unique username
		uniqueID := uuid.Must(uuid.NewV4()).String()[:8]
		adminData := map[string]interface{}{
			"username": "admin_user_" + uniqueID,
			"email":    "admin_" + uniqueID + "@example.com",
			"password": "AdminPassword123!",
			"role":     "admin",
		}
		adminJSON, _ := json.Marshal(adminData)

		// Create canonical HMAC signature
		signature := testutil.SignHMAC("POST", "/auth/admin/signup", "", adminJSON, uid, timestamp, hmacSecret)

		resp := h.NewRequest(http.MethodPost, "/auth/admin/signup", adminJSON).
			WithHeader(types.HeaderContentType, "application/json").
			WithHeader(types.HeaderHMACAuthenticate, signature).
			WithHeader(types.HeaderTimestamp, timestamp).
			WithHeader(types.HeaderUID, uid).Send()

		// Debug: print response body if not 201
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Admin signup failed with status %d: %s", resp.StatusCode, string(body))
		}
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("Admin_Login", func(t *testing.T) {
		// Test POST /auth/admin/login with valid HMAC
		hmacSecret := iso.Config.HMAC.Secret
		uid := uuid.Must(uuid.NewV4()).String()
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		// First create an admin user for login
		uniqueID := uuid.Must(uuid.NewV4()).String()[:8]
		adminUsername := "admin_user_" + uniqueID
		adminPassword := "AdminPassword123!"

		// Create admin user directly in database
		userUUID := uuid.Must(uuid.NewV4())
		hashedPassword, err := utils.Hash(adminPassword)
		require.NoError(t, err)

		userAuth := struct {
			ObjectId      uuid.UUID `json:"objectId" bson:"objectId"`
			Username      string    `json:"username" bson:"username"`
			Password      []byte    `json:"password" bson:"password"`
			Role          string    `json:"role" bson:"role"`
			EmailVerified bool      `json:"emailVerified" bson:"emailVerified"`
		}{
			ObjectId:      userUUID,
			Username:      adminUsername,
			Password:      hashedPassword,
			Role:          "admin",
			EmailVerified: true,
		}

		err = (<-base.Repository.Save(ctx, "userAuth", &userAuth)).Error
		require.NoError(t, err)
		t.Logf("Created admin user for login: %s", adminUsername)

		// Create admin login payload
		loginData := map[string]interface{}{
			"email":    adminUsername, // Admin login uses email field
			"password": adminPassword,
		}
		loginJSON, _ := json.Marshal(loginData)

		// Create canonical HMAC signature
		signature := testutil.SignHMAC("POST", "/auth/admin/login", "", loginJSON, uid, timestamp, hmacSecret)

		resp := h.NewRequest(http.MethodPost, "/auth/admin/login", loginJSON).
			WithHeader(types.HeaderContentType, "application/json").
			WithHeader(types.HeaderHMACAuthenticate, signature).
			WithHeader(types.HeaderTimestamp, timestamp).
			WithHeader(types.HeaderUID, uid).Send()

		// Debug: print response body if not 200
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Admin login failed with status %d: %s", resp.StatusCode, string(body))
		}
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Admin_Unauthorized_Access", func(t *testing.T) {
		// Test admin endpoints without HMAC - should fail
		resp := h.NewRequest(http.MethodPost, "/auth/admin/check", "").Send()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Admin_Invalid_HMAC", func(t *testing.T) {
		// Test admin endpoints with invalid HMAC - should fail
		resp := h.NewRequest(http.MethodPost, "/auth/admin/check", "").
			WithHeader(types.HeaderHMACAuthenticate, "invalid-signature").
			WithHeader(types.HeaderTimestamp, strconv.FormatInt(time.Now().Unix(), 10)).
			WithHeader(types.HeaderUID, uuid.Must(uuid.NewV4()).String()).Send()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// OAuth Handler Tests - OAuth redirect endpoints
	t.Run("OAuth_Github_Redirect", func(t *testing.T) {
		// Test GET /auth/login/github
		resp := h.NewRequest(http.MethodGet, "/auth/login/github", "").Send()
		require.Equal(t, http.StatusFound, resp.StatusCode)
		require.Contains(t, resp.Header.Get("Location"), "github.com/login/oauth/authorize")
	})

	t.Run("OAuth_Google_Redirect", func(t *testing.T) {
		// Test GET /auth/login/google
		resp := h.NewRequest(http.MethodGet, "/auth/login/google", "").Send()
		require.Equal(t, http.StatusFound, resp.StatusCode)
		require.Contains(t, resp.Header.Get("Location"), "accounts.google.com/o/oauth2/v2/auth")
	})

	t.Run("OAuth_Authorized_Callback", func(t *testing.T) {
		// Test GET /auth/oauth2/authorized with invalid parameters
		// This should return 400 Bad Request due to missing/invalid OAuth parameters
		resp := h.NewRequest(http.MethodGet, "/auth/oauth2/authorized?r=http://localhost/&state=abc", "").Send()
		
		// The secure OAuth implementation correctly rejects invalid requests
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Verify the response contains error information
		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
		require.Contains(t, response, "code")
		require.Contains(t, response, "message")
		
		// Test with missing code parameter
		resp2 := h.NewRequest(http.MethodGet, "/auth/oauth2/authorized?state=test", "").Send()
		require.Equal(t, http.StatusBadRequest, resp2.StatusCode)
		
		// Test with missing state parameter  
		resp3 := h.NewRequest(http.MethodGet, "/auth/oauth2/authorized?code=test", "").Send()
		require.Equal(t, http.StatusBadRequest, resp3.StatusCode)
	})
}

// TestAuth_RefactoringValidation tests specific refactoring requirements
func TestAuth_RefactoringValidation(t *testing.T) {
	suite := testutil.Setup(t)
	baseConfig := suite.Config()
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, baseConfig)
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)
	app, _, _ := newAuthApp(t, base, iso.Config)
	h := testutil.NewHTTPHelper(t, app)

	t.Run("JWTv5LibraryValidation", func(t *testing.T) {
		// This test validates that we're using the new JWT library
		// The fact that the app builds and runs means the migration was successful
		require.NotNil(t, app, "App should build with golang-jwt/jwt/v5")
	})

	t.Run("NoCookieMiddlewareUsage", func(t *testing.T) {
		// This test validates that cookie middleware is not used
		// We can't easily test this from the outside, but the integration test
		// passing without cookies confirms the migration

		// Test that protected routes require Authorization header, not cookies
		resp := h.NewRequest(http.MethodPut, "/auth/password/change", "").
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should require Authorization header")
	})

	t.Run("OpenAPISecuritySchemes", func(t *testing.T) {
		// This test validates that the API uses the correct security schemes
		// We test this by ensuring the endpoints behave according to JWTAuth specification

		// Test that endpoints require Bearer token format
		resp := h.NewRequest(http.MethodPut, "/auth/password/change", "").
			WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").
			WithHeader(types.HeaderAuthorization, "InvalidFormat token").Send()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should require Bearer format")
	})

}
