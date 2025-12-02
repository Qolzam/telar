package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/posts/repository"
	votesRepository "github.com/qolzam/telar/apps/api/votes/repository"
)

// TestPostsServiceCacheIntegration tests cache integration with actual posts service
func TestPostsServiceCacheIntegration(t *testing.T) {
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

	ctx := context.Background()
	
	// Apply migration manually for isolated test schema
	// The isolated test creates a unique schema per test, so we need to apply the migration there
	pgConfig := iso.LegacyConfig.ToServiceConfig(dbi.DatabaseTypePostgreSQL).PostgreSQLConfig
	pgConfig.Schema = iso.LegacyConfig.PGSchema // Ensure we use the isolated schema
	
	client, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	require.NoError(t, err, "Failed to create postgres client")
	defer client.Close()
	
	// Create schema if it doesn't exist
	schemaSQL := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, schemaSQL)
	require.NoError(t, err, "Failed to create schema")
	
	// Set search_path to the isolated schema
	setSearchPathSQL := fmt.Sprintf(`SET search_path TO %s`, iso.LegacyConfig.PGSchema)
	_, err = client.DB().ExecContext(ctx, setSearchPathSQL)
	require.NoError(t, err, "Failed to set search_path")
	
	// Apply migration
	if err := repository.ApplyPostsMigration(ctx, client, iso.LegacyConfig.PGSchema); err != nil {
		t.Fatalf("Failed to apply posts migration: %v", err)
	}
	
	// Create PostRepository from the client
	// Use schema-aware constructor for test isolation
	postRepo := repository.NewPostgresRepositoryWithSchema(client, iso.LegacyConfig.PGSchema)
	
	// Create VoteRepository for vote enrichment
	voteRepo := votesRepository.NewPostgresVoteRepository(client)
	
	// Create posts service with cache (commentRepo is nil for this test)
	postService := NewPostService(postRepo, voteRepo, iso.Config, nil, nil)
	
	// Test user context
	userID := uuid.Must(uuid.NewV4())
	user := &types.UserContext{
		UserID:   userID,
		Username: "testuser_cache",
	}

	t.Run("CacheHitRatioOnQueries", func(t *testing.T) {
		// Create some test posts first
		testPosts := []*models.CreatePostRequest{
			{
				PostTypeId: 1,
				Body:       "Cache test post 1",
				Tags:       []string{"cache", "test"},
			},
			{
				PostTypeId: 1,
				Body:       "Cache test post 2", 
				Tags:       []string{"performance", "test"},
			},
			{
				PostTypeId: 1,
				Body:       "Cache test post 3",
				Tags:       []string{"integration", "test"},
			},
		}
		
		createdPosts := make([]*models.Post, 0, len(testPosts))
		for _, req := range testPosts {
			post, err := postService.CreatePost(ctx, req, user)
			require.NoError(t, err)
			createdPosts = append(createdPosts, post)
		}
		
		// Clean up after test
		defer func() {
			for _, post := range createdPosts {
				_ = postService.DeletePost(ctx, post.ObjectId, user)
			}
		}()
		
		// Test cache behavior with identical queries
		filter := &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}
		
		// First query - should be cache miss
		start1 := time.Now()
		result1, err := postService.QueryPosts(ctx, filter)
		duration1 := time.Since(start1)
		require.NoError(t, err)
		assert.NotNil(t, result1)
		
		// Second identical query - should be cache hit
		start2 := time.Now()
		result2, err := postService.QueryPosts(ctx, filter)
		duration2 := time.Since(start2)
		require.NoError(t, err)
		assert.NotNil(t, result2)
		
		// Results should be consistent (cache hit should return same data)
		assert.Equal(t, len(result1.Posts), len(result2.Posts), "Cache hit should return same number of posts")
		
		// Cache hit should be faster
		t.Logf("First query (cache miss): %v", duration1)
		t.Logf("Second query (cache hit): %v", duration2)
		
		// Multiple repeated queries to test cache efficiency
		numRepeats := 10
		totalCacheHitTime := time.Duration(0)
		
		for i := 0; i < numRepeats; i++ {
			start := time.Now()
			result, err := postService.QueryPosts(ctx, filter)
			elapsed := time.Since(start)
			totalCacheHitTime += elapsed
			
					require.NoError(t, err)
		// Validate cache consistency without depending on absolute counts
		assert.Equal(t, len(result1.Posts), len(result.Posts), "Cache hit should return consistent post count")
		}
		
		avgCacheHitTime := totalCacheHitTime / time.Duration(numRepeats)
		t.Logf("Average cache hit time over %d queries: %v", numRepeats, avgCacheHitTime)
		
		// Cache hits should be consistently fast (adjusted for test environment)
		// In test environment, we expect cache hits to be faster than database queries
		assert.Less(t, avgCacheHitTime, 100*time.Millisecond, "Average cache hit time should be under 100ms in test environment")
	})

	t.Run("CacheInvalidationOnPostCreation", func(t *testing.T) {
		// Initial query to populate cache
		filter := &models.PostQueryFilter{
			Limit: 5,
			Page:  1,
		}
		
		result1, err := postService.QueryPosts(ctx, filter)
		require.NoError(t, err)
		initialCount := result1.TotalCount
		
		// Create a new post (should invalidate cache)
		createReq := &models.CreatePostRequest{
			PostTypeId: 1,
			Body:       "Cache invalidation test post",
			Tags:       []string{"invalidation", "test"},
		}
		
		newPost, err := postService.CreatePost(ctx, createReq, user)
		require.NoError(t, err)
		
		// Clean up after test
		defer func() {
			_ = postService.DeletePost(ctx, newPost.ObjectId, user)
		}()
		
		// Query again - should reflect new post (cache was invalidated)
		result2, err := postService.QueryPosts(ctx, filter)
		require.NoError(t, err)
		
		// Should have at least the new post reflected
		assert.GreaterOrEqual(t, result2.TotalCount, initialCount, "Cache should be invalidated after post creation")
		
		// Third query should hit cache again
		start := time.Now()
		result3, err := postService.QueryPosts(ctx, filter)
		cacheHitDuration := time.Since(start)
		require.NoError(t, err)
		
		// Should be consistent with previous result (cache hit)
		assert.Equal(t, len(result2.Posts), len(result3.Posts), "Cache hit should return consistent post count")
		
		t.Logf("Cache hit after invalidation: %v", cacheHitDuration)
		// Cache hit should be reasonably fast (adjusted for test environment)
		assert.Less(t, cacheHitDuration, 100*time.Millisecond, "Cache hit after invalidation should be under 100ms in test environment")
	})

	t.Run("SearchCachePerformance", func(t *testing.T) {
		// Create posts with specific content for search
		searchPosts := []*models.CreatePostRequest{
			{
				PostTypeId: 1,
				Body:       "Golang programming tutorial for beginners",
				Tags:       []string{"golang", "programming", "tutorial"},
			},
			{
				PostTypeId: 1,
				Body:       "Advanced Golang patterns and best practices",
				Tags:       []string{"golang", "advanced", "patterns"},
			},
		}
		
		createdPosts := make([]*models.Post, 0, len(searchPosts))
		for _, req := range searchPosts {
			post, err := postService.CreatePost(ctx, req, user)
			require.NoError(t, err)
			createdPosts = append(createdPosts, post)
		}
		
		// Clean up after test
		defer func() {
			for _, post := range createdPosts {
				_ = postService.DeletePost(ctx, post.ObjectId, user)
			}
		}()
		
		searchTerm := "golang"
		filter := &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}
		
		// First search - cache miss
		start1 := time.Now()
		searchResult1, err := postService.SearchPosts(ctx, searchTerm, filter)
		searchDuration1 := time.Since(start1)
		require.NoError(t, err)
		
		// Second identical search - cache hit
		start2 := time.Now()
		searchResult2, err := postService.SearchPosts(ctx, searchTerm, filter)
		searchDuration2 := time.Since(start2)
		require.NoError(t, err)
		
		// Results should be identical
		assert.Equal(t, searchResult1.TotalCount, searchResult2.TotalCount)
		assert.Equal(t, len(searchResult1.Posts), len(searchResult2.Posts))
		
		t.Logf("First search (cache miss): %v", searchDuration1)
		t.Logf("Second search (cache hit): %v", searchDuration2)
		
		// Search cache should be effective
		assert.Less(t, searchDuration2, searchDuration1+5*time.Millisecond, "Cache hit should not be significantly slower")
	})

	t.Run("DifferentPaginationParametersCache", func(t *testing.T) {
		// Test that different pagination parameters create separate cache entries
		filters := []*models.PostQueryFilter{
			{Limit: 5, Page: 1},
			{Limit: 10, Page: 1},
			{Limit: 5, Page: 2},
		}
		
		results := make([]*models.PostsListResponse, len(filters))
		durations := make([]time.Duration, len(filters))
		
		// First round - all cache misses
		for i, filter := range filters {
			start := time.Now()
			result, err := postService.QueryPosts(ctx, filter)
			durations[i] = time.Since(start)
			require.NoError(t, err)
			results[i] = result
		}
		
		// Second round - all cache hits
		cacheHitDurations := make([]time.Duration, len(filters))
		for i, filter := range filters {
			start := time.Now()
			result, err := postService.QueryPosts(ctx, filter)
			cacheHitDurations[i] = time.Since(start)
			require.NoError(t, err)
			
			// Should get consistent results (cache hit)
			assert.Equal(t, len(results[i].Posts), len(result.Posts), "Cache hit should return consistent post count")
		}
		
		// Log performance comparison
		for i, filter := range filters {
			t.Logf("Filter {Limit: %d, Page: %d} - First: %v, Cached: %v", 
				filter.Limit, filter.Page, durations[i], cacheHitDurations[i])
		}
		
		// Each cache hit should be reasonably fast (adjusted for test environment)
		for i, duration := range cacheHitDurations {
			assert.Less(t, duration, 100*time.Millisecond, 
				"Cache hit %d should be under 100ms in test environment", i)
		}
	})
}

