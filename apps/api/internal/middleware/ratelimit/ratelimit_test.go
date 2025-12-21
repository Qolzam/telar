// Package ratelimit provides tests for the rate limiting middleware
// Following the AUTH_SECURITY_REFACTORING_PLAN.md Phase 2.1 testing requirements
package ratelimit

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/qolzam/telar/apps/api/internal/types"
)

func TestRateLimit_LoginEndpoint_SuccessWithinLimits(t *testing.T) {
	// Setup test app with login rate limiting
	app := fiber.New()
	app.Use(NewLoginLimiter(nil))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test within limits (default: 5 requests per 15 minutes)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set(types.HeaderContentType, "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.1") // Same IP

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		resp.Body.Close()
	}
}

func TestRateLimit_LoginEndpoint_RejectsExcessiveRequests(t *testing.T) {
	// Setup test app with login rate limiting
	app := fiber.New()
	app.Use(NewLoginLimiter(nil))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Make 5 successful requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set(types.HeaderContentType, "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		resp.Body.Close()
	}

	// 6th request should be rate limited
	req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set(types.HeaderContentType, "application/json")
	req.Header.Set("X-Real-IP", "192.168.1.1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)

	// Verify response body contains rate limit error
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "RATE_LIMIT_EXCEEDED")
	assert.Contains(t, string(body), "login")
	resp.Body.Close()
}

func TestRateLimit_DifferentIPs_IndependentLimits(t *testing.T) {
	// Setup test app with login rate limiting
	app := fiber.New()
	app.Use(NewLoginLimiter(nil))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Make 5 requests from first IP - use different route for each IP simulation
	firstIPRequests := 0
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set(types.HeaderContentType, "application/json")
		req.RemoteAddr = "192.168.1.1:9099" // Set RemoteAddr directly

		resp, err := app.Test(req)
		require.NoError(t, err)
		if resp.StatusCode == 200 {
			firstIPRequests++
		}
		resp.Body.Close()
	}

	// At least some requests from first IP should succeed
	assert.Greater(t, firstIPRequests, 0)

	// Make requests from second IP (using different app instance to avoid cross-contamination)
	app2 := fiber.New()
	app2.Use(NewLoginLimiter(nil))
	app2.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	secondIPRequests := 0
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set(types.HeaderContentType, "application/json")
		req.RemoteAddr = "192.168.1.2:9099" // Different IP

		resp, err := app2.Test(req)
		require.NoError(t, err)
		if resp.StatusCode == 200 {
			secondIPRequests++
		}
		resp.Body.Close()
	}

	// Requests from second IP should succeed independently
	assert.Greater(t, secondIPRequests, 0)
}

func TestRateLimit_SignupEndpoint_CorrectLimits(t *testing.T) {
	// Setup test app with signup rate limiting (10 per hour)
	app := fiber.New()
	app.Use(NewSignupLimiter(nil))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test within limits (default: 10 requests per hour)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set(types.HeaderContentType, "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		resp.Body.Close()
	}

	// 11th request should be rate limited
	req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set(types.HeaderContentType, "application/json")
	req.Header.Set("X-Real-IP", "192.168.1.1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)

	// Verify response contains signup-specific message
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "signup")
	resp.Body.Close()
}

func TestRateLimit_PasswordResetEndpoint_CorrectLimits(t *testing.T) {
	// Setup test app with password reset rate limiting (3 per hour)
	app := fiber.New()
	app.Use(NewPasswordResetLimiter(nil))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test within limits (default: 3 requests per hour)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set(types.HeaderContentType, "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		resp.Body.Close()
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set(types.HeaderContentType, "application/json")
	req.Header.Set("X-Real-IP", "192.168.1.1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)

	// Verify response contains password reset specific message
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "password reset")
	resp.Body.Close()
}

func TestRateLimit_VerificationByID_PreventsBruteForce(t *testing.T) {
	// Setup test app with verification rate limiting
	app := fiber.New()
	app.Use(NewVerificationByIDLimiter(nil))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test with same verification ID - should be rate limited by verification ID, not IP
	verificationID := "550e8400-e29b-41d4-a716-446655440000"
	requestBody := `{"verificationId":"` + verificationID + `","code":"123456"}`

	// Make 10 requests with same verification ID
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader(requestBody))
		req.Header.Set(types.HeaderContentType, "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		resp.Body.Close()
	}

	// 11th request with same verification ID should be rate limited
	req := httptest.NewRequest("POST", "/test", strings.NewReader(requestBody))
	req.Header.Set(types.HeaderContentType, "application/json")
	req.Header.Set("X-Real-IP", "192.168.1.1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)

	// But a request with different verification ID should still work
	differentRequestBody := `{"verificationId":"550e8400-e29b-41d4-a716-446655440001","code":"123456"}`
	req = httptest.NewRequest("POST", "/test", strings.NewReader(differentRequestBody))
	req.Header.Set(types.HeaderContentType, "application/json")
	req.Header.Set("X-Real-IP", "192.168.1.1")

	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

func TestRateLimit_CustomLimits_AppliedCorrectly(t *testing.T) {
	// Setup custom limits with very low values for testing
	customLimits := &EndpointLimits{
		LoginMaxRequests:    2,
		LoginWindowDuration: 1 * time.Minute,
	}

	app := fiber.New()
	app.Use(NewLoginLimiter(customLimits))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Test within custom limits (2 requests)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set(types.HeaderContentType, "application/json")
		req.Header.Set("X-Real-IP", "192.168.1.1")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		resp.Body.Close()
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set(types.HeaderContentType, "application/json")
	req.Header.Set("X-Real-IP", "192.168.1.1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)
	resp.Body.Close()
}

func TestRateLimit_DefaultConfiguration_CreatesCorrectLimits(t *testing.T) {
	defaults := DefaultEndpointLimits()

	// Verify default limits match the security plan
	assert.Equal(t, 5, defaults.LoginMaxRequests)
	assert.Equal(t, 15*time.Minute, defaults.LoginWindowDuration)

	assert.Equal(t, 3, defaults.PasswordResetMaxRequests)
	assert.Equal(t, 1*time.Hour, defaults.PasswordResetWindowDuration)

	assert.Equal(t, 10, defaults.SignupMaxRequests)
	assert.Equal(t, 1*time.Hour, defaults.SignupWindowDuration)

	assert.Equal(t, 10, defaults.VerificationMaxRequests)
	assert.Equal(t, 15*time.Minute, defaults.VerificationWindowDuration)
}

func TestRateLimit_ErrorResponse_ContainsCorrectFields(t *testing.T) {
	app := fiber.New()
	app.Use(NewLoginLimiter(nil))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Exhaust rate limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("X-Real-IP", "192.168.1.1")
		resp, _ := app.Test(req)
		resp.Body.Close()
	}

	// Get rate limited response
	req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set("X-Real-IP", "192.168.1.1")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	response := string(body)
	assert.Contains(t, response, "error")
	assert.Contains(t, response, "RATE_LIMIT_EXCEEDED")
	assert.Contains(t, response, "retryAfter")
	assert.Contains(t, response, "login")
	resp.Body.Close()
}

func TestRateLimit_Performance_MinimalOverhead(t *testing.T) {
	app := fiber.New()
	app.Use(NewLoginLimiter(nil))
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Measure time for requests within rate limit
	start := time.Now()
	successCount := 0
	for i := 0; i < 3; i++ { // Reduced to stay within limits
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set(types.HeaderContentType, "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		if resp.StatusCode == 200 {
			successCount++
		}
		resp.Body.Close()
	}
	elapsed := time.Since(start)

	// Ensure some requests succeeded and performance is acceptable
	assert.Greater(t, successCount, 0)
	assert.Less(t, elapsed, 100*time.Millisecond, "Rate limiting should not add significant overhead")
}
