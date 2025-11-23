package comments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/qolzam/telar/apps/api/comments/handlers"
	"github.com/qolzam/telar/apps/api/comments/models"
	"github.com/qolzam/telar/apps/api/comments/services"
	"github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
)

type mockCommentServiceForRoutes struct {
	services.CommentService
}

func (m *mockCommentServiceForRoutes) GetComment(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
	return &models.Comment{
		ObjectId: id,
		Text:     "test",
	}, nil
}

func (m *mockCommentServiceForRoutes) GetCommentsByPost(ctx context.Context, postID uuid.UUID, filter *models.CommentQueryFilter) (*models.CommentsListResponse, error) {
	return &models.CommentsListResponse{
		Comments: []models.CommentResponse{},
		Page:     1,
		Limit:    10,
	}, nil
}

func (m *mockCommentServiceForRoutes) CreateComment(ctx context.Context, req *models.CreateCommentRequest, user *types.UserContext) (*models.Comment, error) {
	return &models.Comment{
		ObjectId: uuid.Must(uuid.NewV4()),
		PostId:   req.PostId,
		Text:     req.Text,
	}, nil
}

func TestCommentsRoutesPrecedence(t *testing.T) {
	t.Parallel()

	pubKey, privKey := testutil.GenerateECDSAKeyPairPEM(t)
	cfg := &config.Config{
		HMAC: config.HMACConfig{Secret: "test-secret"},
		JWT: config.JWTConfig{
			PublicKey:  pubKey,
			PrivateKey: privKey,
		},
	}

	service := &mockCommentServiceForRoutes{}
	handler := handlers.NewCommentHandler(service, cfg.JWT, cfg.HMAC)
	routes := &CommentsHandlers{
		CommentHandler: handler,
	}

	app := fiber.New()
	RegisterRoutes(app, routes, cfg)

	t.Run("GET comment by ID should hit handler", func(t *testing.T) {
		id := uuid.Must(uuid.NewV4()).String()
		req := httptest.NewRequest(http.MethodGet, "/comments/"+id, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.NotEqual(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Invalid UUID should be rejected before handler logic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/comments/not-a-uuid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound)
	})

	t.Run("POST route should be mounted under /comments/", func(t *testing.T) {
		body := strings.NewReader(`{"postId":"` + uuid.Must(uuid.NewV4()).String() + `","text":"hello"}`)
		req := httptest.NewRequest(http.MethodPost, "/comments/", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.NotEqual(t, http.StatusNotFound, resp.StatusCode)
	})
}
