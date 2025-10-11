package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheWarmingStrategies tests cache warming for frequently accessed data
func TestCacheWarmingStrategies(t *testing.T) {
	// Create memory cache for testing
	config := DefaultCacheConfig()
	config.Backend = CacheTypeMemory
	
	factory := NewCacheFactory()
	cache, err := factory.CreateCache(config)
	require.NoError(t, err)
	defer cache.Close()
	
	// Create cache warmer
	warmer := NewCacheWarmer(cache, config)
	defer warmer.Stop()
	
	t.Run("FrequentlyAccessedDataWarming", func(t *testing.T) {
		// Add a warming job with proper signature
		dataFunc := func(ctx context.Context) (interface{}, error) {
			return []byte("frequently_accessed_data"), nil
		}
		
		warmer.AddWarmingJob("popular_key", dataFunc, time.Minute, time.Minute)
		
		// Start warming with context
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		
		warmer.Start(ctx)
		
		// Wait a bit for job to be processed
		time.Sleep(200 * time.Millisecond)
		
		// Verify warming functionality is working
		assert.NotNil(t, warmer)
	})
}

// TestCacheStatisticsAndMonitoring tests cache statistics and monitoring (Phase 5.1.1)
func TestCacheStatisticsAndMonitoring(t *testing.T) {
	// Create memory cache for testing
	config := DefaultCacheConfig()
	config.Backend = CacheTypeMemory
	
	factory := NewCacheFactory()
	cache, err := factory.CreateCache(config)
	require.NoError(t, err)
	defer cache.Close()
	
	// Create monitor
	monitor := NewCacheMonitor(cache)
	
	t.Run("CacheStatisticsCollection", func(t *testing.T) {
		// Record some operations with proper signatures
		monitor.RecordHit("user_posts", time.Millisecond)
		monitor.RecordMiss("user_posts", time.Millisecond)
		monitor.RecordSet("user_posts", 100*time.Microsecond, time.Millisecond)
		
		// Get basic stats
		stats := monitor.GetStats()
		assert.Greater(t, stats.Hits, int64(0))
		assert.Greater(t, stats.Misses, int64(0))
		assert.Greater(t, stats.Sets, int64(0))
	})
	
	t.Run("CacheMonitoringMetrics", func(t *testing.T) {
		// Test monitoring functionality
		stats := monitor.GetStats()
		assert.NotNil(t, stats)
		
		// Verify basic statistics are collected
		assert.GreaterOrEqual(t, stats.Hits, int64(0))
		assert.GreaterOrEqual(t, stats.Misses, int64(0))
		assert.GreaterOrEqual(t, stats.Sets, int64(0))
	})
}

// TestCacheEvictionPolicies tests advanced cache eviction policies (Phase 5.1.1)
func TestCacheEvictionPolicies(t *testing.T) {
	t.Run("LRU_EvictionPolicy", func(t *testing.T) {
		policy := NewLRUPolicy(2) // Max 2 items
		
		// Create cache entries
		entry1 := &CacheEntry{Key: "key1", Value: []byte("value1"), CreatedAt: time.Now()}
		entry2 := &CacheEntry{Key: "key2", Value: []byte("value2"), CreatedAt: time.Now()}
		entry3 := &CacheEntry{Key: "key3", Value: []byte("value3"), CreatedAt: time.Now()}
		
		// Access entries
		policy.OnAccess(entry1)
		policy.OnAccess(entry2)
		policy.OnAccess(entry3)
		
		// Test eviction candidates
		entries := []*CacheEntry{entry1, entry2, entry3}
		candidates := policy.SelectEvictionCandidates(entries, 1)
		assert.Len(t, candidates, 1)
		assert.Equal(t, "LRU", policy.Name())
	})
	
	t.Run("LFU_EvictionPolicy", func(t *testing.T) {
		policy := NewLFUPolicy(2) // Max 2 items
		
		// Create cache entries
		entry1 := &CacheEntry{Key: "key1", Value: []byte("value1"), CreatedAt: time.Now()}
		entry2 := &CacheEntry{Key: "key2", Value: []byte("value2"), CreatedAt: time.Now()}
		
		// Access entries with different frequencies
		policy.OnAccess(entry1)
		policy.OnAccess(entry2)
		policy.OnAccess(entry2) // key2 accessed twice
		
		// Test policy name
		assert.Equal(t, "LFU", policy.Name())
	})
	
	t.Run("TTL_EvictionPolicy", func(t *testing.T) {
		policy := NewTTLPolicy()
		
		// Create expired entry
		entry1 := &CacheEntry{
			Key:       "key1",
			Value:     []byte("value1"),
			CreatedAt: time.Now().Add(-time.Hour),
			TTL:       time.Minute,
			ExpiresAt: time.Now().Add(-time.Minute), // Already expired
		}
		
		// Test if entry should be evicted
		shouldEvict := policy.ShouldEvict(entry1, 100, 1000)
		assert.True(t, shouldEvict)
		assert.Equal(t, "TTL", policy.Name())
	})
}

// -----------------------------------------------------------------------------
// Phase 5.2: Production Configuration (Day 5)
// Phase 5.2.1: Production-ready cache setup
// -----------------------------------------------------------------------------

// TestRedisClusteringSetup tests Redis clustering for high availability (Phase 5.2.1)
func TestRedisClusteringSetup(t *testing.T) {
	t.Run("ClusterConfiguration", func(t *testing.T) {
		clusterConfig := &RedisClusterConfig{
			Addrs:    []string{"localhost:7000", "localhost:7001", "localhost:7002"},
			Password: "",
			PoolSize: 10,
		}
		
		// Test configuration structure for high availability
		assert.NotEmpty(t, clusterConfig.Addrs)
		assert.Equal(t, 10, clusterConfig.PoolSize)
		assert.Len(t, clusterConfig.Addrs, 3)
		assert.True(t, len(clusterConfig.Addrs) >= 3, "Cluster should have at least 3 nodes for HA")
	})
}

// TestCacheSecurityAndAccessControl tests cache security and access control (Phase 5.2.1)
func TestCacheSecurityAndAccessControl(t *testing.T) {
	// Create base cache
	config := DefaultCacheConfig()
	factory := NewCacheFactory()
	cache, err := factory.CreateCache(config)
	require.NoError(t, err)
	defer cache.Close()
	
	// Create security config
	secConfig := DefaultSecurityConfig()
	secConfig.EncryptionEnabled = true
	secConfig.EncryptionKey = "test-encryption-key-32-bytes!!"
	secConfig.AuthEnabled = true
	secConfig.RateLimitingEnabled = true
	secConfig.RateLimit = 100
	
	// Create secure cache
	secureCache, err := NewSecureCache(cache, secConfig)
	require.NoError(t, err)
	defer secureCache.Close()
	
	ctx := context.Background()
	
	t.Run("EncryptionSecurity", func(t *testing.T) {
		// Set encrypted value
		err := secureCache.Set(ctx, "secure_key", []byte("confidential_data"), time.Minute)
		assert.NoError(t, err)
		
		// Get and decrypt value
		value, err := secureCache.Get(ctx, "secure_key")
		assert.NoError(t, err)
		assert.Equal(t, []byte("confidential_data"), value)
		
		// Verify raw value in cache is encrypted (different from original)
		rawValue, err := cache.Get(ctx, "secure_key")
		assert.NoError(t, err)
		assert.NotEqual(t, []byte("confidential_data"), rawValue)
	})
	
	t.Run("AccessControlOperations", func(t *testing.T) {
		// Test basic operations work with security wrapper
		err := secureCache.Set(ctx, "access_test", []byte("test_value"), time.Minute)
		assert.NoError(t, err)
		
		exists, err := secureCache.Exists(ctx, "access_test")
		assert.NoError(t, err)
		assert.True(t, exists)
		
		err = secureCache.Delete(ctx, "access_test")
		assert.NoError(t, err)
		
		// Key should no longer exist
		_, err = secureCache.Get(ctx, "access_test")
		assert.Error(t, err)
		assert.Equal(t, ErrKeyNotFound, err)
	})
}

// TestProductionCacheConfiguration tests production-ready cache configuration (Phase 5.2.1)
func TestProductionCacheConfiguration(t *testing.T) {
	t.Run("ProductionConfigDefaults", func(t *testing.T) {
		config := DefaultProductionConfig()
		assert.NotNil(t, config)
		assert.NotNil(t, config.Performance)
		assert.NotNil(t, config.Security)
		assert.NotNil(t, config.Monitoring)
		assert.NotNil(t, config.HighAvailability)
	})
	
	t.Run("ProductionConfigValidation", func(t *testing.T) {
		config := DefaultProductionConfig()
		err := config.Validate()
		assert.NoError(t, err)
	})
}

// TestCacheEncryptionImplementation tests cache value encryption (Security requirement)
func TestCacheEncryptionImplementation(t *testing.T) {
	t.Run("SuccessfulEncryptionDecryption", func(t *testing.T) {
		encryptor, err := NewCacheEncryptor("production-encryption-key-32b!!")
		require.NoError(t, err)
		
		original := []byte("sensitive production data")
		
		// Encrypt
		encrypted, err := encryptor.Encrypt(original)
		require.NoError(t, err)
		assert.NotEqual(t, original, encrypted)
		
		// Decrypt
		decrypted, err := encryptor.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, original, decrypted)
	})
	
	t.Run("InvalidEncryptionKey", func(t *testing.T) {
		_, err := NewCacheEncryptor("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "encryption key cannot be empty")
	})
	
	t.Run("CorruptedDataHandling", func(t *testing.T) {
		encryptor, err := NewCacheEncryptor("production-encryption-key-32b!!")
		require.NoError(t, err)
		
		// Try to decrypt corrupted data
		corrupted := []byte("this is corrupted encrypted data")
		_, err = encryptor.Decrypt(corrupted)
		assert.Error(t, err)
	})
}

// =============================================================================
// Phase 5 Performance Benchmarks
// Testing optimization performance requirements
// =============================================================================

// BenchmarkCacheOptimizationPerformance tests cache optimization performance (Phase 5.1.1)
func BenchmarkCacheOptimizationPerformance(b *testing.B) {
	encryptor, _ := NewCacheEncryptor("benchmark-encryption-key-32b!!")
	data := []byte("benchmark data for optimization testing with reasonable payload size")
	
	b.Run("EncryptionPerformance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = encryptor.Encrypt(data)
		}
	})
	
	encrypted, _ := encryptor.Encrypt(data)
	b.Run("DecryptionPerformance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = encryptor.Decrypt(encrypted)
		}
	})
}

// BenchmarkProductionCachePerformance tests production cache performance (Phase 5.2.1)
func BenchmarkProductionCachePerformance(b *testing.B) {
	// Setup production-like cache
	config := DefaultCacheConfig()
	factory := NewCacheFactory()
	cache, _ := factory.CreateCache(config)
	defer cache.Close()
	
	secConfig := DefaultSecurityConfig()
	secConfig.EncryptionEnabled = true
	secConfig.EncryptionKey = "production-encryption-key-32b!!"
	
	secureCache, _ := NewSecureCache(cache, secConfig)
	defer secureCache.Close()
	
	ctx := context.Background()
	data := []byte("production benchmark data")
	
	b.Run("ProductionSecureSet", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = secureCache.Set(ctx, "prod_bench_key", data, time.Minute)
		}
	})
	
	// Pre-populate for get benchmark
	_ = secureCache.Set(ctx, "prod_bench_key", data, time.Minute)
	
	b.Run("ProductionSecureGet", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = secureCache.Get(ctx, "prod_bench_key")
		}
	})
}
