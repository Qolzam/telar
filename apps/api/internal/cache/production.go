package cache

import (
	"context"
	"fmt"
	"time"
)

// ProductionCacheConfig represents production-ready cache configuration
type ProductionCacheConfig struct {
	// Basic configuration
	Enabled     bool          `json:"enabled" yaml:"enabled"`
	Backend     string        `json:"backend" yaml:"backend"` // "memory", "redis", "hybrid"
	Prefix      string        `json:"prefix" yaml:"prefix"`
	DefaultTTL  time.Duration `json:"defaultTTL" yaml:"defaultTTL"`
	
	// Memory cache configuration
	Memory MemoryCacheConfig `json:"memory" yaml:"memory"`
	
	// Redis configuration
	Redis RedisCacheConfig `json:"redis" yaml:"redis"`
	
	// Performance tuning
	Performance PerformanceConfig `json:"performance" yaml:"performance"`
	
	// Monitoring and observability
	Monitoring MonitoringConfig `json:"monitoring" yaml:"monitoring"`
	
	// Security settings
	Security SecurityConfig `json:"security" yaml:"security"`
	
	// High availability settings
	HighAvailability HAConfig `json:"highAvailability" yaml:"highAvailability"`
}

// MemoryCacheConfig configures in-memory caching
type MemoryCacheConfig struct {
	MaxMemory        int64         `json:"maxMemory" yaml:"maxMemory"`
	CleanupInterval  time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"`
	EvictionPolicy   string        `json:"evictionPolicy" yaml:"evictionPolicy"` // "lru", "lfu", "ttl"
	MaxKeys          int64         `json:"maxKeys" yaml:"maxKeys"`
	ConcurrencyLevel int           `json:"concurrencyLevel" yaml:"concurrencyLevel"`
}

// RedisCacheConfig configures Redis caching
type RedisCacheConfig struct {
	// Connection settings
	Addresses        []string      `json:"addresses" yaml:"addresses"`
	Password         string        `json:"password" yaml:"password"`
	Database         int           `json:"database" yaml:"database"`
	
	// Connection pool settings
	PoolSize         int           `json:"poolSize" yaml:"poolSize"`
	MinIdleConns     int           `json:"minIdleConns" yaml:"minIdleConns"`
	MaxConnAge       time.Duration `json:"maxConnAge" yaml:"maxConnAge"`
	PoolTimeout      time.Duration `json:"poolTimeout" yaml:"poolTimeout"`
	IdleTimeout      time.Duration `json:"idleTimeout" yaml:"idleTimeout"`
	IdleCheckFreq    time.Duration `json:"idleCheckFreq" yaml:"idleCheckFreq"`
	
	// Operational settings
	DialTimeout      time.Duration `json:"dialTimeout" yaml:"dialTimeout"`
	ReadTimeout      time.Duration `json:"readTimeout" yaml:"readTimeout"`
	WriteTimeout     time.Duration `json:"writeTimeout" yaml:"writeTimeout"`
	
	// Cluster settings
	ClusterMode      bool          `json:"clusterMode" yaml:"clusterMode"`
	MasterName       string        `json:"masterName" yaml:"masterName"` // For Redis Sentinel
	
	// Advanced settings
	MaxRetries       int           `json:"maxRetries" yaml:"maxRetries"`
	MinRetryBackoff  time.Duration `json:"minRetryBackoff" yaml:"minRetryBackoff"`
	MaxRetryBackoff  time.Duration `json:"maxRetryBackoff" yaml:"maxRetryBackoff"`
}

// PerformanceConfig configures performance-related settings
type PerformanceConfig struct {
	// Cache warming
	WarmingEnabled   bool          `json:"warmingEnabled" yaml:"warmingEnabled"`
	WarmingInterval  time.Duration `json:"warmingInterval" yaml:"warmingInterval"`
	WarmingJobs      []WarmingJobConfig `json:"warmingJobs" yaml:"warmingJobs"`
	
	// Compression
	CompressionEnabled    bool   `json:"compressionEnabled" yaml:"compressionEnabled"`
	CompressionThreshold  int64  `json:"compressionThreshold" yaml:"compressionThreshold"`
	CompressionAlgorithm  string `json:"compressionAlgorithm" yaml:"compressionAlgorithm"` // "gzip", "lz4", "snappy"
	
	// Batching
	BatchingEnabled       bool          `json:"batchingEnabled" yaml:"batchingEnabled"`
	BatchSize            int           `json:"batchSize" yaml:"batchSize"`
	BatchTimeout         time.Duration `json:"batchTimeout" yaml:"batchTimeout"`
	
	// Prefetching
	PrefetchingEnabled   bool    `json:"prefetchingEnabled" yaml:"prefetchingEnabled"`
	PrefetchThreshold    float64 `json:"prefetchThreshold" yaml:"prefetchThreshold"`
}

// WarmingJobConfig configures cache warming jobs
type WarmingJobConfig struct {
	Key        string        `json:"key" yaml:"key"`
	Interval   time.Duration `json:"interval" yaml:"interval"`
	TTL        time.Duration `json:"ttl" yaml:"ttl"`
	Enabled    bool          `json:"enabled" yaml:"enabled"`
	DataSource string        `json:"dataSource" yaml:"dataSource"` // Identifies the data source
}

// MonitoringConfig configures monitoring and observability
type MonitoringConfig struct {
	// Metrics collection
	MetricsEnabled       bool          `json:"metricsEnabled" yaml:"metricsEnabled"`
	MetricsInterval      time.Duration `json:"metricsInterval" yaml:"metricsInterval"`
	MetricsCollectors    []string      `json:"metricsCollectors" yaml:"metricsCollectors"` // "prometheus", "statsd", "custom"
	
	// Health checks
	HealthCheckEnabled   bool          `json:"healthCheckEnabled" yaml:"healthCheckEnabled"`
	HealthCheckInterval  time.Duration `json:"healthCheckInterval" yaml:"healthCheckInterval"`
	HealthCheckTimeout   time.Duration `json:"healthCheckTimeout" yaml:"healthCheckTimeout"`
	
	// Logging
	LogLevel            string `json:"logLevel" yaml:"logLevel"` // "debug", "info", "warn", "error"
	LogSlowOperations   bool   `json:"logSlowOperations" yaml:"logSlowOperations"`
	SlowOperationThreshold time.Duration `json:"slowOperationThreshold" yaml:"slowOperationThreshold"`
	
	// Alerting
	AlertingEnabled     bool    `json:"alertingEnabled" yaml:"alertingEnabled"`
	AlertThresholds     AlertThresholds `json:"alertThresholds" yaml:"alertThresholds"`
}

// AlertThresholds defines thresholds for alerting
type AlertThresholds struct {
	HitRateBelow         float64       `json:"hitRateBelow" yaml:"hitRateBelow"`
	ErrorRateAbove       float64       `json:"errorRateAbove" yaml:"errorRateAbove"`
	LatencyAbove         time.Duration `json:"latencyAbove" yaml:"latencyAbove"`
	MemoryUsageAbove     float64       `json:"memoryUsageAbove" yaml:"memoryUsageAbove"`
	ConnectionsAbove     int           `json:"connectionsAbove" yaml:"connectionsAbove"`
}

// SecurityConfig configures security settings
type SecurityConfig struct {
	// Encryption
	EncryptionEnabled   bool   `json:"encryptionEnabled" yaml:"encryptionEnabled"`
	EncryptionKey       string `json:"encryptionKey" yaml:"encryptionKey"`
	EncryptionAlgorithm string `json:"encryptionAlgorithm" yaml:"encryptionAlgorithm"` // "aes-256-gcm"
	
	// Access control
	AuthEnabled         bool     `json:"authEnabled" yaml:"authEnabled"`
	AllowedOrigins      []string `json:"allowedOrigins" yaml:"allowedOrigins"`
	AllowedMethods      []string `json:"allowedMethods" yaml:"allowedMethods"`
	
	// Key security
	KeyHashingEnabled   bool   `json:"keyHashingEnabled" yaml:"keyHashingEnabled"`
	KeyHashingAlgorithm string `json:"keyHashingAlgorithm" yaml:"keyHashingAlgorithm"` // "sha256"
	
	// Rate limiting
	RateLimitingEnabled bool  `json:"rateLimitingEnabled" yaml:"rateLimitingEnabled"`
	RateLimit          int64 `json:"rateLimit" yaml:"rateLimit"` // requests per second
}

// HAConfig configures high availability settings
type HAConfig struct {
	// Replication
	ReplicationEnabled   bool     `json:"replicationEnabled" yaml:"replicationEnabled"`
	ReplicationFactor    int      `json:"replicationFactor" yaml:"replicationFactor"`
	ReplicationNodes     []string `json:"replicationNodes" yaml:"replicationNodes"`
	
	// Failover
	FailoverEnabled      bool          `json:"failoverEnabled" yaml:"failoverEnabled"`
	FailoverTimeout      time.Duration `json:"failoverTimeout" yaml:"failoverTimeout"`
	MaxFailoverAttempts  int           `json:"maxFailoverAttempts" yaml:"maxFailoverAttempts"`
	
	// Load balancing
	LoadBalancingEnabled bool   `json:"loadBalancingEnabled" yaml:"loadBalancingEnabled"`
	LoadBalancingStrategy string `json:"loadBalancingStrategy" yaml:"loadBalancingStrategy"` // "round-robin", "least-connections", "weighted"
	
	// Health monitoring
	NodeHealthCheckEnabled  bool          `json:"nodeHealthCheckEnabled" yaml:"nodeHealthCheckEnabled"`
	NodeHealthCheckInterval time.Duration `json:"nodeHealthCheckInterval" yaml:"nodeHealthCheckInterval"`
}

// DefaultProductionConfig returns a production-ready default configuration
func DefaultProductionConfig() *ProductionCacheConfig {
	return &ProductionCacheConfig{
		Enabled:    true,
		Backend:    "redis",
		Prefix:     "app",
		DefaultTTL: 15 * time.Minute,
		
		Memory: MemoryCacheConfig{
			MaxMemory:        100 * 1024 * 1024, // 100MB
			CleanupInterval:  5 * time.Minute,
			EvictionPolicy:   "lru",
			MaxKeys:          10000,
			ConcurrencyLevel: 16,
		},
		
		Redis: RedisCacheConfig{
			Addresses:       []string{"localhost:6379"},
			Database:        0,
			PoolSize:        100,
			MinIdleConns:    10,
			MaxConnAge:      30 * time.Minute,
			PoolTimeout:     4 * time.Second,
			IdleTimeout:     5 * time.Minute,
			IdleCheckFreq:   time.Minute,
			DialTimeout:     5 * time.Second,
			ReadTimeout:     3 * time.Second,
			WriteTimeout:    3 * time.Second,
			ClusterMode:     false,
			MaxRetries:      3,
			MinRetryBackoff: 100 * time.Millisecond,
			MaxRetryBackoff: 1 * time.Second,
		},
		
		Performance: PerformanceConfig{
			WarmingEnabled:       false,
			WarmingInterval:      10 * time.Minute,
			CompressionEnabled:   true,
			CompressionThreshold: 1024, // 1KB
			CompressionAlgorithm: "gzip",
			BatchingEnabled:      true,
			BatchSize:           100,
			BatchTimeout:        100 * time.Millisecond,
			PrefetchingEnabled:  false,
			PrefetchThreshold:   0.8,
		},
		
		Monitoring: MonitoringConfig{
			MetricsEnabled:         true,
			MetricsInterval:        30 * time.Second,
			MetricsCollectors:      []string{"prometheus"},
			HealthCheckEnabled:     true,
			HealthCheckInterval:    30 * time.Second,
			HealthCheckTimeout:     5 * time.Second,
			LogLevel:              "info",
			LogSlowOperations:     true,
			SlowOperationThreshold: 100 * time.Millisecond,
			AlertingEnabled:       true,
			AlertThresholds: AlertThresholds{
				HitRateBelow:     0.8,
				ErrorRateAbove:   0.05,
				LatencyAbove:     100 * time.Millisecond,
				MemoryUsageAbove: 0.9,
				ConnectionsAbove: 90,
			},
		},
		
		Security: SecurityConfig{
			EncryptionEnabled:   false,
			EncryptionAlgorithm: "aes-256-gcm",
			AuthEnabled:         false,
			KeyHashingEnabled:   true,
			KeyHashingAlgorithm: "sha256",
			RateLimitingEnabled: false,
			RateLimit:          1000,
		},
		
		HighAvailability: HAConfig{
			ReplicationEnabled:      false,
			ReplicationFactor:       3,
			FailoverEnabled:         true,
			FailoverTimeout:         30 * time.Second,
			MaxFailoverAttempts:     3,
			LoadBalancingEnabled:    false,
			LoadBalancingStrategy:   "round-robin",
			NodeHealthCheckEnabled:  true,
			NodeHealthCheckInterval: 10 * time.Second,
		},
	}
}

// Validate validates the production configuration
func (pcc *ProductionCacheConfig) Validate() error {
	if !pcc.Enabled {
		return nil // No validation needed if cache is disabled
	}
	
	// Validate backend
	switch pcc.Backend {
	case "memory", "redis", "hybrid":
		// Valid backends
	default:
		return fmt.Errorf("invalid cache backend: %s", pcc.Backend)
	}
	
	// Validate TTL
	if pcc.DefaultTTL <= 0 {
		return fmt.Errorf("defaultTTL must be positive")
	}
	
	// Validate memory config
	if pcc.Memory.MaxMemory <= 0 {
		return fmt.Errorf("memory maxMemory must be positive")
	}
	
	// Validate Redis config if using Redis backend
	if pcc.Backend == "redis" || pcc.Backend == "hybrid" {
		if len(pcc.Redis.Addresses) == 0 {
			return fmt.Errorf("redis addresses cannot be empty")
		}
		
		if pcc.Redis.PoolSize <= 0 {
			return fmt.Errorf("redis poolSize must be positive")
		}
	}
	
	// Validate monitoring config
	if pcc.Monitoring.MetricsEnabled && pcc.Monitoring.MetricsInterval <= 0 {
		return fmt.Errorf("metricsInterval must be positive when metrics are enabled")
	}
	
	return nil
}

// ProductionCacheManager manages production cache with advanced features
type ProductionCacheManager struct {
	config     *ProductionCacheConfig
	cache      Cache
	monitor    *CacheMonitor
	warmer     *CacheWarmer
	eviction   *EvictionManager
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewProductionCacheManager creates a new production cache manager
func NewProductionCacheManager(config *ProductionCacheConfig) (*ProductionCacheManager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create cache based on backend configuration
	var cache Cache
	var err error
	
	switch config.Backend {
	case "memory":
		cache = NewMemoryCache(&CacheConfig{
			Enabled: config.Enabled,
			TTL:     config.DefaultTTL,
			Prefix:  config.Prefix,
		})
	case "redis":
		cache, err = NewRedisCache(&CacheConfig{
			Enabled: config.Enabled,
			TTL:     config.DefaultTTL,
			Prefix:  config.Prefix,
		})
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create Redis cache: %w", err)
		}
	case "hybrid":
		// TODO: Implement hybrid cache (L1 memory + L2 Redis)
		cancel()
		return nil, fmt.Errorf("hybrid cache not yet implemented")
	default:
		cancel()
		return nil, fmt.Errorf("unsupported cache backend: %s", config.Backend)
	}
	
	// Create monitor
	monitor := NewCacheMonitor(cache)
	
	// Create cache warmer
	warmer := NewCacheWarmer(cache, &CacheConfig{
		Enabled: config.Enabled,
		TTL:     config.DefaultTTL,
		Prefix:  config.Prefix,
	})
	
	// Create eviction manager
	var evictionPolicy EvictionPolicy
	switch config.Memory.EvictionPolicy {
	case "lru":
		evictionPolicy = NewLRUPolicy(config.Memory.MaxMemory)
	case "lfu":
		evictionPolicy = NewLFUPolicy(config.Memory.MaxMemory)
	case "ttl":
		evictionPolicy = NewTTLPolicy()
	default:
		evictionPolicy = NewLRUPolicy(config.Memory.MaxMemory)
	}
	
	eviction := NewEvictionManager(cache, evictionPolicy, config.Memory.MaxMemory)
	
	pcm := &ProductionCacheManager{
		config:   config,
		cache:    cache,
		monitor:  monitor,
		warmer:   warmer,
		eviction: eviction,
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// Start background services
	if err := pcm.startServices(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start services: %w", err)
	}
	
	return pcm, nil
}

// startServices starts background services
func (pcm *ProductionCacheManager) startServices() error {
	// Start monitoring
	if pcm.config.Monitoring.MetricsEnabled {
		go pcm.monitor.StartCollecting(pcm.ctx, pcm.config.Monitoring.MetricsInterval)
	}
	
	// Start cache warming
	if pcm.config.Performance.WarmingEnabled {
		pcm.warmer.Start(pcm.ctx)
	}
	
	return nil
}

// GetCache returns the underlying cache instance
func (pcm *ProductionCacheManager) GetCache() Cache {
	return pcm.cache
}

// GetMonitor returns the cache monitor
func (pcm *ProductionCacheManager) GetMonitor() *CacheMonitor {
	return pcm.monitor
}

// GetWarmer returns the cache warmer
func (pcm *ProductionCacheManager) GetWarmer() *CacheWarmer {
	return pcm.warmer
}

// HealthCheck performs a comprehensive health check
func (pcm *ProductionCacheManager) HealthCheck(ctx context.Context) error {
	// Basic connectivity test
	testKey := "health_check_" + fmt.Sprintf("%d", time.Now().UnixNano())
	testValue := []byte("health_check_value")
	
	// Test set
	if err := pcm.cache.Set(ctx, testKey, testValue, time.Minute); err != nil {
		return fmt.Errorf("cache set failed: %w", err)
	}
	
	// Test get
	if _, err := pcm.cache.Get(ctx, testKey); err != nil {
		return fmt.Errorf("cache get failed: %w", err)
	}
	
	// Test delete
	if err := pcm.cache.Delete(ctx, testKey); err != nil {
		return fmt.Errorf("cache delete failed: %w", err)
	}
	
	return nil
}

// Shutdown gracefully shuts down the cache manager
func (pcm *ProductionCacheManager) Shutdown() error {
	pcm.cancel()
	
	// Stop cache warmer
	if pcm.warmer != nil {
		pcm.warmer.Stop()
	}
	
	// Close cache if it implements closer
	if closer, ok := pcm.cache.(interface{ Close() error }); ok {
		return closer.Close()
	}
	
	return nil
}
