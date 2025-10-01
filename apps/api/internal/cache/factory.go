package cache

import (
	"fmt"
	"time"
)

// CacheFactory creates cache instances based on configuration
type CacheFactory struct{}

// NewCacheFactory creates a new cache factory
func NewCacheFactory() *CacheFactory {
	return &CacheFactory{}
}

// CreateCache creates a cache instance based on the provided configuration
func (f *CacheFactory) CreateCache(config *CacheConfig) (Cache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	if !config.Backend.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrInvalidCacheType, config.Backend)
	}

	switch config.Backend {
	case CacheTypeMemory:
		return NewMemoryCache(config), nil
	case CacheTypeRedis:
		return NewRedisCache(config)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidCacheType, config.Backend)
	}
}

// CreateMemoryCache creates a memory cache instance with custom configuration
func (f *CacheFactory) CreateMemoryCache(maxSize int64, cleanupInterval int) Cache {
	config := DefaultCacheConfig()
	config.Backend = CacheTypeMemory
	config.MaxMemory = maxSize
	config.CleanupInterval = time.Duration(cleanupInterval) * time.Second
	
	return NewMemoryCache(config)
}

// CreateRedisCache creates a Redis cache instance with custom configuration
func (f *CacheFactory) CreateRedisCache(address, password string, database int) (Cache, error) {
	config := DefaultCacheConfig()
	config.Backend = CacheTypeRedis
	config.Redis.Address = address
	config.Redis.Password = password
	config.Redis.Database = database
	
	return NewRedisCache(config)
}

// CreateRedisClusterCache creates a Redis cluster cache instance
func (f *CacheFactory) CreateRedisClusterCache(addresses []string, password string) (Cache, error) {
	config := DefaultCacheConfig()
	config.Backend = CacheTypeRedis
	config.Redis.Password = password
	config.Redis.Cluster.Enabled = true
	config.Redis.Cluster.Addresses = addresses
	
	return NewRedisCache(config)
}

// Global factory instance for convenience
var DefaultFactory = NewCacheFactory()

// Convenience functions using the default factory

// NewCache creates a cache instance using the default factory
func NewCache(config *CacheConfig) (Cache, error) {
	return DefaultFactory.CreateCache(config)
}

// NewMemoryCacheWithConfig creates a memory cache with custom configuration
func NewMemoryCacheWithConfig(maxSize int64, cleanupInterval int) Cache {
	return DefaultFactory.CreateMemoryCache(maxSize, cleanupInterval)
}

// NewRedisCacheWithConfig creates a Redis cache with custom configuration
func NewRedisCacheWithConfig(address, password string, database int) (Cache, error) {
	return DefaultFactory.CreateRedisCache(address, password, database)
}

// NewRedisClusterCacheWithConfig creates a Redis cluster cache with custom configuration
func NewRedisClusterCacheWithConfig(addresses []string, password string) (Cache, error) {
	return DefaultFactory.CreateRedisClusterCache(addresses, password)
}

// MustNewCache creates a cache or panics if configuration is invalid
func MustNewCache(config *CacheConfig) Cache {
	cache, err := NewCache(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create cache: %v", err))
	}
	return cache
}

// CacheBuilder provides a fluent interface for building cache configurations
type CacheBuilder struct {
	config *CacheConfig
}

// NewCacheBuilder creates a new cache builder with default configuration
func NewCacheBuilder() *CacheBuilder {
	return &CacheBuilder{
		config: DefaultCacheConfig(),
	}
}

// WithBackend sets the cache backend type
func (b *CacheBuilder) WithBackend(backend CacheType) *CacheBuilder {
	b.config.Backend = backend
	return b
}

// WithMemoryBackend configures for memory cache
func (b *CacheBuilder) WithMemoryBackend() *CacheBuilder {
	b.config.Backend = CacheTypeMemory
	return b
}

// WithRedisBackend configures for Redis cache
func (b *CacheBuilder) WithRedisBackend() *CacheBuilder {
	b.config.Backend = CacheTypeRedis
	return b
}

// WithMaxSize sets the maximum cache size (for memory cache)
func (b *CacheBuilder) WithMaxSize(size int64) *CacheBuilder {
	b.config.MaxMemory = size
	return b
}

// WithCleanupInterval sets the cleanup interval in seconds
func (b *CacheBuilder) WithCleanupInterval(seconds int) *CacheBuilder {
	b.config.CleanupInterval = time.Duration(seconds) * time.Second
	return b
}

// WithRedisAddress sets the Redis server address
func (b *CacheBuilder) WithRedisAddress(address string) *CacheBuilder {
	b.config.Redis.Address = address
	return b
}

// WithRedisPassword sets the Redis password
func (b *CacheBuilder) WithRedisPassword(password string) *CacheBuilder {
	b.config.Redis.Password = password
	return b
}

// WithRedisDatabase sets the Redis database number
func (b *CacheBuilder) WithRedisDatabase(database int) *CacheBuilder {
	b.config.Redis.Database = database
	return b
}

// WithRedisCluster enables Redis cluster mode with addresses
func (b *CacheBuilder) WithRedisCluster(addresses []string) *CacheBuilder {
	b.config.Backend = CacheTypeRedis
	b.config.Redis.Cluster.Enabled = true
	b.config.Redis.Cluster.Addresses = addresses
	return b
}

// WithRedisPoolSize sets the Redis connection pool size
func (b *CacheBuilder) WithRedisPoolSize(size int) *CacheBuilder {
	b.config.Redis.PoolSize = size
	return b
}

// Build creates the cache instance
func (b *CacheBuilder) Build() (Cache, error) {
	return NewCache(b.config)
}

// MustBuild creates the cache instance or panics on error
func (b *CacheBuilder) MustBuild() Cache {
	cache, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build cache: %v", err))
	}
	return cache
}

// GetConfig returns the current configuration
func (b *CacheBuilder) GetConfig() *CacheConfig {
	// Return a copy to prevent external modifications
	configCopy := *b.config
	return &configCopy
}
