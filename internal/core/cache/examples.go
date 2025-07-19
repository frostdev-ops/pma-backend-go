package cache

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// CacheExample demonstrates how to use the cache management system
type CacheExample struct {
	manager CacheManager
	logger  *logrus.Logger
}

// NewCacheExample creates a new cache example
func NewCacheExample(logger *logrus.Logger) *CacheExample {
	registry := NewRegistry(logger)
	manager := NewManager(registry, logger)

	return &CacheExample{
		manager: manager,
		logger:  logger,
	}
}

// DemoBasicOperations demonstrates basic cache operations
func (e *CacheExample) DemoBasicOperations() error {
	e.logger.Info("=== Cache Management System Demo ===")

	// Create and register some test caches
	testCaches := CreateTestCaches()
	for _, cache := range testCaches {
		if err := e.manager.Registry().Register(cache); err != nil {
			e.logger.WithError(err).Error("Failed to register test cache")
			return err
		}
	}

	// Add some default caches
	defaultCaches := CreateDefaultCaches()
	for _, cache := range defaultCaches {
		if err := e.manager.Registry().Register(cache); err != nil {
			e.logger.WithError(err).Error("Failed to register default cache")
			return err
		}
	}

	e.logger.Info("✓ Registered all test caches")

	// Demonstrate cache statistics
	stats := e.manager.GetStats()
	e.logger.WithFields(logrus.Fields{
		"total_caches":     stats.TotalCaches,
		"healthy_caches":   stats.HealthyCaches,
		"total_memory":     stats.TotalMemoryUsage,
		"overall_hit_rate": stats.OverallHitRate,
	}).Info("Cache manager statistics")

	// Demonstrate health check
	health := e.manager.HealthCheck()
	e.logger.WithFields(logrus.Fields{
		"overall_health":  health.OverallHealth,
		"healthy_count":   len(health.HealthyCaches),
		"unhealthy_count": len(health.UnhealthyCaches),
		"issues_count":    len(health.Issues),
	}).Info("Cache health report")

	// Demonstrate refreshing caches
	e.logger.Info("Refreshing all caches...")
	ctx := context.Background()
	if err := e.manager.WarmCaches(ctx); err != nil {
		e.logger.WithError(err).Error("Failed to warm caches")
		return err
	}
	e.logger.Info("✓ All caches refreshed")

	// Demonstrate clearing specific caches
	e.logger.Info("Clearing entity caches...")
	if err := e.manager.ClearByType(ctx, CacheTypeEntity); err != nil {
		e.logger.WithError(err).Error("Failed to clear entity caches")
		return err
	}
	e.logger.Info("✓ Entity caches cleared")

	// Demonstrate memory management
	memoryReport := e.manager.GetMemoryUsage()
	e.logger.WithFields(logrus.Fields{
		"total_memory_mb": memoryReport.TotalMemoryUsage / (1024 * 1024),
		"memory_pressure": memoryReport.MemoryPressure,
		"largest_cache":   getLargestCacheName(memoryReport),
		"recommendations": len(memoryReport.RecommendedActions),
	}).Info("Cache memory usage")

	if len(memoryReport.RecommendedActions) > 0 {
		for i, action := range memoryReport.RecommendedActions {
			e.logger.WithField("action", i+1).Info(action)
		}
	}

	e.logger.Info("=== Cache Management Demo Complete ===")
	return nil
}

// DemoIndividualCacheOperations demonstrates operations on individual caches
func (e *CacheExample) DemoIndividualCacheOperations() error {
	e.logger.Info("=== Individual Cache Operations Demo ===")

	// Get a specific cache
	caches := e.manager.Registry().List()
	if len(caches) == 0 {
		e.logger.Warn("No caches registered for demo")
		return nil
	}

	cache := caches[0]
	e.logger.WithField("cache", cache.Name()).Info("Working with cache")

	// Demonstrate cache operations
	e.logger.Info("Adding test data to cache...")
	cache.Set("demo_key_1", "demo_value_1", 5*time.Minute)
	cache.Set("demo_key_2", "demo_value_2", 5*time.Minute)
	cache.Set("demo_key_3", map[string]interface{}{
		"type":    "complex_data",
		"value":   42,
		"created": time.Now(),
	}, 10*time.Minute)

	// Test retrieval
	if value, found := cache.Get("demo_key_1"); found {
		e.logger.WithField("value", value).Info("✓ Retrieved value from cache")
	} else {
		e.logger.Warn("✗ Failed to retrieve value from cache")
	}

	// Show cache statistics
	stats := cache.Stats()
	e.logger.WithFields(logrus.Fields{
		"size":         stats.Size,
		"hit_count":    stats.HitCount,
		"miss_count":   stats.MissCount,
		"hit_rate":     stats.HitRate,
		"memory_usage": stats.MemoryUsage,
		"healthy":      stats.IsHealthy,
	}).Info("Cache statistics")

	// Demonstrate key listing
	keys := cache.Keys()
	e.logger.WithFields(logrus.Fields{
		"key_count": len(keys),
		"keys":      keys,
	}).Info("Cache keys")

	// Demonstrate cache clearing
	e.logger.Info("Clearing cache...")
	if err := cache.Clear(); err != nil {
		e.logger.WithError(err).Error("Failed to clear cache")
		return err
	}
	e.logger.WithField("size_after_clear", cache.Size()).Info("✓ Cache cleared")

	e.logger.Info("=== Individual Cache Operations Demo Complete ===")
	return nil
}

// DemoTypeBasedOperations demonstrates operations on cache types
func (e *CacheExample) DemoTypeBasedOperations() error {
	e.logger.Info("=== Type-Based Cache Operations Demo ===")

	ctx := context.Background()

	// List caches by type
	entityCaches := e.manager.Registry().ListByType(CacheTypeEntity)
	e.logger.WithField("count", len(entityCaches)).Info("Entity caches found")

	displayCaches := e.manager.Registry().ListByType(CacheTypeDisplay)
	e.logger.WithField("count", len(displayCaches)).Info("Display caches found")

	// Demonstrate type-based clearing
	if len(entityCaches) > 0 {
		e.logger.Info("Clearing all entity caches...")
		if err := e.manager.ClearByType(ctx, CacheTypeEntity); err != nil {
			e.logger.WithError(err).Error("Failed to clear entity caches")
			return err
		}
		e.logger.Info("✓ Entity caches cleared")
	}

	// Demonstrate type-based refreshing
	if len(displayCaches) > 0 {
		e.logger.Info("Refreshing all display caches...")
		if err := e.manager.RefreshByType(ctx, CacheTypeDisplay); err != nil {
			e.logger.WithError(err).Error("Failed to refresh display caches")
			return err
		}
		e.logger.Info("✓ Display caches refreshed")
	}

	e.logger.Info("=== Type-Based Cache Operations Demo Complete ===")
	return nil
}

// DemoOptimization demonstrates cache optimization features
func (e *CacheExample) DemoOptimization() error {
	e.logger.Info("=== Cache Optimization Demo ===")

	ctx := context.Background()

	// Get initial state
	initialStats := e.manager.GetStats()
	e.logger.WithField("initial_memory", initialStats.TotalMemoryUsage).Info("Initial memory usage")

	// Run optimization
	e.logger.Info("Running cache optimization...")
	if err := e.manager.OptimizeCaches(ctx); err != nil {
		e.logger.WithError(err).Error("Failed to optimize caches")
		return err
	}
	e.logger.Info("✓ Cache optimization completed")

	// Check results
	finalStats := e.manager.GetStats()
	e.logger.WithFields(logrus.Fields{
		"initial_memory": initialStats.TotalMemoryUsage,
		"final_memory":   finalStats.TotalMemoryUsage,
		"memory_saved":   initialStats.TotalMemoryUsage - finalStats.TotalMemoryUsage,
	}).Info("Optimization results")

	// Demonstrate memory freeing
	memoryReport := e.manager.GetMemoryUsage()
	if memoryReport.TotalMemoryUsage > 1024*1024 { // If using more than 1MB
		e.logger.Info("Attempting to free memory...")
		targetMB := int(memoryReport.TotalMemoryUsage / (2 * 1024 * 1024)) // Free half
		if targetMB < 1 {
			targetMB = 1
		}

		if err := e.manager.FreeMemory(ctx, targetMB); err != nil {
			e.logger.WithError(err).Error("Failed to free memory")
			return err
		}

		newReport := e.manager.GetMemoryUsage()
		e.logger.WithFields(logrus.Fields{
			"before_mb": memoryReport.TotalMemoryUsage / (1024 * 1024),
			"after_mb":  newReport.TotalMemoryUsage / (1024 * 1024),
			"freed_mb":  (memoryReport.TotalMemoryUsage - newReport.TotalMemoryUsage) / (1024 * 1024),
		}).Info("✓ Memory freed")
	}

	e.logger.Info("=== Cache Optimization Demo Complete ===")
	return nil
}

// RunFullDemo runs all cache management demonstrations
func (e *CacheExample) RunFullDemo() error {
	if err := e.DemoBasicOperations(); err != nil {
		return err
	}

	if err := e.DemoIndividualCacheOperations(); err != nil {
		return err
	}

	if err := e.DemoTypeBasedOperations(); err != nil {
		return err
	}

	if err := e.DemoOptimization(); err != nil {
		return err
	}

	return nil
}

// getLargestCacheName returns the name of the cache using the most memory
func getLargestCacheName(report CacheMemoryReport) string {
	if len(report.LargestCaches) == 0 {
		return "none"
	}
	return report.LargestCaches[0].Name
}
