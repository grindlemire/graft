package graft

import (
	"context"
	"sync"
)

// defaultCache is the global cache used by Execute/ExecuteFor.
// Similar to the global registry, this provides automatic caching
// without requiring explicit configuration.
var defaultCache = NewMemoryCache()

// DefaultCache returns the global cache instance.
// This can be used to inspect or clear cached values.
func DefaultCache() *MemoryCache {
	return defaultCache
}

// Cache defines the interface for node output caching.
// Implementations can be in-memory, Redis, disk, etc.
type Cache interface {
	// Snapshot returns a copy of the cache.
	Snapshot() map[ID]any

	// Get retrieves a cached value. Returns (value, true, nil) on hit,
	// (nil, false, nil) on miss, or (nil, false, err) on failure.
	Get(ctx context.Context, id ID) (any, bool, error)

	// Set stores a value in the cache.
	Set(ctx context.Context, id ID, value any) error
}

// MemoryCache is a simple thread-safe in-memory cache.
type MemoryCache struct {
	mu    sync.RWMutex
	store map[ID]any
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{store: make(map[ID]any)}
}

// Get retrieves a value from the cache.
func (m *MemoryCache) Get(_ context.Context, id ID) (any, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.store[id]
	return val, ok, nil
}

// Set stores a value in the cache.
func (m *MemoryCache) Set(_ context.Context, id ID, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[id] = value
	return nil
}

// Delete removes specific entries from the cache.
func (m *MemoryCache) Delete(ids ...ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		delete(m.store, id)
	}
}

// Clear removes all entries from the cache.
func (m *MemoryCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[ID]any)
}

// Snapshot returns a copy of all cached values (useful for debugging/inspection).
func (m *MemoryCache) Snapshot() map[ID]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make(map[ID]any, len(m.store))
	for k, v := range m.store {
		cp[k] = v
	}
	return cp
}

// ResetDefaultCache clears the global default cache.
// This is primarily useful for test isolation.
func ResetDefaultCache() {
	defaultCache.Clear()
}
