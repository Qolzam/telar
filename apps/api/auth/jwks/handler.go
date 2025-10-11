package jwks

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/auth/errors"
)

type Handler struct {
	publicKey string
	keyID     string
	config    *HandlerConfig
}

type HandlerConfig struct {
	PublicKey string
	KeyID     string
}

func NewHandler(publicKey, keyID string) *Handler {
	return &Handler{
		publicKey: publicKey,
		keyID:     keyID,
		config: &HandlerConfig{
			PublicKey: publicKey,
			KeyID:     keyID,
		},
	}
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"` // Key Type
	Use string `json:"use"` // Public Key Use
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm
	Crv string `json:"crv"` // Curve (for EC keys)
	X   string `json:"x"`   // X coordinate
	Y   string `json:"y"`   // Y coordinate
}

// Handle returns the JWKS for JWT validation
func (h *Handler) Handle(c *fiber.Ctx) error {
	// Parse the public key
	block, _ := pem.Decode([]byte(h.publicKey))
	if block == nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to parse public key"))
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return errors.HandleServiceError(c, fmt.Errorf("failed to parse public key: %w", err))
	}

	ecdsaKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.HandleServiceError(c, fmt.Errorf("public key is not ECDSA"))
	}

	// Convert to JWK format
	jwk := JWK{
		Kty: "EC",
		Use: "sig",
		Kid: h.keyID,
		Alg: "ES256",
		Crv: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(ecdsaKey.X.Bytes()),
		Y:   base64.RawURLEncoding.EncodeToString(ecdsaKey.Y.Bytes()),
	}

	jwks := JWKS{
		Keys: []JWK{jwk},
	}

	return c.JSON(jwks)
}
