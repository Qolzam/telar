package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache implements Cache interface using Redis
type RedisCache struct {
	client    redis.UniversalClient
	config    *CacheConfig
	hits      int64
	misses    int64
	evictions int64
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config *CacheConfig) (*RedisCache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	var client redis.UniversalClient

	if config.Redis.Cluster.Enabled && len(config.Redis.Cluster.Addresses) > 0 {
		// Use Redis Cluster
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Redis.Cluster.Addresses,
			Password:     config.Redis.Password,
			PoolSize:     config.Redis.PoolSize,
			MinIdleConns: config.Redis.MinIdleConns,
			MaxConnAge:   config.Redis.MaxConnAge,
		})
	} else {
		// Use single Redis instance
		client = redis.NewClient(&redis.Options{
			Addr:         config.Redis.Address,
			Password:     config.Redis.Password,
			DB:           config.Redis.Database,
			PoolSize:     config.Redis.PoolSize,
			MinIdleConns: config.Redis.MinIdleConns,
			MaxConnAge:   config.Redis.MaxConnAge,
		})
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("%w: %v", ErrCacheUnavailable, err)
	}

	return &RedisCache{
		client: client,
		config: config,
	}, nil
}

// Get retrieves a value from Redis cache
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&r.misses, 1)
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	atomic.AddInt64(&r.hits, 1)
	return result, nil
}

// Set stores a value in Redis cache with TTL
func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}
	return nil
}

// Delete removes a value from Redis cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}
	return nil
}

// DeletePattern removes all keys matching the given pattern using SCAN
func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	// Use SCAN to find matching keys to avoid blocking Redis
	var cursor uint64
	var deletedCount int64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis scan error: %w", err)
		}

		if len(keys) > 0 {
			// Delete keys in batch
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("redis batch delete error: %w", err)
			}
			deletedCount += int64(len(keys))
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Exists checks if a key exists in Redis cache
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists error: %w", err)
	}
	return result > 0, nil
}

// Increment atomically increments a numeric value in Redis
func (r *RedisCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	result, err := r.client.IncrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("redis increment error: %w", err)
	}

	// Set TTL if this is a new key
	ttl, err := r.client.TTL(ctx, key).Result()
	if err == nil && ttl == -1 { // Key exists but has no TTL
		r.client.Expire(ctx, key, r.config.TTL)
	}

	return result, nil
}

// SetAdd adds a member to a Redis set at the given key.
func (r *RedisCache) SetAdd(ctx context.Context, key string, member string) error {
	if err := r.client.SAdd(ctx, key, member).Err(); err != nil {
		return fmt.Errorf("redis sadd error: %w", err)
	}
	return nil
}

// SetIsMember checks if a member exists in a Redis set at the given key.
func (r *RedisCache) SetIsMember(ctx context.Context, key string, member string) (bool, error) {
	isMember, err := r.client.SIsMember(ctx, key, member).Result()
	if err != nil {
		return false, fmt.Errorf("redis sismember error: %w", err)
	}
	return isMember, nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Stats returns cache statistics from Redis
func (r *RedisCache) Stats() CacheStats {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hits := atomic.LoadInt64(&r.hits)
	misses := atomic.LoadInt64(&r.misses)
	total := hits + misses
	hitRatio := 0.0
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}

	stats := CacheStats{
		Hits:      hits,
		Misses:    misses,
		HitRatio:  hitRatio,
		Evictions: atomic.LoadInt64(&r.evictions),
	}

	// Get Redis INFO statistics if available
	if info, err := r.client.Info(ctx, "memory", "keyspace").Result(); err == nil {
		stats.MemoryUsage = r.parseRedisMemoryUsage(info)
		stats.Keys = r.parseRedisKeyCount(info)
	}

	return stats
}

// parseRedisMemoryUsage extracts memory usage from Redis INFO output
func (r *RedisCache) parseRedisMemoryUsage(info string) int64 {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				if memory, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					return memory
				}
			}
		}
	}
	return 0
}

// parseRedisKeyCount extracts key count from Redis INFO output
func (r *RedisCache) parseRedisKeyCount(info string) int64 {
	lines := strings.Split(info, "\r\n")
	var totalKeys int64
	
	for _, line := range lines {
		if strings.HasPrefix(line, "db") && strings.Contains(line, "keys=") {
			// Parse line like "db0:keys=10,expires=0,avg_ttl=0"
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				keyValuePairs := strings.Split(parts[1], ",")
				for _, pair := range keyValuePairs {
					if strings.HasPrefix(pair, "keys=") {
						keysPart := strings.TrimPrefix(pair, "keys=")
						if keys, err := strconv.ParseInt(keysPart, 10, 64); err == nil {
							totalKeys += keys
						}
					}
				}
			}
		}
	}
	
	return totalKeys
}

// Ping tests the Redis connection
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// FlushDB removes all keys from the current Redis database (use with caution)
func (r *RedisCache) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// GetClient returns the underlying Redis client for advanced operations
func (r *RedisCache) GetClient() redis.UniversalClient {
	return r.client
}

// SetWithNX sets a key only if it doesn't exist (Redis SET NX)
func (r *RedisCache) SetWithNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	result, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx error: %w", err)
	}
	return result, nil
}

// GetWithTTL returns both value and remaining TTL
func (r *RedisCache) GetWithTTL(ctx context.Context, key string) ([]byte, time.Duration, error) {
	pipe := r.client.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, 0, fmt.Errorf("redis pipeline error: %w", err)
	}

	value, err := getCmd.Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&r.misses, 1)
			return nil, 0, ErrKeyNotFound
		}
		return nil, 0, fmt.Errorf("redis get error: %w", err)
	}

	ttl, err := ttlCmd.Result()
	if err != nil {
		ttl = -1 // Unknown TTL
	}

	atomic.AddInt64(&r.hits, 1)
	return value, ttl, nil
}

// ExpireKey sets TTL for an existing key
func (r *RedisCache) ExpireKey(ctx context.Context, key string, ttl time.Duration) error {
	if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("redis expire error: %w", err)
	}
	return nil
}
