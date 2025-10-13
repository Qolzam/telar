package cache

import (
	"context"
	"sync"
	"time"
)

// EvictionPolicy defines different cache eviction strategies
type EvictionPolicy interface {
	// ShouldEvict determines if a cache entry should be evicted
	ShouldEvict(entry *CacheEntry, currentMemory, maxMemory int64) bool
	
	// OnAccess is called when a cache entry is accessed
	OnAccess(entry *CacheEntry)
	
	// OnInsert is called when a new entry is inserted
	OnInsert(entry *CacheEntry)
	
	// SelectEvictionCandidates returns entries that should be evicted
	SelectEvictionCandidates(entries []*CacheEntry, targetCount int) []*CacheEntry
	
	// Name returns the policy name
	Name() string
}

// CacheEntry represents a cache entry with metadata for eviction policies
type CacheEntry struct {
	Key          string
	Value        []byte
	Size         int64
	CreatedAt    time.Time
	LastAccessed time.Time
	AccessCount  int64
	TTL          time.Duration
	ExpiresAt    time.Time
	mu           sync.RWMutex
}

// UpdateAccess updates access metadata for the entry
func (ce *CacheEntry) UpdateAccess() {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.LastAccessed = time.Now()
	ce.AccessCount++
}

// IsExpired checks if the entry has expired
func (ce *CacheEntry) IsExpired() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return time.Now().After(ce.ExpiresAt)
}

// Age returns the age of the entry
func (ce *CacheEntry) Age() time.Duration {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return time.Since(ce.CreatedAt)
}

// TimeSinceLastAccess returns time since last access
func (ce *CacheEntry) TimeSinceLastAccess() time.Duration {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return time.Since(ce.LastAccessed)
}

// LRUPolicy implements Least Recently Used eviction policy
type LRUPolicy struct {
	maxMemory int64
	mu        sync.RWMutex
}

// NewLRUPolicy creates a new LRU eviction policy
func NewLRUPolicy(maxMemory int64) *LRUPolicy {
	return &LRUPolicy{
		maxMemory: maxMemory,
	}
}

// ShouldEvict checks if eviction is needed based on memory usage
func (lru *LRUPolicy) ShouldEvict(entry *CacheEntry, currentMemory, maxMemory int64) bool {
	return currentMemory > maxMemory
}

// OnAccess updates access time for LRU tracking
func (lru *LRUPolicy) OnAccess(entry *CacheEntry) {
	entry.UpdateAccess()
}

// OnInsert is called when a new entry is inserted
func (lru *LRUPolicy) OnInsert(entry *CacheEntry) {
	entry.UpdateAccess()
}

// SelectEvictionCandidates selects entries for eviction based on LRU
func (lru *LRUPolicy) SelectEvictionCandidates(entries []*CacheEntry, targetCount int) []*CacheEntry {
	// Sort by last accessed time (oldest first)
	sortedEntries := make([]*CacheEntry, len(entries))
	copy(sortedEntries, entries)
	
	// Simple bubble sort for demonstration (use heap or other efficient sort in production)
	for i := 0; i < len(sortedEntries)-1; i++ {
		for j := 0; j < len(sortedEntries)-i-1; j++ {
			if sortedEntries[j].LastAccessed.After(sortedEntries[j+1].LastAccessed) {
				sortedEntries[j], sortedEntries[j+1] = sortedEntries[j+1], sortedEntries[j]
			}
		}
	}
	
	if targetCount > len(sortedEntries) {
		targetCount = len(sortedEntries)
	}
	
	return sortedEntries[:targetCount]
}

// Name returns the policy name
func (lru *LRUPolicy) Name() string {
	return "LRU"
}

// LFUPolicy implements Least Frequently Used eviction policy
type LFUPolicy struct {
	maxMemory int64
}

// NewLFUPolicy creates a new LFU eviction policy
func NewLFUPolicy(maxMemory int64) *LFUPolicy {
	return &LFUPolicy{
		maxMemory: maxMemory,
	}
}

// ShouldEvict checks if eviction is needed
func (lfu *LFUPolicy) ShouldEvict(entry *CacheEntry, currentMemory, maxMemory int64) bool {
	return currentMemory > maxMemory
}

// OnAccess updates access count for LFU tracking
func (lfu *LFUPolicy) OnAccess(entry *CacheEntry) {
	entry.UpdateAccess()
}

// OnInsert is called when a new entry is inserted
func (lfu *LFUPolicy) OnInsert(entry *CacheEntry) {
	entry.UpdateAccess()
}

// SelectEvictionCandidates selects entries based on access frequency
func (lfu *LFUPolicy) SelectEvictionCandidates(entries []*CacheEntry, targetCount int) []*CacheEntry {
	// Sort by access count (least accessed first)
	sortedEntries := make([]*CacheEntry, len(entries))
	copy(sortedEntries, entries)
	
	// Simple sort by access count
	for i := 0; i < len(sortedEntries)-1; i++ {
		for j := 0; j < len(sortedEntries)-i-1; j++ {
			if sortedEntries[j].AccessCount > sortedEntries[j+1].AccessCount {
				sortedEntries[j], sortedEntries[j+1] = sortedEntries[j+1], sortedEntries[j]
			}
		}
	}
	
	if targetCount > len(sortedEntries) {
		targetCount = len(sortedEntries)
	}
	
	return sortedEntries[:targetCount]
}

// Name returns the policy name
func (lfu *LFUPolicy) Name() string {
	return "LFU"
}

// TTLPolicy implements Time To Live based eviction policy
type TTLPolicy struct{}

// NewTTLPolicy creates a new TTL eviction policy
func NewTTLPolicy() *TTLPolicy {
	return &TTLPolicy{}
}

// ShouldEvict checks if entry has expired
func (ttl *TTLPolicy) ShouldEvict(entry *CacheEntry, currentMemory, maxMemory int64) bool {
	return entry.IsExpired()
}

// OnAccess does nothing for TTL policy
func (ttl *TTLPolicy) OnAccess(entry *CacheEntry) {
	// TTL policy doesn't need to track access
}

// OnInsert does nothing for TTL policy
func (ttl *TTLPolicy) OnInsert(entry *CacheEntry) {
	// TTL policy doesn't need to track insertion
}

// SelectEvictionCandidates selects expired entries
func (ttl *TTLPolicy) SelectEvictionCandidates(entries []*CacheEntry, targetCount int) []*CacheEntry {
	var expired []*CacheEntry
	for _, entry := range entries {
		if entry.IsExpired() {
			expired = append(expired, entry)
		}
	}
	
	if targetCount > len(expired) {
		targetCount = len(expired)
	}
	
	return expired[:targetCount]
}

// Name returns the policy name
func (ttl *TTLPolicy) Name() string {
	return "TTL"
}

// CompositePolicy combines multiple eviction policies
type CompositePolicy struct {
	policies []EvictionPolicy
	strategy CompositeStrategy
}

// CompositeStrategy defines how multiple policies are combined
type CompositeStrategy int

const (
	// StrategyAny - evict if ANY policy says to evict
	StrategyAny CompositeStrategy = iota
	// StrategyAll - evict only if ALL policies say to evict
	StrategyAll
	// StrategyPriority - use policies in priority order
	StrategyPriority
)

// NewCompositePolicy creates a new composite eviction policy
func NewCompositePolicy(strategy CompositeStrategy, policies ...EvictionPolicy) *CompositePolicy {
	return &CompositePolicy{
		policies: policies,
		strategy: strategy,
	}
}

// ShouldEvict combines decisions from multiple policies
func (cp *CompositePolicy) ShouldEvict(entry *CacheEntry, currentMemory, maxMemory int64) bool {
	switch cp.strategy {
	case StrategyAny:
		for _, policy := range cp.policies {
			if policy.ShouldEvict(entry, currentMemory, maxMemory) {
				return true
			}
		}
		return false
	case StrategyAll:
		for _, policy := range cp.policies {
			if !policy.ShouldEvict(entry, currentMemory, maxMemory) {
				return false
			}
		}
		return len(cp.policies) > 0
	case StrategyPriority:
		// Use first policy that has an opinion
		for _, policy := range cp.policies {
			if policy.ShouldEvict(entry, currentMemory, maxMemory) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// OnAccess notifies all policies
func (cp *CompositePolicy) OnAccess(entry *CacheEntry) {
	for _, policy := range cp.policies {
		policy.OnAccess(entry)
	}
}

// OnInsert notifies all policies
func (cp *CompositePolicy) OnInsert(entry *CacheEntry) {
	for _, policy := range cp.policies {
		policy.OnInsert(entry)
	}
}

// SelectEvictionCandidates combines candidate selection from all policies
func (cp *CompositePolicy) SelectEvictionCandidates(entries []*CacheEntry, targetCount int) []*CacheEntry {
	candidateMap := make(map[string]*CacheEntry)
	
	// Collect candidates from all policies
	for _, policy := range cp.policies {
		candidates := policy.SelectEvictionCandidates(entries, targetCount)
		for _, candidate := range candidates {
			candidateMap[candidate.Key] = candidate
		}
	}
	
	// Convert map to slice
	var allCandidates []*CacheEntry
	for _, candidate := range candidateMap {
		allCandidates = append(allCandidates, candidate)
	}
	
	if targetCount > len(allCandidates) {
		targetCount = len(allCandidates)
	}
	
	return allCandidates[:targetCount]
}

// Name returns the policy name
func (cp *CompositePolicy) Name() string {
	return "Composite"
}

// EvictionManager manages cache eviction with configurable policies
type EvictionManager struct {
	policy       EvictionPolicy
	cache        Cache
	maxMemory    int64
	evictionRate float64 // Percentage of entries to evict when threshold is hit
	mu           sync.RWMutex
}

// NewEvictionManager creates a new eviction manager
func NewEvictionManager(cache Cache, policy EvictionPolicy, maxMemory int64) *EvictionManager {
	return &EvictionManager{
		policy:       policy,
		cache:        cache,
		maxMemory:    maxMemory,
		evictionRate: 0.1, // Evict 10% by default
	}
}

// SetEvictionRate sets the percentage of entries to evict
func (em *EvictionManager) SetEvictionRate(rate float64) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.evictionRate = rate
}

// CheckAndEvict checks if eviction is needed and performs it
func (em *EvictionManager) CheckAndEvict(ctx context.Context, entries []*CacheEntry, currentMemory int64) error {
	em.mu.RLock()
	policy := em.policy
	maxMemory := em.maxMemory
	evictionRate := em.evictionRate
	em.mu.RUnlock()
	
	// Check if any entry should be evicted based on policy
	needsEviction := false
	for _, entry := range entries {
		if policy.ShouldEvict(entry, currentMemory, maxMemory) {
			needsEviction = true
			break
		}
	}
	
	if !needsEviction {
		return nil
	}
	
	// Calculate how many entries to evict
	targetEvictionCount := int(float64(len(entries)) * evictionRate)
	if targetEvictionCount < 1 {
		targetEvictionCount = 1
	}
	
	// Select candidates for eviction
	candidates := policy.SelectEvictionCandidates(entries, targetEvictionCount)
	
	// Perform eviction
	for _, candidate := range candidates {
		if err := em.cache.Delete(ctx, candidate.Key); err != nil {
			// Log error but continue with other deletions
			continue
		}
	}
	
	return nil
}

// GetPolicy returns the current eviction policy
func (em *EvictionManager) GetPolicy() EvictionPolicy {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.policy
}

// SetPolicy changes the eviction policy
func (em *EvictionManager) SetPolicy(policy EvictionPolicy) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.policy = policy
}
