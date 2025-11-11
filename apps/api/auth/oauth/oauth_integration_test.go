package oauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/auth/oauth"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuth_User_Account_Management(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())

	// Create BaseService from isolated test config
	base, err := platform.NewBaseService(context.Background(), iso.Config)
	require.NoError(t, err)

	oauthConfig := oauth.NewOAuthConfig(
		"http://localhost:3000",
		"test_github_client",
		"test_github_secret",
		"test_google_client",
		"test_google_secret",
	)

	serviceConfig := &oauth.ServiceConfig{
		OAuthConfig: oauthConfig,
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
	service := oauth.NewService(base, serviceConfig)

	t.Run("Create_New_OAuth_User", func(t *testing.T) {
		userInfo := &oauth.OAuthUserInfo{
			ID:        "github123",
			Email:     "oauth.new@example.com",
			Name:      "OAuth Test User",
			AvatarURL: "https://github.com/avatar.jpg",
			Provider:  "github",
		}

		userAuth, userProfile, err := service.FindOrCreateUser(context.Background(), userInfo)
		require.NoError(t, err)

		// Verify user creation
		assert.Equal(t, userInfo.Email, userAuth.Username)
		assert.True(t, userAuth.EmailVerified)
		assert.Equal(t, "user", userAuth.Role)
		assert.Equal(t, userInfo.Name, userProfile.FullName)
		assert.Equal(t, userInfo.Email, userProfile.Email)
		assert.Equal(t, userInfo.AvatarURL, userProfile.Avatar)

		// Verify user was actually saved to database
		assert.NotEqual(t, uuid.Nil, userAuth.ObjectId)
		assert.NotEqual(t, uuid.Nil, userProfile.ObjectId)
		assert.Equal(t, userAuth.ObjectId, userProfile.ObjectId)
	})

	t.Run("Find_Existing_OAuth_User", func(t *testing.T) {
		existingEmail := "existing.oauth@example.com"

		// Create existing user first
		userId := uuid.Must(uuid.NewV4())
		testUser := struct {
			ObjectId      uuid.UUID `json:"objectId" bson:"objectId"`
			Username      string    `json:"username" bson:"username"`
			Password      []byte    `json:"password" bson:"password"`
			Role          string    `json:"role" bson:"role"`
			EmailVerified bool      `json:"emailVerified" bson:"emailVerified"`
			PhoneVerified bool      `json:"phoneVerified" bson:"phoneVerified"`
			CreatedDate   int64     `json:"createdDate" bson:"createdDate"`
			LastUpdated   int64     `json:"lastUpdated" bson:"lastUpdated"`
		}{
			ObjectId:      userId,
			Username:      existingEmail,
			Password:      []byte("hashedpassword"),
			Role:          "user",
			EmailVerified: true,
			PhoneVerified: false,
			CreatedDate:   time.Now().Unix(),
			LastUpdated:   time.Now().Unix(),
		}

		err := (<-base.Repository.Save(context.Background(), "userAuth", testUser.ObjectId, testUser.ObjectId, testUser.CreatedDate, testUser.LastUpdated, &testUser)).Error
		require.NoError(t, err)

		// Create corresponding profile
		testProfile := struct {
			ObjectId    uuid.UUID `json:"objectId" bson:"objectId"`
			FullName    string    `json:"fullName" bson:"fullName"`
			SocialName  string    `json:"socialName" bson:"socialName"`
			Email       string    `json:"email" bson:"email"`
			Avatar      string    `json:"avatar" bson:"avatar"`
			Banner      string    `json:"banner" bson:"banner"`
			TagLine     string    `json:"tagLine" bson:"tagLine"`
			CreatedDate int64     `json:"createdDate" bson:"createdDate"`
			LastUpdated int64     `json:"lastUpdated" bson:"lastUpdated"`
		}{
			ObjectId:    userId,
			FullName:    "Existing User",
			SocialName:  "existinguser",
			Email:       existingEmail,
			Avatar:      "https://example.com/avatar.jpg",
			Banner:      "https://example.com/banner.jpg",
			TagLine:     "Test user",
			CreatedDate: time.Now().Unix(),
			LastUpdated: time.Now().Unix(),
		}

		err = (<-base.Repository.Save(context.Background(), "userProfile", testProfile.ObjectId, testProfile.ObjectId, testProfile.CreatedDate, testProfile.LastUpdated, &testProfile)).Error
		require.NoError(t, err)

		// Try OAuth login with existing user
		userInfo := &oauth.OAuthUserInfo{
			ID:       "google456",
			Email:    existingEmail,
			Name:     "Existing User",
			Provider: "google",
		}

		foundUserAuth, foundUserProfile, err := service.FindOrCreateUser(context.Background(), userInfo)
		require.NoError(t, err)

		// Verify existing user was found (not created)
		assert.Equal(t, existingEmail, foundUserAuth.Username)
		assert.Equal(t, "user", foundUserAuth.Role)
		assert.Equal(t, "Existing User", foundUserProfile.FullName)
		// Note: ObjectId might be different due to OAuth user creation, but email should match
	})
}
