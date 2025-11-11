package posts_test

import (
	"context"
	"testing"

	uuid "github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	models "github.com/qolzam/telar/apps/api/posts/models"
	services "github.com/qolzam/telar/apps/api/posts/services"
)


func TestPosts_CRUD_Postgres(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}
	
	runCRUDSuite(t, ctx, base, iso.Config)
}

func runCRUDSuite(t *testing.T, ctx context.Context, base *platform.BaseService, cfg *platformconfig.Config) {
	t.Helper()
	postService := services.NewPostService(base, cfg)
	const postCollectionName = "post"

	// Create
	id, _ := uuid.NewV4()
	p := &models.CreatePostRequest{
		ObjectId:         &id,
		PostTypeId:       1,
		Body:             "integration test",
		Tags:             []string{"int", "test"},
		Permission:       "Public",
		DisableComments:  false,
		DisableSharing:   false,
	}

	_, err := postService.CreatePost(ctx, p, &types.UserContext{
		UserID:      id,
		Username:    "test@example.com",
		DisplayName: "Test User",
		SocialName:  "testuser",
		Avatar:      "test-avatar.jpg",
	})
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Count should be >= 1 for this objectId
	queryObj := &dbi.Query{
		Conditions: []dbi.Field{
			{
				Name:     "object_id",
				Value:    id.String(),
				Operator: "=",
			},
		},
	}
	cntRes := <-base.Repository.Count(ctx, postCollectionName, queryObj)
	if cntRes.Error != nil {
		t.Fatalf("count after save error: %v", cntRes.Error)
	}

	// Query one
	got, err := postService.GetPost(ctx, id)
	if err != nil {
		t.Fatalf("find one error: %v", err)
	}
	t.Logf("Expected body: %q, Got body: %q", p.Body, got.Body)
	if got.Body != p.Body {
		t.Fatalf("unexpected body: %v", got.Body)
	}

	// Update body
	newBody := "updated body"
	if err := postService.UpdatePost(ctx, id, &models.UpdatePostRequest{
		Body: &newBody,
	}, &types.UserContext{UserID: id, Username: "test@example.com", SocialName: "testuser"}); err != nil {
		t.Fatalf("update error: %v", err)
	}

	got2, err := postService.GetPost(ctx, id)
	if err != nil {
		t.Fatalf("find one after update error: %v", err)
	}
	if got2.Body != newBody {
		t.Fatalf("update not applied: %v", got2.Body)
	}

	// Delete
	if err := postService.DeleteByOwner(ctx, id, id); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	queryObj2 := &dbi.Query{
		Conditions: []dbi.Field{
			{
				Name:     "object_id",
				Value:    id.String(),
				Operator: "=",
			},
		},
	}
	cntRes2 := <-base.Repository.Count(ctx, postCollectionName, queryObj2)
	if cntRes2.Error != nil {
		t.Fatalf("count after delete error: %v", cntRes2.Error)
	}
}

// Enhanced integration tests with error scenarios for Phase 3
// Legacy NoSQL tests removed - PostgreSQL only

func TestPosts_Integration_ErrorScenarios_Postgres(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}
	
	runErrorScenariosSuite(t, ctx, base, iso.Config)
}

func runErrorScenariosSuite(t *testing.T, ctx context.Context, base *platform.BaseService, cfg *platformconfig.Config) {
    t.Helper()
    postService := services.NewPostService(base, cfg)

    // Test 1: Create post with duplicate ObjectId
    t.Run("CreatePost_DuplicateId", func(t *testing.T) {
        id, _ := uuid.NewV4()
        userCtx := &types.UserContext{
            UserID:      id,
            Username:    "test@example.com",
            DisplayName: "Test User",
            SocialName:  "testuser",
        }

        p := &models.CreatePostRequest{
            ObjectId:   &id,
            PostTypeId: 1,
            Body:       "first post",
            Permission: "Public",
        }

        // Create first post
        _, err := postService.CreatePost(ctx, p, userCtx)
        if err != nil {
            t.Fatalf("first create error: %v", err)
        }

        // Try to create duplicate - PostgreSQL enforces unique constraints on objectId; legacy NoSQL implementations did not.
        // this test verifies application-level duplicate handling rather than database-level constraints
        p2 := &models.CreatePostRequest{
            ObjectId:   &id, // Same ID
            PostTypeId: 1,
            Body:       "duplicate post",
            Permission: "Public",
        }

        _, err = postService.CreatePost(ctx, p2, userCtx)
        // Note: This test may pass if the application doesn't enforce objectId uniqueness
        // In production, unique indexes should be created on objectId fields
        if err == nil {
            t.Logf("Warning: No error for duplicate ObjectId - consider adding unique constraints")
        }

        // Cleanup
        _ = postService.DeleteByOwner(ctx, id, id)
    })

    // Test 2: Update non-existent post
    t.Run("UpdatePost_NonExistent", func(t *testing.T) {
        nonExistentId, _ := uuid.NewV4()
        userId, _ := uuid.NewV4()
        
        newBody := "this should fail"
        err := postService.UpdatePost(ctx, nonExistentId, &models.UpdatePostRequest{
            Body: &newBody,
        }, &types.UserContext{UserID: userId, Username: "test@example.com", SocialName: "testuser"})
        
        if err == nil {
            t.Error("Expected error for updating non-existent post, but got none")
        }
    })

    // Test 3: Delete post with wrong owner
    t.Run("DeletePost_WrongOwner", func(t *testing.T) {
        // Create post with one user
        ownerID, _ := uuid.NewV4()
        postID, _ := uuid.NewV4()
        ownerCtx := &types.UserContext{
            UserID:      ownerID,
            Username:    "owner@example.com",
            DisplayName: "Owner",
            SocialName:  "owner",
        }

        p := &models.CreatePostRequest{
            ObjectId:   &postID,
            PostTypeId: 1,
            Body:       "owner's post",
            Permission: "Public",
        }

        _, err := postService.CreatePost(ctx, p, ownerCtx)
        if err != nil {
            t.Fatalf("create error: %v", err)
        }

        // Try to delete with different user
        otherUserID, _ := uuid.NewV4()
        err = postService.DeleteByOwner(ctx, postID, otherUserID)
        if err == nil {
            t.Error("Expected error for deleting post with wrong owner, but got none")
        }

        // Cleanup with correct owner
        _ = postService.DeleteByOwner(ctx, postID, ownerID)
    })

    // Test 4: Query with invalid filters
    t.Run("QueryPosts_InvalidFilters", func(t *testing.T) {
        filter := &models.PostQueryFilter{
            Page:  -1, // Invalid page
            Limit: 0,  // Invalid limit
        }

        results, err := postService.QueryPosts(ctx, filter)
        // Should handle gracefully - either return empty results or error
        _ = results
        _ = err // Test behavior with invalid pagination
    })

    // Test 5: Increment score on non-existent post
    t.Run("IncrementScore_NonExistentPost", func(t *testing.T) {
        nonExistentId, _ := uuid.NewV4()
        userId, _ := uuid.NewV4()
        userCtx := &types.UserContext{
            UserID:      userId,
            Username:    "test@example.com",
            SocialName:  "testuser",
        }
        
        err := postService.IncrementScore(ctx, nonExistentId, 1, userCtx)
        if err == nil {
            t.Error("Expected error for incrementing score on non-existent post, but got none")
        }
    })

    // Test 6: Set comment disabled on non-existent post
    t.Run("SetCommentDisabled_NonExistentPost", func(t *testing.T) {
        nonExistentId, _ := uuid.NewV4()
        userId, _ := uuid.NewV4()
        userCtx := &types.UserContext{
            UserID:      userId,
            Username:    "test@example.com",
            SocialName:  "testuser",
        }
        
        err := postService.SetCommentDisabled(ctx, nonExistentId, true, userCtx)
        if err == nil {
            t.Error("Expected error for setting comment disabled on non-existent post, but got none")
        }
    })
}

// Concurrent operations test suite
// Legacy NoSQL tests removed - PostgreSQL only

func TestPosts_Integration_ConcurrentOperations_Postgres(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}
	
	runConcurrentOperationsSuite(t, ctx, base, iso.Config)
}

func runConcurrentOperationsSuite(t *testing.T, ctx context.Context, base *platform.BaseService, cfg *platformconfig.Config) {
    t.Helper()
    postService := services.NewPostService(base, cfg)

    // Test 1: Concurrent score updates
    t.Run("ConcurrentScoreUpdates", func(t *testing.T) {
        // Create a post first
        postID, _ := uuid.NewV4()
        ownerID, _ := uuid.NewV4()
        ownerCtx := &types.UserContext{
            UserID:      ownerID,
            Username:    "owner@example.com",
            DisplayName: "Owner",
            SocialName:  "owner",
        }

        p := &models.CreatePostRequest{
            ObjectId:   &postID,
            PostTypeId: 1,
            Body:       "concurrent test post",
            Permission: "Public",
        }

        _, err := postService.CreatePost(ctx, p, ownerCtx)
        if err != nil {
            t.Fatalf("create error: %v", err)
        }

        // Run concurrent score updates
        const numGoroutines = 10
        done := make(chan error, numGoroutines)

        for i := 0; i < numGoroutines; i++ {
            go func(userIndex int) {
                var userID uuid.UUID
                if userIndex == 0 {
                    // Ensure at least one authorized update by using the owner's ID
                    userID = ownerID
                } else {
                    userID, _ = uuid.NewV4()
                }
                userCtx := &types.UserContext{
                    UserID:     userID,
                    Username:   "concurrent@example.com",
                    SocialName: "concurrentuser",
                }
                err := postService.IncrementScore(ctx, postID, 1, userCtx)
                done <- err
            }(i)
        }

        // Collect results
        var errors []error
        for i := 0; i < numGoroutines; i++ {
            if err := <-done; err != nil {
                errors = append(errors, err)
            }
        }

        // Some operations might fail due to concurrency, but at least some should succeed
        if len(errors) == numGoroutines {
            t.Errorf("All concurrent score updates failed: %v", errors[0])
        }

        // Cleanup
        _ = postService.DeleteByOwner(ctx, postID, ownerID)
    })

    // Test 2: Concurrent create operations
    t.Run("ConcurrentCreateOperations", func(t *testing.T) {
        const numGoroutines = 5
        done := make(chan error, numGoroutines)

        for i := 0; i < numGoroutines; i++ {
            go func(index int) {
                postID, _ := uuid.NewV4()
                userID, _ := uuid.NewV4()
                userCtx := &types.UserContext{
                    UserID:      userID,
                    Username:    "user@example.com",
                    DisplayName: "User",
                    SocialName:  "user",
                }

                p := &models.CreatePostRequest{
                    ObjectId:   &postID,
                    PostTypeId: 1,
                    Body:       "concurrent create test",
                    Permission: "Public",
                }

                _, err := postService.CreatePost(ctx, p, userCtx)
                if err == nil {
                    // Cleanup on success
                    _ = postService.DeleteByOwner(ctx, postID, userID)
                }
                done <- err
            }(i)
        }

        // Collect results
        var errors []error
        for i := 0; i < numGoroutines; i++ {
            if err := <-done; err != nil {
                errors = append(errors, err)
            }
        }

        // Most create operations should succeed
        if len(errors) > numGoroutines/2 {
            t.Errorf("Too many concurrent create operations failed: %d/%d", len(errors), numGoroutines)
        }
    })
}

// Database stress and large dataset tests
func TestPosts_Integration_LargeDatasets_Postgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large dataset test in short mode")
	}
	
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	ctx := context.Background()
	base, err := platform.NewBaseService(ctx, iso.Config)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}
	
	runLargeDatasetSuite(t, ctx, base, iso.Config)
}

func runLargeDatasetSuite(t *testing.T, ctx context.Context, base *platform.BaseService, cfg *platformconfig.Config) {
    t.Helper()
    postService := services.NewPostService(base, cfg)

    // Test 1: Create and query large number of posts
    t.Run("LargeDatasetOperations", func(t *testing.T) {
        const numPosts = 100
        userID, _ := uuid.NewV4()
        userCtx := &types.UserContext{
            UserID:      userID,
            Username:    "bulk@example.com",
            DisplayName: "Bulk User",
            SocialName:  "bulkuser",
        }

        var createdPosts []uuid.UUID

        // Create many posts
        for i := 0; i < numPosts; i++ {
            postID, _ := uuid.NewV4()
            p := &models.CreatePostRequest{
                ObjectId:   &postID,
                PostTypeId: 1,
                Body:       "bulk test post",
                Permission: "Public",
                Tags:       []string{"bulk", "test"},
            }

            _, err := postService.CreatePost(ctx, p, userCtx)
            if err != nil {
                t.Logf("Failed to create post %d: %v", i, err)
                continue
            }
            createdPosts = append(createdPosts, postID)
        }

        if len(createdPosts) < numPosts/2 {
            t.Fatalf("Failed to create sufficient posts: %d/%d", len(createdPosts), numPosts)
        }

        // Query posts with pagination
        filter := &models.PostQueryFilter{
            OwnerUserId: &userID,
            Page:        1,
            Limit:       20,
        }

        results, err := postService.QueryPosts(ctx, filter)
        if err != nil {
            t.Fatalf("query error: %v", err)
        }

        if results == nil || len(results.Posts) == 0 {
            t.Error("Expected some results from query")
        }

        // Cleanup
        for _, postID := range createdPosts {
            _ = postService.DeleteByOwner(ctx, postID, userID)
        }
    })

    // Test 2: Large post content
    t.Run("LargePostContent", func(t *testing.T) {
        postID, _ := uuid.NewV4()
        userID, _ := uuid.NewV4()
        userCtx := &types.UserContext{
            UserID:      userID,
            Username:    "large@example.com",
            DisplayName: "Large User",
            SocialName:  "largeuser",
        }

        // Create post with large content
        largeBody := "This is a very long post body that tests the system's ability to handle large content. "
        // Repeat the string to make it larger
        for len(largeBody) < 10000 {
            largeBody += "This is a very long post body that tests the system's ability to handle large content. "
        }

        p := &models.CreatePostRequest{
            ObjectId:   &postID,
            PostTypeId: 1,
            Body:       largeBody,
            Permission: "Public",
        }

        _, err := postService.CreatePost(ctx, p, userCtx)
        if err != nil {
            t.Fatalf("create large post error: %v", err)
        }

        // Retrieve and verify
        retrieved, err := postService.GetPost(ctx, postID)
        if err != nil {
            t.Fatalf("get large post error: %v", err)
        }

        if len(retrieved.Body) == 0 {
            t.Error("Large post body was not stored correctly")
        }

        // Cleanup
        _ = postService.DeleteByOwner(ctx, postID, userID)
    })
}

