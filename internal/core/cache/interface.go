package cache

import (
	"context"
	"time"
)

// Cache defines the interface that all system caches must implement
type Cache interface {
	// Clear removes all entries from the cache
	Clear() error

	// Refresh reloads cache data from the source
	Refresh() error

	// Size returns the number of entries in the cache
	Size() int

	// Stats returns cache performance statistics
	Stats() CacheStats

	// Healthy returns true if the cache is functioning properly
	Healthy() bool

	// Name returns the cache identifier
	Name() string

	// Type returns the cache category/type
	Type() CacheType

	// Get retrieves a value by key (optional, not all caches support this)
	Get(key string) (interface{}, bool)

	// Set stores a value by key (optional, not all caches support this)
	Set(key string, value interface{}, ttl time.Duration) error

	// Delete removes a specific key from the cache
	Delete(key string) error

	// Keys returns all cache keys (for debugging/inspection)
	Keys() []string
}

// CacheType represents different categories of caches
type CacheType string

const (
	CacheTypeEntity        CacheType = "entity"
	CacheTypeDisplay       CacheType = "display"
	CacheTypeNetwork       CacheType = "network"
	CacheTypeEnergy        CacheType = "energy"
	CacheTypeConfiguration CacheType = "configuration"
	CacheTypeQuery         CacheType = "query"
	CacheTypeResponse      CacheType = "response"
	CacheTypeSession       CacheType = "session"
	CacheTypeWebSocket     CacheType = "websocket"
	CacheTypeAnalytics     CacheType = "analytics"
	CacheTypeSystem        CacheType = "system"
	CacheTypeOther         CacheType = "other"
)

// CacheStats contains cache performance metrics
type CacheStats struct {
	Name              string        `json:"name"`
	Type              CacheType     `json:"type"`
	Size              int           `json:"size"`
	MemoryUsage       int64         `json:"memory_usage_bytes"`
	HitCount          uint64        `json:"hit_count"`
	MissCount         uint64        `json:"miss_count"`
	HitRate           float64       `json:"hit_rate"`
	AverageAccessTime time.Duration `json:"average_access_time"`
	LastAccessed      time.Time     `json:"last_accessed"`
	CreatedAt         time.Time     `json:"created_at"`
	TTL               time.Duration `json:"ttl"`
	IsHealthy         bool          `json:"is_healthy"`
	ErrorCount        uint64        `json:"error_count"`
	LastError         string        `json:"last_error,omitempty"`
}

// CacheRegistry manages all system caches
type CacheRegistry interface {
	// Register adds a cache to the registry
	Register(cache Cache) error

	// Unregister removes a cache from the registry
	Unregister(name string) error

	// Get retrieves a cache by name
	Get(name string) (Cache, bool)

	// List returns all registered caches
	List() []Cache

	// ListByType returns caches of a specific type
	ListByType(cacheType CacheType) []Cache

	// ClearAll clears all registered caches
	ClearAll(ctx context.Context) error

	// RefreshAll refreshes all registered caches
	RefreshAll(ctx context.Context) error

	// StatsAll returns statistics for all caches
	StatsAll() []CacheStats

	// HealthCheck checks the health of all caches
	HealthCheck() map[string]bool
}

// CacheManager provides high-level cache management operations
type CacheManager interface {
	// Registry access
	Registry() CacheRegistry

	// Bulk operations
	ClearCaches(ctx context.Context, names []string) map[string]error
	RefreshCaches(ctx context.Context, names []string) map[string]error

	// Type-based operations
	ClearByType(ctx context.Context, cacheType CacheType) error
	RefreshByType(ctx context.Context, cacheType CacheType) error

	// Optimization operations
	OptimizeCaches(ctx context.Context) error
	WarmCaches(ctx context.Context) error

	// Statistics and monitoring
	GetStats() CacheManagerStats
	GetCacheStats(name string) (CacheStats, error)

	// Health monitoring
	HealthCheck() CacheHealthReport

	// Memory management
	FreeMemory(ctx context.Context, targetMB int) error
	GetMemoryUsage() CacheMemoryReport
}

// CacheManagerStats contains overall cache manager statistics
type CacheManagerStats struct {
	TotalCaches      int               `json:"total_caches"`
	HealthyCaches    int               `json:"healthy_caches"`
	TotalMemoryUsage int64             `json:"total_memory_usage_bytes"`
	TotalHits        uint64            `json:"total_hits"`
	TotalMisses      uint64            `json:"total_misses"`
	OverallHitRate   float64           `json:"overall_hit_rate"`
	CachesByType     map[CacheType]int `json:"caches_by_type"`
	LastUpdated      time.Time         `json:"last_updated"`
}

// CacheHealthReport contains health information for all caches
type CacheHealthReport struct {
	OverallHealth   bool               `json:"overall_health"`
	HealthyCaches   []string           `json:"healthy_caches"`
	UnhealthyCaches []string           `json:"unhealthy_caches"`
	CacheHealth     map[string]bool    `json:"cache_health"`
	Issues          []CacheHealthIssue `json:"issues"`
	LastCheck       time.Time          `json:"last_check"`
}

// CacheHealthIssue represents a cache health problem
type CacheHealthIssue struct {
	CacheName   string    `json:"cache_name"`
	Severity    string    `json:"severity"` // "warning", "error", "critical"
	Issue       string    `json:"issue"`
	Description string    `json:"description"`
	Suggestion  string    `json:"suggestion"`
	DetectedAt  time.Time `json:"detected_at"`
}

// CacheMemoryReport contains memory usage information
type CacheMemoryReport struct {
	TotalMemoryUsage   int64               `json:"total_memory_usage_bytes"`
	MemoryByCache      map[string]int64    `json:"memory_by_cache"`
	MemoryByType       map[CacheType]int64 `json:"memory_by_type"`
	LargestCaches      []CacheMemoryInfo   `json:"largest_caches"`
	MemoryPressure     bool                `json:"memory_pressure"`
	RecommendedActions []string            `json:"recommended_actions"`
}

// CacheMemoryInfo contains memory information for a specific cache
type CacheMemoryInfo struct {
	Name        string    `json:"name"`
	Type        CacheType `json:"type"`
	MemoryUsage int64     `json:"memory_usage_bytes"`
	Percentage  float64   `json:"percentage_of_total"`
}

// CacheOperation represents the result of a cache operation
type CacheOperation struct {
	CacheName       string        `json:"cache_name"`
	Operation       string        `json:"operation"`
	Success         bool          `json:"success"`
	Error           string        `json:"error,omitempty"`
	Duration        time.Duration `json:"duration"`
	EntriesAffected int           `json:"entries_affected"`
	MemoryFreed     int64         `json:"memory_freed_bytes"`
	Timestamp       time.Time     `json:"timestamp"`
}

// CacheOperationResult contains the results of multiple cache operations
type CacheOperationResult struct {
	Operations           []CacheOperation `json:"operations"`
	SuccessCount         int              `json:"success_count"`
	ErrorCount           int              `json:"error_count"`
	TotalDuration        time.Duration    `json:"total_duration"`
	TotalMemoryFreed     int64            `json:"total_memory_freed_bytes"`
	TotalEntriesAffected int              `json:"total_entries_affected"`
}
