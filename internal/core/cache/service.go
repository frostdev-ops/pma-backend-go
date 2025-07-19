package cache

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
)

// registryImpl implements CacheRegistry
type registryImpl struct {
	caches map[string]Cache
	mutex  sync.RWMutex
	logger *logrus.Logger
}

// NewRegistry creates a new cache registry
func NewRegistry(logger *logrus.Logger) CacheRegistry {
	return &registryImpl{
		caches: make(map[string]Cache),
		logger: logger,
	}
}

// Register adds a cache to the registry
func (r *registryImpl) Register(cache Cache) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := cache.Name()
	if name == "" {
		return fmt.Errorf("cache name cannot be empty")
	}

	if _, exists := r.caches[name]; exists {
		return fmt.Errorf("cache with name %s already exists", name)
	}

	r.caches[name] = cache
	r.logger.WithField("cache", name).Debug("Cache registered")
	return nil
}

// Unregister removes a cache from the registry
func (r *registryImpl) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.caches[name]; !exists {
		return fmt.Errorf("cache with name %s not found", name)
	}

	delete(r.caches, name)
	r.logger.WithField("cache", name).Debug("Cache unregistered")
	return nil
}

// Get retrieves a cache by name
func (r *registryImpl) Get(name string) (Cache, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	cache, exists := r.caches[name]
	return cache, exists
}

// List returns all registered caches
func (r *registryImpl) List() []Cache {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	caches := make([]Cache, 0, len(r.caches))
	for _, cache := range r.caches {
		caches = append(caches, cache)
	}
	return caches
}

// ListByType returns caches of a specific type
func (r *registryImpl) ListByType(cacheType CacheType) []Cache {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var caches []Cache
	for _, cache := range r.caches {
		if cache.Type() == cacheType {
			caches = append(caches, cache)
		}
	}
	return caches
}

// ClearAll clears all registered caches
func (r *registryImpl) ClearAll(ctx context.Context) error {
	r.mutex.RLock()
	caches := make([]Cache, 0, len(r.caches))
	for _, cache := range r.caches {
		caches = append(caches, cache)
	}
	r.mutex.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(caches))

	for _, cache := range caches {
		wg.Add(1)
		go func(c Cache) {
			defer wg.Done()
			if err := c.Clear(); err != nil {
				errChan <- fmt.Errorf("failed to clear cache %s: %w", c.Name(), err)
			}
		}(cache)
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors clearing caches: %v", errors)
	}

	return nil
}

// RefreshAll refreshes all registered caches
func (r *registryImpl) RefreshAll(ctx context.Context) error {
	r.mutex.RLock()
	caches := make([]Cache, 0, len(r.caches))
	for _, cache := range r.caches {
		caches = append(caches, cache)
	}
	r.mutex.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(caches))

	for _, cache := range caches {
		wg.Add(1)
		go func(c Cache) {
			defer wg.Done()
			if err := c.Refresh(); err != nil {
				errChan <- fmt.Errorf("failed to refresh cache %s: %w", c.Name(), err)
			}
		}(cache)
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors refreshing caches: %v", errors)
	}

	return nil
}

// StatsAll returns statistics for all caches
func (r *registryImpl) StatsAll() []CacheStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := make([]CacheStats, 0, len(r.caches))
	for _, cache := range r.caches {
		stats = append(stats, cache.Stats())
	}
	return stats
}

// HealthCheck checks the health of all caches
func (r *registryImpl) HealthCheck() map[string]bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	health := make(map[string]bool)
	for name, cache := range r.caches {
		health[name] = cache.Healthy()
	}
	return health
}

// managerImpl implements CacheManager
type managerImpl struct {
	registry  CacheRegistry
	logger    *logrus.Logger
	startTime time.Time
}

// NewManager creates a new cache manager
func NewManager(registry CacheRegistry, logger *logrus.Logger) CacheManager {
	return &managerImpl{
		registry:  registry,
		logger:    logger,
		startTime: time.Now(),
	}
}

// Registry returns the cache registry
func (m *managerImpl) Registry() CacheRegistry {
	return m.registry
}

// ClearCaches clears specified caches by name
func (m *managerImpl) ClearCaches(ctx context.Context, names []string) map[string]error {
	results := make(map[string]error)

	for _, name := range names {
		cache, exists := m.registry.Get(name)
		if !exists {
			results[name] = fmt.Errorf("cache not found")
			continue
		}

		start := time.Now()
		err := cache.Clear()
		duration := time.Since(start)

		if err != nil {
			results[name] = err
			m.logger.WithFields(logrus.Fields{
				"cache":    name,
				"duration": duration,
				"error":    err,
			}).Error("Failed to clear cache")
		} else {
			m.logger.WithFields(logrus.Fields{
				"cache":    name,
				"duration": duration,
			}).Info("Cache cleared successfully")
		}
	}

	return results
}

// RefreshCaches refreshes specified caches by name
func (m *managerImpl) RefreshCaches(ctx context.Context, names []string) map[string]error {
	results := make(map[string]error)

	for _, name := range names {
		cache, exists := m.registry.Get(name)
		if !exists {
			results[name] = fmt.Errorf("cache not found")
			continue
		}

		start := time.Now()
		err := cache.Refresh()
		duration := time.Since(start)

		if err != nil {
			results[name] = err
			m.logger.WithFields(logrus.Fields{
				"cache":    name,
				"duration": duration,
				"error":    err,
			}).Error("Failed to refresh cache")
		} else {
			m.logger.WithFields(logrus.Fields{
				"cache":    name,
				"duration": duration,
			}).Info("Cache refreshed successfully")
		}
	}

	return results
}

// ClearByType clears all caches of a specific type
func (m *managerImpl) ClearByType(ctx context.Context, cacheType CacheType) error {
	caches := m.registry.ListByType(cacheType)

	var wg sync.WaitGroup
	errChan := make(chan error, len(caches))

	for _, cache := range caches {
		wg.Add(1)
		go func(c Cache) {
			defer wg.Done()
			if err := c.Clear(); err != nil {
				errChan <- fmt.Errorf("failed to clear cache %s: %w", c.Name(), err)
			}
		}(cache)
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors clearing caches of type %s: %v", cacheType, errors)
	}

	return nil
}

// RefreshByType refreshes all caches of a specific type
func (m *managerImpl) RefreshByType(ctx context.Context, cacheType CacheType) error {
	caches := m.registry.ListByType(cacheType)

	var wg sync.WaitGroup
	errChan := make(chan error, len(caches))

	for _, cache := range caches {
		wg.Add(1)
		go func(c Cache) {
			defer wg.Done()
			if err := c.Refresh(); err != nil {
				errChan <- fmt.Errorf("failed to refresh cache %s: %w", c.Name(), err)
			}
		}(cache)
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors refreshing caches of type %s: %v", cacheType, errors)
	}

	return nil
}

// OptimizeCaches performs cache optimization operations
func (m *managerImpl) OptimizeCaches(ctx context.Context) error {
	caches := m.registry.List()
	optimizedCount := 0

	for _, cache := range caches {
		stats := cache.Stats()

		// Check if cache needs optimization based on hit rate
		if stats.HitRate < 0.5 && stats.HitCount+stats.MissCount > 100 {
			// Low hit rate - consider clearing and refreshing
			if err := cache.Clear(); err != nil {
				m.logger.WithFields(logrus.Fields{
					"cache": cache.Name(),
					"error": err,
				}).Error("Failed to clear cache during optimization")
				continue
			}

			if err := cache.Refresh(); err != nil {
				m.logger.WithFields(logrus.Fields{
					"cache": cache.Name(),
					"error": err,
				}).Error("Failed to refresh cache during optimization")
				continue
			}

			optimizedCount++
			m.logger.WithField("cache", cache.Name()).Info("Cache optimized")
		}
	}

	m.logger.WithField("optimized_count", optimizedCount).Info("Cache optimization completed")
	return nil
}

// WarmCaches preloads frequently accessed data
func (m *managerImpl) WarmCaches(ctx context.Context) error {
	caches := m.registry.List()
	warmedCount := 0

	for _, cache := range caches {
		// Refresh cache to warm it up
		if err := cache.Refresh(); err != nil {
			m.logger.WithFields(logrus.Fields{
				"cache": cache.Name(),
				"error": err,
			}).Error("Failed to warm cache")
			continue
		}

		warmedCount++
		m.logger.WithField("cache", cache.Name()).Debug("Cache warmed")
	}

	m.logger.WithField("warmed_count", warmedCount).Info("Cache warming completed")
	return nil
}

// GetStats returns overall cache manager statistics
func (m *managerImpl) GetStats() CacheManagerStats {
	caches := m.registry.List()
	stats := CacheManagerStats{
		TotalCaches:  len(caches),
		CachesByType: make(map[CacheType]int),
		LastUpdated:  time.Now(),
	}

	var totalHits, totalMisses uint64
	var totalMemory int64

	for _, cache := range caches {
		cacheStats := cache.Stats()

		totalHits += cacheStats.HitCount
		totalMisses += cacheStats.MissCount
		totalMemory += cacheStats.MemoryUsage
		stats.CachesByType[cacheStats.Type]++

		if cacheStats.IsHealthy {
			stats.HealthyCaches++
		}
	}

	stats.TotalHits = totalHits
	stats.TotalMisses = totalMisses
	stats.TotalMemoryUsage = totalMemory

	if totalHits+totalMisses > 0 {
		stats.OverallHitRate = float64(totalHits) / float64(totalHits+totalMisses)
	}

	return stats
}

// GetCacheStats returns statistics for a specific cache
func (m *managerImpl) GetCacheStats(name string) (CacheStats, error) {
	cache, exists := m.registry.Get(name)
	if !exists {
		return CacheStats{}, fmt.Errorf("cache %s not found", name)
	}

	return cache.Stats(), nil
}

// HealthCheck performs a comprehensive health check
func (m *managerImpl) HealthCheck() CacheHealthReport {
	caches := m.registry.List()
	report := CacheHealthReport{
		OverallHealth:   true,
		HealthyCaches:   make([]string, 0),
		UnhealthyCaches: make([]string, 0),
		CacheHealth:     make(map[string]bool),
		Issues:          make([]CacheHealthIssue, 0),
		LastCheck:       time.Now(),
	}

	for _, cache := range caches {
		healthy := cache.Healthy()
		name := cache.Name()

		report.CacheHealth[name] = healthy

		if healthy {
			report.HealthyCaches = append(report.HealthyCaches, name)
		} else {
			report.UnhealthyCaches = append(report.UnhealthyCaches, name)
			report.OverallHealth = false

			stats := cache.Stats()
			issue := CacheHealthIssue{
				CacheName:   name,
				Severity:    "error",
				Issue:       "Cache unhealthy",
				Description: fmt.Sprintf("Cache %s is reporting unhealthy status", name),
				Suggestion:  "Check cache implementation and refresh/clear if needed",
				DetectedAt:  time.Now(),
			}

			if stats.ErrorCount > 0 {
				issue.Description += fmt.Sprintf(", %d errors recorded", stats.ErrorCount)
			}

			report.Issues = append(report.Issues, issue)
		}

		// Check for performance issues
		stats := cache.Stats()
		if stats.HitRate < 0.3 && stats.HitCount+stats.MissCount > 50 {
			issue := CacheHealthIssue{
				CacheName:   name,
				Severity:    "warning",
				Issue:       "Low hit rate",
				Description: fmt.Sprintf("Cache %s has low hit rate: %.2f%%", name, stats.HitRate*100),
				Suggestion:  "Consider cache optimization or TTL adjustment",
				DetectedAt:  time.Now(),
			}
			report.Issues = append(report.Issues, issue)
		}
	}

	return report
}

// FreeMemory attempts to free cache memory to reach target
func (m *managerImpl) FreeMemory(ctx context.Context, targetMB int) error {
	targetBytes := int64(targetMB * 1024 * 1024)
	report := m.GetMemoryUsage()

	if report.TotalMemoryUsage <= targetBytes {
		return nil // Already under target
	}

	// Sort caches by memory usage (largest first)
	sort.Slice(report.LargestCaches, func(i, j int) bool {
		return report.LargestCaches[i].MemoryUsage > report.LargestCaches[j].MemoryUsage
	})

	freedMemory := int64(0)
	for _, cacheInfo := range report.LargestCaches {
		if report.TotalMemoryUsage-freedMemory <= targetBytes {
			break
		}

		cache, exists := m.registry.Get(cacheInfo.Name)
		if !exists {
			continue
		}

		beforeSize := cache.Size()
		if err := cache.Clear(); err != nil {
			m.logger.WithFields(logrus.Fields{
				"cache": cacheInfo.Name,
				"error": err,
			}).Error("Failed to clear cache for memory optimization")
			continue
		}

		freedMemory += cacheInfo.MemoryUsage
		m.logger.WithFields(logrus.Fields{
			"cache":        cacheInfo.Name,
			"freed_memory": cacheInfo.MemoryUsage,
			"entries":      beforeSize,
		}).Info("Cache cleared for memory optimization")
	}

	// Force garbage collection
	runtime.GC()

	m.logger.WithFields(logrus.Fields{
		"target_bytes": targetBytes,
		"freed_bytes":  freedMemory,
	}).Info("Memory optimization completed")

	return nil
}

// GetMemoryUsage returns detailed memory usage information
func (m *managerImpl) GetMemoryUsage() CacheMemoryReport {
	caches := m.registry.List()
	report := CacheMemoryReport{
		MemoryByCache:      make(map[string]int64),
		MemoryByType:       make(map[CacheType]int64),
		LargestCaches:      make([]CacheMemoryInfo, 0),
		RecommendedActions: make([]string, 0),
	}

	for _, cache := range caches {
		stats := cache.Stats()
		name := cache.Name()

		report.MemoryByCache[name] = stats.MemoryUsage
		report.MemoryByType[stats.Type] += stats.MemoryUsage
		report.TotalMemoryUsage += stats.MemoryUsage

		report.LargestCaches = append(report.LargestCaches, CacheMemoryInfo{
			Name:        name,
			Type:        stats.Type,
			MemoryUsage: stats.MemoryUsage,
		})
	}

	// Calculate percentages
	for i := range report.LargestCaches {
		if report.TotalMemoryUsage > 0 {
			report.LargestCaches[i].Percentage = float64(report.LargestCaches[i].MemoryUsage) / float64(report.TotalMemoryUsage) * 100
		}
	}

	// Sort by memory usage
	sort.Slice(report.LargestCaches, func(i, j int) bool {
		return report.LargestCaches[i].MemoryUsage > report.LargestCaches[j].MemoryUsage
	})

		// Check for memory pressure
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	if float64(report.TotalMemoryUsage)/float64(memStats.Alloc) > 0.5 {
		report.MemoryPressure = true
		report.RecommendedActions = append(report.RecommendedActions, 
			"High cache memory usage detected - consider clearing least used caches")
	}

	if len(report.LargestCaches) > 0 && report.LargestCaches[0].Percentage > 40 {
		report.RecommendedActions = append(report.RecommendedActions,
			fmt.Sprintf("Cache '%s' uses %.1f%% of total cache memory - consider optimization",
				report.LargestCaches[0].Name, report.LargestCaches[0].Percentage))
	}

	return report
}

// estimateMemoryUsage provides a rough estimate of memory usage for basic data structures
func estimateMemoryUsage(data interface{}) int64 {
	if data == nil {
		return 0
	}

	// This is a simplified estimation - real implementations would be more sophisticated
	return int64(unsafe.Sizeof(data))
}
