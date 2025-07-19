package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/cache"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CacheHandler handles cache management API endpoints
type CacheHandler struct {
	manager cache.CacheManager
	logger  *logrus.Logger
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(manager cache.CacheManager, logger *logrus.Logger) *CacheHandler {
	return &CacheHandler{
		manager: manager,
		logger:  logger,
	}
}

// RegisterRoutes registers cache-related routes
func (h *CacheHandler) RegisterRoutes(router *gin.RouterGroup) {
	cacheGroup := router.Group("/cache")
	{
		// Core operations
		cacheGroup.POST("/clear", h.ClearCaches)
		cacheGroup.POST("/refresh", h.RefreshCaches)
		cacheGroup.GET("/status", h.GetCacheStatus)
		cacheGroup.POST("/warm", h.WarmCaches)
		cacheGroup.GET("/stats", h.GetCacheStats)
		cacheGroup.POST("/invalidate", h.InvalidateKeys)
		cacheGroup.GET("/health", h.GetCacheHealth)
		cacheGroup.POST("/optimize", h.OptimizeCaches)

		// Advanced operations
		cacheGroup.GET("/memory", h.GetMemoryUsage)
		cacheGroup.POST("/memory/free", h.FreeMemory)
		cacheGroup.GET("/list", h.ListCaches)
		cacheGroup.GET("/types", h.GetCacheTypes)

		// Type-based operations
		cacheGroup.POST("/clear/:type", h.ClearCachesByType)
		cacheGroup.POST("/refresh/:type", h.RefreshCachesByType)

		// Individual cache operations
		cacheGroup.GET("/:name/stats", h.GetIndividualCacheStats)
		cacheGroup.POST("/:name/clear", h.ClearIndividualCache)
		cacheGroup.POST("/:name/refresh", h.RefreshIndividualCache)
		cacheGroup.GET("/:name/keys", h.GetCacheKeys)
	}
}

// ClearCaches clears specified caches
func (h *CacheHandler) ClearCaches(c *gin.Context) {
	var request struct {
		Caches []string `json:"caches"`
		Force  bool     `json:"force"`
		Async  bool     `json:"async"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	// Default to clearing all caches if none specified
	if len(request.Caches) == 0 {
		caches := h.manager.Registry().List()
		for _, cache := range caches {
			request.Caches = append(request.Caches, cache.Name())
		}
	}

	ctx := context.Background()
	if timeout := c.Query("timeout"); timeout != "" {
		if timeoutSec, err := strconv.Atoi(timeout); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()
		}
	}

	start := time.Now()
	results := h.manager.ClearCaches(ctx, request.Caches)
	duration := time.Since(start)

	// Build response
	operations := make([]cache.CacheOperation, 0, len(request.Caches))
	successCount := 0
	errorCount := 0

	for _, cacheName := range request.Caches {
		op := cache.CacheOperation{
			CacheName: cacheName,
			Operation: "clear",
			Success:   results[cacheName] == nil,
			Duration:  duration / time.Duration(len(request.Caches)), // Approximation
			Timestamp: time.Now(),
		}

		if results[cacheName] != nil {
			op.Error = results[cacheName].Error()
			errorCount++
		} else {
			successCount++
			// Try to get cache stats for entries affected
			if cacheObj, exists := h.manager.Registry().Get(cacheName); exists {
				op.EntriesAffected = cacheObj.Size()
			}
		}

		operations = append(operations, op)
	}

	result := cache.CacheOperationResult{
		Operations:    operations,
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		TotalDuration: duration,
	}

	status := http.StatusOK
	if errorCount > 0 && successCount == 0 {
		status = http.StatusInternalServerError
	} else if errorCount > 0 {
		status = http.StatusPartialContent
	}

	c.JSON(status, gin.H{
		"status": "success",
		"data":   result,
	})
}

// RefreshCaches refreshes specified caches
func (h *CacheHandler) RefreshCaches(c *gin.Context) {
	var request struct {
		Caches []string `json:"caches"`
		Force  bool     `json:"force"`
		Async  bool     `json:"async"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	// Default to refreshing all caches if none specified
	if len(request.Caches) == 0 {
		caches := h.manager.Registry().List()
		for _, cache := range caches {
			request.Caches = append(request.Caches, cache.Name())
		}
	}

	ctx := context.Background()
	if timeout := c.Query("timeout"); timeout != "" {
		if timeoutSec, err := strconv.Atoi(timeout); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()
		}
	}

	start := time.Now()
	results := h.manager.RefreshCaches(ctx, request.Caches)
	duration := time.Since(start)

	// Build response
	operations := make([]cache.CacheOperation, 0, len(request.Caches))
	successCount := 0
	errorCount := 0

	for _, cacheName := range request.Caches {
		op := cache.CacheOperation{
			CacheName: cacheName,
			Operation: "refresh",
			Success:   results[cacheName] == nil,
			Duration:  duration / time.Duration(len(request.Caches)), // Approximation
			Timestamp: time.Now(),
		}

		if results[cacheName] != nil {
			op.Error = results[cacheName].Error()
			errorCount++
		} else {
			successCount++
			// Try to get cache stats for entries affected
			if cacheObj, exists := h.manager.Registry().Get(cacheName); exists {
				op.EntriesAffected = cacheObj.Size()
			}
		}

		operations = append(operations, op)
	}

	result := cache.CacheOperationResult{
		Operations:    operations,
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		TotalDuration: duration,
	}

	status := http.StatusOK
	if errorCount > 0 && successCount == 0 {
		status = http.StatusInternalServerError
	} else if errorCount > 0 {
		status = http.StatusPartialContent
	}

	c.JSON(status, gin.H{
		"status": "success",
		"data":   result,
	})
}

// GetCacheStatus returns cache status and statistics
func (h *CacheHandler) GetCacheStatus(c *gin.Context) {
	stats := h.manager.GetStats()
	health := h.manager.HealthCheck()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"statistics": stats,
			"health":     health,
			"timestamp":  time.Now(),
		},
	})
}

// WarmCaches preloads caches with fresh data
func (h *CacheHandler) WarmCaches(c *gin.Context) {
	var request struct {
		Caches []string `json:"caches"`
		Async  bool     `json:"async"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		// Allow empty body - warm all caches
		request.Caches = []string{}
	}

	ctx := context.Background()
	start := time.Now()

	var err error
	if len(request.Caches) > 0 {
		results := h.manager.RefreshCaches(ctx, request.Caches)
		// Check if any failed
		for _, e := range results {
			if e != nil {
				err = e
				break
			}
		}
	} else {
		err = h.manager.WarmCaches(ctx)
	}

	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"message":   "Cache warming completed",
			"duration":  duration.String(),
			"warmed_at": time.Now(),
		},
	})
}

// GetCacheStats returns detailed cache statistics
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	cacheType := c.Query("type")
	cacheName := c.Query("cache")

	if cacheName != "" {
		// Get stats for specific cache
		stats, err := h.manager.GetCacheStats(cacheName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   stats,
		})
		return
	}

	// Get stats for all caches or by type
	allStats := h.manager.Registry().StatsAll()

	if cacheType != "" {
		filteredStats := make([]cache.CacheStats, 0)
		for _, stats := range allStats {
			if string(stats.Type) == cacheType {
				filteredStats = append(filteredStats, stats)
			}
		}
		allStats = filteredStats
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   allStats,
	})
}

// InvalidateKeys invalidates specific cache keys
func (h *CacheHandler) InvalidateKeys(c *gin.Context) {
	var request struct {
		CacheName string   `json:"cache_name"`
		Keys      []string `json:"keys"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	cacheObj, exists := h.manager.Registry().Get(request.CacheName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  "Cache not found",
		})
		return
	}

	results := make(map[string]string)
	successCount := 0

	for _, key := range request.Keys {
		if err := cacheObj.Delete(key); err != nil {
			results[key] = err.Error()
		} else {
			results[key] = "deleted"
			successCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"cache_name":    request.CacheName,
			"results":       results,
			"success_count": successCount,
			"error_count":   len(request.Keys) - successCount,
		},
	})
}

// GetCacheHealth returns cache health monitoring information
func (h *CacheHandler) GetCacheHealth(c *gin.Context) {
	health := h.manager.HealthCheck()

	status := http.StatusOK
	if !health.OverallHealth {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": "success",
		"data":   health,
	})
}

// OptimizeCaches performs cache optimization operations
func (h *CacheHandler) OptimizeCaches(c *gin.Context) {
	ctx := context.Background()
	start := time.Now()

	err := h.manager.OptimizeCaches(ctx)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"message":      "Cache optimization completed",
			"duration":     duration.String(),
			"optimized_at": time.Now(),
		},
	})
}

// GetMemoryUsage returns cache memory usage information
func (h *CacheHandler) GetMemoryUsage(c *gin.Context) {
	report := h.manager.GetMemoryUsage()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   report,
	})
}

// FreeMemory attempts to free cache memory to reach target
func (h *CacheHandler) FreeMemory(c *gin.Context) {
	var request struct {
		TargetMB int `json:"target_mb"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}

	if request.TargetMB <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Target MB must be positive",
		})
		return
	}

	ctx := context.Background()
	start := time.Now()
	beforeReport := h.manager.GetMemoryUsage()

	err := h.manager.FreeMemory(ctx, request.TargetMB)
	duration := time.Since(start)
	afterReport := h.manager.GetMemoryUsage()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	freedMemory := beforeReport.TotalMemoryUsage - afterReport.TotalMemoryUsage

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"target_mb":          request.TargetMB,
			"freed_memory_bytes": freedMemory,
			"freed_memory_mb":    freedMemory / (1024 * 1024),
			"before_usage":       beforeReport.TotalMemoryUsage,
			"after_usage":        afterReport.TotalMemoryUsage,
			"duration":           duration.String(),
			"completed_at":       time.Now(),
		},
	})
}

// ListCaches returns a list of all registered caches
func (h *CacheHandler) ListCaches(c *gin.Context) {
	caches := h.manager.Registry().List()
	cacheType := c.Query("type")

	cacheList := make([]gin.H, 0, len(caches))
	for _, cacheObj := range caches {
		if cacheType != "" && string(cacheObj.Type()) != cacheType {
			continue
		}

		stats := cacheObj.Stats()
		cacheList = append(cacheList, gin.H{
			"name":         cacheObj.Name(),
			"type":         cacheObj.Type(),
			"size":         cacheObj.Size(),
			"healthy":      cacheObj.Healthy(),
			"memory_usage": stats.MemoryUsage,
			"hit_rate":     stats.HitRate,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"caches": cacheList,
			"count":  len(cacheList),
		},
	})
}

// GetCacheTypes returns available cache types
func (h *CacheHandler) GetCacheTypes(c *gin.Context) {
	caches := h.manager.Registry().List()
	typeCount := make(map[cache.CacheType]int)

	for _, cacheObj := range caches {
		typeCount[cacheObj.Type()]++
	}

	types := make([]gin.H, 0, len(typeCount))
	for cacheType, count := range typeCount {
		types = append(types, gin.H{
			"type":  cacheType,
			"count": count,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"types": types,
		},
	})
}

// ClearCachesByType clears all caches of a specific type
func (h *CacheHandler) ClearCachesByType(c *gin.Context) {
	cacheType := cache.CacheType(c.Param("type"))

	ctx := context.Background()
	start := time.Now()

	err := h.manager.ClearByType(ctx, cacheType)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	caches := h.manager.Registry().ListByType(cacheType)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"type":       cacheType,
			"cleared":    len(caches),
			"duration":   duration.String(),
			"cleared_at": time.Now(),
		},
	})
}

// RefreshCachesByType refreshes all caches of a specific type
func (h *CacheHandler) RefreshCachesByType(c *gin.Context) {
	cacheType := cache.CacheType(c.Param("type"))

	ctx := context.Background()
	start := time.Now()

	err := h.manager.RefreshByType(ctx, cacheType)
	duration := time.Since(start)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	caches := h.manager.Registry().ListByType(cacheType)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"type":         cacheType,
			"refreshed":    len(caches),
			"duration":     duration.String(),
			"refreshed_at": time.Now(),
		},
	})
}

// GetIndividualCacheStats returns statistics for a specific cache
func (h *CacheHandler) GetIndividualCacheStats(c *gin.Context) {
	cacheName := c.Param("name")

	stats, err := h.manager.GetCacheStats(cacheName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// ClearIndividualCache clears a specific cache
func (h *CacheHandler) ClearIndividualCache(c *gin.Context) {
	cacheName := c.Param("name")

	ctx := context.Background()
	results := h.manager.ClearCaches(ctx, []string{cacheName})

	if err, exists := results[cacheName]; exists && err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"cache":      cacheName,
			"cleared":    true,
			"cleared_at": time.Now(),
		},
	})
}

// RefreshIndividualCache refreshes a specific cache
func (h *CacheHandler) RefreshIndividualCache(c *gin.Context) {
	cacheName := c.Param("name")

	ctx := context.Background()
	results := h.manager.RefreshCaches(ctx, []string{cacheName})

	if err, exists := results[cacheName]; exists && err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"cache":        cacheName,
			"refreshed":    true,
			"refreshed_at": time.Now(),
		},
	})
}

// GetCacheKeys returns all keys in a specific cache
func (h *CacheHandler) GetCacheKeys(c *gin.Context) {
	cacheName := c.Param("name")

	cacheObj, exists := h.manager.Registry().Get(cacheName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  "Cache not found",
		})
		return
	}

	keys := cacheObj.Keys()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"cache": cacheName,
			"keys":  keys,
			"count": len(keys),
		},
	})
}
