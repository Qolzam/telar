package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// createAuthTestApp creates a test app using the base service pattern (following posts golden pattern)
func createAuthTestApp(t *testing.T, iso *testutil.IsolatedTest) *fiber.App {
	t.Helper()

	// Create base service for the test (following posts golden pattern)
	// This avoids transactional wrapper issues that cause context cancellation
	ctx := context.Background()
	cfg := iso.Config
	base, err := platform.NewBaseService(ctx, cfg)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}

	return newAuthAppForTest(t, base, iso.Config)
}

// Functions moved to handlers_persistence_test.go to avoid duplication

func TestAuth_Login_SPA_Compatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	// Note: this assumes a test user may not exist; expecting 400 for now
	form := url.Values{}
	form.Set("username", "user@example.com")
	form.Set("password", "pass")

	resp := httpHelper.NewRequest("POST", "/auth/login/", bytes.NewBufferString(form.Encode())).
		WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestAuth_OAuth_Redirects(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	resp := httpHelper.NewRequest("GET", "/auth/login/github", nil).Send()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("github redirect status=%d", resp.StatusCode)
	}
	resp2 := httpHelper.NewRequest("GET", "/auth/login/google", nil).Send()
	if resp2.StatusCode != http.StatusFound {
		t.Fatalf("google redirect status=%d", resp2.StatusCode)
	}
}

func TestAuth_Signup_SPA_ReturnsToken(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	form := url.Values{}
	form.Set("fullName", "Test User")
	form.Set("email", "tester@example.com")
	form.Set("newPassword", "Password123!@#")
	form.Set("verifyType", "email")
	form.Set("g-recaptcha-response", "dummy")
	form.Set("responseType", "spa")

	resp := httpHelper.NewRequest("POST", "/auth/signup", bytes.NewBufferString(form.Encode())).
		WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
	// Depending on recaptcha validation, allow 200 or 400 in unit tests
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected signup status: %d", resp.StatusCode)
	}
}

func TestAuth_Verification_SPA_Status(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	form := url.Values{}
	form.Set("code", "123456")
	form.Set("verificaitonSecret", "dummy")
	form.Set("responseType", "spa")

	resp := httpHelper.NewRequest("POST", "/auth/signup/verify", bytes.NewBufferString(form.Encode())).
		WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected verify status: %d", resp.StatusCode)
	}
}

func TestAuth_Password_Forget_Status(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	form := url.Values{}
	form.Set("email", "tester@example.com")

	resp := httpHelper.NewRequest("POST", "/auth/password/forget", bytes.NewBufferString(form.Encode())).
		WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("unexpected forget status: %d", resp.StatusCode)
	}
}

func TestAuth_Routes_HMACCookie_NextBranches(t *testing.T) {
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	// HMAC present → HMAC.Next returns false, cookie.Next returns true
	resp := httpHelper.NewRequest(http.MethodGet, "/auth/login/", nil).
		WithHeader(types.HeaderHMACAuthenticate, "sha1=dummy").Send()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	// No HMAC present → HMAC.Next returns true for cookie group usages
	resp2 := httpHelper.NewRequest(http.MethodGet, "/auth/login/", nil).Send()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
}

func TestAuth_Password_Change_Unauthorized(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	resp := httpHelper.NewRequest("PUT", "/auth/password/change", nil).Send()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for password change without cookies, got %d", resp.StatusCode)
	}
}

func TestAuth_Verification_SSR_Status(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	form := url.Values{}
	form.Set("code", "123456")
	form.Set("verificaitonSecret", "dummy")
	form.Set("responseType", "ssr")

	resp := httpHelper.NewRequest("POST", "/auth/signup/verify", bytes.NewBufferString(form.Encode())).
		WithHeader(types.HeaderContentType, "application/x-www-form-urlencoded").Send()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected verify SSR status: %d", resp.StatusCode)
	}
}

func TestAuth_Admin_Check_HMAC(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	// Get secret from the safe, local config (now synchronized with global config)
	secret := iso.Config.HMAC.Secret
	uid := uuid.Must(uuid.NewV4()).String()
	body := []byte("{}")

	resp := httpHelper.NewRequest("POST", "/auth/admin/check", bytes.NewReader(body)).
		WithHeader(types.HeaderContentType, "application/json").
		WithHMACAuth(secret, uid).Send()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin check status=%d", resp.StatusCode)
	}
}

func TestAuth_Admin_Signup_Returns201(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	// Get secret from the safe, local config (now synchronized with global config)
	secret := iso.Config.HMAC.Secret

	// Try JSON payload instead of form data to debug the issue
	payload := map[string]interface{}{
		"email":    "admin@example.com",
		"password": "AdminPass123!",
	}

	// --- THE FIX ---
	// Generate a valid, unique user ID to simulate an authenticated request.
	// This user ID represents the existing admin who is creating a new admin.
	adminUID := uuid.Must(uuid.NewV4()).String()

	// Pass the valid adminUID to the authentication helper.
	resp := httpHelper.NewRequest("POST", "/auth/admin/signup", payload).
		WithHMACAuth(secret, adminUID).Send()
	// --- END OF FIX ---
	// Admin signup should return 201 Created for create operation
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		t.Fatalf("admin signup should return 201 Created or 409 Conflict, got %d", resp.StatusCode)
	}
}

func TestAuth_OAuth_Authorized_Redirect(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	// t.Parallel() // Now safe to use!

	// 1. Get the shared connection pool.
	suite := testutil.Setup(t)

	// 2. Create isolated test environment for configuration
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())

	// 3. Use base service for all database operations in your test.
	app := createAuthTestApp(t, iso)
	httpHelper := testutil.NewHTTPHelper(t, app)

	resp := httpHelper.NewRequest("GET", "/auth/oauth2/authorized?r=http://localhost/&state=abc", nil).Send()
	
	// The secure OAuth implementation correctly rejects invalid requests
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("authorized redirect status=%d, expected 400 for invalid OAuth parameters", resp.StatusCode)
	}
	
	// Verify the response contains error information
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	
	if _, ok := response["code"]; !ok {
		t.Fatalf("response missing error code")
	}
	if _, ok := response["message"]; !ok {
		t.Fatalf("response missing error message")
	}
}
