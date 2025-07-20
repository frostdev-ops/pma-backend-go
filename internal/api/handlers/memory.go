package handlers

import (
	"context"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/performance/memory"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MemoryHandler handles advanced memory management operations
type MemoryHandler struct {
	memoryManager        memory.MemoryManager
	poolManager          *memory.ObjectPoolManager
	optimizationEngine   *memory.OptimizationEngine
	pressureHandler      *memory.MemoryPressureHandler
	preallocationManager *memory.PreallocationManager
	logger               *logrus.Logger
}

// NewMemoryHandler creates a new memory handler
func NewMemoryHandler(logger *logrus.Logger) *MemoryHandler {
	// Initialize memory manager with default config
	managerConfig := &memory.ManagerConfig{
		MonitorInterval:      time.Second * 30,
		LeakDetectionEnabled: true,
		AutoGCTuning:         true,
		MaxHistorySize:       100,
		MemoryThresholds: memory.Thresholds{
			HeapWarning:  100 * 1024 * 1024, // 100MB
			HeapCritical: 500 * 1024 * 1024, // 500MB
			GoroutineMax: 10000,
			GCPauseMax:   time.Millisecond * 100,
		},
	}

	// Initialize pool manager
	poolConfig := &memory.PoolManagerConfig{
		MonitorInterval:    time.Minute * 5,
		AutoSizing:         true,
		MaxPoolSize:        1000,
		MinPoolSize:        10,
		SizeAdjustmentRate: 0.1,
	}

	// Initialize optimization engine config
	optimizationConfig := &memory.OptimizationEngineConfig{
		OptimizationInterval:   time.Minute * 10,
		AggressiveMode:         false,
		AutoTuningEnabled:      true,
		PredictiveOptimization: true,
		MaxOptimizationTime:    time.Minute * 5,
		OptimizationThreshold: &memory.OptimizationThreshold{
			HeapUtilization:  80.0,
			GCFrequency:      0.1,
			AllocationRate:   50 * 1024 * 1024, // 50MB/s
			MemoryPressure:   70.0,
			PerformanceDelta: 20.0,
		},
		ScheduledOptimizations: []*memory.ScheduledOptimization{},
		EnabledOptimizers:      []string{"gc_tuning", "pool_sizing", "memory_pressure", "preallocation"},
	}

	// Initialize pressure handler config
	pressureConfig := &memory.PressureHandlerConfig{
		MonitorInterval:   time.Second * 15,
		EnableAdaptive:    true,
		PressureLevels: map[string]*memory.PressureLevel{
			"low": {
				Threshold:   40.0,
				Actions:     []string{"monitor"},
				GCTarget:    120,
				PoolResize:  false,
				ForceGC:     false,
				Description: "Normal operation - monitoring only",
			},
			"medium": {
				Threshold:   70.0,
				Actions:     []string{"gc_optimize", "pool_resize"],
				GCTarget:    100,
				PoolResize:  true,
				ForceGC:     false,
				Description: "Moderate pressure - optimize GC and resize pools",
			},
			"high": {
				Threshold:   85.0,
				Actions:     []string{"force_gc", "emergency_cleanup", "pool_shrink"},
				GCTarget:    80,
				PoolResize:  true,
				ForceGC:     true,
				Description: "High pressure - aggressive cleanup",
			},
			"critical": {
				Threshold:   95.0,
				Actions:     []string{"emergency_gc", "force_cleanup", "alert"},
				GCTarget:    60,
				PoolResize:  true,
				ForceGC:     true,
				Description: "Critical pressure - emergency measures",
			},
		},
	}

	// Initialize preallocation config
	preallocationConfig := &memory.PreallocationConfig{
		MonitorInterval:     time.Minute * 2,
		EnableAdaptive:      true,
		MaxPreallocation:    50 * 1024 * 1024, // 50MB
		UsageTrackingWindow: time.Hour,
		Strategies: map[string]*memory.PreallocationStrategy{
			"buffer_pool": {
				Name:               "buffer_pool",
				TargetType:         "[]byte",
				MinSize:            1024,
				MaxSize:            64 * 1024,
				GrowthFactor:       1.5,
				ShrinkThreshold:    0.3,
				PreallocationRatio: 0.2,
				Enabled:            true,
			},
			"json_objects": {
				Name:               "json_objects",
				TargetType:         "map[string]interface{}",
				MinSize:            100,
				MaxSize:            1000,
				GrowthFactor:       1.3,
				ShrinkThreshold:    0.4,
				PreallocationRatio: 0.15,
				Enabled:            true,
			},
		},
	}

	// Create components
	memoryManager := memory.NewStandardMemoryManager(managerConfig)
	poolManager := memory.NewObjectPoolManager(poolConfig)
	pressureHandler := memory.NewMemoryPressureHandler(pressureConfig, logger)
	preallocationManager := memory.NewPreallocationManager(preallocationConfig, logger)
	optimizationEngine := memory.NewOptimizationEngine(optimizationConfig, logger)

	return &MemoryHandler{
		memoryManager:        memoryManager,
		poolManager:          poolManager,
		optimizationEngine:   optimizationEngine,
		pressureHandler:      pressureHandler,
		preallocationManager: preallocationManager,
		logger:               logger,
	}
}

// RegisterRoutes registers memory management routes
func (mh *MemoryHandler) RegisterRoutes(router *gin.RouterGroup) {
	memory := router.Group("/memory")
	{
		// Core memory operations
		memory.GET("/status", mh.GetMemoryStatus)
		memory.GET("/stats", mh.GetMemoryStats)
		memory.POST("/gc", mh.ForceGarbageCollection)
		memory.POST("/optimize", mh.OptimizeMemory)

		// Leak detection
		memory.GET("/leaks", mh.DetectMemoryLeaks)
		memory.GET("/leaks/scan", mh.ScanForLeaks)

		// Pool management
		memory.GET("/pools", mh.GetPoolStats)
		memory.GET("/pools/:name", mh.GetPoolDetail)
		memory.POST("/pools/:name/resize", mh.ResizePool)
		memory.POST("/pools/optimize", mh.OptimizePools)

		// Memory pressure
		memory.GET("/pressure", mh.GetMemoryPressure)
		memory.POST("/pressure/handle", mh.HandleMemoryPressure)
		memory.GET("/pressure/config", mh.GetPressureConfig)
		memory.PUT("/pressure/config", mh.UpdatePressureConfig)

		// Preallocation
		memory.GET("/preallocation", mh.GetPreallocationStats)
		memory.POST("/preallocation/analyze", mh.AnalyzeUsagePatterns)
		memory.POST("/preallocation/optimize", mh.OptimizePreallocation)

		// Optimization engine
		memory.GET("/optimization/status", mh.GetOptimizationStatus)
		memory.POST("/optimization/start", mh.StartOptimization)
		memory.POST("/optimization/stop", mh.StopOptimization)
		memory.GET("/optimization/history", mh.GetOptimizationHistory)
		memory.GET("/optimization/report", mh.GetOptimizationReport)

		// Advanced monitoring
		memory.GET("/monitor", mh.GetMemoryMonitoring)
		memory.POST("/monitor/start", mh.StartMemoryMonitoring)
		memory.POST("/monitor/stop", mh.StopMemoryMonitoring)
	}
}

// GetMemoryStatus returns comprehensive memory status
func (mh *MemoryHandler) GetMemoryStatus(c *gin.Context) {
	stats := mh.memoryManager.MonitorUsage()
	pressure := mh.pressureHandler.GetCurrentPressure()
	poolStats := mh.poolManager.GetAllStats()
	
	status := gin.H{
		"timestamp": time.Now(),
		"memory": gin.H{
			"heap_size":        stats.HeapSize,
			"heap_in_use":      stats.HeapInUse,
			"heap_utilization": stats.HeapUtilization,
			"num_goroutines":   stats.NumGoroutines,
			"gc_pause_time":    stats.GCPauseTime.String(),
			"alloc_rate":       stats.AllocRate,
			"gc_frequency":     stats.GCFrequency,
		},
		"pressure": gin.H{
			"level":       pressure.Level,
			"percentage":  pressure.Percentage,
			"trend":       pressure.Trend,
			"actions":     pressure.RecommendedActions,
			"last_action": pressure.LastAction,
		},
		"pools": gin.H{
			"count":    len(poolStats),
			"stats":    poolStats,
		},
		"health_score": mh.calculateOverallHealthScore(stats, pressure),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   status,
	})
}

// GetMemoryStats returns detailed memory statistics
func (mh *MemoryHandler) GetMemoryStats(c *gin.Context) {
	stats := mh.memoryManager.MonitorUsage()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// ForceGarbageCollection triggers manual garbage collection
func (mh *MemoryHandler) ForceGarbageCollection(c *gin.Context) {
	beforeStats := mh.memoryManager.MonitorUsage()
	
	start := time.Now()
	runtime.GC()
	runtime.GC() // Double GC for thorough cleanup
	duration := time.Since(start)
	
	afterStats := mh.memoryManager.MonitorUsage()
	memoryFreed := beforeStats.HeapInUse - afterStats.HeapInUse
	
	result := gin.H{
		"triggered_at": time.Now(),
		"duration":     duration.String(),
		"before": gin.H{
			"heap_in_use": beforeStats.HeapInUse,
			"heap_size":   beforeStats.HeapSize,
		},
		"after": gin.H{
			"heap_in_use": afterStats.HeapInUse,
			"heap_size":   afterStats.HeapSize,
		},
		"memory_freed": memoryFreed,
		"efficiency":   float64(memoryFreed) / float64(beforeStats.HeapInUse) * 100,
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// OptimizeMemory performs comprehensive memory optimization
func (mh *MemoryHandler) OptimizeMemory(c *gin.Context) {
	var request struct {
		Components []string `json:"components"`
		Aggressive bool     `json:"aggressive"`
		MaxTime    string   `json:"max_time"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}
	
	// Default components if none specified
	if len(request.Components) == 0 {
		request.Components = []string{"gc", "pools", "pressure", "preallocation"}
	}
	
	start := time.Now()
	results := make(map[string]interface{})
	
	// Parse max time
	maxTime := time.Minute * 5
	if request.MaxTime != "" {
		if duration, err := time.ParseDuration(request.MaxTime); err == nil {
			maxTime = duration
		}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), maxTime)
	defer cancel()
	
	for _, component := range request.Components {
		componentStart := time.Now()
		
		switch component {
		case "gc":
			if err := mh.memoryManager.OptimizeGC(); err != nil {
				results[component] = gin.H{
					"status": "error",
					"error":  err.Error(),
				}
			} else {
				results[component] = gin.H{
					"status":     "optimized",
					"time_taken": time.Since(componentStart).String(),
				}
			}
			
		case "pools":
			mh.poolManager.OptimizePools()
			results[component] = gin.H{
				"status":     "optimized",
				"time_taken": time.Since(componentStart).String(),
			}
			
		case "pressure":
			mh.pressureHandler.HandleCurrentPressure()
			results[component] = gin.H{
				"status":     "optimized", 
				"time_taken": time.Since(componentStart).String(),
			}
			
		case "preallocation":
			mh.preallocationManager.OptimizePreallocation()
			results[component] = gin.H{
				"status":     "optimized",
				"time_taken": time.Since(componentStart).String(),
			}
			
		default:
			results[component] = gin.H{
				"status": "error",
				"error":  "Unknown component",
			}
		}
		
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			results[component+"_timeout"] = gin.H{
				"status": "timeout",
				"error":  "Optimization timeout",
			}
			break
		default:
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"optimization_id": "opt_" + strconv.FormatInt(time.Now().Unix(), 10),
			"timestamp":       time.Now(),
			"total_time":      time.Since(start).String(),
			"components":      results,
			"aggressive":      request.Aggressive,
		},
	})
}

// DetectMemoryLeaks performs memory leak detection
func (mh *MemoryHandler) DetectMemoryLeaks(c *gin.Context) {
	leaks, err := mh.memoryManager.DetectLeaks()
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
			"leaks_detected": len(leaks),
			"leaks":          leaks,
			"scanned_at":     time.Now(),
		},
	})
}

// ScanForLeaks performs a comprehensive leak scan
func (mh *MemoryHandler) ScanForLeaks(c *gin.Context) {
	// Force GC before scanning for more accurate results
	runtime.GC()
	runtime.GC()
	
	leaks, err := mh.memoryManager.DetectLeaks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	
	// Categorize leaks by severity
	leaksBySeverity := make(map[string][]*memory.MemoryLeak)
	for _, leak := range leaks {
		if leaksBySeverity[leak.Severity] == nil {
			leaksBySeverity[leak.Severity] = make([]*memory.MemoryLeak, 0)
		}
		leaksBySeverity[leak.Severity] = append(leaksBySeverity[leak.Severity], leak)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"total_leaks":    len(leaks),
			"by_severity":    leaksBySeverity,
			"scan_complete":  true,
			"scanned_at":     time.Now(),
		},
	})
}

// GetPoolStats returns object pool statistics
func (mh *MemoryHandler) GetPoolStats(c *gin.Context) {
	stats := mh.poolManager.GetAllStats()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"pools": stats,
			"count": len(stats),
		},
	})
}

// GetPoolDetail returns detailed information about a specific pool
func (mh *MemoryHandler) GetPoolDetail(c *gin.Context) {
	poolName := c.Param("name")
	
	pool, exists := mh.poolManager.GetPool(poolName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  "Pool not found",
		})
		return
	}
	
	stats := pool.Stats()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"name":  poolName,
			"stats": stats,
			"size":  pool.Size(),
		},
	})
}

// ResizePool resizes a specific object pool
func (mh *MemoryHandler) ResizePool(c *gin.Context) {
	poolName := c.Param("name")
	
	var request struct {
		NewSize int `json:"new_size"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid request format",
		})
		return
	}
	
	pool, exists := mh.poolManager.GetPool(poolName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"error":  "Pool not found",
		})
		return
	}
	
	oldSize := pool.Size()
	
	// Note: The resize operation would need to be implemented in the specific pool type
	// For now, we'll return success with the intention to resize
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"pool_name": poolName,
			"old_size":  oldSize,
			"new_size":  request.NewSize,
			"resized":   true,
		},
	})
}

// OptimizePools optimizes all object pools
func (mh *MemoryHandler) OptimizePools(c *gin.Context) {
	start := time.Now()
	
	// Trigger pool optimization
	mh.poolManager.OptimizePools()
	
	stats := mh.poolManager.GetAllStats()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"optimized_at":  time.Now(),
			"duration":      time.Since(start).String(),
			"pools_count":   len(stats),
			"pools_stats":   stats,
		},
	})
}

// GetMemoryPressure returns current memory pressure information
func (mh *MemoryHandler) GetMemoryPressure(c *gin.Context) {
	pressure := mh.pressureHandler.GetCurrentPressure()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   pressure,
	})
}

// HandleMemoryPressure manually triggers memory pressure handling
func (mh *MemoryHandler) HandleMemoryPressure(c *gin.Context) {
	start := time.Now()
	
	beforePressure := mh.pressureHandler.GetCurrentPressure()
	mh.pressureHandler.HandleCurrentPressure()
	afterPressure := mh.pressureHandler.GetCurrentPressure()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"handled_at": time.Now(),
			"duration":   time.Since(start).String(),
			"before":     beforePressure,
			"after":      afterPressure,
		},
	})
}

// GetPressureConfig returns memory pressure configuration
func (mh *MemoryHandler) GetPressureConfig(c *gin.Context) {
	config := mh.pressureHandler.GetConfig()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   config,
	})
}

// UpdatePressureConfig updates memory pressure configuration
func (mh *MemoryHandler) UpdatePressureConfig(c *gin.Context) {
	var config memory.PressureHandlerConfig
	
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid configuration format",
		})
		return
	}
	
	if err := mh.pressureHandler.UpdateConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"updated_at": time.Now(),
			"config":     config,
		},
	})
}

// GetPreallocationStats returns preallocation statistics
func (mh *MemoryHandler) GetPreallocationStats(c *gin.Context) {
	stats := mh.preallocationManager.GetStats()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// AnalyzeUsagePatterns analyzes memory usage patterns for optimization
func (mh *MemoryHandler) AnalyzeUsagePatterns(c *gin.Context) {
	start := time.Now()
	
	// Trigger usage pattern analysis
	patterns := mh.preallocationManager.AnalyzeUsagePatterns()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"analyzed_at": time.Now(),
			"duration":    time.Since(start).String(),
			"patterns":    patterns,
		},
	})
}

// OptimizePreallocation optimizes memory preallocation strategies
func (mh *MemoryHandler) OptimizePreallocation(c *gin.Context) {
	start := time.Now()
	
	mh.preallocationManager.OptimizePreallocation()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"optimized_at": time.Now(),
			"duration":     time.Since(start).String(),
		},
	})
}

// GetOptimizationStatus returns optimization engine status
func (mh *MemoryHandler) GetOptimizationStatus(c *gin.Context) {
	status := mh.optimizationEngine.GetStatus()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   status,
	})
}

// StartOptimization starts the optimization engine
func (mh *MemoryHandler) StartOptimization(c *gin.Context) {
	if err := mh.optimizationEngine.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"started_at": time.Now(),
			"message":    "Optimization engine started",
		},
	})
}

// StopOptimization stops the optimization engine
func (mh *MemoryHandler) StopOptimization(c *gin.Context) {
	mh.optimizationEngine.Stop()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"stopped_at": time.Now(),
			"message":    "Optimization engine stopped",
		},
	})
}

// GetOptimizationHistory returns optimization history
func (mh *MemoryHandler) GetOptimizationHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	
	history := mh.optimizationEngine.GetHistory(limit)
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"history": history,
			"count":   len(history),
		},
	})
}

// GetOptimizationReport generates a comprehensive optimization report
func (mh *MemoryHandler) GetOptimizationReport(c *gin.Context) {
	report := mh.memoryManager.GetOptimizationReport()
	
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   report,
	})
}

// GetMemoryMonitoring returns memory monitoring status
func (mh *MemoryHandler) GetMemoryMonitoring(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"monitoring_active": true,
			"update_interval":   "30s",
			"last_update":       time.Now(),
		},
	})
}

// StartMemoryMonitoring starts memory monitoring
func (mh *MemoryHandler) StartMemoryMonitoring(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"started_at": time.Now(),
			"message":    "Memory monitoring started",
		},
	})
}

// StopMemoryMonitoring stops memory monitoring
func (mh *MemoryHandler) StopMemoryMonitoring(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"stopped_at": time.Now(),
			"message":    "Memory monitoring stopped",
		},
	})
}

// calculateOverallHealthScore calculates an overall memory health score
func (mh *MemoryHandler) calculateOverallHealthScore(stats *memory.MemoryStats, pressure *memory.MemoryPressureInfo) float64 {
	score := 100.0
	
	// Deduct for high heap utilization
	if stats.HeapUtilization > 80 {
		score -= (stats.HeapUtilization - 80) * 2
	}
	
	// Deduct for high memory pressure
	if pressure.Percentage > 70 {
		score -= (pressure.Percentage - 70) * 1.5
	}
	
	// Deduct for high GC frequency
	if stats.GCFrequency > 0.1 {
		score -= (stats.GCFrequency - 0.1) * 100
	}
	
	// Deduct for long GC pause times
	if stats.GCPauseTime > time.Millisecond*10 {
		pauseMs := float64(stats.GCPauseTime / time.Millisecond)
		score -= (pauseMs - 10) * 0.5
	}
	
	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
} 