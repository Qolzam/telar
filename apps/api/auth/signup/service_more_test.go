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
	"github.com/qolzam/telar/apps/api/internal/utils"
)

func TestSignupService_SaveAndUpdateVerification_Coverage(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
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
	verifyId := uuid.Must(uuid.NewV4())
	doc := &models.UserVerification{
		ObjectId:    verifyId,
		UserId:      uuid.Must(uuid.NewV4()),
		Code:        "000000",
		Target:      "u@example.com",
		TargetType:  "email",
		Counter:     1,
		CreatedDate: utils.UTCNowUnix(),
	}
	_ = s.SaveUserVerification(ctx, doc)
	_ = s.UpdateVerification(ctx, &models.DatabaseFilter{ObjectId: &verifyId}, &models.DatabaseUpdate{Set: map[string]interface{}{"counter": 2}})
}
