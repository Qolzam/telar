package cache

import (
	"context"
	"errors"
	"time"
)

// Cache defines the generic cache interface for all cache implementations
type Cache interface {
	// Get retrieves a value from cache by key
	Get(ctx context.Context, key string) ([]byte, error)
	
	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	
	// Delete removes a value from cache by key
	Delete(ctx context.Context, key string) error
	
	// DeletePattern removes all keys matching the given pattern
	DeletePattern(ctx context.Context, pattern string) error
	
	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// Increment atomically increments a numeric value
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	
	// Close closes the cache connection
	Close() error
	
	// Stats returns cache statistics
	Stats() CacheStats
}

// CacheConfig holds configuration for cache instances
type CacheConfig struct {
	// Enabled indicates if caching is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// TTL is the default time-to-live for cache entries
	TTL time.Duration `json:"ttl" yaml:"ttl"`
	
	// Prefix is added to all cache keys
	Prefix string `json:"prefix" yaml:"prefix"`
	
	// Backend specifies the cache backend (memory, redis)
	Backend CacheType `json:"backend" yaml:"backend"`
	
	// MaxMemory is the maximum memory usage for memory cache (in bytes)
	MaxMemory int64 `json:"max_memory" yaml:"max_memory"`
	
	// CleanupInterval for expired item cleanup
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`
	
	// MaxRetries for backward compatibility
	MaxRetries int `json:"maxRetries" yaml:"max_retries"`
	
	// Redis configuration
	Redis RedisConfig `json:"redis" yaml:"redis"`
}

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	// Address is the Redis server address
	Address string `json:"address" yaml:"address"`
	
	// Password for Redis authentication
	Password string `json:"password" yaml:"password"`
	
	// Database number
	Database int `json:"database" yaml:"database"`
	
	// PoolSize is the maximum number of connections
	PoolSize int `json:"pool_size" yaml:"pool_size"`
	
	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns"`
	
	// MaxConnAge is the maximum connection age
	MaxConnAge time.Duration `json:"max_conn_age" yaml:"max_conn_age"`
	
	// Cluster settings
	Cluster ClusterConfig `json:"cluster" yaml:"cluster"`
}

// ClusterConfig holds Redis cluster configuration
type ClusterConfig struct {
	// Enabled indicates if cluster mode is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// Addresses is the list of cluster node addresses
	Addresses []string `json:"addresses" yaml:"addresses"`
}

// CacheStats provides cache performance statistics
type CacheStats struct {
	// Hits is the number of cache hits
	Hits int64 `json:"hits"`
	
	// Misses is the number of cache misses
	Misses int64 `json:"misses"`
	
	// HitRatio is the cache hit ratio (hits / (hits + misses))
	HitRatio float64 `json:"hit_ratio"`
	
	// Keys is the current number of keys in cache
	Keys int64 `json:"keys"`
	
	// MemoryUsage is the current memory usage in bytes
	MemoryUsage int64 `json:"memory_usage"`
	
	// Evictions is the number of evicted items
	Evictions int64 `json:"evictions"`
}

// Common cache errors
var (
	// ErrKeyNotFound is returned when a key is not found in cache
	ErrKeyNotFound = errors.New("key not found")
	
	// ErrCacheUnavailable is returned when cache backend is unavailable
	ErrCacheUnavailable = errors.New("cache unavailable")
	
	// ErrInvalidTTL is returned when TTL is invalid
	ErrInvalidTTL = errors.New("invalid TTL")
	
	// ErrInvalidCacheType is returned when cache type is invalid
	ErrInvalidCacheType = errors.New("invalid cache type")
	
	// ErrCacheDisabled is returned when cache is disabled
	ErrCacheDisabled = errors.New("cache disabled")
	
	// ErrSerializationFailed is returned when data serialization fails
	ErrSerializationFailed = errors.New("serialization failed")
	
	// ErrDeserializationFailed is returned when data deserialization fails
	ErrDeserializationFailed = errors.New("deserialization failed")
	
	// ErrInvalidKey is returned when a cache key is invalid
	ErrInvalidKey = errors.New("invalid cache key")
)

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enabled:         true,
		TTL:             15 * time.Minute,
		Prefix:          "telar:",
		Backend:         "memory",
		MaxMemory:       100 * 1024 * 1024, // 100MB
		CleanupInterval: 5 * time.Minute,
		MaxRetries:      3, // For backward compatibility
		Redis: RedisConfig{
			Address:      "localhost:6379",
			Password:     "",
			Database:     0,
			PoolSize:     10,
			MinIdleConns: 5,
			MaxConnAge:   30 * time.Minute,
			Cluster: ClusterConfig{
				Enabled:   false,
				Addresses: []string{},
			},
		},
	}
}

// CacheType represents different cache backend types
type CacheType string

const (
	// CacheTypeMemory represents in-memory cache
	CacheTypeMemory CacheType = "memory"
	
	// CacheTypeRedis represents Redis cache
	CacheTypeRedis CacheType = "redis"
)

// IsValid checks if the cache type is valid
func (ct CacheType) IsValid() bool {
	switch ct {
	case CacheTypeMemory, CacheTypeRedis:
		return true
	default:
		return false
	}
}
