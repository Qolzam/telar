package cache

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// cacheItem represents an item in the memory cache
type cacheItem struct {
	value      []byte
	expiration time.Time
}

// MemoryCache implements Cache interface using in-memory storage
type MemoryCache struct {
	items           map[string]*cacheItem
	mutex           sync.RWMutex
	maxMemory       int64
	currentMemory   int64
	hits            int64
	misses          int64
	evictions       int64
	cleanupTicker   *time.Ticker
	cleanupDone     chan bool
	config          *CacheConfig
	closed          bool
	closeMutex      sync.Mutex  // Protects cleanup resources during close
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(config *CacheConfig) *MemoryCache {
	if config == nil {
		config = DefaultCacheConfig()
	}
	
	cache := &MemoryCache{
		items:         make(map[string]*cacheItem),
		maxMemory:     config.MaxMemory,
		currentMemory: 0,
		cleanupDone:   make(chan bool),
		config:        config,
		closed:        false,
	}
	
	// Start cleanup goroutine
	go cache.startCleanup()
	
	return cache
}

// Get retrieves a value from cache
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Check if cache is closed
	if c.closed {
		return nil, ErrCacheDisabled
	}

	item, exists := c.items[key]
	if !exists {
		atomic.AddInt64(&c.misses, 1)
		return nil, ErrKeyNotFound
	}

	// Check if item has expired
	if time.Now().After(item.expiration) {
		atomic.AddInt64(&c.misses, 1)
		// Remove expired item (upgrade to write lock)
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.items, key)
		c.updateMemoryUsage(key, nil, item)
		c.mutex.Unlock()
		c.mutex.RLock()
		return nil, ErrKeyNotFound
	}

	atomic.AddInt64(&c.hits, 1)
	// Return a copy of the value
	result := make([]byte, len(item.value))
	copy(result, item.value)
	return result, nil
}

// Set stores a value in cache with expiration
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if cache is closed
	if c.closed {
		return ErrCacheDisabled
	}

	// Make a copy of the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	newItem := &cacheItem{
		value:      valueCopy,
		expiration: time.Now().Add(ttl),
	}

	// Check memory usage and evict if necessary
	oldItem := c.items[key]
	newMemory := c.calculateMemoryUsage(key, newItem)
	
	if oldItem != nil {
		// Replace existing item
		oldMemory := c.calculateMemoryUsage(key, oldItem)
		c.currentMemory = c.currentMemory - oldMemory + newMemory
	} else {
		// New item
		c.currentMemory += newMemory
	}

	// Evict items if memory limit exceeded
	c.evictIfNeeded()

	c.items[key] = newItem
	return nil
}

// Delete removes a value from cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if item, exists := c.items[key]; exists {
		delete(c.items, key)
		c.updateMemoryUsage(key, nil, item)
	}
	return nil
}

// DeletePattern removes all keys matching the given pattern
func (c *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Simple pattern matching (supports * wildcard)
	keysToDelete := make([]string, 0)
	
	for key := range c.items {
		if c.matchPattern(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		if item := c.items[key]; item != nil {
			delete(c.items, key)
			c.updateMemoryUsage(key, nil, item)
		}
	}

	return nil
}

// Exists checks if a key exists in cache
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return false, nil
	}

	// Check if item has expired
	if time.Now().After(item.expiration) {
		return false, nil
	}

	return true, nil
}

// Increment atomically increments a numeric value
func (c *MemoryCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, exists := c.items[key]
	var currentValue int64 = 0

	if exists && !time.Now().After(item.expiration) {
		// Try to parse existing value as integer
		if val, err := strconv.ParseInt(string(item.value), 10, 64); err == nil {
			currentValue = val
		}
	}

	newValue := currentValue + delta
	newValueBytes := []byte(strconv.FormatInt(newValue, 10))

	// Create new item with default TTL
	newItem := &cacheItem{
		value:      newValueBytes,
		expiration: time.Now().Add(c.config.TTL),
	}

	// Update memory usage
	if exists {
		c.updateMemoryUsage(key, newItem, item)
	} else {
		c.currentMemory += c.calculateMemoryUsage(key, newItem)
	}

	c.items[key] = newItem
	return newValue, nil
}

// Close closes the cache connection
func (c *MemoryCache) Close() error {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if already closed
	if c.closed {
		return nil
	}

	// Stop cleanup goroutine
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
		close(c.cleanupDone)
	}

	// Clear all items
	c.items = make(map[string]*cacheItem)
	c.currentMemory = 0
	c.closed = true
	return nil
}

// Stats returns cache statistics
func (c *MemoryCache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	expired := int64(0)
	active := int64(0)
	now := time.Now()
	
	for _, item := range c.items {
		if now.After(item.expiration) {
			expired++
		} else {
			active++
		}
	}

	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)
	total := hits + misses
	hitRatio := 0.0
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}

	return CacheStats{
		Hits:        hits,
		Misses:      misses,
		HitRatio:    hitRatio,
		Keys:        active,
		MemoryUsage: c.currentMemory,
		Evictions:   atomic.LoadInt64(&c.evictions),
	}
}

// startCleanup runs a background goroutine to clean up expired items
func (c *MemoryCache) startCleanup() {
	c.closeMutex.Lock()
	c.cleanupTicker = time.NewTicker(c.config.CleanupInterval)
	ticker := c.cleanupTicker  // Create local copy
	c.closeMutex.Unlock()
	
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.cleanupDone:
			return
		}
	}
}

// cleanupExpired removes expired items from the cache
func (c *MemoryCache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	keysToDelete := make([]string, 0)
	
	for key, item := range c.items {
		if now.After(item.expiration) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		if item := c.items[key]; item != nil {
			delete(c.items, key)
			c.updateMemoryUsage(key, nil, item)
		}
	}
}

// evictIfNeeded removes items if memory limit is exceeded (LRU-like eviction)
func (c *MemoryCache) evictIfNeeded() {
	if c.maxMemory <= 0 || c.currentMemory <= c.maxMemory {
		return
	}

	// Simple eviction: remove expired items first, then oldest items
	now := time.Now()
	keysToEvict := make([]string, 0)
	
	// First pass: collect expired items
	for key, item := range c.items {
		if now.After(item.expiration) {
			keysToEvict = append(keysToEvict, key)
		}
	}

	// Second pass: if still over limit, evict oldest items
	if len(keysToEvict) == 0 && c.currentMemory > c.maxMemory {
		// Simple approach: evict 25% of items (oldest by iteration order)
		count := 0
		targetCount := len(c.items) / 4
		if targetCount == 0 {
			targetCount = 1
		}
		
		for key := range c.items {
			if count >= targetCount {
				break
			}
			keysToEvict = append(keysToEvict, key)
			count++
		}
	}

	// Evict selected items
	for _, key := range keysToEvict {
		if item := c.items[key]; item != nil {
			delete(c.items, key)
			c.updateMemoryUsage(key, nil, item)
			atomic.AddInt64(&c.evictions, 1)
		}
	}
}

// calculateMemoryUsage estimates memory usage for a cache item
func (c *MemoryCache) calculateMemoryUsage(key string, item *cacheItem) int64 {
	if item == nil {
		return 0
	}
	// Rough estimation: key + value + overhead
	return int64(len(key) + len(item.value) + 64) // 64 bytes estimated overhead
}

// updateMemoryUsage updates current memory usage when items change
func (c *MemoryCache) updateMemoryUsage(key string, newItem, oldItem *cacheItem) {
	var newMem, oldMem int64
	
	if newItem != nil {
		newMem = c.calculateMemoryUsage(key, newItem)
	}
	if oldItem != nil {
		oldMem = c.calculateMemoryUsage(key, oldItem)
	}
	
	c.currentMemory = c.currentMemory - oldMem + newMem
}

// matchPattern implements simple pattern matching with * wildcard
func (c *MemoryCache) matchPattern(text, pattern string) bool {
	// Handle simple cases
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return text == pattern
	}

	// Split pattern by * and check each part
	parts := strings.Split(pattern, "*")
	if len(parts) == 0 {
		return true
	}

	// Check if text starts with first part
	if len(parts[0]) > 0 && !strings.HasPrefix(text, parts[0]) {
		return false
	}

	// Check if text ends with last part
	if len(parts[len(parts)-1]) > 0 && !strings.HasSuffix(text, parts[len(parts)-1]) {
		return false
	}

	// For more complex patterns, check middle parts
	currentPos := len(parts[0])
	for i := 1; i < len(parts)-1; i++ {
		part := parts[i]
		if len(part) == 0 {
			continue
		}
		
		pos := strings.Index(text[currentPos:], part)
		if pos == -1 {
			return false
		}
		currentPos += pos + len(part)
	}

	return true
}
