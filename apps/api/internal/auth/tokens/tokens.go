package tokens

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TelarSocialClaims mirrors legacy envelope containing user Claim
type TelarSocialClaims struct {
    Name          string                 `json:"name"`
    Organizations string                 `json:"organizations"`
    AccessToken   string                 `json:"access_token"`
    Claim         map[string]interface{} `json:"claim"`
    jwt.RegisteredClaims
}

// CreateToken creates ES256 signed JWT with TelarSocialClaims
func CreateToken(privateKeyPEM, providerName string, profile map[string]string, organizationList string, claim map[string]interface{}) (string, error) {
    // Parse EC private key from parameter
    privateKey, keyErr := jwt.ParseECPrivateKeyFromPEM([]byte(privateKeyPEM))
    if keyErr != nil {
        return "", keyErr
    }
    method := jwt.GetSigningMethod(jwt.SigningMethodES256.Name)
    claims := TelarSocialClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            ID:        profile["id"],
            Issuer:    "telar-social@" + providerName,
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(48 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Subject:   profile["login"],
            Audience:  []string{profile["audience"]},
        },
        Organizations: organizationList,
        Name:          profile["name"],
        AccessToken:   "",
        Claim:         claim,
    }
    
    // Create token with kid header
    token := jwt.NewWithClaims(method, claims)
    token.Header["kid"] = "telar-auth-key-1" // Consistent key ID
    
    return token.SignedString(privateKey)
}

// CreateTokenWithKey creates ES256 signed JWT with TelarSocialClaims using provided private key
func CreateTokenWithKey(providerName string, profile map[string]string, organizationList string, claim map[string]interface{}, privateKeyPEM string) (string, error) {
	// Parse EC private key from provided parameter
	privateKey, keyErr := jwt.ParseECPrivateKeyFromPEM([]byte(privateKeyPEM))
	if keyErr != nil {
		return "", keyErr
	}
	method := jwt.GetSigningMethod(jwt.SigningMethodES256.Name)
	claims := TelarSocialClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        profile["id"],
			Issuer:    "telar-social@" + providerName,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(48 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   profile["login"],
			Audience:  []string{profile["audience"]},
		},
		Organizations: organizationList,
		Name:          profile["name"],
		AccessToken:   "",
		Claim:         claim,
	}
	
	// Create token with kid header
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = "telar-auth-key-1" // Consistent key ID
	
	return token.SignedString(privateKey)
}

// Note: Cookie-writing functions removed as part of migration to header-based JWT authentication.
// Clients should now use Authorization: Bearer <token> header instead of cookies.

