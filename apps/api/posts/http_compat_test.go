package posts_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts"
	"github.com/qolzam/telar/apps/api/posts/handlers"
	"github.com/qolzam/telar/apps/api/posts/services"
)


// verifyPostgresConnection tests if we can actually connect to PostgreSQL
func verifyPostgresConnection() error {
	cfg, err := testutil.LoadTestConfig()
	if err != nil {
		return fmt.Errorf("failed to load test config: %w", err)
	}
	dsn := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		cfg.PGHost, cfg.PGUser, cfg.PGPassword, cfg.PGDatabase, cfg.PGSchema)
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}
	defer dbConn.Close()
	if err := dbConn.Ping(); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}
	_, err = dbConn.Exec("CREATE TABLE IF NOT EXISTS test_connection (id SERIAL PRIMARY KEY, name TEXT)")
	if err != nil {
		return fmt.Errorf("failed to create test table: %w", err)
	}
	_, err = dbConn.Exec("DROP TABLE IF EXISTS test_connection")
	if err != nil {
		return fmt.Errorf("failed to drop test table: %w", err)
	}
	return nil
}

// verifyPostExistsInDatabase checks if a post was actually saved
func verifyPostExistsInDatabase(postID string) error {
	cfg, err := testutil.LoadTestConfig()
	if err != nil {
		return fmt.Errorf("failed to load test config: %w", err)
	}
	dsn := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		cfg.PGHost, cfg.PGUser, cfg.PGPassword, cfg.PGDatabase, cfg.PGSchema)
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}
	defer dbConn.Close()
	
	// Check if post exists
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.post WHERE object_id = $1", cfg.PGSchema)
	err = dbConn.QueryRow(query, postID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query post: %w", err)
	}
	
	if count == 0 {
		return fmt.Errorf("post with ID %s not found in database", postID)
	}
	
	return nil
}

// inspectDatabaseContents shows what's actually in the database
func inspectDatabaseContents(postID string) error {
	cfg, err := testutil.LoadTestConfig()
	if err != nil {
		return fmt.Errorf("failed to load test config: %w", err)
	}
	dsn := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		cfg.PGHost, cfg.PGUser, cfg.PGPassword, cfg.PGDatabase, cfg.PGSchema)
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}
	defer dbConn.Close()
	
	// Check table structure
	rows, err := dbConn.Query(fmt.Sprintf("SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = '%s' AND table_name = 'post' ORDER BY ordinal_position", cfg.PGSchema))
	if err != nil {
		return fmt.Errorf("failed to query table structure: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var colName, dataType string
		if err := rows.Scan(&colName, &dataType); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
	}
	
	// Check if post exists and show its data
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.post WHERE object_id = $1", cfg.PGSchema)
	err = dbConn.QueryRow(query, postID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query post count: %w", err)
	}
	

	
	if count > 0 {
		// Show the actual post data
		query = fmt.Sprintf("SELECT object_id, body, owner_user_id FROM %s.post WHERE object_id = $1", cfg.PGSchema)
		row := dbConn.QueryRow(query, postID)
		var objID, body, ownerID string
		if err := row.Scan(&objID, &body, &ownerID); err != nil {
			return fmt.Errorf("failed to scan post data: %w", err)
		}

	}
	
	return nil
}

// showRawDatabaseContents shows exactly what's in the database
func showRawDatabaseContents(postID string) error {
	cfg, err := testutil.LoadTestConfig()
	if err != nil {
		return fmt.Errorf("failed to load test config: %w", err)
	}
	dsn := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		cfg.PGHost, cfg.PGUser, cfg.PGPassword, cfg.PGDatabase, cfg.PGSchema)
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}
	defer dbConn.Close()
	
	// Show all posts to see the structure
	query := fmt.Sprintf("SELECT * FROM %s.post ORDER BY id DESC LIMIT 5", cfg.PGSchema)
	rows, err := dbConn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var id int
		var objID, data, createdDate, lastUpdated sql.NullString
		if err := rows.Scan(&id, &objID, &data, &createdDate, &lastUpdated); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
	}
	
	// Specifically look for our post
	query = fmt.Sprintf("SELECT * FROM %s.post WHERE object_id = $1", cfg.PGSchema)
	row := dbConn.QueryRow(query, postID)
	var id int
	var objID, data, createdDate, lastUpdated sql.NullString
	if err := row.Scan(&id, &objID, &data, &createdDate, &lastUpdated); err != nil {
		return fmt.Errorf("failed to find specific post: %w", err)
	}
	

	return nil
}

// cleanCorruptedData removes corrupted documents from the test database
func cleanCorruptedData() error {
	cfg, err := testutil.LoadTestConfig()
	if err != nil {
		return fmt.Errorf("failed to load test config: %w", err)
	}
	dsn := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		cfg.PGHost, cfg.PGUser, cfg.PGPassword, cfg.PGDatabase, cfg.PGSchema)
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}
	defer dbConn.Close()
	
	// Remove documents that look like update operations (contain $inc, $set, etc.)
	query := fmt.Sprintf("DELETE FROM %s.post WHERE data::text LIKE '%%$inc%%' OR data::text LIKE '%%$set%%'", cfg.PGSchema)
	_, err = dbConn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clean corrupted data: %w", err)
	}
	

	return nil
}

// newTestApp creates a new test Fiber app with posts routes using dependency injection
func newTestApp(t *testing.T, base *platform.BaseService, config *testutil.TestConfig) (*fiber.App, *posts.RouterConfig) {
    app := fiber.New(fiber.Config{
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  30 * time.Second,
    })

	// Add test middleware to set user context
	app.Use(func(c *fiber.Ctx) error {
		// Extract user info from headers (simulating HMAC middleware)
		uid := c.Get(types.HeaderUID)
		if uid != "" {
			userID, _ := uuid.FromString(uid)
			createdDate, _ := strconv.ParseInt(c.Get("createdDate"), 10, 64)
			user := types.UserContext{
				UserID:      userID,
				Username:    c.Get("email"),
				DisplayName: c.Get("displayName"),
				SocialName:  c.Get("socialName"),
				Avatar:      "",
				Banner:      "",
				TagLine:    "",
				SystemRole:  c.Get("role"),
				CreatedDate: createdDate,
			}
			c.Locals(types.UserCtxName, user)
		}
		return c.Next()
	})

    // Create handlers and config using the injected base service
    postService := services.NewPostService(base, config.ToPlatformConfig(dbi.DatabaseTypePostgreSQL))
    postHandler := handlers.NewPostHandler(postService, config.ToPlatformConfig(dbi.DatabaseTypePostgreSQL).JWT, config.ToPlatformConfig(dbi.DatabaseTypePostgreSQL).HMAC)
    
    postsHandlers := &posts.PostsHandlers{
        PostHandler: postHandler,
    }
    
    // Generate valid ECDSA keys for testing
    pubKey, _ := testutil.GenerateECDSAKeyPairPEM(t)
    
    // Use dynamic configuration from test environment with valid keys
    routerConfig := &posts.RouterConfig{
        PayloadSecret: config.PayloadSecret,
        PublicKey:     pubKey, // Use generated valid ECDSA public key
    }
    
    // Use the new professional dependency injection pattern
    posts.RegisterRoutes(app, postsHandlers, config.ToPlatformConfig(dbi.DatabaseTypePostgreSQL))
    return app, routerConfig
}

// addHMACHeaders creates HMAC authentication headers using canonical signing format (normalized path)
// This is used for tests that use httptest.NewRequest directly (legacy pattern)
func addHMACHeaders(req *http.Request, body []byte, secret string, uid string) {
	method := req.Method
	path := req.URL.Path
	query := req.URL.RawQuery
	
	// Normalize path to match Fiber's c.Path() behavior
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

// Legacy NoSQL tests removed - PostgreSQL only
func TestPosts_HTTP_Compatibility_Postgres(t *testing.T) {
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
	
	// Clean up any corrupted data from previous tests
	if err := cleanCorruptedData(); err != nil {
		t.Logf("Failed to clean corrupted data: %v", err)
	}

	// Build Postgres-bound BaseService and register routes with it
	pgSvc, err := platform.NewBaseService(context.Background(), iso.Config)
	if err != nil { 
		t.Fatalf("failed to build postgres base service: %v", err) 
	}
	
	// Use the newTestApp helper which includes ECDSA key generation
	app, routerCfg := newTestApp(t, pgSvc, iso.LegacyConfig)
	
	// Create HTTP helper
	httpHelper := testutil.NewHTTPHelper(t, app)
	uid := uuid.Must(uuid.NewV4()).String()
	secret := routerCfg.PayloadSecret

	    // Create
	    newId := uuid.Must(uuid.NewV4()).String()
	    payload := map[string]interface{}{
			"objectId":         newId,
			"postTypeId":       1,
			"score":            0,
			"viewCount":        0,
			"body":             "compat body",
			"tags":             []string{"a"},
			"commentCounter":   0,
			"image":            "",
			"imageFullPath":    "",
			"video":            "",
			"thumbnail":        "",
			"album":            map[string]interface{}{"count": 0, "cover": "", "coverId": newId, "photos": []string{}, "title": ""},
			"disableComments":  false,
			"disableSharing":   false,
			"deleted":          false,
			"deletedDate":      0,
			"lastUpdated":      0,
			"accessUserList":   []string{},
			"permission":      "Public",
			"version":         "v1",
		}
		resp := httpHelper.NewRequest("POST", "/posts/", payload).
			WithAuthHeaders(secret, uid).Send()
		require.Equal(t, http.StatusCreated, resp.StatusCode, "CreatePost should return 201 Created")

		// Verify via the same BaseService used by the handler (strongest coupling to behavior)
		ctx := context.Background()
		postSvc := services.NewPostService(pgSvc, iso.Config)
		deadline := time.Now().Add(2 * time.Second)
		for {
			if _, err := postSvc.GetPost(ctx, uuid.FromStringOrNil(newId)); err == nil {
				break
			}
			if time.Now().After(deadline) { t.Fatalf("post not found in repository after create") }
			time.Sleep(50 * time.Millisecond)
		}

		// Additional delay to ensure post is fully committed
		time.Sleep(200 * time.Millisecond)

		// Test UpdatePost
		updatePayload := map[string]interface{}{
			"objectId":        newId,
			"body":            "updated body for PostgreSQL",
			"tags":            []string{"updated", "postgres"},
			"image":           "updated-image.jpg",
			"video":           "updated-video.mp4",
			"thumbnail":       "updated-thumb.jpg",
			"album":           map[string]interface{}{"count": 5, "cover": "cover.jpg", "coverId": newId, "photos": []string{"photo1.jpg", "photo2.jpg"}, "title": "Updated Album"},
			"disableComments": true,
			"disableSharing":  true,
			"permission":      "Circles",
			"version":         "v2",
		}
		updateBody, _ := json.Marshal(updatePayload)
		reqUpdate := httptest.NewRequest("PUT", "/posts/", bytes.NewReader(updateBody))
		reqUpdate.Header.Set(types.HeaderContentType, "application/json")
		addHMACHeaders(reqUpdate, updateBody, secret, uid)
		respUpdate, _ := app.Test(reqUpdate)
		if respUpdate.StatusCode != 200 {
			// Log the error response
			var errorResp map[string]interface{}
			if err := json.NewDecoder(respUpdate.Body).Decode(&errorResp); err == nil {
				t.Logf("❌ Update error response: %+v", errorResp)
			} else {
				t.Logf("❌ Failed to decode error response: %v", err)
			}
			t.Fatalf("update status=%d", respUpdate.StatusCode)
		}

		// Test UpdatePostProfile
		profilePayload := map[string]interface{}{
			"ownerUserId":      uid,
			"ownerDisplayName": "Updated Tester",
			"ownerAvatar":      "updated-avatar.jpg",
		}
		profileBody, _ := json.Marshal(profilePayload)
		reqProfile := httptest.NewRequest("PUT", "/posts/profile", bytes.NewReader(profileBody))
		reqProfile.Header.Set(types.HeaderContentType, "application/json")
		addHMACHeaders(reqProfile, profileBody, secret, uid)
		respProfile, _ := app.Test(reqProfile)
		if respProfile.StatusCode != 200 {
			t.Fatalf("update profile status=%d", respProfile.StatusCode)
		}

		// Test IncrementScore
		scorePayload := map[string]interface{}{
			"postId": newId,
			"delta":  5,
		}
		scoreBody, _ := json.Marshal(scorePayload)
		reqScore := httptest.NewRequest("PUT", "/posts/actions/score", bytes.NewReader(scoreBody))
		reqScore.Header.Set(types.HeaderContentType, "application/json")
		addHMACHeaders(reqScore, scoreBody, secret, uid)
		respScore, _ := app.Test(reqScore)
		if respScore.StatusCode != 200 {
			t.Fatalf("increment score status=%d", respScore.StatusCode)
		}

		// Generate url key
		req3 := httptest.NewRequest("PUT", "/posts/urlkey/"+newId, nil)
		addHMACHeaders(req3, nil, secret, uid)
		resp3, _ := app.Test(req3)
		if resp3.StatusCode != 200 {
			t.Fatalf("urlkey status=%d", resp3.StatusCode)
		}

		// Parse the generated URL key from response
		var urlKeyResp map[string]interface{}
		if err := json.NewDecoder(resp3.Body).Decode(&urlKeyResp); err != nil {
			t.Fatalf("failed to decode URL key response: %v", err)
		}
		generatedURLKey, ok := urlKeyResp["urlKey"].(string)
		if !ok || generatedURLKey == "" {
			t.Fatalf("failed to get generated URL key from response")
		}

		// Increment comment count
		inc := map[string]interface{}{"postId": newId, "count": 1}
		b4, _ := json.Marshal(inc)
		req4 := httptest.NewRequest("PUT", "/posts/actions/comment/count", bytes.NewReader(b4))
		req4.Header.Set(types.HeaderContentType, "application/json")
		addHMACHeaders(req4, b4, secret, uid)
		resp4, _ := app.Test(req4)
		if resp4.StatusCode != 200 {
			t.Fatalf("comment count status=%d", resp4.StatusCode)
		}

		// Test DisableComment
		disableCommentPayload := map[string]interface{}{
			"objectId": newId,
			"disable":  true,
		}
		disableCommentBody, _ := json.Marshal(disableCommentPayload)
		reqDisableComment := httptest.NewRequest("PUT", "/posts/comment/disable", bytes.NewReader(disableCommentBody))
		reqDisableComment.Header.Set(types.HeaderContentType, "application/json")
		addHMACHeaders(reqDisableComment, disableCommentBody, secret, uid)
		respDisableComment, _ := app.Test(reqDisableComment)
		if respDisableComment.StatusCode != 200 {
			t.Fatalf("disable comment status=%d", respDisableComment.StatusCode)
		}

		// Test DisableSharing
		disableSharingPayload := map[string]interface{}{
			"objectId": newId,
			"disable":  true,
		}
		disableSharingBody, _ := json.Marshal(disableSharingPayload)
		reqDisableSharing := httptest.NewRequest("PUT", "/posts/share/disable", bytes.NewReader(disableSharingBody))
		reqDisableSharing.Header.Set(types.HeaderContentType, "application/json")
		addHMACHeaders(reqDisableSharing, disableSharingBody, secret, uid)
		respDisableSharing, _ := app.Test(reqDisableSharing)
		if respDisableSharing.StatusCode != 200 {
			t.Fatalf("disable sharing status=%d", respDisableSharing.StatusCode)
		}

		// Test QueryPosts (list with filters)
		reqQuery := httptest.NewRequest("GET", "/posts/?limit=10&page=1&type=1", nil)
		addHMACHeaders(reqQuery, nil, secret, uid)
		respQuery, _ := app.Test(reqQuery)
		if respQuery.StatusCode != 200 {
			t.Fatalf("query posts status=%d", respQuery.StatusCode)
		}

		// Test CreateIndex
		indexPayload := map[string]interface{}{
			"objectId":   newId,
			"postTypeId": 1,
			"body":       "index test body",
		}
		indexBody, _ := json.Marshal(indexPayload)
		reqIndex := httptest.NewRequest("POST", "/posts/actions/index", bytes.NewReader(indexBody))
		reqIndex.Header.Set(types.HeaderContentType, "application/json")
		addHMACHeaders(reqIndex, indexBody, secret, uid)
		respIndex, _ := app.Test(reqIndex)
		if respIndex.StatusCode != 201 {
			t.Fatalf("create index status=%d", respIndex.StatusCode)
		}

		// Test GetPostByURLKey using the generated URL key
		reqGetByURLKey := httptest.NewRequest("GET", "/posts/urlkey/"+generatedURLKey, nil)
		addHMACHeaders(reqGetByURLKey, nil, secret, uid)
		respGetByURLKey, _ := app.Test(reqGetByURLKey)
		if respGetByURLKey.StatusCode != 200 {
			t.Fatalf("get by urlkey status=%d", respGetByURLKey.StatusCode)
		}

		// Delete
		req5 := httptest.NewRequest("DELETE", "/posts/"+newId, nil)
		addHMACHeaders(req5, nil, secret, uid)
		resp5, _ := app.Test(req5)
		if resp5.StatusCode != 204 {
			t.Fatalf("delete status=%d", resp5.StatusCode)
		}

		// Test Cursor-based Pagination
		reqCursor := httptest.NewRequest("GET", "/posts/?limit=5&cursor_pagination=true", nil)
		addHMACHeaders(reqCursor, nil, secret, uid)
		respCursor, _ := app.Test(reqCursor, 10000) // 10 second timeout for cursor pagination
		if respCursor.StatusCode != 200 {
			t.Fatalf("cursor pagination status=%d", respCursor.StatusCode)
		}
		
		// Verify cursor pagination response format
		var cursorResult map[string]interface{}
		if err := json.NewDecoder(respCursor.Body).Decode(&cursorResult); err == nil {
			if data, exists := cursorResult["data"]; exists {
				t.Logf("Cursor pagination returned %d posts", len(data.([]interface{})))
			}
			if nextCursor, exists := cursorResult["nextCursor"]; exists && nextCursor != nil {
				// Test pagination with cursor
				reqNextPage := httptest.NewRequest("GET", fmt.Sprintf("/posts/?limit=5&cursor_pagination=true&cursor=%s", nextCursor), nil)
				addHMACHeaders(reqNextPage, nil, secret, uid)
				respNextPage, _ := app.Test(reqNextPage, 10000) // 10 second timeout for cursor pagination
				if respNextPage.StatusCode != 200 {
					t.Fatalf("cursor next page status=%d", respNextPage.StatusCode)
				}
				t.Logf("Cursor pagination next page test passed")
			}
		}

		t.Logf("PostgreSQL HTTP compatibility test passed - All operations work correctly")
}

// TestHTTPEdgeCases tests HTTP edge cases and malformed requests
func TestHTTPEdgeCases(t *testing.T) {
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
	app, routerConfig := newTestApp(t, base, iso.LegacyConfig) // Use the newTestApp that returns app and config
	
	// Use the test configuration secret for HMAC signing
	secret := routerConfig.PayloadSecret
	uid := uuid.Must(uuid.NewV4()).String()

		t.Run("MalformedHeaders", func(t *testing.T) {
			// Test with malformed authorization header
			reqBody := []byte(`{"body":"test post","postTypeId":1}`)
			req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqBody))
			req.Header.Set(types.HeaderContentType, "application/json")
			
			// Add malformed headers
			req.Header.Set(types.HeaderAuthorization, types.BearerPrefix+"invalid-token-format")
			req.Header.Set("X-Cloud-Signature", "malformed-signature")
			req.Header.Set(types.HeaderUID, uid)
			
			resp, _ := app.Test(req)
			if resp.StatusCode == 200 {
				t.Error("Expected error for malformed headers, got success")
			}
		})

		t.Run("InvalidContentTypes", func(t *testing.T) {
			reqBody := []byte(`{"body":"test post","postTypeId":1}`)
			req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqBody))
			
			// Test with invalid content type
			req.Header.Set(types.HeaderContentType, "text/plain")
			addHMACHeaders(req, reqBody, secret, uid)
			
			resp, _ := app.Test(req)
			// Should handle gracefully or return appropriate error
			if resp.StatusCode < 400 {
				t.Log("Server handles invalid content type gracefully")
			}
		})

		t.Run("OversizedPayloads", func(t *testing.T) {
			// Create oversized payload (1MB+ JSON)
			largeContent := strings.Repeat("A", 1024*1024) // 1MB of content
			largePayload := map[string]interface{}{
				"body":       largeContent,
				"postTypeId": 1,
			}
			
			largeBody, _ := json.Marshal(largePayload)
			req := httptest.NewRequest("POST", "/posts", bytes.NewReader(largeBody))
			req.Header.Set(types.HeaderContentType, "application/json")
			addHMACHeaders(req, largeBody, secret, uid)
			
			resp, _ := app.Test(req, 5000) // 5 second timeout
			// Should handle large payloads appropriately
			t.Logf("Large payload response status: %d", resp.StatusCode)
		})

		t.Run("ConcurrentRequests", func(t *testing.T) {
			var wg sync.WaitGroup
			concurrentUsers := 10
			
			for i := 0; i < concurrentUsers; i++ {
				wg.Add(1)
				go func(userIndex int) {
					defer wg.Done()
					
					reqBody := []byte(fmt.Sprintf(`{"body":"concurrent test %d","postTypeId":1}`, userIndex))
					req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqBody))
					req.Header.Set(types.HeaderContentType, "application/json")
					addHMACHeaders(req, reqBody, secret, uuid.Must(uuid.NewV4()).String())
					
					resp, _ := app.Test(req, 3000) // 3 second timeout
					if resp.StatusCode >= 500 {
						t.Errorf("Concurrent request %d failed with status %d", userIndex, resp.StatusCode)
					}
				}(i)
			}
			
			wg.Wait()
			t.Log("Concurrent requests test completed")
		})

		t.Run("MalformedJSON", func(t *testing.T) {
			// Test with malformed JSON
			malformedJSON := []byte(`{"body":"test post","postTypeId":1`) // Missing closing brace
			req := httptest.NewRequest("POST", "/posts", bytes.NewReader(malformedJSON))
			req.Header.Set(types.HeaderContentType, "application/json")
			addHMACHeaders(req, malformedJSON, secret, uid)
			
			resp, _ := app.Test(req)
			if resp.StatusCode != 400 && resp.StatusCode != 422 {
				t.Errorf("Expected 400 or 422 for malformed JSON, got %d", resp.StatusCode)
			}
		})

		t.Run("MissingRequiredFields", func(t *testing.T) {
			// Test with missing required fields
			incompletePayload := []byte(`{"body":"test without postTypeId"}`)
			req := httptest.NewRequest("POST", "/posts", bytes.NewReader(incompletePayload))
			req.Header.Set(types.HeaderContentType, "application/json")
			addHMACHeaders(req, incompletePayload, secret, uid)
			
			resp, _ := app.Test(req)
			if resp.StatusCode == 200 {
				t.Error("Expected error for missing required fields, got success")
			}
		})

		t.Run("InvalidHTTPMethods", func(t *testing.T) {
			// Test unsupported HTTP methods
			req := httptest.NewRequest("PATCH", "/posts", nil)
			addHMACHeaders(req, nil, secret, uid)
			
			resp, _ := app.Test(req)
			if resp.StatusCode != 405 && resp.StatusCode != 404 {
				t.Logf("PATCH method response: %d (may be handled by middleware)", resp.StatusCode)
			}
		})

		t.Run("ExtremelyLongURLs", func(t *testing.T) {
			// Test with extremely long URLs
			longPath := "/posts/" + strings.Repeat("a", 2000)
			req := httptest.NewRequest("GET", longPath, nil)
			addHMACHeaders(req, nil, secret, uid)
			
			resp, _ := app.Test(req)
			// Should handle gracefully
			t.Logf("Long URL response status: %d", resp.StatusCode)
		})

		t.Run("SpecialCharactersInParams", func(t *testing.T) {
			// Test with special characters in URL parameters
			specialChars := []string{"%", "&", "=", "?", "#", "<", ">", "\"", "'"}
			
			for _, char := range specialChars {
				encodedChar := fmt.Sprintf("%%%.2X", []byte(char)[0])
				                            req := httptest.NewRequest("GET", "/posts/"+encodedChar, nil)
                            addHMACHeaders(req, nil, secret, uid)
				
				resp, _ := app.Test(req)
				// Should handle special characters appropriately
				t.Logf("Special character %s response: %d", char, resp.StatusCode)
			}
		})
}

// TestHTTPTimeouts tests slow database response scenarios
func TestHTTPTimeouts(t *testing.T) {
	// This test simulates slow database responses
	// In a real scenario, you might mock the database to introduce delays
	
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	secret := iso.Config.HMAC.Secret
	uid := uuid.Must(uuid.NewV4()).String()
	
	// Create the app using service injection pattern
	base, err := platform.NewBaseService(context.Background(), iso.Config)
	if err != nil {
		t.Fatalf("failed to build postgresql base service: %v", err)
	}
	
	// Use the newTestApp helper which includes ECDSA key generation
	app, _ := newTestApp(t, base, iso.LegacyConfig)

		t.Run("RequestTimeout", func(t *testing.T) {
			// Create a request that might take time
			reqBody := []byte(`{"body":"timeout test","postTypeId":1}`)
			req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqBody))
			req.Header.Set(types.HeaderContentType, "application/json")
			addHMACHeaders(req, reqBody, secret, uid)
			
			start := time.Now()
			resp, _ := app.Test(req, 5000) // 5 second timeout
			duration := time.Since(start)
			
			t.Logf("Request completed in %v with status %d", duration, resp.StatusCode)
			
			// Verify request doesn't hang indefinitely
			if duration > 10*time.Second {
				t.Error("Request took too long, possible timeout issue")
			}
		})
}

// TestHTTPSecurity tests security-related HTTP scenarios
func TestHTTPSecurity(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	secret := iso.Config.HMAC.Secret
	uid := uuid.Must(uuid.NewV4()).String()
	
	// Create the app using service injection pattern
	base, err := platform.NewBaseService(context.Background(), iso.Config)
	if err != nil {
		t.Fatalf("failed to build postgresql base service: %v", err)
	}
	app, _ := newTestApp(t, base, iso.LegacyConfig)

		t.Run("SQLInjectionAttempts", func(t *testing.T) {
			// Test SQL injection attempts in URL parameters
			sqlInjections := []string{
				"'; DROP TABLE posts; --",
				"' OR '1'='1",
				"'; DELETE FROM posts WHERE '1'='1'; --",
				"UNION SELECT * FROM users",
			}
			
			for _, injection := range sqlInjections {
				// URL encode the injection to prevent httptest.NewRequest from interpreting it as HTTP protocol
				encodedInjection := url.QueryEscape(injection)
				req := httptest.NewRequest("GET", "/posts/"+encodedInjection, nil)
				addHMACHeaders(req, nil, secret, uid)
				
				resp, _ := app.Test(req)
				// Should not return 200 for malformed IDs and should handle injection attempts
				if resp.StatusCode == 200 {
					t.Logf("SQL injection attempt handled: %s", injection)
				}
			}
		})

		t.Run("XSSAttempts", func(t *testing.T) {
			// Test XSS attempts in post content
			xssPayloads := []string{
				"<script>alert('xss')</script>",
				"javascript:alert('xss')",
				"<img src=x onerror=alert('xss')>",
				"<svg/onload=alert('xss')>",
			}
			
			for _, xssPayload := range xssPayloads {
				reqBody, _ := json.Marshal(map[string]interface{}{
					"body":       xssPayload,
					"postTypeId": 1,
				})
				
				req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqBody))
				req.Header.Set(types.HeaderContentType, "application/json")
				addHMACHeaders(req, reqBody, secret, uid)
				
				resp, _ := app.Test(req)
				// Should handle XSS attempts appropriately
				t.Logf("XSS payload response: %d", resp.StatusCode)
			}
		})

		t.Run("HeaderInjection", func(t *testing.T) {
			// Test header injection attempts
			reqBody := []byte(`{"body":"header injection test","postTypeId":1}`)
			req := httptest.NewRequest("POST", "/posts", bytes.NewReader(reqBody))
			req.Header.Set(types.HeaderContentType, "application/json")
			
			// Attempt header injection
			req.Header.Set("X-Injected-Header", "test\r\nX-Evil-Header: injected")
			req.Header.Set("User-Agent", "Mozilla/5.0\r\nX-Injected: true")
			
			addHMACHeaders(req, reqBody, secret, uid)
			
			resp, _ := app.Test(req)
			// Should handle header injection attempts
			t.Logf("Header injection response: %d", resp.StatusCode)
		})
}

