package cache

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// CacheMonitor provides comprehensive cache monitoring and statistics
type CacheMonitor struct {
	cache      Cache
	stats      *DetailedCacheStats
	mu         sync.RWMutex
	collectors []MetricCollector
	startTime  time.Time
}

// DetailedCacheStats provides comprehensive cache statistics
type DetailedCacheStats struct {
	// Basic counters
	Hits              int64 `json:"hits"`
	Misses            int64 `json:"misses"`
	Sets              int64 `json:"sets"`
	Deletes           int64 `json:"deletes"`
	Errors            int64 `json:"errors"`
	
	// Performance metrics
	AvgHitTime        time.Duration `json:"avgHitTime"`
	AvgMissTime       time.Duration `json:"avgMissTime"`
	AvgSetTime        time.Duration `json:"avgSetTime"`
	
	// Memory metrics
	MemoryUsage       int64 `json:"memoryUsage"`
	KeyCount          int64 `json:"keyCount"`
	EvictionCount     int64 `json:"evictionCount"`
	
	// Time-based metrics
	HitRate           float64   `json:"hitRate"`
	LastResetTime     time.Time `json:"lastResetTime"`
	UptimeSeconds     int64     `json:"uptimeSeconds"`
	
	// Pattern statistics
	PatternStats      map[string]*PatternMetrics `json:"patternStats"`
}

// PatternMetrics tracks metrics for specific cache key patterns
type PatternMetrics struct {
	Pattern      string        `json:"pattern"`
	Hits         int64         `json:"hits"`
	Misses       int64         `json:"misses"`
	Sets         int64         `json:"sets"`
	HitRate      float64       `json:"hitRate"`
	AvgTTL       time.Duration `json:"avgTTL"`
	LastAccess   time.Time     `json:"lastAccess"`
}

// MetricCollector interface for custom metric collection
type MetricCollector interface {
	Collect(stats *DetailedCacheStats) error
	Name() string
}

// NewCacheMonitor creates a new cache monitor
func NewCacheMonitor(cache Cache) *CacheMonitor {
	return &CacheMonitor{
		cache: cache,
		stats: &DetailedCacheStats{
			PatternStats:  make(map[string]*PatternMetrics),
			LastResetTime: time.Now(),
		},
		startTime: time.Now(),
	}
}

// AddCollector adds a metric collector
func (cm *CacheMonitor) AddCollector(collector MetricCollector) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.collectors = append(cm.collectors, collector)
}

// RecordHit records a cache hit with timing
func (cm *CacheMonitor) RecordHit(pattern string, duration time.Duration) {
	atomic.AddInt64(&cm.stats.Hits, 1)
	cm.updatePatternStats(pattern, "hit", duration)
	cm.updateAverageTime(&cm.stats.AvgHitTime, duration, cm.stats.Hits)
}

// RecordMiss records a cache miss with timing
func (cm *CacheMonitor) RecordMiss(pattern string, duration time.Duration) {
	atomic.AddInt64(&cm.stats.Misses, 1)
	cm.updatePatternStats(pattern, "miss", duration)
	cm.updateAverageTime(&cm.stats.AvgMissTime, duration, cm.stats.Misses)
}

// RecordSet records a cache set operation with timing
func (cm *CacheMonitor) RecordSet(pattern string, duration time.Duration, ttl time.Duration) {
	atomic.AddInt64(&cm.stats.Sets, 1)
	cm.updatePatternStats(pattern, "set", duration)
	cm.updatePatternTTL(pattern, ttl)
	cm.updateAverageTime(&cm.stats.AvgSetTime, duration, cm.stats.Sets)
}

// RecordDelete records a cache delete operation
func (cm *CacheMonitor) RecordDelete(pattern string) {
	atomic.AddInt64(&cm.stats.Deletes, 1)
	cm.updatePatternStats(pattern, "delete", 0)
}

// RecordError records a cache error
func (cm *CacheMonitor) RecordError(pattern string) {
	atomic.AddInt64(&cm.stats.Errors, 1)
}

// RecordEviction records a cache eviction
func (cm *CacheMonitor) RecordEviction() {
	atomic.AddInt64(&cm.stats.EvictionCount, 1)
}

// GetStats returns current cache statistics
func (cm *CacheMonitor) GetStats() *DetailedCacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Calculate derived metrics
	totalOps := cm.stats.Hits + cm.stats.Misses
	if totalOps > 0 {
		cm.stats.HitRate = float64(cm.stats.Hits) / float64(totalOps)
	}
	
	cm.stats.UptimeSeconds = int64(time.Since(cm.startTime).Seconds())
	
	// Update pattern hit rates
	for _, pattern := range cm.stats.PatternStats {
		total := pattern.Hits + pattern.Misses
		if total > 0 {
			pattern.HitRate = float64(pattern.Hits) / float64(total)
		}
	}
	
	return cm.stats
}

// ResetStats resets all statistics
func (cm *CacheMonitor) ResetStats() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.stats = &DetailedCacheStats{
		PatternStats:  make(map[string]*PatternMetrics),
		LastResetTime: time.Now(),
	}
}

// StartCollecting starts metric collection with specified interval
func (cm *CacheMonitor) StartCollecting(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.collectMetrics()
		}
	}
}

// updatePatternStats updates statistics for a specific pattern
func (cm *CacheMonitor) updatePatternStats(pattern, operation string, duration time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cm.stats.PatternStats[pattern] == nil {
		cm.stats.PatternStats[pattern] = &PatternMetrics{
			Pattern: pattern,
		}
	}
	
	patternStats := cm.stats.PatternStats[pattern]
	patternStats.LastAccess = time.Now()
	
	switch operation {
	case "hit":
		patternStats.Hits++
	case "miss":
		patternStats.Misses++
	case "set":
		patternStats.Sets++
	case "delete":
		// No specific counter for deletes in pattern stats
	}
}

// updatePatternTTL updates average TTL for a pattern
func (cm *CacheMonitor) updatePatternTTL(pattern string, ttl time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cm.stats.PatternStats[pattern] == nil {
		cm.stats.PatternStats[pattern] = &PatternMetrics{
			Pattern: pattern,
		}
	}
	
	patternStats := cm.stats.PatternStats[pattern]
	
	// Simple moving average for TTL
	if patternStats.Sets > 0 {
		patternStats.AvgTTL = time.Duration(
			(int64(patternStats.AvgTTL) * (patternStats.Sets - 1) + int64(ttl)) / patternStats.Sets,
		)
	} else {
		patternStats.AvgTTL = ttl
	}
}

// updateAverageTime updates running average for timing metrics
func (cm *CacheMonitor) updateAverageTime(avgTime *time.Duration, newTime time.Duration, count int64) {
	if count > 0 {
		*avgTime = time.Duration(
			(int64(*avgTime) * (count - 1) + int64(newTime)) / count,
		)
	}
}

// collectMetrics triggers all registered metric collectors
func (cm *CacheMonitor) collectMetrics() {
	stats := cm.GetStats()
	
	cm.mu.RLock()
	collectors := make([]MetricCollector, len(cm.collectors))
	copy(collectors, cm.collectors)
	cm.mu.RUnlock()
	
	for _, collector := range collectors {
		if err := collector.Collect(stats); err != nil {
			// Log error but continue with other collectors
		}
	}
}

// ExportStatsJSON exports statistics as JSON
func (cm *CacheMonitor) ExportStatsJSON() ([]byte, error) {
	stats := cm.GetStats()
	return json.MarshalIndent(stats, "", "  ")
}

// PrometheusCollector implements MetricCollector for Prometheus metrics
type PrometheusCollector struct {
	// This would integrate with Prometheus client library
	// For now, it's a placeholder
}

// Collect implements MetricCollector interface
func (pc *PrometheusCollector) Collect(stats *DetailedCacheStats) error {
	// Here you would push metrics to Prometheus
	// Example metrics:
	// - cache_hits_total
	// - cache_misses_total
	// - cache_hit_ratio
	// - cache_operation_duration_seconds
	// - cache_memory_usage_bytes
	// - cache_keys_total
	return nil
}

// Name returns the collector name
func (pc *PrometheusCollector) Name() string {
	return "prometheus"
}

// LogCollector implements MetricCollector for structured logging
type LogCollector struct {
	logger Logger // Assume we have a logger interface
}

// Collect implements MetricCollector interface
func (lc *LogCollector) Collect(stats *DetailedCacheStats) error {
	// Log key metrics at specified intervals
	lc.logger.Info("Cache Statistics",
		"hits", stats.Hits,
		"misses", stats.Misses,
		"hit_rate", stats.HitRate,
		"memory_usage", stats.MemoryUsage,
		"key_count", stats.KeyCount,
		"uptime_seconds", stats.UptimeSeconds,
	)
	return nil
}

// Name returns the collector name
func (lc *LogCollector) Name() string {
	return "logger"
}

// Logger interface placeholder
type Logger interface {
	Info(msg string, fields ...interface{})
}
