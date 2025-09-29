package tokens

import (
	b64 "encoding/base64"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type resetPasswordClaims struct {
    VerifyId string `json:"verifyId"`
    jwt.RegisteredClaims
}

// DEPRECATED: GenerateResetPasswordToken uses JWT tokens which are insecure for password reset
// Use GenerateSecureResetToken from reset_secure.go instead
func GenerateResetPasswordToken(privateKeyPEM, verifyId string) (string, error) {
    privateKeyEnc := b64.StdEncoding.EncodeToString([]byte(privateKeyPEM))
    jwtKey := []byte(privateKeyEnc[:20])
    expirationTime := time.Now().Add(5 * time.Minute)
    claims := &resetPasswordClaims{
        VerifyId: verifyId,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expirationTime),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString(jwtKey)
    if err != nil { return "", err }
    // Legacy encodes the token again with base64
    return b64.StdEncoding.EncodeToString([]byte(signed)), nil
}

// DEPRECATED: DecodeResetPasswordToken decodes JWT tokens which are insecure for password reset
// Use ValidateResetToken from reset_secure.go instead
func DecodeResetPasswordToken(privateKeyPEM, token string) (*resetPasswordClaims, error) {
    privateKeyEnc := b64.StdEncoding.EncodeToString([]byte(privateKeyPEM))
    jwtKey := []byte(privateKeyEnc[:20])
    decoded, err := b64.StdEncoding.DecodeString(token)
    if err != nil { return nil, err }
    claims := new(resetPasswordClaims)
    tkn, err := jwt.ParseWithClaims(string(decoded), claims, func(token *jwt.Token) (interface{}, error) { return jwtKey, nil })
    if err != nil { return nil, err }
    if !tkn.Valid { return nil, jwt.ErrSignatureInvalid }
    return claims, nil
}

