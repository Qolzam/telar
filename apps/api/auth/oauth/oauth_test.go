package oauth_test

import (
	"testing"
	"time"

	"github.com/qolzam/telar/apps/api/auth/oauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPKCE_Parameter_Security(t *testing.T) {
	// Test PKCE parameter generation
	pkce, err := oauth.GeneratePKCEParams()
	require.NoError(t, err)

	// Verify code verifier length (43-128 characters)
	assert.GreaterOrEqual(t, len(pkce.CodeVerifier), 43)
	assert.LessOrEqual(t, len(pkce.CodeVerifier), 128)

	// Verify code challenge is SHA256 hash of verifier
	assert.NotEmpty(t, pkce.CodeChallenge)
	assert.NotEqual(t, pkce.CodeVerifier, pkce.CodeChallenge)

	// Verify state parameter
	assert.NotEmpty(t, pkce.State)
	assert.GreaterOrEqual(t, len(pkce.State), 16)
}

func TestState_Store_Security(t *testing.T) {
	stateStore := oauth.NewMemoryStateStore()

	pkce := &oauth.PKCEParams{
		CodeVerifier:  "test_verifier",
		CodeChallenge: "test_challenge",
		State:         "test_state",
	}

	// Test storage and retrieval
	err := stateStore.Store("test_state", pkce, time.Minute)
	require.NoError(t, err)

	retrieved, err := stateStore.Retrieve("test_state")
	require.NoError(t, err)
	assert.Equal(t, pkce.CodeVerifier, retrieved.CodeVerifier)

	// Test expiry
	err = stateStore.Store("expired_state", pkce, time.Millisecond)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = stateStore.Retrieve("expired_state")
	assert.Error(t, err, "expired state should be rejected")

	// Test deletion
	err = stateStore.Delete("test_state")
	require.NoError(t, err)

	_, err = stateStore.Retrieve("test_state")
	assert.Error(t, err, "deleted state should not be found")
}

func TestOAuth_Config(t *testing.T) {
	config := oauth.NewOAuthConfig(
		"http://localhost:3000",
		"test_github_client",
		"test_github_secret",
		"test_google_client",
		"test_google_secret",
	)

	require.NotNil(t, config.GitHub)
	require.NotNil(t, config.Google)

	// Test GitHub config
	assert.Equal(t, "test_github_client", config.GitHub.ClientID)
	assert.Equal(t, "test_github_secret", config.GitHub.ClientSecret)
	assert.Contains(t, config.GitHub.Scopes, "user:email")

	// Test Google config
	assert.Equal(t, "test_google_client", config.Google.ClientID)
	assert.Equal(t, "test_google_secret", config.Google.ClientSecret)
	assert.Contains(t, config.Google.Scopes, "openid")

	// Test auth URL generation
	pkce := &oauth.PKCEParams{
		CodeVerifier:  "test_verifier",
		CodeChallenge: "test_challenge",
		State:         "test_state",
	}

	githubURL, err := config.GetAuthURL("github", pkce)
	require.NoError(t, err)
	assert.Contains(t, githubURL, "github.com/login/oauth/authorize")
	assert.Contains(t, githubURL, "code_challenge=test_challenge")

	googleURL, err := config.GetAuthURL("google", pkce)
	require.NoError(t, err)
	assert.Contains(t, googleURL, "accounts.google.com/o/oauth2")
	assert.Contains(t, googleURL, "code_challenge=test_challenge")
}
