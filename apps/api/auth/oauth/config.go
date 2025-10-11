package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

type OAuthConfig struct {
	GitHub *oauth2.Config
	Google *oauth2.Config
}

type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
	State         string
}

// NewOAuthConfig creates OAuth configurations for all providers
func NewOAuthConfig(baseURL, githubClientID, githubSecret, googleClientID, googleSecret string) *OAuthConfig {
	redirectURL := baseURL + "/auth/oauth2/authorized"

	return &OAuthConfig{
		GitHub: &oauth2.Config{
			ClientID:     githubClientID,
			ClientSecret: githubSecret,
			Endpoint:     github.Endpoint,
			RedirectURL:  redirectURL,
			Scopes:       []string{"user:email", "read:user"},
		},
		Google: &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "profile", "email"},
		},
	}
}

// GeneratePKCEParams generates PKCE parameters for secure OAuth flow
func GeneratePKCEParams() (*PKCEParams, error) {
	// Generate code verifier (43-128 characters)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeVerifier := base64.URLEncoding.EncodeToString(verifierBytes)

	// Generate code challenge (SHA256 hash of verifier)
	challenge := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.URLEncoding.EncodeToString(challenge[:])

	// Generate state parameter (CSRF protection)
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	return &PKCEParams{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		State:         state,
	}, nil
}

// GetAuthURL generates OAuth authorization URL with PKCE and state
func (cfg *OAuthConfig) GetAuthURL(provider string, pkce *PKCEParams) (string, error) {
	var config *oauth2.Config

	switch provider {
	case "github":
		config = cfg.GitHub
	case "google":
		config = cfg.Google
	default:
		return "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	// Build authorization URL with PKCE
	authURL := config.AuthCodeURL(pkce.State,
		oauth2.SetAuthURLParam("code_challenge", pkce.CodeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	return authURL, nil
}
