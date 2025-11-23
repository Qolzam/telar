package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/qolzam/telar/apps/api/internal/pkg/log"
)

// GenericCacheService provides a generic caching service for all microservices
type GenericCacheService struct {
	cache  Cache
	config *CacheConfig
	stats  *serviceStats
}

// serviceStats tracks cache service statistics with atomic operations for thread safety
type serviceStats struct {
	hits     int64
	misses   int64
	errors   int64
	sets     int64
	deletes  int64
}

// Thread-safe increment methods
func (s *serviceStats) incHits() {
	atomic.AddInt64(&s.hits, 1)
}

func (s *serviceStats) incMisses() {
	atomic.AddInt64(&s.misses, 1)
}

func (s *serviceStats) incErrors() {
	atomic.AddInt64(&s.errors, 1)
}

func (s *serviceStats) incSets() {
	atomic.AddInt64(&s.sets, 1)
}

func (s *serviceStats) incDeletes() {
	atomic.AddInt64(&s.deletes, 1)
}

// Thread-safe getters
func (s *serviceStats) getHits() int64 {
	return atomic.LoadInt64(&s.hits)
}

func (s *serviceStats) getMisses() int64 {
	return atomic.LoadInt64(&s.misses)
}

func (s *serviceStats) getErrors() int64 {
	return atomic.LoadInt64(&s.errors)
}

func (s *serviceStats) getSets() int64 {
	return atomic.LoadInt64(&s.sets)
}

func (s *serviceStats) getDeletes() int64 {
	return atomic.LoadInt64(&s.deletes)
}

// NewGenericCacheService creates a new generic cache service
func NewGenericCacheService(cache Cache, config *CacheConfig) *GenericCacheService {
	if config == nil {
		config = DefaultCacheConfig()
	}
	
	return &GenericCacheService{
		cache:  cache,
		config: config,
		stats:  &serviceStats{},
	}
}

// GetCached retrieves and unmarshals cached data into the target interface
func (gcs *GenericCacheService) GetCached(ctx context.Context, key string, target interface{}) error {
	if !gcs.config.Enabled || gcs.cache == nil {
		gcs.stats.incMisses()
		return ErrCacheDisabled
	}
	
	// Build the full cache key with prefix
	fullKey := gcs.buildKey(key)
	
	// Get data from cache
	data, err := gcs.cache.Get(ctx, fullKey)
	if err != nil {
		if err == ErrKeyNotFound {
			gcs.stats.incMisses()
		} else {
			gcs.stats.incErrors()
			log.Error("Cache get error for key %s: %v", fullKey, err)
		}
		return err
	}
	
	// Unmarshal the data
	if err := json.Unmarshal(data, target); err != nil {
		gcs.stats.incErrors()
		log.Error("Cache data unmarshal error for key %s: %v", fullKey, err)
		return fmt.Errorf("%w: %v", ErrDeserializationFailed, err)
	}
	
	gcs.stats.incHits()
	return nil
}

// CacheData marshals and stores data in cache with TTL
func (gcs *GenericCacheService) CacheData(ctx context.Context, key string, data interface{}, ttl ...time.Duration) error {
	if !gcs.config.Enabled || gcs.cache == nil {
		return ErrCacheDisabled
	}
	
	// Use provided TTL or default
	cacheTTL := gcs.config.TTL
	if len(ttl) > 0 && ttl[0] > 0 {
		cacheTTL = ttl[0]
	}
	
	// Marshal the data
	jsonData, err := json.Marshal(data)
	if err != nil {
		gcs.stats.incErrors()
		log.Error("Cache data marshal error for key %s: %v", key, err)
		return fmt.Errorf("%w: %v", ErrSerializationFailed, err)
	}
	
	// Build the full cache key with prefix
	fullKey := gcs.buildKey(key)
	
	// Store in cache
	if err := gcs.cache.Set(ctx, fullKey, jsonData, cacheTTL); err != nil {
		gcs.stats.incErrors()
		log.Error("Cache set error for key %s: %v", fullKey, err)
		return err
	}
	
	gcs.stats.incSets()
	return nil
}

// InvalidatePattern removes all cache keys matching the given pattern
func (gcs *GenericCacheService) InvalidatePattern(ctx context.Context, pattern string) error {
	if !gcs.config.Enabled || gcs.cache == nil {
		return ErrCacheDisabled
	}
	
	// Build the full pattern with prefix
	fullPattern := gcs.buildKey(pattern)
	
	if err := gcs.cache.DeletePattern(ctx, fullPattern); err != nil {
		gcs.stats.incErrors()
		log.Error("Cache pattern invalidation error for pattern %s: %v", fullPattern, err)
		return err
	}
	
	gcs.stats.incDeletes()
	return nil
}

// InvalidateKey removes a specific key from cache
func (gcs *GenericCacheService) InvalidateKey(ctx context.Context, key string) error {
	if !gcs.config.Enabled || gcs.cache == nil {
		return ErrCacheDisabled
	}
	
	// Build the full cache key with prefix
	fullKey := gcs.buildKey(key)
	
	if err := gcs.cache.Delete(ctx, fullKey); err != nil {
		gcs.stats.incErrors()
		log.Error("Cache key invalidation error for key %s: %v", fullKey, err)
		return err
	}
	
	gcs.stats.incDeletes()
	return nil
}

// setOps defines optional set operations supported by some cache backends (e.g., Redis).
// Implementations that don't support sets can ignore this (GenericCacheService checks at runtime).
type setOps interface {
	SetAdd(ctx context.Context, key string, member string) error
	SetIsMember(ctx context.Context, key string, member string) (bool, error)
}

// SetAdd adds a member to a set stored at key. Best-effort; returns ErrCacheDisabled if unsupported/disabled.
func (gcs *GenericCacheService) SetAdd(ctx context.Context, key string, member string) error {
	if !gcs.IsEnabled() {
		return ErrCacheDisabled
	}
	setCache, ok := gcs.cache.(setOps)
	if !ok {
		return ErrCacheDisabled
	}
	fullKey := gcs.buildKey(key)
	if err := setCache.SetAdd(ctx, fullKey, member); err != nil {
		gcs.stats.incErrors()
		log.Error("Cache set add error for key %s: %v", fullKey, err)
		return err
	}
	gcs.stats.incSets()
	return nil
}

// SetIsMember checks if a member is part of the set at key. Returns false with ErrCacheDisabled if unsupported/disabled.
func (gcs *GenericCacheService) SetIsMember(ctx context.Context, key string, member string) (bool, error) {
	if !gcs.IsEnabled() {
		return false, ErrCacheDisabled
	}
	setCache, ok := gcs.cache.(setOps)
	if !ok {
		return false, ErrCacheDisabled
	}
	fullKey := gcs.buildKey(key)
	isMember, err := setCache.SetIsMember(ctx, fullKey, member)
	if err != nil {
		gcs.stats.incErrors()
		log.Error("Cache set isMember error for key %s: %v", fullKey, err)
		return false, err
	}
	return isMember, nil
}

// GenerateHashKey creates a deterministic hash-based cache key from parameters
func (gcs *GenericCacheService) GenerateHashKey(prefix string, params map[string]interface{}) string {
	// Create a hash from sorted parameters for deterministic keys
	h := sha256.New()
	
	// Add prefix
	h.Write([]byte(prefix + ":"))
	
	// Sort keys for consistent hashing
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// Add sorted parameters to hash
	for _, k := range keys {
		v := params[k]
		var valueStr string
		
		switch val := v.(type) {
		case string:
			valueStr = val
		case nil:
			valueStr = "nil"
		default:
			// Convert to JSON for complex types
			if jsonVal, err := json.Marshal(val); err == nil {
				valueStr = string(jsonVal)
			} else {
				valueStr = fmt.Sprintf("%v", val)
			}
		}
		
		h.Write([]byte(fmt.Sprintf("%s=%s;", k, valueStr)))
	}
	
	// Generate hash
	hash := hex.EncodeToString(h.Sum(nil))[:16] // Use first 16 chars for brevity
	return fmt.Sprintf("%s:%s", prefix, hash)
}

// Exists checks if a key exists in cache
func (gcs *GenericCacheService) Exists(ctx context.Context, key string) (bool, error) {
	if !gcs.config.Enabled || gcs.cache == nil {
		return false, ErrCacheDisabled
	}
	
	fullKey := gcs.buildKey(key)
	return gcs.cache.Exists(ctx, fullKey)
}

// Increment atomically increments a numeric value in cache
func (gcs *GenericCacheService) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	if !gcs.config.Enabled || gcs.cache == nil {
		return 0, ErrCacheDisabled
	}
	
	fullKey := gcs.buildKey(key)
	return gcs.cache.Increment(ctx, fullKey, delta)
}

// GetStats returns cache service statistics
func (gcs *GenericCacheService) GetStats() CacheStats {
	cacheStats := gcs.cache.Stats()
	
	// Calculate hit ratio from service stats
	hits := gcs.stats.getHits()
	misses := gcs.stats.getMisses()
	total := hits + misses
	hitRatio := 0.0
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}
	
	// Merge cache and service stats
	return CacheStats{
		Hits:        hits,
		Misses:      misses,
		HitRatio:    hitRatio,
		Keys:        cacheStats.Keys,
		MemoryUsage: cacheStats.MemoryUsage,
		Evictions:   cacheStats.Evictions,
	}
}

// Close closes the cache service
func (gcs *GenericCacheService) Close() error {
	if gcs.cache != nil {
		return gcs.cache.Close()
	}
	return nil
}

// IsEnabled returns whether caching is enabled
func (gcs *GenericCacheService) IsEnabled() bool {
	return gcs.config.Enabled && gcs.cache != nil
}

// GetConfig returns the cache configuration
func (gcs *GenericCacheService) GetConfig() *CacheConfig {
	return gcs.config
}

// buildKey constructs the full cache key with prefix
func (gcs *GenericCacheService) buildKey(key string) string {
	if gcs.config.Prefix == "" {
		return key
	}
	
	// Ensure prefix ends with a colon
	prefix := gcs.config.Prefix
	if !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	
	return prefix + key
}

// validateKey ensures the cache key is valid
func (gcs *GenericCacheService) validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	
	// Check for invalid characters (spaces, control characters)
	for _, char := range key {
		if char <= 32 || char >= 127 {
			return fmt.Errorf("%w: contains invalid character", ErrInvalidKey)
		}
	}
	
	// Check key length (Redis has a limit of 512MB, but practical limit is much lower)
	if len(key) > 250 {
		return fmt.Errorf("%w: key too long (max 250 characters)", ErrInvalidKey)
	}
	
	return nil
}

// Warm preloads cache with data for performance optimization
func (gcs *GenericCacheService) Warm(ctx context.Context, data map[string]interface{}) error {
	if !gcs.config.Enabled || gcs.cache == nil {
		return ErrCacheDisabled
	}
	
	for key, value := range data {
		if err := gcs.CacheData(ctx, key, value); err != nil {
			log.Error("Cache warming failed for key %s: %v", key, err)
			// Continue with other keys even if one fails
		}
	}
	
	return nil
}

// Flush removes all cache entries (use with caution)
func (gcs *GenericCacheService) Flush(ctx context.Context) error {
	if !gcs.config.Enabled || gcs.cache == nil {
		return ErrCacheDisabled
	}
	
	// Use prefix pattern to remove all keys with our prefix
	pattern := gcs.config.Prefix + "*"
	if gcs.config.Prefix == "" {
		pattern = "*"
	}
	
	return gcs.cache.DeletePattern(ctx, pattern)
}
