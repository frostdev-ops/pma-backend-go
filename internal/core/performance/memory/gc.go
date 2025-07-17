package memory

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
)

// GCTuner manages garbage collection optimization
type GCTuner struct {
	currentGCPercent    int
	targets             *GCTargets
	mu                  sync.RWMutex
	optimizationHistory []GCOptimization
}

// GCOptimization represents a GC optimization action
type GCOptimization struct {
	Timestamp  int64  `json:"timestamp"`
	Action     string `json:"action"`
	OldPercent int    `json:"old_percent"`
	NewPercent int    `json:"new_percent"`
	Reason     string `json:"reason"`
}

// NewGCTuner creates a new GC tuner
func NewGCTuner() *GCTuner {
	// Get current GOGC value
	currentGC := 100 // Default value
	if gcEnv := os.Getenv("GOGC"); gcEnv != "" {
		if parsed, err := strconv.Atoi(gcEnv); err == nil {
			currentGC = parsed
		}
	}

	return &GCTuner{
		currentGCPercent: currentGC,
		targets: &GCTargets{
			TargetGCPercent: 100,
			MaxHeapSize:     500 * 1024 * 1024, // 500MB default
		},
		optimizationHistory: make([]GCOptimization, 0),
	}
}

// SetTargets sets GC optimization targets
func (gt *GCTuner) SetTargets(targets GCTargets) error {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	gt.targets = &targets

	// Apply the target GC percent immediately
	if targets.TargetGCPercent > 0 {
		return gt.setGCPercent(targets.TargetGCPercent, "target_set")
	}

	return nil
}

// OptimizeSettings optimizes GC settings based on current memory stats
func (gt *GCTuner) OptimizeSettings(stats *MemoryStats) error {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	if gt.targets == nil {
		return fmt.Errorf("no GC targets set")
	}

	// Determine optimal GC percent based on current conditions
	optimalPercent := gt.calculateOptimalGCPercent(stats)

	if optimalPercent != gt.currentGCPercent {
		reason := gt.generateOptimizationReason(stats, optimalPercent)
		return gt.setGCPercent(optimalPercent, reason)
	}

	return nil
}

// calculateOptimalGCPercent calculates the optimal GOGC value
func (gt *GCTuner) calculateOptimalGCPercent(stats *MemoryStats) int {
	basePercent := gt.targets.TargetGCPercent

	// Adjust based on heap utilization
	if stats.HeapUtilization > 80 {
		// High heap utilization - more aggressive GC
		basePercent = max(basePercent-20, 50)
	} else if stats.HeapUtilization < 30 {
		// Low heap utilization - less aggressive GC
		basePercent = min(basePercent+30, 200)
	}

	// Adjust based on allocation rate
	if stats.AllocRate > 50*1024*1024 { // 50MB/s
		// High allocation rate - more frequent GC
		basePercent = max(basePercent-10, 50)
	} else if stats.AllocRate < 10*1024*1024 { // 10MB/s
		// Low allocation rate - less frequent GC
		basePercent = min(basePercent+20, 200)
	}

	// Adjust based on heap size relative to target
	if gt.targets.MaxHeapSize > 0 {
		heapRatio := float64(stats.HeapInUse) / float64(gt.targets.MaxHeapSize)
		if heapRatio > 0.8 {
			// Approaching heap limit - aggressive GC
			basePercent = max(basePercent-30, 25)
		} else if heapRatio < 0.3 {
			// Well below heap limit - relaxed GC
			basePercent = min(basePercent+40, 300)
		}
	}

	return basePercent
}

// generateOptimizationReason generates a human-readable reason for the optimization
func (gt *GCTuner) generateOptimizationReason(stats *MemoryStats, newPercent int) string {
	if newPercent < gt.currentGCPercent {
		if stats.HeapUtilization > 80 {
			return "high_heap_utilization"
		}
		if stats.AllocRate > 50*1024*1024 {
			return "high_allocation_rate"
		}
		if gt.targets.MaxHeapSize > 0 && float64(stats.HeapInUse)/float64(gt.targets.MaxHeapSize) > 0.8 {
			return "approaching_heap_limit"
		}
		return "memory_pressure"
	} else {
		if stats.HeapUtilization < 30 {
			return "low_heap_utilization"
		}
		if stats.AllocRate < 10*1024*1024 {
			return "low_allocation_rate"
		}
		return "memory_available"
	}
}

// setGCPercent sets the GOGC environment variable and debug setting
func (gt *GCTuner) setGCPercent(percent int, reason string) error {
	oldPercent := gt.currentGCPercent

	// Set the GOGC environment variable
	os.Setenv("GOGC", strconv.Itoa(percent))

	// Set the debug GC percent
	debug.SetGCPercent(percent)

	gt.currentGCPercent = percent

	// Record the optimization
	gt.optimizationHistory = append(gt.optimizationHistory, GCOptimization{
		Timestamp:  int64(len(gt.optimizationHistory)), // Simplified timestamp
		Action:     "gc_percent_change",
		OldPercent: oldPercent,
		NewPercent: percent,
		Reason:     reason,
	})

	// Limit history size
	if len(gt.optimizationHistory) > 100 {
		gt.optimizationHistory = gt.optimizationHistory[1:]
	}

	return nil
}

// ReduceGCTarget reduces GC target for more aggressive collection
func (gt *GCTuner) ReduceGCTarget() {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	newPercent := max(gt.currentGCPercent-10, 25)
	gt.setGCPercent(newPercent, "manual_reduction")
}

// IncreaseGCTarget increases GC target for less aggressive collection
func (gt *GCTuner) IncreaseGCTarget() {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	newPercent := min(gt.currentGCPercent+10, 200)
	gt.setGCPercent(newPercent, "manual_increase")
}

// GetCurrentPercent returns the current GOGC percent
func (gt *GCTuner) GetCurrentPercent() int {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	return gt.currentGCPercent
}

// GetOptimizationHistory returns the history of GC optimizations
func (gt *GCTuner) GetOptimizationHistory() []GCOptimization {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	// Return a copy to avoid race conditions
	history := make([]GCOptimization, len(gt.optimizationHistory))
	copy(history, gt.optimizationHistory)

	return history
}

// ResetToDefault resets GC settings to default values
func (gt *GCTuner) ResetToDefault() error {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	return gt.setGCPercent(100, "reset_to_default")
}

// GetGCReport generates a comprehensive GC tuning report
func (gt *GCTuner) GetGCReport() *GCReport {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	return &GCReport{
		CurrentPercent:      gt.currentGCPercent,
		Targets:             *gt.targets,
		OptimizationHistory: gt.GetOptimizationHistory(),
		Recommendations:     gt.generateGCRecommendations(),
	}
}

// GCReport contains comprehensive GC tuning information
type GCReport struct {
	CurrentPercent      int              `json:"current_percent"`
	Targets             GCTargets        `json:"targets"`
	OptimizationHistory []GCOptimization `json:"optimization_history"`
	Recommendations     []string         `json:"recommendations"`
}

// generateGCRecommendations generates GC tuning recommendations
func (gt *GCTuner) generateGCRecommendations() []string {
	recommendations := []string{}

	if gt.currentGCPercent < 50 {
		recommendations = append(recommendations, "Very aggressive GC settings - monitor for high CPU usage")
	} else if gt.currentGCPercent > 150 {
		recommendations = append(recommendations, "Relaxed GC settings - monitor heap growth")
	}

	if len(gt.optimizationHistory) > 10 {
		// Check for frequent changes
		recentChanges := 0
		for i := len(gt.optimizationHistory) - 10; i < len(gt.optimizationHistory); i++ {
			if gt.optimizationHistory[i].Action == "gc_percent_change" {
				recentChanges++
			}
		}

		if recentChanges > 5 {
			recommendations = append(recommendations, "Frequent GC adjustments detected - consider manual tuning")
		}
	}

	return recommendations
}

// Note: Helper functions min/max are defined in manager.go
