package database

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// QueryCache defines the interface for query result caching
type QueryCache interface {
	Get(query string, params []interface{}) ([]byte, bool)
	Set(query string, params []interface{}, result []byte, ttl time.Duration) error
	Invalidate(pattern string) error
	InvalidateTable(tableName string) error
	GetStats() *CacheStats
	WarmCache(queries []CacheWarmupQuery) error
	Clear() error
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	HitRate      float64       `json:"hit_rate"`
	TotalHits    int64         `json:"total_hits"`
	TotalMisses  int64         `json:"total_misses"`
	TotalQueries int64         `json:"total_queries"`
	MemoryUsage  int64         `json:"memory_usage"`
	EntryCount   int           `json:"entry_count"`
	AvgTTL       time.Duration `json:"avg_ttl"`
	LastCleared  time.Time     `json:"last_cleared"`
}

// CacheEntry represents a cached query result
type CacheEntry struct {
	Key         string    `json:"key"`
	Query       string    `json:"query"`
	Params      string    `json:"params"` // JSON encoded
	Result      []byte    `json:"result"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
	Tables      []string  `json:"tables"` // Tables involved in the query
}

// CacheWarmupQuery represents a query to pre-warm in cache
type CacheWarmupQuery struct {
	Query  string        `json:"query"`
	Params []interface{} `json:"params"`
	TTL    time.Duration `json:"ttl"`
}

// MemoryQueryCache implements QueryCache using in-memory storage
type MemoryQueryCache struct {
	entries        map[string]*CacheEntry
	tableIndex     map[string][]string // table -> cache keys
	stats          *CacheStats
	mu             sync.RWMutex
	cleanupTicker  *time.Ticker
	stopCleanup    chan bool
	maxMemoryBytes int64
	maxEntries     int
	defaultTTL     time.Duration
}

// NewMemoryQueryCache creates a new in-memory query cache
func NewMemoryQueryCache(maxMemoryBytes int64, maxEntries int, defaultTTL time.Duration) *MemoryQueryCache {
	cache := &MemoryQueryCache{
		entries:        make(map[string]*CacheEntry),
		tableIndex:     make(map[string][]string),
		stats:          &CacheStats{LastCleared: time.Now()},
		stopCleanup:    make(chan bool),
		maxMemoryBytes: maxMemoryBytes,
		maxEntries:     maxEntries,
		defaultTTL:     defaultTTL,
	}

	cache.startCleanupRoutine()
	return cache
}

// generateCacheKey creates a unique cache key for a query and its parameters
func (c *MemoryQueryCache) generateCacheKey(query string, params []interface{}) string {
	// Normalize query (remove extra whitespace, convert to lowercase)
	normalizedQuery := strings.ToLower(regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(query), " "))

	// Serialize parameters
	paramBytes, _ := json.Marshal(params)

	// Create hash
	hasher := md5.New()
	hasher.Write([]byte(normalizedQuery))
	hasher.Write(paramBytes)

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// extractTables extracts table names from a SQL query
func (c *MemoryQueryCache) extractTables(query string) []string {
	tables := []string{}
	queryUpper := strings.ToUpper(query)

	// Simple regex patterns for common SQL operations
	patterns := []string{
		`FROM\s+(\w+)`,
		`JOIN\s+(\w+)`,
		`UPDATE\s+(\w+)`,
		`INSERT\s+INTO\s+(\w+)`,
		`DELETE\s+FROM\s+(\w+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(queryUpper, -1)
		for _, match := range matches {
			if len(match) > 1 {
				tableName := strings.ToLower(match[1])
				// Avoid duplicates
				found := false
				for _, existing := range tables {
					if existing == tableName {
						found = true
						break
					}
				}
				if !found {
					tables = append(tables, tableName)
				}
			}
		}
	}

	return tables
}

// Get retrieves a cached query result
func (c *MemoryQueryCache) Get(query string, params []interface{}) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateCacheKey(query, params)
	entry, exists := c.entries[key]

	c.stats.TotalQueries++

	if !exists {
		c.stats.TotalMisses++
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		c.stats.TotalMisses++
		// Remove expired entry (will be cleaned up by cleanup routine)
		return nil, false
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccess = time.Now()

	c.stats.TotalHits++
	c.updateHitRate()

	// Return a copy of the result
	result := make([]byte, len(entry.Result))
	copy(result, entry.Result)

	return result, true
}

// Set stores a query result in the cache
func (c *MemoryQueryCache) Set(query string, params []interface{}, result []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	key := c.generateCacheKey(query, params)
	tables := c.extractTables(query)

	// Check memory limits before adding
	if c.shouldEvict(len(result)) {
		c.evictLRU()
	}

	// Serialize parameters for storage
	paramBytes, _ := json.Marshal(params)

	entry := &CacheEntry{
		Key:         key,
		Query:       query,
		Params:      string(paramBytes),
		Result:      make([]byte, len(result)),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(ttl),
		AccessCount: 0,
		LastAccess:  time.Now(),
		Tables:      tables,
	}

	copy(entry.Result, result)

	// Remove existing entry if it exists
	if existing, exists := c.entries[key]; exists {
		c.removeFromTableIndex(existing)
	}

	// Add to cache
	c.entries[key] = entry
	c.addToTableIndex(entry)

	c.updateStats()

	return nil
}

// shouldEvict determines if cache eviction is needed
func (c *MemoryQueryCache) shouldEvict(newEntrySize int) bool {
	currentMemory := c.calculateMemoryUsage()

	// Check memory limit
	if currentMemory+int64(newEntrySize) > c.maxMemoryBytes {
		return true
	}

	// Check entry count limit
	if len(c.entries) >= c.maxEntries {
		return true
	}

	return false
}

// evictLRU removes least recently used entries
func (c *MemoryQueryCache) evictLRU() {
	if len(c.entries) == 0 {
		return
	}

	// Find LRU entry
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range c.entries {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		entry := c.entries[oldestKey]
		c.removeFromTableIndex(entry)
		delete(c.entries, oldestKey)
	}
}

// addToTableIndex adds an entry to the table index
func (c *MemoryQueryCache) addToTableIndex(entry *CacheEntry) {
	for _, table := range entry.Tables {
		c.tableIndex[table] = append(c.tableIndex[table], entry.Key)
	}
}

// removeFromTableIndex removes an entry from the table index
func (c *MemoryQueryCache) removeFromTableIndex(entry *CacheEntry) {
	for _, table := range entry.Tables {
		keys := c.tableIndex[table]
		for i, key := range keys {
			if key == entry.Key {
				c.tableIndex[table] = append(keys[:i], keys[i+1:]...)
				break
			}
		}

		// Remove empty table entries
		if len(c.tableIndex[table]) == 0 {
			delete(c.tableIndex, table)
		}
	}
}

// Invalidate removes cache entries matching a pattern
func (c *MemoryQueryCache) Invalidate(pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	keysToRemove := []string{}
	for key, entry := range c.entries {
		if regex.MatchString(entry.Query) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		entry := c.entries[key]
		c.removeFromTableIndex(entry)
		delete(c.entries, key)
	}

	c.updateStats()

	return nil
}

// InvalidateTable removes all cache entries that involve a specific table
func (c *MemoryQueryCache) InvalidateTable(tableName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tableName = strings.ToLower(tableName)
	keys := c.tableIndex[tableName]

	for _, key := range keys {
		if entry, exists := c.entries[key]; exists {
			c.removeFromTableIndex(entry)
			delete(c.entries, key)
		}
	}

	c.updateStats()

	return nil
}

// GetStats returns current cache statistics
func (c *MemoryQueryCache) GetStats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := *c.stats // Create a copy
	stats.EntryCount = len(c.entries)
	stats.MemoryUsage = c.calculateMemoryUsage()

	if len(c.entries) > 0 {
		var totalTTL time.Duration
		for _, entry := range c.entries {
			totalTTL += entry.ExpiresAt.Sub(entry.CreatedAt)
		}
		stats.AvgTTL = totalTTL / time.Duration(len(c.entries))
	}

	return &stats
}

// calculateMemoryUsage estimates memory usage of cache entries
func (c *MemoryQueryCache) calculateMemoryUsage() int64 {
	var total int64
	for _, entry := range c.entries {
		// Rough estimation of memory usage
		total += int64(len(entry.Key))
		total += int64(len(entry.Query))
		total += int64(len(entry.Params))
		total += int64(len(entry.Result))
		total += int64(len(entry.Tables) * 20) // Estimate for slice overhead
		total += 100                           // Struct overhead estimation
	}
	return total
}

// updateStats updates cache statistics
func (c *MemoryQueryCache) updateStats() {
	c.updateHitRate()
}

// updateHitRate calculates and updates the cache hit rate
func (c *MemoryQueryCache) updateHitRate() {
	if c.stats.TotalQueries > 0 {
		c.stats.HitRate = float64(c.stats.TotalHits) / float64(c.stats.TotalQueries)
	}
}

// WarmCache preloads queries into the cache
func (c *MemoryQueryCache) WarmCache(queries []CacheWarmupQuery) error {
	// This would typically execute the queries and cache the results
	// For this implementation, we'll just prepare cache entries
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, warmupQuery := range queries {
		key := c.generateCacheKey(warmupQuery.Query, warmupQuery.Params)

		// Skip if already cached
		if _, exists := c.entries[key]; exists {
			continue
		}

		// Create placeholder entry (in real implementation, execute query here)
		tables := c.extractTables(warmupQuery.Query)
		paramBytes, _ := json.Marshal(warmupQuery.Params)

		entry := &CacheEntry{
			Key:         key,
			Query:       warmupQuery.Query,
			Params:      string(paramBytes),
			Result:      []byte{}, // Would contain actual query result
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(warmupQuery.TTL),
			AccessCount: 0,
			LastAccess:  time.Now(),
			Tables:      tables,
		}

		c.entries[key] = entry
		c.addToTableIndex(entry)
	}

	c.updateStats()

	return nil
}

// Clear removes all entries from the cache
func (c *MemoryQueryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.tableIndex = make(map[string][]string)
	c.stats.LastCleared = time.Now()
	c.stats.EntryCount = 0
	c.stats.MemoryUsage = 0

	return nil
}

// startCleanupRoutine starts a routine to clean up expired entries
func (c *MemoryQueryCache) startCleanupRoutine() {
	c.cleanupTicker = time.NewTicker(time.Minute * 5) // Cleanup every 5 minutes

	go func() {
		for {
			select {
			case <-c.cleanupTicker.C:
				c.cleanupExpired()
			case <-c.stopCleanup:
				c.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanupExpired removes expired cache entries
func (c *MemoryQueryCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	keysToRemove := []string{}

	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		entry := c.entries[key]
		c.removeFromTableIndex(entry)
		delete(c.entries, key)
	}

	if len(keysToRemove) > 0 {
		c.updateStats()
	}
}

// Stop stops the cache cleanup routine
func (c *MemoryQueryCache) Stop() {
	close(c.stopCleanup)
}

// GetCacheReport generates a comprehensive cache performance report
func (c *MemoryQueryCache) GetCacheReport() *CacheReport {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.GetStats()

	// Analyze entry distribution
	tableDistribution := make(map[string]int)
	for table, keys := range c.tableIndex {
		tableDistribution[table] = len(keys)
	}

	// Find most accessed entries
	topEntries := []*CacheEntry{}
	for _, entry := range c.entries {
		topEntries = append(topEntries, entry)
	}

	// Sort by access count (simplified - in production use proper sorting)
	// For now, just take first 10 as example
	if len(topEntries) > 10 {
		topEntries = topEntries[:10]
	}

	report := &CacheReport{
		GeneratedAt:       time.Now(),
		Stats:             *stats,
		TableDistribution: tableDistribution,
		TopEntries:        topEntries,
		Recommendations:   c.generateCacheRecommendations(stats),
	}

	return report
}

// CacheReport contains comprehensive cache analysis
type CacheReport struct {
	GeneratedAt       time.Time      `json:"generated_at"`
	Stats             CacheStats     `json:"stats"`
	TableDistribution map[string]int `json:"table_distribution"`
	TopEntries        []*CacheEntry  `json:"top_entries"`
	Recommendations   []string       `json:"recommendations"`
}

// generateCacheRecommendations generates cache optimization recommendations
func (c *MemoryQueryCache) generateCacheRecommendations(stats *CacheStats) []string {
	recommendations := []string{}

	if stats.HitRate < 0.5 {
		recommendations = append(recommendations, "Low cache hit rate - consider increasing TTL or reviewing query patterns")
	}

	if stats.MemoryUsage > c.maxMemoryBytes*8/10 {
		recommendations = append(recommendations, "High memory usage - consider increasing cache size or reducing TTL")
	}

	if stats.EntryCount > c.maxEntries*8/10 {
		recommendations = append(recommendations, "High entry count - consider increasing max entries or implementing better eviction")
	}

	if len(c.tableIndex) > 20 {
		recommendations = append(recommendations, "Many tables cached - consider table-specific cache strategies")
	}

	return recommendations
}

// CacheInvalidationManager handles intelligent cache invalidation
type CacheInvalidationManager struct {
	cache              QueryCache
	writeOperations    []string
	tableInvalidations map[string]time.Time
	mu                 sync.RWMutex
}

// NewCacheInvalidationManager creates a new cache invalidation manager
func NewCacheInvalidationManager(cache QueryCache) *CacheInvalidationManager {
	return &CacheInvalidationManager{
		cache:              cache,
		writeOperations:    []string{"INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP"},
		tableInvalidations: make(map[string]time.Time),
	}
}

// HandleQuery determines if cache should be invalidated based on query type
func (cim *CacheInvalidationManager) HandleQuery(query string) error {
	queryUpper := strings.ToUpper(strings.TrimSpace(query))

	// Check if it's a write operation
	for _, op := range cim.writeOperations {
		if strings.HasPrefix(queryUpper, op) {
			return cim.invalidateAffectedTables(query)
		}
	}

	return nil
}

// invalidateAffectedTables invalidates cache entries for tables affected by the query
func (cim *CacheInvalidationManager) invalidateAffectedTables(query string) error {
	cim.mu.Lock()
	defer cim.mu.Unlock()

	// Extract affected tables
	tables := []string{}
	queryUpper := strings.ToUpper(query)

	patterns := []string{
		`UPDATE\s+(\w+)`,
		`INSERT\s+INTO\s+(\w+)`,
		`DELETE\s+FROM\s+(\w+)`,
		`CREATE\s+TABLE\s+(\w+)`,
		`ALTER\s+TABLE\s+(\w+)`,
		`DROP\s+TABLE\s+(\w+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(queryUpper, -1)
		for _, match := range matches {
			if len(match) > 1 {
				tables = append(tables, strings.ToLower(match[1]))
			}
		}
	}

	// Invalidate cache for affected tables
	for _, table := range tables {
		if err := cim.cache.InvalidateTable(table); err != nil {
			return err
		}
		cim.tableInvalidations[table] = time.Now()
	}

	return nil
}

// GetInvalidationStats returns statistics about cache invalidations
func (cim *CacheInvalidationManager) GetInvalidationStats() map[string]time.Time {
	cim.mu.RLock()
	defer cim.mu.RUnlock()

	stats := make(map[string]time.Time)
	for table, timestamp := range cim.tableInvalidations {
		stats[table] = timestamp
	}

	return stats
}
