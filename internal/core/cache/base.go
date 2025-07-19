package cache

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// BaseCache provides common cache functionality that can be embedded
type BaseCache struct {
	name      string
	cacheType CacheType
	data      map[string]*CacheEntry
	mutex     sync.RWMutex
	stats     *CacheStatistics
	ttl       time.Duration
	createdAt time.Time
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key          string
	Value        interface{}
	ExpiresAt    time.Time
	CreatedAt    time.Time
	AccessCount  uint64
	LastAccessed time.Time
}

// CacheStatistics tracks cache performance metrics
type CacheStatistics struct {
	HitCount     uint64
	MissCount    uint64
	ErrorCount   uint64
	LastError    string
	LastAccessed time.Time
	TotalSets    uint64
	TotalDeletes uint64
	mutex        sync.RWMutex
}

// NewBaseCache creates a new base cache
func NewBaseCache(name string, cacheType CacheType, ttl time.Duration) *BaseCache {
	return &BaseCache{
		name:      name,
		cacheType: cacheType,
		data:      make(map[string]*CacheEntry),
		stats:     &CacheStatistics{},
		ttl:       ttl,
		createdAt: time.Now(),
	}
}

// Name returns the cache identifier
func (c *BaseCache) Name() string {
	return c.name
}

// Type returns the cache category/type
func (c *BaseCache) Type() CacheType {
	return c.cacheType
}

// Size returns the number of entries in the cache
func (c *BaseCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.data)
}

// Clear removes all entries from the cache
func (c *BaseCache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]*CacheEntry)
	return nil
}

// Get retrieves a value by key
func (c *BaseCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		c.recordMiss()
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.data, key)
		c.mutex.Unlock()
		c.mutex.RLock()
		c.recordMiss()
		return nil, false
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccessed = time.Now()

	c.recordHit()
	return entry.Value, true
}

// Set stores a value by key
func (c *BaseCache) Set(key string, value interface{}, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ttl == 0 {
		ttl = c.ttl
	}

	entry := &CacheEntry{
		Key:          key,
		Value:        value,
		ExpiresAt:    time.Now().Add(ttl),
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
	}

	c.data[key] = entry
	c.recordSet()
	return nil
}

// Delete removes a specific key from the cache
func (c *BaseCache) Delete(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
	c.recordDelete()
	return nil
}

// Keys returns all cache keys
func (c *BaseCache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.data))
	for key := range c.data {
		keys = append(keys, key)
	}
	return keys
}

// Stats returns cache performance statistics
func (c *BaseCache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	totalRequests := c.stats.HitCount + c.stats.MissCount
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(c.stats.HitCount) / float64(totalRequests)
	}

	return CacheStats{
		Name:         c.name,
		Type:         c.cacheType,
		Size:         len(c.data),
		MemoryUsage:  c.estimateMemoryUsage(),
		HitCount:     c.stats.HitCount,
		MissCount:    c.stats.MissCount,
		HitRate:      hitRate,
		LastAccessed: c.stats.LastAccessed,
		CreatedAt:    c.createdAt,
		TTL:          c.ttl,
		IsHealthy:    c.Healthy(),
		ErrorCount:   c.stats.ErrorCount,
		LastError:    c.stats.LastError,
	}
}

// Healthy returns true if the cache is functioning properly
func (c *BaseCache) Healthy() bool {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	// Consider unhealthy if there are recent errors
	return c.stats.ErrorCount == 0 || c.stats.LastError == ""
}

// Refresh reloads cache data from the source (base implementation is a no-op)
func (c *BaseCache) Refresh() error {
	// Base implementation - subclasses should override this
	return nil
}

// CleanupExpired removes expired entries
func (c *BaseCache) CleanupExpired() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
			removed++
		}
	}

	return removed
}

// GetExpiredKeys returns keys of expired entries
func (c *BaseCache) GetExpiredKeys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	now := time.Now()
	expired := make([]string, 0)

	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			expired = append(expired, key)
		}
	}

	return expired
}

// recordHit increments hit counter
func (c *BaseCache) recordHit() {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()

	c.stats.HitCount++
	c.stats.LastAccessed = time.Now()
}

// recordMiss increments miss counter
func (c *BaseCache) recordMiss() {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()

	c.stats.MissCount++
	c.stats.LastAccessed = time.Now()
}

// recordSet increments set counter
func (c *BaseCache) recordSet() {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()

	c.stats.TotalSets++
}

// recordDelete increments delete counter
func (c *BaseCache) recordDelete() {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()

	c.stats.TotalDeletes++
}

// recordError records an error
func (c *BaseCache) recordError(err error) {
	if err == nil {
		return
	}

	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()

	c.stats.ErrorCount++
	c.stats.LastError = err.Error()
}

// estimateMemoryUsage provides a rough estimate of memory usage
func (c *BaseCache) estimateMemoryUsage() int64 {
	var total int64

	// Estimate overhead for the map structure
	total += int64(len(c.data)) * 24 // rough estimate for map overhead per entry

	// Estimate memory for entries
	for _, entry := range c.data {
		total += int64(unsafe.Sizeof(*entry))
		total += int64(len(entry.Key))

		// Estimate value size (this is very approximate)
		if entry.Value != nil {
			total += int64(unsafe.Sizeof(entry.Value))
			// Try to estimate string size if it's a string
			if str, ok := entry.Value.(string); ok {
				total += int64(len(str))
			}
		}
	}

	return total
}

// SetTTL updates the default TTL for new entries
func (c *BaseCache) SetTTL(ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.ttl = ttl
}

// GetTTL returns the default TTL
func (c *BaseCache) GetTTL() time.Duration {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.ttl
}

// Has checks if a key exists in the cache (without updating access stats)
func (c *BaseCache) Has(key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return false
	}

	// Check if expired
	return !time.Now().After(entry.ExpiresAt)
}

// TouchKey updates the last accessed time for a key
func (c *BaseCache) TouchKey(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.data[key]
	if !exists {
		return fmt.Errorf("key not found")
	}

	entry.LastAccessed = time.Now()
	return nil
}

// ExtendTTL extends the TTL for a specific key
func (c *BaseCache) ExtendTTL(key string, extension time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.data[key]
	if !exists {
		return fmt.Errorf("key not found")
	}

	entry.ExpiresAt = entry.ExpiresAt.Add(extension)
	return nil
}

// GetEntryInfo returns detailed information about a cache entry
func (c *BaseCache) GetEntryInfo(key string) (*CacheEntry, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	// Return a copy to prevent external modification
	return &CacheEntry{
		Key:          entry.Key,
		Value:        entry.Value,
		ExpiresAt:    entry.ExpiresAt,
		CreatedAt:    entry.CreatedAt,
		AccessCount:  entry.AccessCount,
		LastAccessed: entry.LastAccessed,
	}, nil
}
