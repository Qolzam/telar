package moderation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// InMemoryCache acts as a simple in-memory store for the demo.
//
// IMPORTANT PRODUCTION NOTE: This implementation has no TTL or eviction policy.
// Every unique content hash will remain in memory indefinitely, which can lead
// to memory leaks in long-running services. For production deployments, this
// should be replaced with a Redis-backed cache or an LRU cache with size limits.
//
// The ContentModerator interface allows for hot-swapping implementations without
// changing the pipeline logic.
type InMemoryCache struct {
	store map[string]*ModerationResult
	mu    sync.RWMutex
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		store: make(map[string]*ModerationResult),
	}
}

func (c *InMemoryCache) Name() string {
	return "L1-Hash-Cache"
}

func (c *InMemoryCache) Moderate(ctx context.Context, text string) (*ModerationResult, error) {
	start := time.Now()
	hash := generateHash(text)

	c.mu.RLock()
	result, exists := c.store[hash]
	c.mu.RUnlock()

	if exists {
		// Return a copy to avoid mutating cache state
		cachedResult := *result
		cachedResult.AnalysisTimeMs = time.Since(start).Milliseconds() // It's a cache hit speed!
		cachedResult.ModelUsed = c.Name()
		return &cachedResult, nil
	}

	return nil, nil
}

// Set adds a result to the cache. We will call this from the Pipeline.
func (c *InMemoryCache) Set(text string, result *ModerationResult) {
	hash := generateHash(text)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[hash] = result
}

func generateHash(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}
