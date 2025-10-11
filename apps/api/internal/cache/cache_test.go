package cache_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/qolzam/telar/apps/api/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheFactory(t *testing.T) {
	factory := cache.NewCacheFactory()

	t.Run("CreateMemoryCache", func(t *testing.T) {
		config := cache.DefaultCacheConfig()
		config.Backend = cache.CacheTypeMemory

		c, err := factory.CreateCache(config)
		require.NoError(t, err)
		assert.NotNil(t, c)

		// Test basic operations
		ctx := context.Background()
		err = c.Set(ctx, "test", []byte("value"), time.Minute)
		assert.NoError(t, err)

		value, err := c.Get(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, []byte("value"), value)

		err = c.Delete(ctx, "test")
		assert.NoError(t, err)

		_, err = c.Get(ctx, "test")
		assert.Equal(t, cache.ErrKeyNotFound, err)

		err = c.Close()
		assert.NoError(t, err)
	})

	t.Run("InvalidCacheType", func(t *testing.T) {
		config := cache.DefaultCacheConfig()
		config.Backend = cache.CacheType("invalid")

		_, err := factory.CreateCache(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cache type")
	})
}

func TestCacheBuilder(t *testing.T) {
	t.Run("MemoryCacheBuilder", func(t *testing.T) {
		c := cache.NewCacheBuilder().
			WithMemoryBackend().
			WithMaxSize(1024 * 1024). // 1MB
			WithCleanupInterval(60).   // 60 seconds
			MustBuild()

		assert.NotNil(t, c)

		// Test basic operations
		ctx := context.Background()
		err := c.Set(ctx, "test", []byte("value"), time.Minute)
		assert.NoError(t, err)

		value, err := c.Get(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, []byte("value"), value)

		err = c.Close()
		assert.NoError(t, err)
	})

	t.Run("GetConfig", func(t *testing.T) {
		builder := cache.NewCacheBuilder().
			WithMemoryBackend().
			WithMaxSize(2048)

		config := builder.GetConfig()
		assert.Equal(t, cache.CacheTypeMemory, config.Backend)
		assert.Equal(t, int64(2048), config.MaxMemory)
	})
}

func TestGenericCacheService(t *testing.T) {
	// Create memory cache for testing
	memCache := cache.NewMemoryCache(cache.DefaultCacheConfig())
	defer memCache.Close()

	config := cache.DefaultCacheConfig()
	config.Prefix = "test_service"
	service := cache.NewGenericCacheService(memCache, config)

	t.Run("SetAndGet", func(t *testing.T) {
		ctx := context.Background()
		
		// Test with struct
		data := struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{
			ID:   1,
			Name: "test",
		}

		err := service.CacheData(ctx, "user:1", data, time.Minute)
		assert.NoError(t, err)

		var result struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		err = service.GetCached(ctx, "user:1", &result)
		assert.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		ctx := context.Background()
		
		var result string
		err := service.GetCached(ctx, "user:999", &result)
		assert.Equal(t, cache.ErrKeyNotFound, err)
	})

	t.Run("Delete", func(t *testing.T) {
		ctx := context.Background()
		
		// Set a value
		err := service.CacheData(ctx, "temp:key", "value", time.Minute)
		assert.NoError(t, err)

		// Verify it exists
		var result string
		err = service.GetCached(ctx, "temp:key", &result)
		assert.NoError(t, err)
		assert.Equal(t, "value", result)

		// Delete it
		err = service.InvalidateKey(ctx, "temp:key")
		assert.NoError(t, err)

		// Verify it's gone
		err = service.GetCached(ctx, "temp:key", &result)
		assert.Equal(t, cache.ErrKeyNotFound, err)
	})

	t.Run("InvalidatePattern", func(t *testing.T) {
		ctx := context.Background()
		
		// Set multiple values with same pattern
		err := service.CacheData(ctx, "pattern_test:key1", "value1", time.Minute)
		assert.NoError(t, err)
		err = service.CacheData(ctx, "pattern_test:key2", "value2", time.Minute)
		assert.NoError(t, err)
		err = service.CacheData(ctx, "other:key3", "value3", time.Minute)
		assert.NoError(t, err)

		// Invalidate pattern with wildcard
		err = service.InvalidatePattern(ctx, "pattern_test:*")
		assert.NoError(t, err)

		// Verify pattern keys are gone
		var result string
		err = service.GetCached(ctx, "pattern_test:key1", &result)
		assert.Equal(t, cache.ErrKeyNotFound, err)
		err = service.GetCached(ctx, "pattern_test:key2", &result)
		assert.Equal(t, cache.ErrKeyNotFound, err)

		// Verify other key still exists
		err = service.GetCached(ctx, "other:key3", &result)
		assert.NoError(t, err)
		assert.Equal(t, "value3", result)
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := service.GetStats()
		assert.True(t, stats.Hits >= 0)
		assert.True(t, stats.Misses >= 0)
	})
}

func TestMemoryCache(t *testing.T) {
	config := cache.DefaultCacheConfig()
	config.MaxMemory = 1024 * 1024 // 1MB limit
	config.CleanupInterval = time.Millisecond * 100 // Fast cleanup for testing

	memCache := cache.NewMemoryCache(config)
	defer memCache.Close()

	t.Run("BasicOperations", func(t *testing.T) {
		ctx := context.Background()

		// Test Set and Get
		err := memCache.Set(ctx, "key1", []byte("value1"), time.Minute)
		assert.NoError(t, err)

		value, err := memCache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.Equal(t, []byte("value1"), value)

		// Test Exists
		exists, err := memCache.Exists(ctx, "key1")
		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = memCache.Exists(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Test Delete
		err = memCache.Delete(ctx, "key1")
		assert.NoError(t, err)

		_, err = memCache.Get(ctx, "key1")
		assert.Equal(t, cache.ErrKeyNotFound, err)
	})

	t.Run("TTLExpiration", func(t *testing.T) {
		ctx := context.Background()

		// Set with short TTL
		err := memCache.Set(ctx, "ttl_key", []byte("ttl_value"), time.Millisecond*50)
		assert.NoError(t, err)

		// Should exist immediately
		value, err := memCache.Get(ctx, "ttl_key")
		assert.NoError(t, err)
		assert.Equal(t, []byte("ttl_value"), value)

		// Wait for expiration + cleanup
		time.Sleep(time.Millisecond * 200)

		// Should be expired now
		_, err = memCache.Get(ctx, "ttl_key")
		assert.Equal(t, cache.ErrKeyNotFound, err)
	})

	t.Run("Increment", func(t *testing.T) {
		ctx := context.Background()

		// Test increment non-existent key
		value, err := memCache.Increment(ctx, "counter", 5)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), value)

		// Test increment existing key
		value, err = memCache.Increment(ctx, "counter", 3)
		assert.NoError(t, err)
		assert.Equal(t, int64(8), value)

		// Test negative increment (decrement)
		value, err = memCache.Increment(ctx, "counter", -2)
		assert.NoError(t, err)
		assert.Equal(t, int64(6), value)
	})

	t.Run("DeletePattern", func(t *testing.T) {
		ctx := context.Background()

		// Set multiple keys with pattern
		err := memCache.Set(ctx, "pattern:key1", []byte("value1"), time.Minute)
		assert.NoError(t, err)
		err = memCache.Set(ctx, "pattern:key2", []byte("value2"), time.Minute)
		assert.NoError(t, err)
		err = memCache.Set(ctx, "other:key3", []byte("value3"), time.Minute)
		assert.NoError(t, err)

		// Delete pattern
		err = memCache.DeletePattern(ctx, "pattern:*")
		assert.NoError(t, err)

		// Verify pattern keys are deleted
		_, err = memCache.Get(ctx, "pattern:key1")
		assert.Equal(t, cache.ErrKeyNotFound, err)
		_, err = memCache.Get(ctx, "pattern:key2")
		assert.Equal(t, cache.ErrKeyNotFound, err)

		// Verify other key still exists
		value, err := memCache.Get(ctx, "other:key3")
		assert.NoError(t, err)
		assert.Equal(t, []byte("value3"), value)
	})

	t.Run("Stats", func(t *testing.T) {
		ctx := context.Background()

		// Reset cache for clean stats
		freshCache := cache.NewMemoryCache(config)
		defer freshCache.Close()

		// Perform operations to generate stats
		err := freshCache.Set(ctx, "stats_key", []byte("value"), time.Minute)
		assert.NoError(t, err)

		// Generate hits and misses
		_, err = freshCache.Get(ctx, "stats_key") // hit
		assert.NoError(t, err)
		_, err = freshCache.Get(ctx, "nonexistent") // miss
		assert.Equal(t, cache.ErrKeyNotFound, err)

		stats := freshCache.Stats()
		assert.Equal(t, int64(1), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.Equal(t, 0.5, stats.HitRatio)
		assert.True(t, stats.Keys > 0)
		assert.True(t, stats.MemoryUsage > 0)
	})
}

// Benchmark tests
func BenchmarkMemoryCache(b *testing.B) {
	config := cache.DefaultCacheConfig()
	config.MaxMemory = 10 * 1024 * 1024 // 10MB
	memCache := cache.NewMemoryCache(config)
	defer memCache.Close()

	ctx := context.Background()

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "benchmark_key_" + string(rune(i))
			err := memCache.Set(ctx, key, []byte("benchmark_value"), time.Minute)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Get", func(b *testing.B) {
		// Setup
		for i := 0; i < 1000; i++ {
			key := "get_benchmark_key_" + string(rune(i))
			err := memCache.Set(ctx, key, []byte("benchmark_value"), time.Minute)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := "get_benchmark_key_" + string(rune(i%1000))
			_, err := memCache.Get(ctx, key)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkGenericCacheService(b *testing.B) {
	memCache := cache.NewMemoryCache(cache.DefaultCacheConfig())
	defer memCache.Close()
	
	config := cache.DefaultCacheConfig()
	config.Prefix = "benchmark"
	service := cache.NewGenericCacheService(memCache, config)
	ctx := context.Background()

	b.Run("SetStruct", func(b *testing.B) {
		data := struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{ID: 1, Name: "benchmark"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("user:%d", i)
			err := service.CacheData(ctx, key, data, time.Minute)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetStruct", func(b *testing.B) {
		// Setup
		data := struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{ID: 1, Name: "benchmark"}

		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("user:%d", i)
			err := service.CacheData(ctx, key, data, time.Minute)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}
			key := fmt.Sprintf("user:%d", i%1000)
			err := service.GetCached(ctx, key, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
