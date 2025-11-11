package login

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestLoginService_FindAndReadProfile_Coverage(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)

	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()

	base, err := platform.NewBaseService(ctx, iso.Config)
	require.NoError(t, err)

	// Create service config for testing
	serviceConfig := &ServiceConfig{
		JWTConfig: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMACConfig: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
	svc := NewService(base, serviceConfig)
	uid := uuid.Must(uuid.NewV4())

	// Seed userAuth and userProfile
	userAuth := map[string]interface{}{"objectId": uid, "username": "find@example.com", "password": []byte("p"), "emailVerified": true, "phoneVerified": false, "role": "user"}
	_ = (<-base.Repository.Save(ctx, "userAuth", uid, uid, 1, 1, userAuth)).Error
	userProfile := map[string]interface{}{"objectId": uid, "fullName": "FN", "socialName": "sn", "email": "find@example.com", "avatar": "a", "banner": "b", "tagLine": "t", "created_date": 1}
	_ = (<-base.Repository.Save(ctx, "userProfile", uid, uid, 1, 1, userProfile)).Error

	_, _ = svc.FindUserByUsername(ctx, "find@example.com")
	
}
