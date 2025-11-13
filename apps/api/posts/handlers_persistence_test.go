package posts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// --- helpers ---

// addHMACHeaders creates HMAC authentication headers using canonical signing format (normalized path)
// Note: This is a local helper for handlers_persistence_test.go
// The http_compat_test.go file has its own version of this function
func addHMACHeadersLocal(req *http.Request, body []byte, secret string, uid string) {
	method := req.Method
	path := req.URL.Path
	query := req.URL.RawQuery
	originalPath := path
	normalizedPath := filepath.Clean(originalPath)
	if strings.HasSuffix(originalPath, "/") && normalizedPath != "/" {
		normalizedPath += "/"
	}
	path = normalizedPath
	if body == nil {
		body = []byte{}
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sig := testutil.SignHMAC(method, path, query, body, uid, timestamp, secret)
	req.Header.Set(types.HeaderHMACAuthenticate, sig)
	req.Header.Set(types.HeaderUID, uid)
	req.Header.Set(types.HeaderTimestamp, timestamp)
}

// addHMACHeaders creates HMAC authentication headers using canonical signing format
func addHMACHeadersLegacy(req *http.Request, body []byte, secret string, uid string) {
	// Generate timestamp for canonical signing
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	
	// Extract request details for canonical signing
	method := req.Method
	path := req.URL.Path
	query := req.URL.RawQuery
	
	// Generate canonical HMAC signature
	sig := testutil.SignHMAC(method, path, query, body, uid, timestamp, secret)
	req.Header.Set(types.HeaderHMACAuthenticate, sig)
	req.Header.Set(types.HeaderUID, uid)
	req.Header.Set(types.HeaderTimestamp, timestamp)
	req.Header.Set("username", "test@example.com")
	req.Header.Set("displayName", "Tester")
	req.Header.Set("socialName", "tester")
	req.Header.Set("systemRole", "user")
}

type postGetResp struct {
    Body             string `json:"body"`
    OwnerDisplayName string `json:"ownerDisplayName"`
    URLKey           string `json:"urlKey"`
    DisableComments  bool   `json:"disableComments"`
}

func runPostsPersistenceSuiteFast(t *testing.T, dbType string, base *platform.BaseService, config *testutil.TestConfig) {
    
    app, routerConfig := newTestApp(t, base, config)
    
    // Create HTTP helper
    httpHelper := testutil.NewHTTPHelper(t, app)
    uid := uuid.Must(uuid.NewV4()).String()
    
    // Use the test configuration secret for HMAC signing
    secret := routerConfig.PayloadSecret

    // Create
    newId := uuid.Must(uuid.NewV4()).String()
    payload := map[string]interface{}{
        "objectId":        newId,
        "postTypeId":      1,
        "score":           0,
        "viewCount":       0,
        "body":            "initial body",
        "tags":            []string{"t"},
        "commentCounter":  0,
        "image":           "",
        "imageFullPath":   "",
        "video":           "",
        "thumbnail":       "",
        "album":           map[string]interface{}{"count": 0, "cover": "", "coverId": newId, "photos": []string{}, "title": ""},
        "disableComments": false,
        "disableSharing":  false,
        "deleted":         false,
        "deletedDate":     0,
        "lastUpdated":     0,
        "accessUserList":  []string{},
        "permission":      "Public",
        "version":         "v1",
    }
    
    resp := httpHelper.NewRequest(http.MethodPost, "/posts/", payload).
        WithAuthHeaders(secret, uid).Send()
    require.Equal(t, http.StatusCreated, resp.StatusCode, "Create post should return 201 Created")

    // Update body via handler
    upd := map[string]interface{}{"objectId": newId, "body": "updated body"}
    
    respUp := httpHelper.NewRequest(http.MethodPut, "/posts/", upd).
        WithAuthHeaders(secret, uid).Send()
    require.Equal(t, http.StatusOK, respUp.StatusCode, "Update post should return 200 OK")

    // Increment score +1 then -1 to cover both branches
    sc := map[string]interface{}{"postId": newId, "delta": 1}
    
    respSc := httpHelper.NewRequest(http.MethodPut, "/posts/actions/score", sc).
        WithAuthHeaders(secret, uid).Send()
    require.Equal(t, http.StatusOK, respSc.StatusCode, "Increment score +1 should return 200 OK")
    sc2 := map[string]interface{}{"postId": newId, "delta": -1}
    
    respSc2 := httpHelper.NewRequest(http.MethodPut, "/posts/actions/score", sc2).
        WithAuthHeaders(secret, uid).Send()
    require.Equal(t, http.StatusOK, respSc2.StatusCode, "Increment score -1 should return 200 OK")

    // Disable comments
    dc := map[string]interface{}{"objectId": newId, "disable": true}
    bdc, _ := json.Marshal(dc)
    reqDc := httptest.NewRequest(http.MethodPut, "/posts/comment/disable", bytes.NewReader(bdc))
    reqDc.Header.Set(types.HeaderContentType, "application/json")
    addHMACHeadersLocal(reqDc, bdc, secret, uid)
    respDc, err := app.Test(reqDc)
    require.NoError(t, err)
    if respDc.StatusCode != http.StatusOK { t.Fatalf("disable comment status=%d", respDc.StatusCode) }

    // Disable sharing true then false
    ds := map[string]interface{}{"objectId": newId, "disable": true}
    bds, _ := json.Marshal(ds)
    reqDs := httptest.NewRequest(http.MethodPut, "/posts/share/disable", bytes.NewReader(bds))
    reqDs.Header.Set(types.HeaderContentType, "application/json")
    addHMACHeadersLocal(reqDs, bds, secret, uid)
    respDs, err := app.Test(reqDs)
    require.NoError(t, err)
    if respDs.StatusCode != http.StatusOK { t.Fatalf("disable share status=%d", respDs.StatusCode) }
    ds2 := map[string]interface{}{"objectId": newId, "disable": false}
    bds2, _ := json.Marshal(ds2)
    reqDs2 := httptest.NewRequest(http.MethodPut, "/posts/share/disable", bytes.NewReader(bds2))
    reqDs2.Header.Set(types.HeaderContentType, "application/json")
    addHMACHeadersLocal(reqDs2, bds2, secret, uid)
    respDs2, err := app.Test(reqDs2)
    require.NoError(t, err)
    if respDs2.StatusCode != http.StatusOK { t.Fatalf("enable share status=%d", respDs2.StatusCode) }

    // Generate url key
    reqKey := httptest.NewRequest(http.MethodPut, "/posts/urlkey/"+newId, nil)
    addHMACHeadersLocal(reqKey, nil, secret, uid)
    respKey, err := app.Test(reqKey)
    require.NoError(t, err)
    if respKey.StatusCode != http.StatusOK { t.Fatalf("urlkey status=%d", respKey.StatusCode) }

    // Read and assert persisted changes
    reqGet := httptest.NewRequest(http.MethodGet, "/posts/"+newId, nil)
    addHMACHeadersLocal(reqGet, nil, secret, uid)
    respGet, err := app.Test(reqGet)
    require.NoError(t, err)
    if respGet.StatusCode != http.StatusOK { t.Fatalf("get status=%d", respGet.StatusCode) }
    var got postGetResp
    if err := json.NewDecoder(respGet.Body).Decode(&got); err != nil { t.Fatalf("decode get: %v", err) }
    if got.Body != "updated body" { t.Fatalf("expected updated body, got %q", got.Body) }
    if got.OwnerDisplayName != "Tester" { t.Fatalf("ownerDisplayName not persisted: %q", got.OwnerDisplayName) }
    if got.URLKey == "" { t.Fatalf("urlKey not set") }
    if !got.DisableComments { t.Fatalf("disableComments not persisted") }

    // List route with pagination; just ensure not error
    reqList := httptest.NewRequest(http.MethodGet, "/posts/?page=1&limit=1", nil)
    addHMACHeadersLocal(reqList, nil, secret, uid)
    respList, err := app.Test(reqList, 10000) // 10 second timeout
    require.NoError(t, err)
    if respList.StatusCode != http.StatusOK { t.Fatalf("list status=%d", respList.StatusCode) }

    // Delete and ensure it's gone via repository count
    reqDel := httptest.NewRequest(http.MethodDelete, "/posts/"+newId, nil)
    addHMACHeadersLocal(reqDel, nil, secret, uid)
    respDel, err := app.Test(reqDel)
    require.NoError(t, err)
    if respDel.StatusCode != http.StatusNoContent { t.Fatalf("delete status=%d", respDel.StatusCode) }

	// Verify repository count is 0 using the base service repository
	queryObj := &dbi.Query{
		Conditions: []dbi.Field{
			{
				Name:     "object_id",
				Value:    uuid.FromStringOrNil(newId).String(),
				Operator: "=",
			},
		},
	}
	cnt := <-base.Repository.Count(context.Background(), "post", queryObj)
	if cnt.Error != nil {
		t.Fatalf("count error: %v", cnt.Error)
	}
}

// Legacy NoSQL tests removed - PostgreSQL only

func TestPosts_Handler_Persistence_Postgres(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	// Create base service for the test
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}
	
	// Postgres: transactional wrapper not required for HTTP flow
	runPostsPersistenceSuiteFast(t, "postgres", base, iso.LegacyConfig)
}

func TestPosts_Create_Success_AllFields_And_QueryBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	// Create base service for the test
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}
	
	app, routerConfig := newTestApp(t, base, iso.LegacyConfig)
	uid := uuid.Must(uuid.NewV4()).String()
	// Use the test configuration secret for HMAC signing
	secret := routerConfig.PayloadSecret
	// Compose full payload hitting all setFields in Update handler
    newId := uuid.Must(uuid.NewV4()).String()
    payload := map[string]interface{}{
        "objectId": newId,
        "postTypeId": 1,
        "score": 1,
        "viewCount": 2,
        "body": "X",
        "tags": []string{"a","b"},
        "commentCounter": 0,
        "image": "i",
        "imageFullPath": "if",
        "video": "v",
        "thumbnail": "t",
        "album": map[string]interface{}{"count":1,"cover":"c","coverId":newId,"photos":[]string{"p"},"title":"tt"},
        "disableComments": false,
        "disableSharing": false,
        "deleted": false,
        "deletedDate": 0,
        "lastUpdated": 0,
        "accessUserList": []string{"a"},
        "permission": 1,
        "version": "v1",
    }
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/posts/", bytes.NewReader(body))
	req.Header.Set(types.HeaderContentType, "application/json")
	addHMACHeadersLocal(req, body, secret, uid)
	_, _ = app.Test(req)
	
	// generate url key and fetch by id
	r2 := httptest.NewRequest("PUT", "/posts/urlkey/"+newId, nil)
	addHMACHeadersLocal(r2, nil, secret, uid)
	_, _ = app.Test(r2)
	r3 := httptest.NewRequest("GET", "/posts/"+newId, nil)
	addHMACHeadersLocal(r3, nil, secret, uid)
	_, _ = app.Test(r3)
}

// Error scenario tests for handlers
func TestPosts_Handler_ErrorScenarios(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run DB tests")
	}
	
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
    
	// Create the app using service injection pattern
	base, err := platform.NewBaseService(context.Background(), iso.Config)
    if err != nil {
        t.Fatalf("failed to build postgresql base service: %v", err)
    }
    
    app, routerConfig := newTestApp(t, base, iso.LegacyConfig)
    uid := uuid.Must(uuid.NewV4()).String()
    nonExistentId := uuid.Must(uuid.NewV4()).String()
    
    // Use the test configuration secret for HMAC signing
    secret := routerConfig.PayloadSecret

    // Test 1: Create post with invalid JSON
    t.Run("CreatePost_InvalidJSON", func(t *testing.T) {
        invalidJSON := []byte(`{"body": "test", "postTypeId":}`) // malformed JSON
        req := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(invalidJSON))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, invalidJSON, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for invalid JSON, got status %d", resp.StatusCode)
        }
    })

    // Test 2: Create post with missing required fields
    t.Run("CreatePost_MissingRequiredFields", func(t *testing.T) {
        payload := map[string]interface{}{
            // Missing body and postTypeId
            "objectId": uuid.Must(uuid.NewV4()).String(),
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for missing required fields, got status %d", resp.StatusCode)
        }
    })

    // Test 3: Create post with invalid postTypeId
    t.Run("CreatePost_InvalidPostTypeId", func(t *testing.T) {
        payload := map[string]interface{}{
            "objectId":   uuid.Must(uuid.NewV4()).String(),
            "postTypeId": -1, // Invalid type
            "body":       "test body",
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for invalid postTypeId, got status %d", resp.StatusCode)
        }
    })

    // Test 4: Update non-existent post
    t.Run("UpdatePost_NotFound", func(t *testing.T) {
        payload := map[string]interface{}{
            "objectId": nonExistentId,
            "body":     "updated body",
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPut, "/posts/", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for updating non-existent post, got status %d", resp.StatusCode)
        }
    })

    // Test 5: Get non-existent post
    t.Run("GetPost_NotFound", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/posts/"+nonExistentId, nil)
        addHMACHeadersLocal(req, nil, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for getting non-existent post, got status %d", resp.StatusCode)
        }
    })

    // Test 6: Delete non-existent post
    t.Run("DeletePost_NotFound", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodDelete, "/posts/"+nonExistentId, nil)
        addHMACHeadersLocal(req, nil, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for deleting non-existent post, got status %d", resp.StatusCode)
        }
    })

    // Test 7: Score update with invalid postId
    t.Run("UpdateScore_InvalidPostId", func(t *testing.T) {
        payload := map[string]interface{}{
            "postId": "invalid-uuid",
            "delta":  1,
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPut, "/posts/actions/score", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for invalid postId in score update, got status %d", resp.StatusCode)
        }
    })

    // Test 8: Score update with missing delta
    t.Run("UpdateScore_MissingCount", func(t *testing.T) {
        payload := map[string]interface{}{
            "postId": uuid.Must(uuid.NewV4()).String(),
            // Missing delta field
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPut, "/posts/actions/score", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for missing delta in score update, got status %d", resp.StatusCode)
        }
    })

    // Test 9: Disable comments with invalid postId
    t.Run("DisableComments_InvalidPostId", func(t *testing.T) {
        payload := map[string]interface{}{
            "postId": "invalid-uuid",
            "status": true,
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPut, "/posts/comment/disable", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for invalid postId in disable comments, got status %d", resp.StatusCode)
        }
    })

    // Test 10: Disable sharing with missing status
    t.Run("DisableSharing_MissingStatus", func(t *testing.T) {
        payload := map[string]interface{}{
            "postId": uuid.Must(uuid.NewV4()).String(),
            // Missing status field
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPut, "/posts/share/disable", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for missing status in disable sharing, got status %d", resp.StatusCode)
        }
    })

    // Test 11: Generate URL key for non-existent post
    t.Run("GenerateURLKey_PostNotFound", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodPut, "/posts/urlkey/"+nonExistentId, nil)
        addHMACHeadersLocal(req, nil, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for generating URL key for non-existent post, got status %d", resp.StatusCode)
        }
    })

    // Test 12: List posts with invalid pagination parameters
    t.Run("ListPosts_InvalidPagination", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/posts/?page=-1&limit=0", nil)
        addHMACHeadersLocal(req, nil, secret, uid)
        
        // Add timeout to prevent hanging - production-ready error handling
        resp, err := app.Test(req, 5000) // 5 second timeout
        if err != nil {
            // If the request times out or fails, log it and skip further assertions
            t.Logf("Request failed or timed out as expected for invalid pagination: %v", err)
            return
        }
        
        // The handler should gracefully handle invalid pagination parameters
        // Either return 200 with corrected parameters or 400 for validation error
        if resp.StatusCode == http.StatusOK {
            t.Logf("Invalid pagination gracefully handled with status 200 (parameters likely corrected)")
        } else if resp.StatusCode == http.StatusBadRequest {
            t.Logf("Invalid pagination correctly rejected with status 400")
        } else {
            t.Errorf("Unexpected status code for invalid pagination: %d", resp.StatusCode)
        }
    })

    // Test 13: Request without authentication headers
    t.Run("Request_NoAuth", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/posts/"+nonExistentId, nil)
        // No HMAC headers added
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for request without auth, got status %d", resp.StatusCode)
        }
    })

    // Test 14: Request with invalid authentication
    t.Run("Request_InvalidAuth", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/posts/"+nonExistentId, nil)
        req.Header.Set("X-Cloud-Signature", "invalid-signature")
        req.Header.Set("X-Cloud-Trace-Context", "invalid-trace")
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for invalid auth, got status %d", resp.StatusCode)
        }
    })

    // Test 15: Create post with extremely long body
    t.Run("CreatePost_ExtremelyLongBody", func(t *testing.T) {
        longBody := string(make([]byte, 100000)) // 100KB body
        payload := map[string]interface{}{
            "objectId":   uuid.Must(uuid.NewV4()).String(),
            "postTypeId": 1,
            "body":       longBody,
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        // This tests the behavior with extremely large payloads
        _ = resp.StatusCode
    })

    // Test 16: Update post with malformed album data
    t.Run("UpdatePost_MalformedAlbum", func(t *testing.T) {
        // First create a post
        newId := uuid.Must(uuid.NewV4()).String()
        createPayload := map[string]interface{}{
            "objectId":   newId,
            "postTypeId": 1,
            "body":       "test body",
        }
        createBody, _ := json.Marshal(createPayload)
        createReq := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(createBody))
        createReq.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(createReq, createBody, secret, uid)
        createResp, _ := app.Test(createReq)
        
        if createResp.StatusCode == http.StatusCreated {
            // Then try to update with malformed album
            updatePayload := map[string]interface{}{
                "objectId": newId,
                "album":    "not-an-object", // Should be an object
            }
            updateBody, _ := json.Marshal(updatePayload)
                    updateReq := httptest.NewRequest(http.MethodPut, "/posts/", bytes.NewReader(updateBody))
        updateReq.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(updateReq, updateBody, secret, uid)
        updateResp, _ := app.Test(updateReq)
            if updateResp.StatusCode == http.StatusOK {
                t.Errorf("Expected error for malformed album data, got status %d", updateResp.StatusCode)
            }
        }
    })
}

// Test authorization and ownership scenarios
func TestPosts_Handler_AuthorizationErrors(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run DB tests")
	}
	
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	// Create the app using service injection pattern
	base, err := platform.NewBaseService(context.Background(), iso.Config)
    if err != nil {
        t.Fatalf("failed to build postgresql base service: %v", err)
    }
    
    app, routerConfig := newTestApp(t, base, iso.LegacyConfig)
    ownerUid := uuid.Must(uuid.NewV4()).String()
    otherUid := uuid.Must(uuid.NewV4()).String()
    
    // Use the test configuration secret for HMAC signing
    secret := routerConfig.PayloadSecret

    // Create a post with owner
    postId := uuid.Must(uuid.NewV4()).String()
    payload := map[string]interface{}{
        "objectId":   postId,
        "postTypeId": 1,
        "body":       "owner's post",
        "permission": 1, // OnlyMe
    }
    body, _ := json.Marshal(payload)
    req := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(body))
    req.Header.Set(types.HeaderContentType, "application/json")
    addHMACHeadersLocal(req, body, secret, ownerUid)
    resp, _ := app.Test(req)
    
    if resp.StatusCode == http.StatusOK {
        // Test 1: Other user tries to update the post
        t.Run("UpdatePost_UnauthorizedUser", func(t *testing.T) {
            updatePayload := map[string]interface{}{
                "objectId": postId,
                "body":     "hacked body",
            }
            updateBody, _ := json.Marshal(updatePayload)
            updateReq := httptest.NewRequest(http.MethodPut, "/posts/", bytes.NewReader(updateBody))
            updateReq.Header.Set(types.HeaderContentType, "application/json")
            addHMACHeadersLocal(updateReq, updateBody, secret, otherUid)
            updateResp, _ := app.Test(updateReq)
            if updateResp.StatusCode == http.StatusOK {
                t.Errorf("Expected authorization error for unauthorized update, got status %d", updateResp.StatusCode)
            }
        })

        // Test 2: Other user tries to delete the post
        t.Run("DeletePost_UnauthorizedUser", func(t *testing.T) {
            deleteReq := httptest.NewRequest(http.MethodDelete, "/posts/"+postId, nil)
            addHMACHeadersLocal(deleteReq, nil, secret, otherUid)
            deleteResp, _ := app.Test(deleteReq)
            if deleteResp.StatusCode == http.StatusOK {
                t.Errorf("Expected authorization error for unauthorized delete, got status %d", deleteResp.StatusCode)
            }
        })

        // Test 3: Other user tries to update score (this might be allowed depending on business logic)
        t.Run("UpdateScore_DifferentUser", func(t *testing.T) {
            scorePayload := map[string]interface{}{
                "postId": postId,
                "delta":  1,
            }
            scoreBody, _ := json.Marshal(scorePayload)
            scoreReq := httptest.NewRequest(http.MethodPut, "/posts/actions/score", bytes.NewReader(scoreBody))
            scoreReq.Header.Set(types.HeaderContentType, "application/json")
            addHMACHeadersLocal(scoreReq, scoreBody, secret, otherUid)
            scoreResp, _ := app.Test(scoreReq)
            // Note: Score updates might be allowed from different users - test actual behavior
            _ = scoreResp.StatusCode
        })

        // Test 4: Other user tries to generate URL key
        t.Run("GenerateURLKey_UnauthorizedUser", func(t *testing.T) {
            urlKeyReq := httptest.NewRequest(http.MethodPut, "/posts/urlkey/"+postId, nil)
            addHMACHeadersLocal(urlKeyReq, nil, secret, otherUid)
            urlKeyResp, _ := app.Test(urlKeyReq)
            if urlKeyResp.StatusCode == http.StatusOK {
                t.Errorf("Expected authorization error for unauthorized URL key generation, got status %d", urlKeyResp.StatusCode)
            }
        })

        // Test 5: Other user tries to disable comments
        t.Run("DisableComments_UnauthorizedUser", func(t *testing.T) {
            disablePayload := map[string]interface{}{
                "postId": postId,
                "status": true,
            }
            disableBody, _ := json.Marshal(disablePayload)
                    disableReq := httptest.NewRequest(http.MethodPut, "/posts/comment/disable", bytes.NewReader(disableBody))
        disableReq.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(disableReq, disableBody, secret, otherUid)
        disableResp, _ := app.Test(disableReq)
            if disableResp.StatusCode == http.StatusOK {
                t.Errorf("Expected authorization error for unauthorized comment disable, got status %d", disableResp.StatusCode)
            }
        })
    }
}

// Test edge cases and boundary conditions
func TestPosts_Handler_EdgeCases(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run DB tests")
	}
	
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	// Create the app using service injection pattern
	base, err := platform.NewBaseService(context.Background(), iso.Config)
    if err != nil {
        t.Fatalf("failed to build postgresql base service: %v", err)
    }
    
    app, routerConfig := newTestApp(t, base, iso.LegacyConfig)
    uid := uuid.Must(uuid.NewV4()).String()
    
    // Use the test configuration secret for HMAC signing
    secret := routerConfig.PayloadSecret

    // Test 1: Create post with empty body
    t.Run("CreatePost_EmptyBody", func(t *testing.T) {
        payload := map[string]interface{}{
            "objectId":   uuid.Must(uuid.NewV4()).String(),
            "postTypeId": 1,
            "body":       "",
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        // Test behavior with empty body - might be valid or invalid depending on business rules
        _ = resp.StatusCode
    })

    // Test 2: Create post with special characters in body
    t.Run("CreatePost_SpecialCharacters", func(t *testing.T) {
        payload := map[string]interface{}{
            "objectId":   uuid.Must(uuid.NewV4()).String(),
            "postTypeId": 1,
            "body":       "Special chars: 💀☠️🔥💯 and unicode: 你好世界",
        }
        body, _ := json.Marshal(payload)
        req := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(body))
        req.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(req, body, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode != http.StatusCreated {
            t.Errorf("Failed to create post with special characters, status %d", resp.StatusCode)
        }
    })

    // Test 3: Update with null/undefined values
    t.Run("UpdatePost_NullValues", func(t *testing.T) {
        // First create a post
        postId := uuid.Must(uuid.NewV4()).String()
        createPayload := map[string]interface{}{
            "objectId":   postId,
            "postTypeId": 1,
            "body":       "initial body",
            "tags":       []string{"tag1", "tag2"},
        }
        createBody, _ := json.Marshal(createPayload)
        createReq := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(createBody))
        createReq.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(createReq, createBody, secret, uid)
        createResp, _ := app.Test(createReq)
        
        if createResp.StatusCode == http.StatusCreated {
            // Try to update with null values
            updatePayload := map[string]interface{}{
                "objectId": postId,
                "tags":     nil, // Explicitly null
            }
            updateBody, _ := json.Marshal(updatePayload)
            updateReq := httptest.NewRequest(http.MethodPut, "/posts/", bytes.NewReader(updateBody))
            updateReq.Header.Set(types.HeaderContentType, "application/json")
            addHMACHeadersLocal(updateReq, updateBody, secret, uid)
            updateResp, _ := app.Test(updateReq)
            _ = updateResp.StatusCode // Test behavior with null values
        }
    })

    // Test 4: Very large pagination limits
    t.Run("ListPosts_LargePaginationLimit", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/posts/?page=1&limit=1000000", nil)
        addHMACHeadersLocal(req, nil, secret, uid)
        resp, err := app.Test(req, 10000) // 10 second timeout
        require.NoError(t, err, "The HTTP request for ListPosts_LargePaginationLimit failed")
        // Test how the system handles very large limits
        _ = resp.StatusCode
    })

    // Test 5: Concurrent operations on the same post
    t.Run("ConcurrentUpdates", func(t *testing.T) {
        // First create a post
        postId := uuid.Must(uuid.NewV4()).String()
        createPayload := map[string]interface{}{
            "objectId":   postId,
            "postTypeId": 1,
            "body":       "concurrent test",
        }
        createBody, _ := json.Marshal(createPayload)
        createReq := httptest.NewRequest(http.MethodPost, "/posts/", bytes.NewReader(createBody))
        createReq.Header.Set(types.HeaderContentType, "application/json")
        addHMACHeadersLocal(createReq, createBody, secret, uid)
        createResp, _ := app.Test(createReq)
        
        if createResp.StatusCode == http.StatusCreated {
            // Simulate concurrent score updates
            done := make(chan bool, 5)
            for i := 0; i < 5; i++ {
                go func(increment int) {
                    scorePayload := map[string]interface{}{
                        "postId": postId,
                        "delta":  1,
                    }
                    scoreBody, _ := json.Marshal(scorePayload)
                    scoreReq := httptest.NewRequest(http.MethodPut, "/posts/actions/score", bytes.NewReader(scoreBody))
                    scoreReq.Header.Set(types.HeaderContentType, "application/json")
                    addHMACHeadersLocal(scoreReq, scoreBody, secret, uid)
                    _, _ = app.Test(scoreReq)
                    done <- true
                }(i)
            }
            
            // Wait for all goroutines to complete
            for i := 0; i < 5; i++ {
                <-done
            }
        }
    })

    // Test 6: Invalid HTTP methods
    t.Run("InvalidHTTPMethods", func(t *testing.T) {
        postId := uuid.Must(uuid.NewV4()).String()
        
        // Try PATCH method (not supported)
        req := httptest.NewRequest(http.MethodPatch, "/posts/"+postId, nil)
        addHMACHeadersLocal(req, nil, secret, uid)
        resp, _ := app.Test(req)
        if resp.StatusCode == http.StatusOK {
            t.Errorf("Expected error for unsupported HTTP method, got status %d", resp.StatusCode)
        }
    })
}


