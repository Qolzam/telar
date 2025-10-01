package jwks_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/auth/jwks"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWKS_Endpoint_Basic(t *testing.T) {
	// Use a minimal test - just verify the endpoint responds
	handler := jwks.NewHandler("invalid-key-for-test", "test-key-id")

	app := fiber.New()
	app.Get("/.well-known/jwks.json", handler.Handle)

	h := testutil.NewHTTPHelper(t, app)

	t.Run("JWKS_Endpoint_Exists", func(t *testing.T) {
		resp := h.NewRequest("GET", "/.well-known/jwks.json", nil).Send()

		// Should respond (even if with error due to invalid key)
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)

		// Check response is JSON
		contentType := resp.Header.Get(types.HeaderContentType)
		if resp.StatusCode == http.StatusOK {
			assert.Contains(t, contentType, "application/json")
		}

		// If it's an error response, it should be JSON error format
		if resp.StatusCode >= 400 {
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var errorResp map[string]interface{}
			err = json.Unmarshal(body, &errorResp)
			require.NoError(t, err)

			// Should have error structure
			assert.Contains(t, errorResp, "code")
			assert.Contains(t, errorResp, "message")
		}
	})

	t.Run("JWKS_Handler_Creation", func(t *testing.T) {
		// Test handler creation doesn't panic
		handler := jwks.NewHandler("test-key", "test-id")
		assert.NotNil(t, handler)
	})
}

func TestJWKS_Types(t *testing.T) {
	t.Run("JWKS_Struct", func(t *testing.T) {
		jwks := jwks.JWKS{
			Keys: []jwks.JWK{
				{
					Kty: "EC",
					Use: "sig",
					Kid: "test-key",
					Alg: "ES256",
					Crv: "P-256",
					X:   "test-x",
					Y:   "test-y",
				},
			},
		}

		// Test JSON marshaling
		data, err := json.Marshal(jwks)
		require.NoError(t, err)
		assert.Contains(t, string(data), "test-key")
		assert.Contains(t, string(data), "EC")
		assert.Contains(t, string(data), "ES256")
	})
}
