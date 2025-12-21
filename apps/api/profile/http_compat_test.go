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
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/profile/repository"
	"github.com/qolzam/telar/apps/api/profile/services"
)

func newProfileApp(t *testing.T, profileRepo repository.ProfileRepository, config *testutil.TestConfig, hmacSecret string) *fiber.App {
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
	
	profileService := services.NewProfileService(profileRepo, platformCfg)
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

	ctx := context.Background()
	
	// Create ProfileRepository using test helper which applies migration automatically
	profileRepo, err := repository.NewPostgresProfileRepositoryForTest(ctx, iso)
	if err != nil {
		t.Fatalf("failed to create ProfileRepository: %v", err)
	}

	app := newProfileApp(t, profileRepo, iso.LegacyConfig, iso.Config.HMAC.Secret)

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

	// Create profile first to ensure deterministic state for subsequent operations
	// Use map[string]interface{} to allow proper JSON unmarshaling of UUID
	dtoPayload := map[string]interface{}{
		"objectId": validUUID,
		"fullName": "Test User",
	}
	respC := httpHelper.NewRequest(http.MethodPost, "/profile/dto", dtoPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusCreated, respC.StatusCode,
		"POST /profile/dto should return 201 Created, got %d", respC.StatusCode)

	// Create index is a no-op (indexes created via migrations), should return 200 OK
	respIdx := httpHelper.NewRequest(http.MethodPost, "/profile/index", nil).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respIdx.StatusCode,
		"POST /profile/index should return 200 OK (no-op), got %d", respIdx.StatusCode)

	// Get profiles by IDs (profile exists, so should return 200 OK with profile data)
	idsPayload := []string{validUUID}
	respIds := httpHelper.NewRequest(http.MethodPost, "/profile/ids", idsPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respIds.StatusCode,
		"POST /profile/ids should return 200 OK (profile exists), got %d", respIds.StatusCode)

	// Now update the profile (it exists, so should return 200 OK)
	updPayload := map[string]string{"fullName": "Updated Test User"}
	respUpd := httpHelper.NewRequest(http.MethodPut, "/profile/", updPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respUpd.StatusCode,
		"PUT /profile/ should return 200 OK (profile exists), got %d", respUpd.StatusCode)

	// Update last seen (profile exists, so should return 200 OK)
	lsPayload := map[string]string{"userId": validUUID}
	respLS := httpHelper.NewRequest(http.MethodPut, "/profile/last-seen", lsPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respLS.StatusCode,
		"PUT /profile/last-seen should return 200 OK (profile exists), got %d", respLS.StatusCode)

	// Get profile DTO (profile exists, so should return 200 OK)
	respDto := httpHelper.NewRequest(http.MethodGet, fmt.Sprintf("/profile/dto/id/%s", validUUID), nil).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respDto.StatusCode,
		"GET /profile/dto/id/:id should return 200 OK (profile exists), got %d", respDto.StatusCode)

	// Dispatch is a no-op endpoint, should return 200 OK
	respD := httpHelper.NewRequest(http.MethodPost, "/profile/dispatch", map[string]interface{}{}).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respD.StatusCode,
		"POST /profile/dispatch should return 200 OK (no-op), got %d", respD.StatusCode)

	// Get profiles by IDs via DTO endpoint (profile exists, so should return 200 OK)
	respDis := httpHelper.NewRequest(http.MethodPost, "/profile/dto/ids", idsPayload).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respDis.StatusCode,
		"POST /profile/dto/ids should return 200 OK (profile exists), got %d", respDis.StatusCode)

	// Increment follow count (profile exists, so should return 200 OK)
	respF := httpHelper.NewRequest(http.MethodPut, fmt.Sprintf("/profile/follow/inc/1/%s", validUUID), nil).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respF.StatusCode,
		"PUT /profile/follow/inc/ should return 200 OK (profile exists), got %d", respF.StatusCode)

	// Increment follower count (profile exists, so should return 200 OK)
	respFr := httpHelper.NewRequest(http.MethodPut, fmt.Sprintf("/profile/follower/inc/1/%s", validUUID), nil).
		WithAuthHeaders(secret, validUUID).Send()
	require.Equal(t, http.StatusOK, respFr.StatusCode,
		"PUT /profile/follower/inc/ should return 200 OK (profile exists), got %d", respFr.StatusCode)
}










