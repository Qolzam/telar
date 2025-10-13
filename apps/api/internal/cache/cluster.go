package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClusterConfig configures Redis cluster setup
type RedisClusterConfig struct {
	// Cluster nodes
	Addrs    []string `json:"addrs" yaml:"addrs"`
	Password string   `json:"password" yaml:"password"`
	
	// Connection settings
	MaxRedirects   int           `json:"maxRedirects" yaml:"maxRedirects"`
	ReadOnly       bool          `json:"readOnly" yaml:"readOnly"`
	RouteByLatency bool          `json:"routeByLatency" yaml:"routeByLatency"`
	RouteRandomly  bool          `json:"routeRandomly" yaml:"routeRandomly"`
	
	// Pool settings
	PoolSize        int           `json:"poolSize" yaml:"poolSize"`
	MinIdleConns    int           `json:"minIdleConns" yaml:"minIdleConns"`
	MaxConnAge      time.Duration `json:"maxConnAge" yaml:"maxConnAge"`
	PoolTimeout     time.Duration `json:"poolTimeout" yaml:"poolTimeout"`
	IdleTimeout     time.Duration `json:"idleTimeout" yaml:"idleTimeout"`
	
	// Timeouts
	DialTimeout  time.Duration `json:"dialTimeout" yaml:"dialTimeout"`
	ReadTimeout  time.Duration `json:"readTimeout" yaml:"readTimeout"`
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`
	
	// TLS configuration
	TLSConfig *tls.Config `json:"-" yaml:"-"`
	
	// Monitoring
	OnConnect func(ctx context.Context, cn *redis.Conn) error `json:"-" yaml:"-"`
}

// RedisClusterCache implements Cache interface with Redis cluster support
type RedisClusterCache struct {
	client *redis.ClusterClient
	config *CacheConfig
	stats  *CacheStats
}

// NewRedisClusterCache creates a new Redis cluster cache
func NewRedisClusterCache(config *CacheConfig, clusterConfig *RedisClusterConfig) (*RedisClusterCache, error) {
	if len(clusterConfig.Addrs) == 0 {
		return nil, fmt.Errorf("cluster addresses cannot be empty")
	}
	
	// Create Redis cluster client
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    clusterConfig.Addrs,
		Password: clusterConfig.Password,
		
		MaxRedirects:   clusterConfig.MaxRedirects,
		ReadOnly:       clusterConfig.ReadOnly,
		RouteByLatency: clusterConfig.RouteByLatency,
		RouteRandomly:  clusterConfig.RouteRandomly,
		
		PoolSize:     clusterConfig.PoolSize,
		MinIdleConns: clusterConfig.MinIdleConns,
		MaxConnAge:   clusterConfig.MaxConnAge,
		PoolTimeout:  clusterConfig.PoolTimeout,
		IdleTimeout:  clusterConfig.IdleTimeout,
		
		DialTimeout:  clusterConfig.DialTimeout,
		ReadTimeout:  clusterConfig.ReadTimeout,
		WriteTimeout: clusterConfig.WriteTimeout,
		
		TLSConfig: clusterConfig.TLSConfig,
		OnConnect: clusterConfig.OnConnect,
	})
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis cluster: %w", err)
	}
	
	return &RedisClusterCache{
		client: client,
		config: config,
		stats:  &CacheStats{},
	}, nil
}

// Get retrieves a value from the cache
func (rcc *RedisClusterCache) Get(ctx context.Context, key string) ([]byte, error) {
	if !rcc.config.Enabled {
		return nil, ErrCacheDisabled
	}
	
	fullKey := rcc.buildKey(key)
	
	val, err := rcc.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			rcc.stats.Misses++
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("redis cluster get failed: %w", err)
	}
	
	rcc.stats.Hits++
	return val, nil
}

// Set stores a value in the cache
func (rcc *RedisClusterCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if !rcc.config.Enabled {
		return ErrCacheDisabled
	}
	
	fullKey := rcc.buildKey(key)
	
	if ttl <= 0 {
		ttl = rcc.config.TTL
	}
	
	if err := rcc.client.Set(ctx, fullKey, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis cluster set failed: %w", err)
	}
	
	return nil
}

// Delete removes a key from the cache
func (rcc *RedisClusterCache) Delete(ctx context.Context, key string) error {
	if !rcc.config.Enabled {
		return ErrCacheDisabled
	}
	
	fullKey := rcc.buildKey(key)
	
	if err := rcc.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("redis cluster delete failed: %w", err)
	}
	
	return nil
}

// DeletePattern removes all keys matching the pattern
func (rcc *RedisClusterCache) DeletePattern(ctx context.Context, pattern string) error {
	if !rcc.config.Enabled {
		return ErrCacheDisabled
	}
	
	fullPattern := rcc.buildKey(pattern)
	
	// Use SCAN across all cluster nodes
	var allKeys []string
	err := rcc.client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		iter := client.Scan(ctx, 0, fullPattern, 0).Iterator()
		for iter.Next(ctx) {
			allKeys = append(allKeys, iter.Val())
		}
		return iter.Err()
	})
	
	if err != nil {
		return fmt.Errorf("redis cluster scan failed: %w", err)
	}
	
	// Delete keys in batches
	if len(allKeys) > 0 {
		if err := rcc.client.Del(ctx, allKeys...).Err(); err != nil {
			return fmt.Errorf("redis cluster batch delete failed: %w", err)
		}
	}
	
	return nil
}

// Exists checks if a key exists
func (rcc *RedisClusterCache) Exists(ctx context.Context, key string) (bool, error) {
	if !rcc.config.Enabled {
		return false, ErrCacheDisabled
	}
	
	fullKey := rcc.buildKey(key)
	
	count, err := rcc.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("redis cluster exists failed: %w", err)
	}
	
	return count > 0, nil
}

// Increment increments a numeric value
func (rcc *RedisClusterCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	if !rcc.config.Enabled {
		return 0, ErrCacheDisabled
	}
	
	fullKey := rcc.buildKey(key)
	
	val, err := rcc.client.IncrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("redis cluster increment failed: %w", err)
	}
	
	return val, nil
}

// Stats returns cache statistics
func (rcc *RedisClusterCache) Stats() CacheStats {
	return *rcc.stats
}

// Close closes the Redis cluster connection
func (rcc *RedisClusterCache) Close() error {
	return rcc.client.Close()
}

// buildKey builds the full cache key with prefix
func (rcc *RedisClusterCache) buildKey(key string) string {
	if rcc.config.Prefix == "" {
		return key
	}
	return rcc.config.Prefix + ":" + key
}

// GetClusterInfo returns cluster information
func (rcc *RedisClusterCache) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	var nodes []NodeInfo
	
	// Simplified cluster info - in production use proper cluster node discovery
	clusterSlots, err := rcc.client.ClusterSlots(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster slots: %w", err)
	}
	
	for _, slot := range clusterSlots {
		for _, node := range slot.Nodes {
			nodeInfo := NodeInfo{
				Address:   node.Addr,
				Role:      "master", // Simplified
				Connected: true,
			}
			nodes = append(nodes, nodeInfo)
		}
	}
	
	return &ClusterInfo{
		Nodes:     nodes,
		Timestamp: time.Now(),
	}, nil
}

// ClusterInfo represents Redis cluster information
type ClusterInfo struct {
	Nodes     []NodeInfo `json:"nodes"`
	Timestamp time.Time  `json:"timestamp"`
}

// NodeInfo represents information about a Redis node
type NodeInfo struct {
	Address     string `json:"address"`
	Role        string `json:"role"` // "master" or "slave"
	Connected   bool   `json:"connected"`
	MemoryUsed  int64  `json:"memoryUsed"`
	MemoryTotal int64  `json:"memoryTotal"`
	Version     string `json:"version"`
	Uptime      int64  `json:"uptime"`
}

// parseRedisInfo parses Redis INFO command output
func parseRedisInfo(addr, info string) NodeInfo {
	node := NodeInfo{
		Address:   addr,
		Connected: true,
	}
	
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				key, value := parts[0], parts[1]
				
				switch key {
				case "redis_version":
					node.Version = value
				case "role":
					node.Role = value
				case "used_memory":
					// Parse memory usage (simplified)
					node.MemoryUsed = parseInt64(value)
				case "uptime_in_seconds":
					node.Uptime = parseInt64(value)
				}
			}
		}
	}
	
	return node
}

// parseInt64 safely parses string to int64
func parseInt64(s string) int64 {
	// Simplified implementation - in production use strconv.ParseInt
	var result int64
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int64(r-'0')
		} else {
			break
		}
	}
	return result
}

// RedisFailoverManager manages Redis failover scenarios
type RedisFailoverManager struct {
	primaryCache   Cache
	fallbackCache  Cache
	healthChecker  *HealthChecker
	failoverActive bool
}

// HealthChecker monitors cache health
type HealthChecker struct {
	cache           Cache
	checkInterval   time.Duration
	timeout         time.Duration
	maxFailures     int
	currentFailures int
	lastCheck       time.Time
	isHealthy       bool
}

// NewRedisFailoverManager creates a failover manager
func NewRedisFailoverManager(primary, fallback Cache) *RedisFailoverManager {
	return &RedisFailoverManager{
		primaryCache:  primary,
		fallbackCache: fallback,
		healthChecker: &HealthChecker{
			cache:         primary,
			checkInterval: 30 * time.Second,
			timeout:       5 * time.Second,
			maxFailures:   3,
			isHealthy:     true,
		},
	}
}

// StartHealthChecking starts health monitoring
func (rfm *RedisFailoverManager) StartHealthChecking(ctx context.Context) {
	ticker := time.NewTicker(rfm.healthChecker.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rfm.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck checks cache health and manages failover
func (rfm *RedisFailoverManager) performHealthCheck(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, rfm.healthChecker.timeout)
	defer cancel()
	
	// Simple health check
	testKey := "health_check"
	testValue := []byte("ping")
	
	err := rfm.primaryCache.Set(checkCtx, testKey, testValue, time.Minute)
	if err == nil {
		_, err = rfm.primaryCache.Get(checkCtx, testKey)
	}
	
	if err != nil {
		rfm.healthChecker.currentFailures++
		if rfm.healthChecker.currentFailures >= rfm.healthChecker.maxFailures {
			rfm.activateFailover()
		}
	} else {
		rfm.healthChecker.currentFailures = 0
		if rfm.failoverActive {
			rfm.deactivateFailover()
		}
	}
	
	rfm.healthChecker.lastCheck = time.Now()
	rfm.healthChecker.isHealthy = err == nil
}

// activateFailover switches to fallback cache
func (rfm *RedisFailoverManager) activateFailover() {
	if !rfm.failoverActive {
		rfm.failoverActive = true
		// Log failover activation
	}
}

// deactivateFailover switches back to primary cache
func (rfm *RedisFailoverManager) deactivateFailover() {
	if rfm.failoverActive {
		rfm.failoverActive = false
		// Log failover deactivation
	}
}

// GetActiveCache returns the currently active cache
func (rfm *RedisFailoverManager) GetActiveCache() Cache {
	if rfm.failoverActive {
		return rfm.fallbackCache
	}
	return rfm.primaryCache
}

// IsHealthy returns current health status
func (rfm *RedisFailoverManager) IsHealthy() bool {
	return rfm.healthChecker.isHealthy
}

// GetFailoverStatus returns failover status
func (rfm *RedisFailoverManager) GetFailoverStatus() bool {
	return rfm.failoverActive
}
