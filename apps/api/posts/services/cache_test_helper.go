package services

import (
	"testing"
	"time"

	"github.com/qolzam/telar/apps/api/internal/cache"
)

// CacheTestHelper provides isolated cache instances for testing
type CacheTestHelper struct {
	memCache    cache.Cache
	cacheConfig *cache.CacheConfig
	cacheService *cache.GenericCacheService
}

// NewCacheTestHelper creates a new isolated cache test helper
func NewCacheTestHelper(t *testing.T) *CacheTestHelper {
	// Create unique cache configuration for this test
	cacheConfig := cache.DefaultCacheConfig()
	cacheConfig.Enabled = true
	cacheConfig.Prefix = "posts_test_" + generateTestID()
	cacheConfig.TTL = time.Hour
	cacheConfig.MaxMemory = 10 * 1024 * 1024 // 10MB limit for tests
	
	// Create isolated memory cache instance
	memCache := cache.NewMemoryCache(cacheConfig)
	
	// Create generic cache service
	cacheService := cache.NewGenericCacheService(memCache, cacheConfig)
	
	return &CacheTestHelper{
		memCache:     memCache,
		cacheConfig:  cacheConfig,
		cacheService: cacheService,
	}
}

// GetCacheService returns the isolated cache service
func (h *CacheTestHelper) GetCacheService() *cache.GenericCacheService {
	return h.cacheService
}

// GetMemoryCache returns the isolated memory cache instance
func (h *CacheTestHelper) GetMemoryCache() cache.Cache {
	return h.memCache
}

// GetConfig returns the cache configuration
func (h *CacheTestHelper) GetConfig() *cache.CacheConfig {
	return h.cacheConfig
}

// Cleanup properly cleans up the cache instances
func (h *CacheTestHelper) Cleanup() {
	if h.memCache != nil {
		h.memCache.Close()
	}
}

// generateTestID creates a unique identifier for test isolation
func generateTestID() string {
	return time.Now().Format("20060102150405.000000000")
}

// RunCacheTest runs a cache test with proper isolation
func RunCacheTest(t *testing.T, testName string, testFunc func(*CacheTestHelper)) {
	t.Run(testName, func(t *testing.T) {
		helper := NewCacheTestHelper(t)
		defer helper.Cleanup()
		
		testFunc(helper)
	})
}
