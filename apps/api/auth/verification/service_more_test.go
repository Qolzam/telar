package verification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
)

func TestVerificationService_SuccessBranch_Coverage(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypeMongoDB, suite.Config())
	if iso.Repo == nil {
		t.Skip("MongoDB not available, skipping test")
	}

	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	if err != nil {
		t.Skip("no repo")
	}
	config := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: iso.Config.HMAC.Secret,
		},
		AppConfig: platformconfig.AppConfig{
			OrgName:   "TestOrg",
			WebDomain: "http://localhost:3000",
		},
	}
	s := NewService(base, config)
	uid := uuid.Must(uuid.NewV4())
	vid := uuid.Must(uuid.NewV4())
	now := utils.UTCNowUnix()

	// Test new secure verification method (Phase 1.4: Legacy support removed)
	// Seed verification doc for success path: not verified and correct code
	_ = (<-base.Repository.Save(ctx, "userVerification", map[string]interface{}{
		"objectId":        vid,
		"userId":          uid,
		"code":            "123456",
		"counter":         0,
		"created_date":    now,
		"remoteIpAddress": "127.0.0.1",
		"hashedPassword":  []byte("hashed_password"),
		"target":          "test@example.com",
		"targetType":      "email",
		"used":            false,
		"expiresAt":       now + 3600,
	})).Error

	// Test secure verification (should handle database operations gracefully even if auth creation fails)
	params := VerifySignupParams{
		VerificationId:  vid,
		Code:            "123456",
		RemoteIpAddress: "127.0.0.1",
		ResponseType:    "spa",
	}
	_, _ = s.VerifySignup(ctx, params)
}

func TestVerification_Handler_SSR_Minimal(t *testing.T) {
	// Create handler with local test configuration
	web := "http://localhost"
	org := "Telar"
	pub := "dummy"

	app := fiber.New()
	handlerConfig := &HandlerConfig{
		PublicKey: pub,
		OrgName:   org,
		WebDomain: web,
	}
	handler := NewHandler(&Service{}, handlerConfig)
	app.Post("/verify", handler.Handle)
	// token is invalid but handler will return 400; path executed
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader("verificaitonSecret=bad&responseType=ssr"))
	req.Header.Set(types.HeaderContentType, "application/x-www-form-urlencoded")
	_, _ = app.Test(req)
}

func TestVerification_generateSocialName(t *testing.T) {
	out := generateSocialName("John Doe", "123e4567-e89b-12d3-a456-426614174000")
	if out == "" {
		t.Fatalf("expected non-empty social name")
	}
}
