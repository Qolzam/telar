package cache

import (
	"context"
	"sync"
	"time"
)

// CacheWarmer handles cache warming strategies for frequently accessed data
type CacheWarmer struct {
	cache         Cache
	config        *CacheConfig
	warmingJobs   map[string]*WarmingJob
	mu            sync.RWMutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// WarmingJob defines a cache warming job
type WarmingJob struct {
	Key        string
	DataFunc   func(ctx context.Context) (interface{}, error)
	Interval   time.Duration
	TTL        time.Duration
	LastUpdate time.Time
	Enabled    bool
}

// NewCacheWarmer creates a new cache warmer instance
func NewCacheWarmer(cache Cache, config *CacheConfig) *CacheWarmer {
	return &CacheWarmer{
		cache:       cache,
		config:      config,
		warmingJobs: make(map[string]*WarmingJob),
		stopChan:    make(chan struct{}),
	}
}

// AddWarmingJob adds a new cache warming job
func (cw *CacheWarmer) AddWarmingJob(key string, dataFunc func(ctx context.Context) (interface{}, error), interval, ttl time.Duration) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	cw.warmingJobs[key] = &WarmingJob{
		Key:      key,
		DataFunc: dataFunc,
		Interval: interval,
		TTL:      ttl,
		Enabled:  true,
	}
}

// RemoveWarmingJob removes a cache warming job
func (cw *CacheWarmer) RemoveWarmingJob(key string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	delete(cw.warmingJobs, key)
}

// Start begins the cache warming process
func (cw *CacheWarmer) Start(ctx context.Context) {
	cw.wg.Add(1)
	go cw.warmingLoop(ctx)
}

// Stop stops the cache warming process
func (cw *CacheWarmer) Stop() {
	close(cw.stopChan)
	cw.wg.Wait()
}

// warmingLoop runs the main cache warming loop
func (cw *CacheWarmer) warmingLoop(ctx context.Context) {
	defer cw.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-cw.stopChan:
			return
		case <-ticker.C:
			cw.processWarmingJobs(ctx)
		}
	}
}

// processWarmingJobs processes all active warming jobs
func (cw *CacheWarmer) processWarmingJobs(ctx context.Context) {
	cw.mu.RLock()
	jobs := make([]*WarmingJob, 0, len(cw.warmingJobs))
	for _, job := range cw.warmingJobs {
		if job.Enabled {
			jobs = append(jobs, job)
		}
	}
	cw.mu.RUnlock()
	
	for _, job := range jobs {
		if time.Since(job.LastUpdate) >= job.Interval {
			cw.wg.Add(1)
			go cw.executeWarmingJob(ctx, job)
		}
	}
}

// executeWarmingJob executes a single warming job
func (cw *CacheWarmer) executeWarmingJob(ctx context.Context, job *WarmingJob) {
	defer cw.wg.Done()
	
	// Execute the data function
	data, err := job.DataFunc(ctx)
	if err != nil {
		// Log error but don't stop warming
		return
	}
	
	// Serialize and cache the data
	serializedData, err := serializeData(data)
	if err != nil {
		return
	}
	
	// Store in cache
	if err := cw.cache.Set(ctx, job.Key, serializedData, job.TTL); err != nil {
		return
	}
	
	// Update last execution time
	cw.mu.Lock()
	if cachedJob, exists := cw.warmingJobs[job.Key]; exists {
		cachedJob.LastUpdate = time.Now()
	}
	cw.mu.Unlock()
}

// GetWarmingJobStatus returns the status of all warming jobs
func (cw *CacheWarmer) GetWarmingJobStatus() map[string]WarmingJobStatus {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	
	status := make(map[string]WarmingJobStatus)
	for key, job := range cw.warmingJobs {
		status[key] = WarmingJobStatus{
			Key:        job.Key,
			Enabled:    job.Enabled,
			Interval:   job.Interval,
			LastUpdate: job.LastUpdate,
			NextUpdate: job.LastUpdate.Add(job.Interval),
		}
	}
	
	return status
}

// WarmingJobStatus represents the status of a warming job
type WarmingJobStatus struct {
	Key        string        `json:"key"`
	Enabled    bool          `json:"enabled"`
	Interval   time.Duration `json:"interval"`
	LastUpdate time.Time     `json:"lastUpdate"`
	NextUpdate time.Time     `json:"nextUpdate"`
}

// serializeData helper function (assumes JSON serialization)
func serializeData(data interface{}) ([]byte, error) {
	// This would use the same serialization logic as GenericCacheService
	// For now, return placeholder
	return []byte{}, nil
}
