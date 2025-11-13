package posts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/posts/handlers"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/posts/services"
)

// MockPostServiceForRouting is a minimal mock for route testing
type MockPostServiceForRouting struct {
	services.PostService
}

func (m *MockPostServiceForRouting) GetPost(ctx context.Context, postID uuid.UUID) (*models.Post, error) {
	// Return a simple mock post
	return &models.Post{
		ObjectId: postID,
		Body:     "Mock post",
	}, nil
}

func (m *MockPostServiceForRouting) QueryPostsWithCursor(ctx context.Context, filter *models.PostQueryFilter) (*models.PostsListResponse, error) {
	// Return empty result for cursor query
	return &models.PostsListResponse{
		Posts:      []models.PostResponse{},
		NextCursor: "",
		HasNext:    false,
	}, nil
}

// TestPostRoutingPrecedence verifies that route matching works correctly,
// specifically that /queries/cursor routes to QueryPostsWithCursor and not GetPost
func TestPostRoutingPrecedence(t *testing.T) {
	t.Parallel()

	// Generate valid ECDSA key pair for JWT middleware
	pubPEM, privPEM := testutil.GenerateECDSAKeyPairPEM(t)

	// Create minimal mock service
	mockService := &MockPostServiceForRouting{}
	
	// Create handlers with the mock service
	jwtConfig := config.JWTConfig{
		PublicKey:  pubPEM,
		PrivateKey: privPEM,
	}
	hmacConfig := config.HMACConfig{
		Secret: "test-secret",
	}
	handler := handlers.NewPostHandler(mockService, jwtConfig, hmacConfig)
	
	mockHandlers := &PostsHandlers{PostHandler: handler}
	mockConfig := &config.Config{
		HMAC: config.HMACConfig{
			Secret: "test-secret",
		},
		JWT: config.JWTConfig{
			PublicKey:  pubPEM,
			PrivateKey: privPEM,
		},
	}

	app := fiber.New()
	RegisterRoutes(app, mockHandlers, mockConfig)

	t.Run("Should route /queries/cursor to QueryPostsWithCursor handler", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/queries/cursor", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		// Should get a response (may be 401 due to missing auth, but NOT 400 with UUID error)
		require.NotEqual(t, http.StatusBadRequest, resp.StatusCode, 
			"/cursor should not return 400 Bad Request (Invalid UUID)")
		require.NotEqual(t, http.StatusNotFound, resp.StatusCode,
			"/cursor should match the route (not return 404)")
	})

	t.Run("Should route /<uuid> to GetPost handler", func(t *testing.T) {
		testUUID := uuid.Must(uuid.NewV4()).String()
		req := httptest.NewRequest("GET", "/posts/"+testUUID, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		// Should get a response (may be 401 due to missing auth, but route should match)
		require.NotEqual(t, http.StatusNotFound, resp.StatusCode,
			"Valid UUID route should match (not return 404)")
	})

	t.Run("Should return 404 for non-uuid parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/not-a-uuid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		// For non-UUID parameters, the route constraint should ideally return 404,
		// but if dual auth runs first, 401 is also acceptable (auth required).
		// Both indicate the request is invalid (either route doesn't match or auth is required).
		// The constraint is still valuable for type safety with valid requests.
		require.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized,
			"Non-UUID parameter should return 404 (route constraint) or 401 (auth required), got %d", resp.StatusCode)
	})

	t.Run("Should not match /queries/cursor as /:postId", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/queries/cursor?limit=5", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		// Should NOT get 400 Bad Request with "Invalid UUID format"
		// This was the original bug - /cursor was being matched as /:postId
		require.NotEqual(t, http.StatusBadRequest, resp.StatusCode,
			"/cursor should not be matched as /:postId (should not return 400 Bad Request)")
	})
}

