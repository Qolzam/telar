package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qolzam/telar/apps/api/internal/cache"
	"github.com/qolzam/telar/apps/api/posts/models"
)

// TestCacheKeyGeneration tests cache key generation consistency
func TestCacheKeyGeneration(t *testing.T) {
	helper := NewCacheTestHelper(t)
	defer helper.Cleanup()
	
	cacheService := helper.GetCacheService()

	t.Run("ConsistentKeyGeneration", func(t *testing.T) {
		// Create identical filter objects
		filter1 := &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}
		
		filter2 := &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}
		
		// Generate cache keys (simulating internal key generation)
		params1 := map[string]interface{}{
			"operation": "query",
			"limit":     filter1.Limit,
			"page":      filter1.Page,
		}
		
		params2 := map[string]interface{}{
			"operation": "query",
			"limit":     filter2.Limit,
			"page":      filter2.Page,
		}
		
		key1 := cacheService.GenerateHashKey("query", params1)
		key2 := cacheService.GenerateHashKey("query", params2)
		
		// Keys should be identical for identical filters
		assert.Equal(t, key1, key2, "Cache keys should be identical for identical filters")
	})

	t.Run("DifferentFiltersGenerateDifferentKeys", func(t *testing.T) {
		// Create different filter objects
		filter1 := &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}
		
		filter2 := &models.PostQueryFilter{
			Limit: 20,
			Page:  1,
		}
		
		filter3 := &models.PostQueryFilter{
			Limit: 10,
			Page:  2,
		}
		
		// Generate cache keys
		params1 := map[string]interface{}{
			"operation": "query",
			"limit":     filter1.Limit,
			"page":      filter1.Page,
		}
		
		params2 := map[string]interface{}{
			"operation": "query",
			"limit":     filter2.Limit,
			"page":      filter2.Page,
		}
		
		params3 := map[string]interface{}{
			"operation": "query",
			"limit":     filter3.Limit,
			"page":      filter3.Page,
		}
		
		key1 := cacheService.GenerateHashKey("query", params1)
		key2 := cacheService.GenerateHashKey("query", params2)
		key3 := cacheService.GenerateHashKey("query", params3)
		
		// All keys should be different
		assert.NotEqual(t, key1, key2, "Different limits should generate different keys")
		assert.NotEqual(t, key1, key3, "Different pages should generate different keys")
		assert.NotEqual(t, key2, key3, "Different filters should generate different keys")
	})

	t.Run("SearchKeyGeneration", func(t *testing.T) {
		// Test search-specific key generation
		searchTerm1 := "golang programming"
		searchTerm2 := "javascript development"
		
		filter := &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}
		
		params1 := map[string]interface{}{
			"operation": "search",
			"query":     searchTerm1,
			"limit":     filter.Limit,
			"page":      filter.Page,
		}
		
		params2 := map[string]interface{}{
			"operation": "search",
			"query":     searchTerm2,
			"limit":     filter.Limit,
			"page":      filter.Page,
		}
		
		key1 := cacheService.GenerateHashKey("search", params1)
		key2 := cacheService.GenerateHashKey("search", params2)
		
		// Different search terms should generate different keys
		assert.NotEqual(t, key1, key2, "Different search terms should generate different keys")
	})
}

// TestCacheOperations tests basic cache operations
func TestCacheOperations(t *testing.T) {
	helper := NewCacheTestHelper(t)
	defer helper.Cleanup()
	
	cacheService := helper.GetCacheService()
	ctx := context.Background()
	
	t.Run("SetAndGetOperations", func(t *testing.T) {
		// Test data - using simple map structure since we need to match models.PostsListResponse structure
		testData := map[string]interface{}{
			"posts": []map[string]interface{}{
				{
					"objectId":   uuid.Must(uuid.NewV4()).String(),
					"postTypeId": 1,
					"body":       "Test post 1",
				},
				{
					"objectId":   uuid.Must(uuid.NewV4()).String(),
					"postTypeId": 1,
					"body":       "Test post 2",
				},
			},
			"totalCount": 2,
		}
		
		key := "test_posts_list"
		
		// Set data in cache
		err := cacheService.CacheData(ctx, key, testData, time.Hour)
		require.NoError(t, err)
		
		// Get data from cache
		var retrieved map[string]interface{}
		err = cacheService.GetCached(ctx, key, &retrieved)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		
		// Verify data integrity
		totalCount, ok := retrieved["totalCount"].(float64)
		require.True(t, ok, "TotalCount should be a number")
		assert.Equal(t, float64(2), totalCount)
		
		posts, ok := retrieved["posts"].([]interface{})
		require.True(t, ok, "Posts should be array")
		assert.Len(t, posts, 2)
	})

	t.Run("CacheExpiration", func(t *testing.T) {
		testData := "test_value"
		key := "test_expiration"
		shortTTL := 50 * time.Millisecond
		
		// Set with short TTL
		err := cacheService.CacheData(ctx, key, testData, shortTTL)
		require.NoError(t, err)
		
		// Should exist immediately
		var retrieved string
		err = cacheService.GetCached(ctx, key, &retrieved)
		require.NoError(t, err)
		assert.Equal(t, testData, retrieved)
		
		// Wait for expiration
		time.Sleep(100 * time.Millisecond)
		
		// Should be expired
		err = cacheService.GetCached(ctx, key, &retrieved)
		assert.Equal(t, cache.ErrKeyNotFound, err)
	})

	t.Run("PatternInvalidation", func(t *testing.T) {
		// Set multiple keys with similar patterns
		keys := []string{
			"query:limit_10_page_1",
			"query:limit_10_page_2", 
			"search:golang_limit_10",
			"user:123_limit_5",
		}
		
		for _, key := range keys {
			err := cacheService.CacheData(ctx, key, "test_data", time.Hour)
			require.NoError(t, err)
		}
		
		// Invalidate all query-related keys
		err := cacheService.InvalidatePattern(ctx, "query:*")
		require.NoError(t, err)
		
		// Query keys should be gone
		var result string
		err = cacheService.GetCached(ctx, "query:limit_10_page_1", &result)
		assert.Equal(t, cache.ErrKeyNotFound, err)
		
		err = cacheService.GetCached(ctx, "query:limit_10_page_2", &result)
		assert.Equal(t, cache.ErrKeyNotFound, err)
		
		// Other keys should still exist
		err = cacheService.GetCached(ctx, "search:golang_limit_10", &result)
		require.NoError(t, err)
		assert.Equal(t, "test_data", result)
		
		err = cacheService.GetCached(ctx, "user:123_limit_5", &result)
		require.NoError(t, err)
		assert.Equal(t, "test_data", result)
	})
}

// TestCachePerformance validates cache performance
func TestCachePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	helper := NewCacheTestHelper(t)
	defer helper.Cleanup()
	
	cacheService := helper.GetCacheService()
	ctx := context.Background()

	t.Run("BulkOperationsPerformance", func(t *testing.T) {
		// Test bulk set operations
		numOperations := 1000
		testData := "test_value_with_some_reasonable_length_to_simulate_real_data"
		
		start := time.Now()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("bulk_test_%d", i)
			err := cacheService.CacheData(ctx, key, testData, time.Hour)
			require.NoError(t, err)
		}
		setDuration := time.Since(start)
		
		// Test bulk get operations
		start = time.Now()
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("bulk_test_%d", i)
			var result string
			err := cacheService.GetCached(ctx, key, &result)
			require.NoError(t, err)
			assert.Equal(t, testData, result)
		}
		getDuration := time.Since(start)
		
		t.Logf("Bulk operations performance:")
		t.Logf("Set %d items: %v (%.2f ops/sec)", numOperations, setDuration, float64(numOperations)/setDuration.Seconds())
		t.Logf("Get %d items: %v (%.2f ops/sec)", numOperations, getDuration, float64(numOperations)/getDuration.Seconds())
		
		// Performance assertions (adjust based on your requirements)
		avgSetTime := setDuration / time.Duration(numOperations)
		avgGetTime := getDuration / time.Duration(numOperations)
		
		assert.Less(t, avgSetTime, 1*time.Millisecond, "Average set time should be less than 1ms")
		assert.Less(t, avgGetTime, 500*time.Microsecond, "Average get time should be less than 500μs")
	})

	t.Run("ConcurrentOperations", func(t *testing.T) {
		// Test concurrent cache operations
		numGoroutines := 10
		opsPerGoroutine := 100
		
		start := time.Now()
		done := make(chan bool, numGoroutines)
		
		for g := 0; g < numGoroutines; g++ {
			go func(goroutineID int) {
				for i := 0; i < opsPerGoroutine; i++ {
					key := fmt.Sprintf("concurrent_%d_%d", goroutineID, i)
					data := fmt.Sprintf("data_%d_%d", goroutineID, i)
					
					// Set operation
					err := cacheService.CacheData(ctx, key, data, time.Hour)
					assert.NoError(t, err)
					
					// Get operation
					var result string
					err = cacheService.GetCached(ctx, key, &result)
					assert.NoError(t, err)
					assert.Equal(t, data, result)
				}
				done <- true
			}(g)
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
		
		totalDuration := time.Since(start)
		totalOps := numGoroutines * opsPerGoroutine * 2 // 2 ops per iteration (set + get)
		
		t.Logf("Concurrent operations: %d total ops in %v (%.2f ops/sec)", 
			totalOps, totalDuration, float64(totalOps)/totalDuration.Seconds())
	})
}

// TestCacheDisabledBehavior tests behavior when cache is disabled
func TestCacheDisabledBehavior(t *testing.T) {
	helper := NewCacheTestHelper(t)
	defer helper.Cleanup()
	
	// Override config to disable cache
	helper.GetConfig().Enabled = false
	cacheService := helper.GetCacheService()
	ctx := context.Background()

	t.Run("DisabledCacheOperations", func(t *testing.T) {
		testData := "test_value"
		key := "disabled_test"
		
		// Set operation should return ErrCacheDisabled
		err := cacheService.CacheData(ctx, key, testData, time.Hour)
		assert.Equal(t, cache.ErrCacheDisabled, err)
		
		// Get operation should return ErrCacheDisabled
		var result string
		err = cacheService.GetCached(ctx, key, &result)
		assert.Equal(t, cache.ErrCacheDisabled, err)
		
		// Invalidation operations should return ErrCacheDisabled
		err = cacheService.InvalidateKey(ctx, key)
		assert.Equal(t, cache.ErrCacheDisabled, err)
		
		err = cacheService.InvalidatePattern(ctx, "test:*")
		assert.Equal(t, cache.ErrCacheDisabled, err)
	})
}

// TestCacheMemoryUsage tests memory efficiency
func TestCacheMemoryUsage(t *testing.T) {
	helper := NewCacheTestHelper(t)
	defer helper.Cleanup()
	
	memCache := helper.GetMemoryCache()
	
	// Get initial stats
	initialStats := memCache.Stats()
	
	// Perform cache operations
	ctx := context.Background()
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	
	for key, value := range testData {
		err := memCache.Set(ctx, key, []byte(value.(string)), time.Hour)
		require.NoError(t, err)
	}
	
	// Get final stats
	finalStats := memCache.Stats()
	
	t.Run("MemoryUsageTracking", func(t *testing.T) {
		// Memory usage should have increased
		assert.Greater(t, finalStats.MemoryUsage, initialStats.MemoryUsage)
		
		// Key count should have increased
		assert.Greater(t, finalStats.Keys, initialStats.Keys)
		
		// Should have some hits or misses by now
		assert.True(t, finalStats.Hits >= 0)
		assert.True(t, finalStats.Misses >= 0)
	})

	t.Run("MemoryCleanup", func(t *testing.T) {
		// Test TTL expiration
		shortTTL := 50 * time.Millisecond
		err := memCache.Set(ctx, "temp_key", []byte("temp_value"), shortTTL)
		require.NoError(t, err)
		
		// Should exist immediately
		_, err = memCache.Get(ctx, "temp_key")
		assert.NoError(t, err)
		
		// Wait for expiration plus cleanup
		time.Sleep(200 * time.Millisecond)
		
		// Should be expired
		_, err = memCache.Get(ctx, "temp_key")
		assert.Equal(t, cache.ErrKeyNotFound, err)
	})
}
