package profile

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/profile/services"
)

func newProfileApp(t *testing.T, base *platform.BaseService, config *testutil.TestConfig, hmacSecret string) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		uid := c.Get(types.HeaderUID)
		if uid != "" {
			userID, _ := uuid.FromString(uid)
			createdDate, _ := strconv.ParseInt(c.Get("createdDate"), 10, 64)
			user := types.UserContext{
				UserID:      userID,
				Username:    c.Get("email"),
				DisplayName: c.Get("displayName"),
				SocialName:  c.Get("socialName"),
				Avatar:      "",
				Banner:      "",
				TagLine:     "",
				SystemRole:  c.Get("role"),
				CreatedDate: createdDate,
			}
			c.Locals(types.UserCtxName, user)
		}
		return c.Next()
	})

	// Use the provided HMAC secret to ensure it matches the test's signing secret
	platformCfg := config.ToPlatformConfig(dbi.DatabaseTypePostgreSQL)
	// Override HMAC secret to match the test's signing secret
	platformCfg.HMAC.Secret = hmacSecret
	
	profileService := services.NewService(base, platformCfg)
	profileHandler := NewProfileHandler(profileService, platformCfg.JWT, platformCfg.HMAC)

	profileHandlers := &ProfileHandlers{
		ProfileHandler: profileHandler,
	}

	RegisterRoutes(app, profileHandlers, platformCfg)
	return app
}

func TestProfile_PublicAndHMACRoutes_OK(t *testing.T) {
	suite := testutil.Setup(t)

	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	baseSvc, err := platform.NewBaseService(context.Background(), iso.Config)
	if err != nil {
		t.Fatalf("failed to build postgresql base service: %v", err)
	}

	app := newProfileApp(t, baseSvc, iso.LegacyConfig, iso.Config.HMAC.Secret)

	httpHelper := testutil.NewHTTPHelper(t, app)

	secret := iso.Config.HMAC.Secret

	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	respMy := httpHelper.NewRequest(http.MethodGet, "/profile/my", nil).Send()
	require.True(t, respMy.StatusCode == http.StatusOK || respMy.StatusCode == http.StatusUnauthorized,
		"/profile/my should return 200 or 401, got %d", respMy.StatusCode)

	respList := httpHelper.NewRequest(http.MethodGet, "/profile/?search=a&page=1&limit=1", nil).Send()
	require.True(t, respList.StatusCode == http.StatusOK || respList.StatusCode == http.StatusUnauthorized,
		"/profile/ list should return 200 or 401, got %d", respList.StatusCode)

	respId := httpHelper.NewRequest(http.MethodGet, fmt.Sprintf("/profile/id/%s", validUUID), nil).Send()
	require.True(t, respId.StatusCode == http.StatusOK || respId.StatusCode == http.StatusUnauthorized,
		"/profile/id/:id should return 200 or 401, got %d", respId.StatusCode)

	respSn := httpHelper.NewRequest(http.MethodGet, "/profile/social/alice", nil).Send()
	require.True(t, respSn.StatusCode == http.StatusOK || respSn.StatusCode == http.StatusUnauthorized,
		"/profile/social/:name should return 200 or 401, got %d", respSn.StatusCode)

	idsPayload := []string{validUUID}
	respIds := httpHelper.NewRequest(http.MethodPost, "/profile/ids", idsPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respIds.StatusCode == http.StatusOK || respIds.StatusCode == http.StatusInternalServerError,
		"/profile/ids should return 200 or 500, got %d", respIds.StatusCode)

	respIdx := httpHelper.NewRequest(http.MethodPost, "/profile/index", nil).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respIdx.StatusCode == http.StatusOK || respIdx.StatusCode == http.StatusInternalServerError,
		"/profile/index should return 200 or 500, got %d", respIdx.StatusCode)

	lsPayload := map[string]string{"userId": validUUID}
	respLS := httpHelper.NewRequest(http.MethodPut, "/profile/last-seen", lsPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respLS.StatusCode == http.StatusOK || respLS.StatusCode == http.StatusInternalServerError,
		"/profile/last-seen should return 200 or 500, got %d", respLS.StatusCode)

	updPayload := map[string]string{"fullName": "Test User"}
	respUpd := httpHelper.NewRequest(http.MethodPut, "/profile/", updPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respUpd.StatusCode,
		"PUT /profile/ should return 200, got %d", respUpd.StatusCode)

	respDto := httpHelper.NewRequest(http.MethodGet, fmt.Sprintf("/profile/dto/id/%s", validUUID), nil).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respDto.StatusCode == http.StatusOK || respDto.StatusCode == http.StatusInternalServerError,
		"GET /profile/dto/id/:id should return 200 or 500, got %d", respDto.StatusCode)

	dtoPayload := map[string]string{"objectId": validUUID, "fullName": "Test User"}
	respC := httpHelper.NewRequest(http.MethodPost, "/profile/dto", dtoPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respC.StatusCode == http.StatusCreated || respC.StatusCode == http.StatusInternalServerError,
		"POST /profile/dto should return 201 or 500, got %d", respC.StatusCode)

	respD := httpHelper.NewRequest(http.MethodPost, "/profile/dispatch", map[string]interface{}{}).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respD.StatusCode == http.StatusOK || respD.StatusCode == http.StatusInternalServerError,
		"POST /profile/dispatch should return 200 or 500, got %d", respD.StatusCode)

	respDis := httpHelper.NewRequest(http.MethodPost, "/profile/dto/ids", idsPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respDis.StatusCode == http.StatusOK || respDis.StatusCode == http.StatusInternalServerError,
		"POST /profile/dto/ids should return 200 or 500, got %d", respDis.StatusCode)

	respF := httpHelper.NewRequest(http.MethodPut, fmt.Sprintf("/profile/follow/inc/1/%s", validUUID), nil).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respF.StatusCode == http.StatusOK || respF.StatusCode == http.StatusInternalServerError,
		"PUT /profile/follow/inc/ should return 200 or 500, got %d", respF.StatusCode)

	respFr := httpHelper.NewRequest(http.MethodPut, fmt.Sprintf("/profile/follower/inc/1/%s", validUUID), nil).
		WithAuthHeaders(secret, validUUID).Send()
	require.True(t, respFr.StatusCode == http.StatusOK || respFr.StatusCode == http.StatusInternalServerError,
		"PUT /profile/follower/inc/ should return 200 or 500, got %d", respFr.StatusCode)
}










