package performance

import (
	"context"
	"fmt"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/posts/services"
)

// Benchmark single post creation

func BenchmarkPostService_CreatePost(b *testing.B) {
	// Skip if not running database tests
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("RUN_DB_TESTS not set, skipping benchmark")
	}

	ctx := context.Background()
	cfg, err := testutil.LoadTestConfig()
	if err != nil {
		b.Fatalf("failed to load test config: %v", err)
	}

	// Create platform config for the service
	platformCfg := cfg.ToPlatformConfig("postgresql")
	base, err := platform.NewBaseService(ctx, platformCfg)
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	postService := services.NewPostService(base, platformCfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	b.ResetTimer()
	var createdPosts []uuid.UUID

	for i := 0; i < b.N; i++ {
		postID := uuid.Must(uuid.NewV4())
		req := &models.CreatePostRequest{
			ObjectId:   &postID,
			PostTypeId: 1,
			Body:       "benchmark test post",
			Permission: "Public",
		}

		_, err := postService.CreatePost(ctx, req, userCtx)
		if err != nil {
			b.Fatalf("create post error: %v", err)
		}
		createdPosts = append(createdPosts, postID)
	}

	b.StopTimer()
	// Cleanup
	for _, postID := range createdPosts {
		_ = postService.DeleteByOwner(ctx, postID, userCtx.UserID)
	}
}


// TestPostService_CursorPaginationPerformance tests cursor pagination performance with PostgreSQL
func TestPostService_CursorPaginationPerformance(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	ctx := context.Background()
	
	// Create platform config for the service
	platformCfg := iso.Config
	base, err := platform.NewBaseService(ctx, platformCfg)
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}

	postService := services.NewPostService(base, platformCfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "perftest_pg",
		DisplayName: "Performance Test User PostgreSQL",
		SystemRole:  "user",
	}

	// Create test data for pagination performance testing
	numPosts := 500 // Slightly fewer for PostgreSQL to avoid timeout
	postIDs := make([]uuid.UUID, numPosts)
	
	t.Logf("Creating %d test posts for PostgreSQL cursor pagination performance test...", numPosts)
	start := time.Now()
	
	for i := 0; i < numPosts; i++ {
		createReq := &models.CreatePostRequest{
			PostTypeId: 1,
			Body:       fmt.Sprintf("PostgreSQL performance test post #%d for cursor pagination", i),
		}
		
		postID, err := postService.CreatePost(ctx, createReq, userCtx)
		if err != nil {
			t.Fatalf("Failed to create test post %d: %v", i, err)
		}
		postIDs[i] = postID.ObjectId
		
		// Add slight delay to ensure different timestamps
		time.Sleep(time.Millisecond)
	}
	
	setupDuration := time.Since(start)
	t.Logf("Created %d posts in %v", numPosts, setupDuration)

	// Test cursor pagination with PostgreSQL optimizations
	pageSize := 25
	numPages := 5

	t.Logf("Testing PostgreSQL cursor pagination performance...")
	start = time.Now()
	
	filter := &models.PostQueryFilter{
		Limit:         pageSize,
		SortField:     "createdDate",
		SortDirection: "desc",
	}
	
	var cursor string
	totalPosts := 0
	
	for page := 0; page < numPages; page++ {
		if cursor != "" {
			filter.Cursor = cursor
		}
		
		pageStart := time.Now()
		result, err := postService.QueryPostsWithCursor(ctx, filter)
		pageDuration := time.Since(pageStart)
		
		if err != nil {
			t.Fatalf("PostgreSQL cursor pagination failed at page %d: %v", page, err)
		}
		
		totalPosts += len(result.Posts)
		cursor = result.NextCursor
		
		t.Logf("PostgreSQL cursor page %d: %d posts in %v (hasNext: %v)", 
			page+1, len(result.Posts), pageDuration, result.HasNext)
		
		// PostgreSQL should also be fast
		if pageDuration > 150*time.Millisecond {
			t.Logf("PostgreSQL cursor page %d took longer than expected: %v", page+1, pageDuration)
		}
		
		if !result.HasNext {
			break
		}
	}
	
	totalDuration := time.Since(start)
	t.Logf("PostgreSQL cursor pagination: %d posts across %d pages in %v", 
		totalPosts, numPages, totalDuration)

	// Cleanup
	t.Logf("Cleaning up %d test posts...", len(postIDs))
	for _, postID := range postIDs {
		_ = postService.DeleteByOwner(ctx, postID, userCtx.UserID)
	}
}
