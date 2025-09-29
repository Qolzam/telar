package authhmac

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/qolzam/telar/apps/api/internal/types"
)

func sign(method, path, query string, body []byte, uid, timestamp, secret string) string {
	bodyHash := sha256.Sum256(body)
	canonicalString := fmt.Sprintf("%s\n%s\n%s\n%x\n%s\n%s",
		method,
		path,
		query,
		bodyHash,
		uid,
		timestamp,
	)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonicalString))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestAuthHMAC_UnauthorizedWithoutHeader(t *testing.T) {
	app := fiber.New()
	app.Post("/", New(Config{PayloadSecret: "s"}), func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte("{}")))
	req.Header.Set(types.HeaderContentType, "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthHMAC_AuthorizedWithValidSignature(t *testing.T) {
	app := fiber.New()
	app.Post("/", New(Config{PayloadSecret: "s"}), func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	body := []byte("{}")
	uid := "123e4567-e89b-12d3-a456-426614174000"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set(types.HeaderContentType, "application/json")
	req.Header.Set(types.HeaderHMACAuthenticate, sign("POST", "/", "", body, uid, timestamp, "s"))
	req.Header.Set(types.HeaderUID, uid)
	req.Header.Set(types.HeaderTimestamp, timestamp)

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
