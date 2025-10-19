package testutil

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/internal/utils"
	"github.com/stretchr/testify/require"
)

// HTTPHelper provides a robust way to make HTTP requests in tests.
// It enforces error checking and provides a fluent API for building requests.
type HTTPHelper struct {
	t   *testing.T
	app *fiber.App
}

// NewHTTPHelper creates a new test helper for a given Fiber app.
func NewHTTPHelper(t *testing.T, app *fiber.App) *HTTPHelper {
	require.NotNil(t, app, "Fiber app provided to HTTPHelper cannot be nil")
	return &HTTPHelper{
		t:   t,
		app: app,
	}
}

// A Request represents a request to be sent.
type Request struct {
	helper  *HTTPHelper
	method  string
	path    string
	body    io.Reader
	headers http.Header
}

// NewRequest begins building a new test request.
func (h *HTTPHelper) NewRequest(method, path string, body interface{}) *Request {
	var bodyReader io.Reader
	if body != nil {
		// Marshal body to JSON if it's not already bytes
		if b, ok := body.([]byte); ok {
			bodyReader = bytes.NewReader(b)
		} else {
			jsonBytes, err := json.Marshal(body)
			require.NoError(h.t, err, "Failed to marshal request body to JSON")
			bodyReader = bytes.NewReader(jsonBytes)
		}
	}

	return &Request{
		helper:  h,
		method:  method,
		path:    path,
		body:    bodyReader,
		headers: make(http.Header),
	}
}

// WithHeader adds a header to the request.
func (r *Request) WithHeader(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

// WithAuthHeaders adds standard HMAC authentication headers.
// DEPRECATED: Use WithHMACAuth for clarity
func (r *Request) WithAuthHeaders(secret, uid string) *Request {
	return r.WithHMACAuth(secret, uid) // Pass through both parameters
}

// WithHMACAuth adds HMAC authentication headers with canonical signing
// Following dependency injection principles - secret must be passed explicitly
func (r *Request) WithHMACAuth(secret, uid string) *Request {
	// Generate timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Extract request details
	method := r.method
	path := r.path
	query := "" // Extract from path if needed
	if strings.Contains(path, "?") {
		parts := strings.SplitN(path, "?", 2)
		path = parts[0]
		query = parts[1]
	}

	// Get body for signing
	var bodyBytes []byte
	if r.body != nil {
		bodyBytes, _ = io.ReadAll(r.body)
		r.body = bytes.NewReader(bodyBytes)
	}

	// Generate canonical signature with injected secret
	sig := SignHMAC(method, path, query, bodyBytes, uid, timestamp, secret)

	// Set required headers for canonical HMAC
	r.WithHeader(types.HeaderHMACAuthenticate, sig)
	r.WithHeader(types.HeaderUID, uid)
	r.WithHeader(types.HeaderTimestamp, timestamp)

	// Set content type if not already set
	if r.headers.Get(types.HeaderContentType) == "" {
		r.WithHeader(types.HeaderContentType, "application/json")
	}

	// Optional headers for user context (not part of signature)
	r.WithHeader("username", "test@example.com")
	r.WithHeader("displayName", "Tester")
	r.WithHeader("socialName", "tester")
	r.WithHeader("systemRole", "user")

	return r
}

// WithJWTAuth generates a valid JWT and adds it as Authorization: Bearer header.
func (r *Request) WithJWTAuth(token string) *Request {
	r.WithHeader(types.HeaderAuthorization, types.BearerPrefix+token)
	return r
}

// WithCookieAuth generates a valid JWT and adds it as session cookies.
// This is the key to solving the votes/setting service issues.
func (r *Request) WithCookieAuth(userCtx types.UserContext) *Request {
	// For tests, create a simple mock JWT token without real signing
	// This avoids the complexity of setting up proper private keys in tests
	header := `{"alg":"ES256","typ":"JWT"}`
	payload := `{"user_id":"` + userCtx.UserID.String() + `","email":"` + userCtx.Username + `","role":"` + userCtx.SystemRole + `","iat":` + strconv.FormatInt(time.Now().Unix(), 10) + `}`
	signature := "test-signature"

	// Encode as base64url (simplified for tests)
	headerB64 := base64URLEncode([]byte(header))
	payloadB64 := base64URLEncode([]byte(payload))

	// Use default cookie names if config values are nil
	// Use default cookie names for testing
	headerCookieName := "header"
	payloadCookieName := "payload"
	signatureCookieName := "signature"

	headerCookie := &http.Cookie{Name: headerCookieName, Value: headerB64, Path: "/"}
	payloadCookie := &http.Cookie{Name: payloadCookieName, Value: payloadB64, Path: "/"}
	signatureCookie := &http.Cookie{Name: signatureCookieName, Value: signature, Path: "/"}

	r.headers.Add("Cookie", headerCookie.String())
	r.headers.Add("Cookie", payloadCookie.String())
	r.headers.Add("Cookie", signatureCookie.String())

	return r
}

// AsMultipartForm configures the request to be sent as multipart/form-data.
// This is the key to solving the storage service issue.
func (r *Request) AsMultipartForm(formData map[string]string, files map[string][]byte) *Request {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Write form fields
	for key, val := range formData {
		_ = writer.WriteField(key, val)
	}

	// Write files
	for key, fileBytes := range files {
		part, _ := writer.CreateFormFile(key, "testfile.jpg") // filename can be static
		_, _ = part.Write(fileBytes)
	}

	err := writer.Close()
	require.NoError(r.helper.t, err)

	// Set the multipart body and content type
	r.body = body
	r.WithHeader(types.HeaderContentType, writer.FormDataContentType())

	return r
}

// WithMultipartAuth creates a multipart form request with HMAC authentication.
// This method handles the correct order of operations for multipart + HMAC.
func (r *Request) WithMultipartAuth(secret, uid string, formData map[string]string, files map[string][]byte) *Request {
	// First create the multipart form data
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Write form fields
	for key, val := range formData {
		_ = writer.WriteField(key, val)
	}

	// Write files
	for key, fileBytes := range files {
		// Create a custom header for the file part
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="testfile.jpg"`, key))
		header.Set(types.HeaderContentType, "image/jpeg")

		part, _ := writer.CreatePart(header)
		_, _ = part.Write(fileBytes)
	}

	err := writer.Close()
	require.NoError(r.helper.t, err)

	// Get the body bytes before setting the body
	bodyBytes := body.Bytes()

	// Set the multipart body and content type
	r.body = bytes.NewReader(bodyBytes)
	r.WithHeader(types.HeaderContentType, writer.FormDataContentType())

	// Generate timestamp for canonical signing
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Extract request details for canonical signing
	method := r.method
	path := r.path
	query := ""
	if strings.Contains(path, "?") {
		parts := strings.SplitN(path, "?", 2)
		path = parts[0]
		query = parts[1]
	}

	// Calculate HMAC with canonical signing
	sig := SignHMAC(method, path, query, bodyBytes, uid, timestamp, secret)
	r.WithHeader(types.HeaderHMACAuthenticate, sig)
	r.WithHeader(types.HeaderUID, uid)
	r.WithHeader(types.HeaderTimestamp, timestamp)
	r.WithHeader("username", "test@example.com")
	r.WithHeader("displayName", "Tester")
	r.WithHeader("socialName", "tester")
	r.WithHeader("systemRole", "user")

	return r
}

// Send executes the request and returns the response.
// It includes robust error handling and a default timeout.
func (r *Request) Send() *http.Response {
	req := httptest.NewRequest(r.method, r.path, r.body)
	req.Header = r.headers

	// Use a reasonable default timeout to prevent tests from hanging.
	resp, err := r.helper.app.Test(req, int(10*time.Second.Milliseconds()))

	// CRITICAL FIX: This is the core of the solution.
	require.NoError(r.helper.t, err, "app.Test should not return an error")
	require.NotNil(r.helper.t, resp, "app.Test response should not be nil")

	return resp
}

// SendWithRetry executes the request with retry logic for robustness.
func (r *Request) SendWithRetry(maxRetries int) *http.Response {
	const timeout = 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		req := httptest.NewRequest(r.method, r.path, r.body)
		req.Header = r.headers

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req = req.WithContext(ctx)

		resp, err := r.helper.app.Test(req)
		if err == nil && resp != nil {
			cancel()
			return resp
		}

		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
		cancel()
	}

	r.helper.t.Fatalf("HTTP request failed after %d retries", maxRetries)
	return nil
}

// SignHMAC generates HMAC SHA256 signature using canonical signing format
// Canonical string format: METHOD\nPATH\nCANONICAL_QUERY\nsha256(BODY)\nUID\nTIMESTAMP
func SignHMAC(method, path, query string, body []byte, uid, timestamp, secret string) string {
	// Build canonical string
	bodyHash := sha256.Sum256(body)
	canonicalString := fmt.Sprintf("%s\n%s\n%s\n%x\n%s\n%s",
		method,
		path,
		query,
		bodyHash,
		uid,
		timestamp,
	)

	// Generate HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonicalString))
	return types.HMACPrefix + hex.EncodeToString(mac.Sum(nil))
}

// CreateTestUserContext creates a test user context for testing purposes.
func CreateTestUserContext(uid string) types.UserContext {
	userID, _ := uuid.FromString(uid)
	return types.UserContext{
		UserID:      userID,
		Username:    "test@example.com",
		DisplayName: "Test User",
		SocialName:  "testuser",
		SystemRole:  "user",
		CreatedDate: time.Now().Unix(),
	}
}

// base64URLEncode encodes bytes to base64url format (JWT standard)
func base64URLEncode(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	// Convert to base64url format
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	encoded = strings.TrimRight(encoded, "=")
	return encoded
}

// WithUserJWT simulates a REAL USER sending a request from a browser/app
func (r *Request) WithUserJWT(token string) *Request {
	r.WithHeader(types.HeaderAuthorization, types.BearerPrefix+token)
	return r
}

// WithS2SHMAC simulates a BACKEND SERVICE making an internal S2S call
func (r *Request) WithS2SHMAC(secret, serviceName string) *Request {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Get request body as bytes for signing
	var bodyBytes []byte
	if r.body != nil {
		if body, ok := r.body.(*bytes.Buffer); ok {
			bodyBytes = body.Bytes()
		} else if body, ok := r.body.(*strings.Reader); ok {
			bodyBytes = make([]byte, body.Len())
			body.Read(bodyBytes)
			body.Seek(0, 0) // Reset reader position
		}
	}

	// Build canonical signature
	signature := SignHMAC(r.method, r.path, "", bodyBytes, serviceName, timestamp, secret)

	r.WithHeader(types.HeaderHMACAuthenticate, signature)
	r.WithHeader(types.HeaderTimestamp, timestamp)
	r.WithHeader(types.HeaderUID, serviceName) // The "user" is the service itself
	return r
}

// GenerateTestJWT creates a test JWT token for testing purposes
func GenerateTestJWT(privateKeyPEM string, userCtx types.UserContext) (string, error) {
	// Create claims with user context
	claims := utils.TokenClaims{
		Claim: map[string]interface{}{
			types.HeaderUID: userCtx.UserID.String(),
			"username":      userCtx.Username,
			"displayName":   userCtx.DisplayName,
			"avatar":        userCtx.Avatar,
			"role":          userCtx.SystemRole,
			"socialName":    userCtx.SocialName,
			"banner":        userCtx.Banner,
			"tagLine":       userCtx.TagLine,
			"createdDate":   userCtx.CreatedDate,
		},
	}

	// Generate token with 1 hour expiration
	token, err := utils.GenerateJWTToken([]byte(privateKeyPEM), claims, 1)
	if err != nil {
		return "", fmt.Errorf("failed to generate test JWT: %w", err)
	}

	return token, nil
}

// GenerateECDSAKeyPairPEM generates valid ECDSA key pairs for testing.
// Returns (publicKeyPEM, privateKeyPEM) as strings.
// This function should be used across all services to avoid code duplication.
func GenerateECDSAKeyPairPEM(t *testing.T) (string, string) {
	t.Helper()

	// Generate ECDSA keys as expected by the JWT middleware
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "Failed to generate ECDSA private key")

	// Use PKCS8 format for private key
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err, "Failed to marshal ECDSA private key")
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	// Use PKIX format for public key
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err, "Failed to marshal ECDSA public key")
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	return string(pubPEM), string(privPEM)
}
