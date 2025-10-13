package oauth

import (
	"fmt"
	"sync"
	"time"
)

// MemoryStateStore implements StateStore using in-memory storage with TTL
type MemoryStateStore struct {
	store map[string]*stateEntry
	mutex sync.RWMutex
}

type stateEntry struct {
	pkce      *PKCEParams
	expiresAt time.Time
}

func NewMemoryStateStore() *MemoryStateStore {
	store := &MemoryStateStore{
		store: make(map[string]*stateEntry),
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

// Store saves PKCE parameters with TTL
func (s *MemoryStateStore) Store(state string, pkce *PKCEParams, ttl time.Duration) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.store[state] = &stateEntry{
		pkce:      pkce,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Retrieve gets PKCE parameters by state
func (s *MemoryStateStore) Retrieve(state string) (*PKCEParams, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	entry, exists := s.store[state]
	if !exists {
		return nil, fmt.Errorf("state not found")
	}

	if time.Now().After(entry.expiresAt) {
		return nil, fmt.Errorf("state expired")
	}

	return entry.pkce, nil
}

// Delete removes state entry
func (s *MemoryStateStore) Delete(state string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.store, state)
	return nil
}

// cleanup removes expired entries
func (s *MemoryStateStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mutex.Lock()
		now := time.Now()
		for state, entry := range s.store {
			if now.After(entry.expiresAt) {
				delete(s.store, state)
			}
		}
		s.mutex.Unlock()
	}
}
