package utils

import (
    "fmt"

	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	// Meta data
	Claim interface{} `json:"claim"`

	// Inherit from registered claims
	jwt.RegisteredClaims
}

// GenerateJWTToken
func GenerateJWTToken(privateKeydata []byte, claim TokenClaims, expireOffsetHour int64) (string, error) {

    privateKey, keyErr := jwt.ParseECPrivateKeyFromPEM(privateKeydata)
    if keyErr != nil {
        return "", fmt.Errorf("unable to parse private key: %w", keyErr)
    }

	method := jwt.GetSigningMethod(jwt.SigningMethodES256.Name)

	session, err := jwt.NewWithClaims(method, claim).SignedString(privateKey)
	return session, err
}

// ValidateToken
func ValidateToken(keydata []byte, token string) (jwt.MapClaims, error) {

	publicKey, keyErr := jwt.ParseECPublicKeyFromPEM(keydata)
	if keyErr != nil {
		return nil, keyErr
	}

	parsed, parseErr := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	if parseErr != nil {
		return nil, parseErr
	}

	if claims, ok := parsed.Claims.(jwt.MapClaims); ok && parsed.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("Token claim is not valid!")
	}
}
