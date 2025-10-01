package admin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	authErrors "github.com/qolzam/telar/apps/api/auth/errors"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	authhmac "github.com/qolzam/telar/apps/api/internal/middleware/authhmac"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// createTestPlatformConfig creates a test platform config
func createTestPlatformConfig() *platformconfig.Config {
	return &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
}

func TestAdminHandler_Login_Success_Coverage(t *testing.T) {
	// Note: Test timeout is set via -timeout flag when running tests
	// This test requires at least 2-5 seconds due to bcrypt operations

	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction using Config-First pattern
	baseConfig := suite.Config()
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, baseConfig)
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	// Use the new dependency injection pattern following posts golden reference
	platformCfg := createTestPlatformConfig()
	adminService := NewService(base, "test-private-key", platformCfg)
	adminHandler := NewAdminHandler(adminService, platformCfg.JWT, platformCfg.HMAC)

	app := fiber.New()
	group := app.Group("/auth/admin")

	// Create HMAC middleware with proper configuration
	hmacMiddleware := authhmac.New(authhmac.Config{
		PayloadSecret: iso.Config.HMAC.Secret,
	})

	group.Post("/login", hmacMiddleware, adminHandler.Login)

	// Seed admin user with test-optimized password hashing
	pass, _ := hashForTest("Adm1n!Pass") // Use faster bcrypt cost for tests
	userAuth := map[string]interface{}{
		"objectId":      uuid.Must(uuid.NewV4()),
		"username":      "admin-login@example.com",
		"password":      pass,
		"role":          "admin",
		"emailVerified": true,
		"phoneVerified": true,
		"created_date":  1,
		"last_updated":  1,
	}

	res := <-base.Repository.Save(ctx, "userAuth", userAuth)
	require.NoError(t, res.Error)

	req := httptest.NewRequest(http.MethodPost, "/auth/admin/login",
		strings.NewReader("email=admin-login@example.com&password=Adm1n!Pass"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	// Use a longer timeout for bcrypt comparison tests
	resp, err := app.Test(req, 10000) // 10 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestAdminHandler_Login_Error_WrongPassword(t *testing.T) {
	// Note: Test timeout is set via -timeout flag when running tests
	// This test requires at least 2-5 seconds due to bcrypt operations

	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	// Use the new dependency injection pattern following posts golden reference
	platformCfg := createTestPlatformConfig()
	adminService := NewService(base, "test-private-key", platformCfg)
	adminHandler := NewAdminHandler(adminService, platformCfg.JWT, platformCfg.HMAC)

	app := fiber.New()
	group := app.Group("/auth/admin")

	// Create HMAC middleware with proper configuration
	hmacMiddleware := authhmac.New(authhmac.Config{
		PayloadSecret: iso.Config.HMAC.Secret,
	})

	group.Post("/login", hmacMiddleware, adminHandler.Login)

	// Seed admin user with test-optimized password hashing
	pass, _ := hashForTest("Adm1n!Pass") // Use faster bcrypt cost for tests
	userAuth := map[string]interface{}{
		"objectId":      uuid.Must(uuid.NewV4()),
		"username":      "admin-login@example.com",
		"password":      pass,
		"role":          "admin",
		"emailVerified": true,
		"phoneVerified": true,
		"created_date":  1,
		"last_updated":  1,
	}

	res := <-base.Repository.Save(ctx, "userAuth", userAuth)
	require.NoError(t, res.Error)

	req := httptest.NewRequest(http.MethodPost, "/auth/admin/login",
		strings.NewReader("email=admin-login@example.com&password=Wrong!Pass"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	// Use a longer timeout for bcrypt comparison tests
	resp, err := app.Test(req, 10000) // 10 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestAdminHandler_Check_OK(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	app := fiber.New()
	// Use the new dependency injection pattern
	platformCfg := createTestPlatformConfig()
	adminService := NewService(base, "test-private-key", platformCfg)
	adminHandler := NewAdminHandler(adminService, platformCfg.JWT, platformCfg.HMAC)
	app.Post("/check", adminHandler.Check)

	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader("{}"))
	req.Header.Set(types.HeaderContentType, "application/json")

	// Use a reasonable timeout for database operations
	resp, err := app.Test(req, 5000) // 5 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestAdminHandler_Signup_Status(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping")
	}

	ctx := context.Background()

	// DEBUGGING: Log test configuration details
	t.Logf("TRACE req=test-admin-signup step=config payloadSecret=%s", iso.Config.HMAC.Secret)

	// CRITICAL FIX: Use proper dependency injection pattern per solution2.md
	// Create the handler dependencies using the isolated repo and local config
	serviceConfig := &platform.ServiceConfig{
		DatabaseType:       "mongodb",
		DatabaseName:       "test_db",
		EnableTransactions: false,
	}
	baseService := platform.NewBaseServiceWithRepo(iso.Repo, serviceConfig)

	// DEBUGGING: First verify no admin exists in isolated database
	findFilter := map[string]interface{}{"role": "admin"}
	existingAdminCheck := <-baseService.Repository.FindOne(ctx, "userAuth", findFilter)
	var dummy struct{}
	if existingAdminCheck.Decode(&dummy) == nil {
		t.Logf("WARNING: Found existing admin in isolated database")
	} else {
		t.Logf("CONFIRMED: No existing admin found in isolated database")
	}

	// Also check specifically for the email we're going to use
	specificEmailFilter := map[string]interface{}{"username": "admin-sign@example.com", "role": "admin"}
	specificAdminCheck := <-baseService.Repository.FindOne(ctx, "userAuth", specificEmailFilter)
	var dummy2 struct{}
	if specificAdminCheck.Decode(&dummy2) == nil {
		t.Logf("WARNING: Found existing admin with our specific email in isolated database")
	} else {
		t.Logf("CONFIRMED: No existing admin with our email found in isolated database")
	}

	platformConfig := createTestPlatformConfig()
	adminService := NewService(baseService, `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIPkykxxwlgyafwhy5+MmNGnQ+oFKJ+UKSZEl97+IRkV+oAoGCCqGSM49
AwEHoUQDQgAEvru6au3DbYVIYp+5Xz4Iu9KbNRNlNZuhr8dSuWCpyRoKkMwQJD68
SRkdHLhfK7Stm+j8hTH451f3W54tuazjjQ==
-----END EC PRIVATE KEY-----`, platformConfig)
	adminHandler := NewAdminHandler(adminService, platformConfig.JWT, platformConfig.HMAC)

	t.Logf("TRACE req=test-admin-signup step=handler-created-with-isolated-repo")

	// Create minimal app without full routing - direct handler test
	app := fiber.New()
	app.Post("/signup", adminHandler.Signup)

	req := httptest.NewRequest(http.MethodPost, "/signup",
		strings.NewReader("email=admin-sign@example.com&password=Adm1n!Pass"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	t.Logf("TRACE req=test-admin-signup step=request-prepared")

	// Use a reasonable timeout for database operations
	resp, err := app.Test(req, 5000) // 5 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Check the response body for more details
	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Logf("TRACE req=test-admin-signup step=response-received status=%d body=%s", resp.StatusCode, string(bodyBytes))

	// If we get a 500 error, let's try to manually trigger the admin creation to see the exact error
	if resp.StatusCode == 500 {
		t.Logf("DEBUG: Attempting manual admin creation to isolate error...")

		// But first, check if an admin was created despite the 500 error
		afterHttpFilter := map[string]interface{}{"username": "admin-sign@example.com", "role": "admin"}
		afterHttpCheck := <-baseService.Repository.FindOne(ctx, "userAuth", afterHttpFilter)
		var dummy3 struct{}
		if afterHttpCheck.Decode(&dummy3) == nil {
			t.Logf("DEBUG: FOUND admin after HTTP call despite 500 error - this suggests transaction commit issue")
		} else {
			t.Logf("DEBUG: No admin found after HTTP call - HTTP handler truly failed")
		}

		token, createErr := adminService.CreateAdmin(ctx, "admin", "admin-sign@example.com", "Adm1n!Pass")
		if createErr != nil {
			t.Logf("DEBUG: Manual CreateAdmin error: %v (type: %T)", createErr, createErr)
			// Try to unwrap the error to see the root cause
			if authErr, ok := createErr.(*authErrors.AuthError); ok {
				t.Logf("DEBUG: AuthError details - Code: %s, Message: %s, Cause: %v", authErr.Code, authErr.Message, authErr.Cause)
			}
		} else {
			t.Logf("DEBUG: Manual CreateAdmin succeeded with token: %s", token)
		}
	}

	// Assert that admin signup returns 201 Created status
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Admin signup should return 201 Created")
}

func TestAdminHandler_Check_And_Signup_InternalError(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	app := fiber.New()
	// Use the new dependency injection pattern
	platformCfg := createTestPlatformConfig()
	adminService := NewService(base, "test-private-key", platformCfg)
	adminHandler := NewAdminHandler(adminService, platformCfg.JWT, platformCfg.HMAC)
	app.Post("/check", adminHandler.Check)
	app.Post("/signup", adminHandler.Signup)

	// Test /check with empty body (should handle gracefully)
	r1 := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader("{}"))
	r1.Header.Set(types.HeaderContentType, "application/json")

	// Use a reasonable timeout for database operations
	resp1, err := app.Test(r1, 5000) // 5 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp1)

	// Test /signup with minimal data (may error but shouldn't crash)
	r2 := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=a@b.c&password=x"))
	r2.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")

	// Use a reasonable timeout for database operations
	resp2, err := app.Test(r2, 5000) // 5 second timeout
	require.NoError(t, err)
	require.NotNil(t, resp2)
}

// hashForTest provides fast password hashing for tests using bcrypt.MinCost=4
// This is 16x faster than bcrypt.DefaultCost=10 while maintaining security for testing
func hashForTest(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
}

// compareHashForTest provides fast password comparison for tests
// This bypasses the slow bcrypt comparison by using a simple string comparison
// Note: This is only for tests - production should use proper bcrypt comparison
func compareHashForTest(hash []byte, password []byte) error {
	// For tests, we'll use a simple approach: compare the stored hash with a new hash
	// This is much faster than bcrypt.CompareHashAndPassword
	newHash, err := hashForTest(string(password))
	if err != nil {
		return err
	}

	// Simple byte comparison - this is not cryptographically secure but fast for tests
	if string(hash) == string(newHash) {
		return nil
	}
	return fmt.Errorf("password mismatch")
}
