package services

import (
	"testing"
	"time"

	"github.com/qolzam/telar/apps/api/internal/cache"
)

// CacheTestHelper provides isolated cache utilities for comment service tests.
type CacheTestHelper struct {
	memCache     cache.Cache
	cacheConfig  *cache.CacheConfig
	cacheService *cache.GenericCacheService
}

// NewCacheTestHelper returns a helper that isolates cache state per test.
func NewCacheTestHelper(t *testing.T) *CacheTestHelper {
	t.Helper()

	cfg := cache.DefaultCacheConfig()
	cfg.Enabled = true
	cfg.Prefix = "comments_test_" + time.Now().Format("20060102150405.000000000")
	cfg.TTL = time.Hour
	cfg.MaxMemory = 5 * 1024 * 1024

	mem := cache.NewMemoryCache(cfg)
	service := cache.NewGenericCacheService(mem, cfg)

	return &CacheTestHelper{
		memCache:     mem,
		cacheConfig:  cfg,
		cacheService: service,
	}
}

// GetCacheService exposes the isolated generic cache service.
func (h *CacheTestHelper) GetCacheService() *cache.GenericCacheService {
	return h.cacheService
}

// GetConfig exposes the cache configuration used for the helper.
func (h *CacheTestHelper) GetConfig() *cache.CacheConfig {
	return h.cacheConfig
}

// Cleanup releases resources associated with the helper.
func (h *CacheTestHelper) Cleanup() {
	if h.memCache != nil {
		h.memCache.Close()
	}
}

// RunCacheTest executes a cache test with automatic cleanup.
func RunCacheTest(t *testing.T, name string, fn func(helper *CacheTestHelper)) {
	t.Run(name, func(t *testing.T) {
		helper := NewCacheTestHelper(t)
		defer helper.Cleanup()
		fn(helper)
	})
}

