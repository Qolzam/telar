package cache

import (
	"os"
	"strings"
	"time"
)

// toCacheConfig creates a default cache config with optional prefix
func toCacheConfig(prefix string) *CacheConfig {
	cfg := DefaultCacheConfig()
	
	// Override with service-specific prefix when provided
	if prefix != "" {
		cfg.Prefix = prefix
	}
	
	// Use environment variables for configuration
	// Note: This is a legacy integration function - new code should use platform config
	if redisAddr := os.Getenv("REDIS_ADDRESS"); redisAddr != "" {
		cfg.Redis.Address = redisAddr
	}
	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		cfg.Redis.Password = redisPass
	}

	return cfg
}

// NewGenericCacheServiceFor creates a GenericCacheService using app config, isolated in cache layer
// prefix sets a service-specific key prefix, e.g. "posts"
func NewGenericCacheServiceFor(prefix string) *GenericCacheService {
	cfg := toCacheConfig(prefix)

	// Optional backend switch via env: memory | redis | hybrid
	// Note: This is a legacy integration function - new code should use platform config
	backend := strings.ToLower(os.Getenv("CACHE_BACKEND"))
	if backend == "" {
		backend = "memory"
	}

	var cacheBackend Cache

	switch backend {
	case "disabled":
		cfg.Enabled = false
		cacheBackend = nil

	case "test":
		cfg.Enabled = true
		cfg.Backend = CacheTypeMemory
		cfg.TTL = 60 * time.Second
		if cfg.Prefix == "" {
			cfg.Prefix = prefix
		}
		cfg.Prefix = cfg.Prefix + "_test"
		cacheBackend = NewMemoryCache(cfg)

	case "production":
		pc := DefaultProductionConfig()
		pc.Enabled = cfg.Enabled
		pc.Prefix = cfg.Prefix
		pc.DefaultTTL = cfg.TTL
		if cfg.Backend == CacheTypeRedis {
			pc.Backend = "redis"
			if cfg.Redis.Cluster.Enabled && len(cfg.Redis.Cluster.Addresses) > 0 {
				pc.Redis.Addresses = cfg.Redis.Cluster.Addresses
				pc.Redis.ClusterMode = true
			} else if cfg.Redis.Address != "" {
				pc.Redis.Addresses = []string{cfg.Redis.Address}
			}
			pc.Redis.Password = cfg.Redis.Password
			pc.Redis.Database = cfg.Redis.Database
			pc.Redis.PoolSize = cfg.Redis.PoolSize
			pc.Redis.MinIdleConns = cfg.Redis.MinIdleConns
			pc.Redis.MaxConnAge = cfg.Redis.MaxConnAge
		} else {
			pc.Backend = "memory"
		}
		mgr, err := NewProductionCacheManager(pc)
		if err == nil && mgr != nil {
			cacheBackend = mgr.GetCache()
		}
		if cacheBackend == nil {
			cacheBackend = NewMemoryCache(cfg)
		}

	case "auto":
		fallthrough
	default:
		factory := NewCacheFactory()
		b, err := factory.CreateCache(cfg)
		if err == nil {
			cacheBackend = b
		}
		if cacheBackend == nil {
			cacheBackend = NewMemoryCache(cfg)
		}
	}

	return NewGenericCacheService(cacheBackend, cfg)
}


