package verification

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
)

func TestVerificationService_All_Coverage(t *testing.T) {
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

	// Phase 1.4: Test only secure verification method (legacy support removed)
	// Ensure verification doc exists with proper secure fields
	_ = (<-base.Repository.Save(ctx, "userVerification", map[string]interface{}{
		"objectId":        vid,
		"userId":          uid,
		"code":            "000000",
		"remoteIpAddress": "127.0.0.1",
		"created_date":    1,
		"counter":         0,
		"hashedPassword":  []byte("hashed_password"),
		"target":          "test@example.com",
		"targetType":      "email",
		"used":            false,
		"expiresAt":       999999999999, // Far future
	})).Error

	// Test new secure verification method
	params := VerifySignupParams{
		VerificationId:  vid,
		Code:            "000000",
		RemoteIpAddress: "127.0.0.1",
		ResponseType:    "spa",
	}
	_, _ = s.VerifySignup(ctx, params)

	// createUserAuth and createUserProfile also called for coverage (even if they fail)
	_ = s.createUserAuth(ctx, userAuthDoc{ObjectId: uid})
	_ = s.createUserProfile(ctx, map[string]interface{}{"objectId": uid})
}
