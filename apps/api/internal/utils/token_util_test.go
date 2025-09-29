package utils

import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/x509"
    "encoding/pem"
    "testing"
)

func TestGenerateAndValidateToken(t *testing.T) {
    ecKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    privDER, _ := x509.MarshalECPrivateKey(ecKey)
    privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})
    pubDER, _ := x509.MarshalPKIXPublicKey(&ecKey.PublicKey)
    pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
    claim := TokenClaims{}
    tok, err := GenerateJWTToken(privPEM, claim, 1)
    if err != nil { t.Fatalf("gen: %v", err) }
    if _, err := ValidateToken(pubPEM, tok); err != nil { t.Fatalf("validate: %v", err) }
}


