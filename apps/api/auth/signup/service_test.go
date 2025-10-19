package signup

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/models"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"

	"github.com/qolzam/telar/apps/api/internal/testutil"
)

func TestSignupService_Tokens_Coverage(t *testing.T) {
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
	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		AppConfig: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}
	s := NewService(base, serviceConfig)
	uid := uuid.Must(uuid.NewV4())
	// Test the new secure verification methods
	_, _ = s.InitiateEmailVerification(ctx, EmailVerificationRequest{
		UserId:          uid,
		EmailTo:         "u@example.com",
		RemoteIpAddress: "127.0.0.1",
		FullName:        "U",
		UserPassword:    "p",
	})
	_, _ = s.InitiatePhoneVerification(ctx, PhoneVerificationRequest{
		UserId:          uid,
		PhoneNumber:     "+1234567890",
		FullName:        "U",
		UserPassword:    "p",
		RemoteIpAddress: "127.0.0.1",
	})
	_ = s.UpdateVerification(ctx, &models.DatabaseFilter{ObjectId: &uid}, &models.DatabaseUpdate{Set: map[string]interface{}{"counter": 2}})
}
