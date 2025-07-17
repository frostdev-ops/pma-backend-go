package memory

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// MemoryManager defines the interface for memory management and optimization
type MemoryManager interface {
	MonitorUsage() *MemoryStats
	OptimizeGC() error
	DetectLeaks() ([]*MemoryLeak, error)
	SetGCTargets(targets GCTargets) error
	GetOptimizationReport() *OptimizationReport
}

// MemoryStats contains detailed memory usage statistics
type MemoryStats struct {
	HeapSize        uint64        `json:"heap_size"`
	HeapInUse       uint64        `json:"heap_in_use"`
	HeapIdle        uint64        `json:"heap_idle"`
	HeapReleased    uint64        `json:"heap_released"`
	StackInUse      uint64        `json:"stack_in_use"`
	StackSys        uint64        `json:"stack_sys"`
	MSpanInUse      uint64        `json:"mspan_in_use"`
	MSpanSys        uint64        `json:"mspan_sys"`
	MCacheInUse     uint64        `json:"mcache_in_use"`
	MCacheSys       uint64        `json:"mcache_sys"`
	BuckHashSys     uint64        `json:"buck_hash_sys"`
	GCSys           uint64        `json:"gc_sys"`
	OtherSys        uint64        `json:"other_sys"`
	NextGC          uint64        `json:"next_gc"`
	LastGC          time.Time     `json:"last_gc"`
	GCPauseTime     time.Duration `json:"gc_pause_time"`
	NumGoroutines   int           `json:"num_goroutines"`
	NumGC           uint32        `json:"num_gc"`
	AllocRate       float64       `json:"alloc_rate"`       // bytes per second
	GCFrequency     float64       `json:"gc_frequency"`     // GCs per second
	HeapUtilization float64       `json:"heap_utilization"` // percentage
}

// MemoryLeak represents a detected memory leak
type MemoryLeak struct {
	Type        string    `json:"type"`
	Size        uint64    `json:"size"`
	Location    string    `json:"location"`
	StackTrace  string    `json:"stack_trace"`
	DetectedAt  time.Time `json:"detected_at"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	Suggestions []string  `json:"suggestions"`
}

// GCTargets contains garbage collection optimization targets
type GCTargets struct {
	MaxHeapSize     uint64        `json:"max_heap_size"`
	TargetGCPercent int           `json:"target_gc_percent"`
	MaxGCPause      time.Duration `json:"max_gc_pause"`
	MemoryLimit     uint64        `json:"memory_limit"`
}

// OptimizationReport contains comprehensive memory optimization analysis
type OptimizationReport struct {
	GeneratedAt     time.Time     `json:"generated_at"`
	CurrentStats    MemoryStats   `json:"current_stats"`
	DetectedLeaks   []*MemoryLeak `json:"detected_leaks"`
	GCEfficiency    GCEfficiency  `json:"gc_efficiency"`
	Recommendations []string      `json:"recommendations"`
	HealthScore     float64       `json:"health_score"` // 0-100
}

// GCEfficiency contains garbage collection efficiency metrics
type GCEfficiency struct {
	AveragePauseTime time.Duration `json:"average_pause_time"`
	MaxPauseTime     time.Duration `json:"max_pause_time"`
	GCOverhead       float64       `json:"gc_overhead"`      // percentage of CPU time
	CollectionRate   float64       `json:"collection_rate"`  // MB/s
	EfficiencyScore  float64       `json:"efficiency_score"` // 0-100
}

// StandardMemoryManager implements MemoryManager
type StandardMemoryManager struct {
	config           *ManagerConfig
	stats            *MemoryStats
	gcStats          runtime.MemStats
	previousStats    runtime.MemStats
	leakDetector     *LeakDetector
	gcTuner          *GCTuner
	mu               sync.RWMutex
	monitoringTicker *time.Ticker
	stopMonitoring   chan bool
	statsHistory     []*MemoryStats
	maxHistorySize   int
}

// ManagerConfig contains configuration for memory management
type ManagerConfig struct {
	MonitorInterval      time.Duration `json:"monitor_interval"`
	LeakDetectionEnabled bool          `json:"leak_detection_enabled"`
	AutoGCTuning         bool          `json:"auto_gc_tuning"`
	MaxHistorySize       int           `json:"max_history_size"`
	MemoryThresholds     Thresholds    `json:"memory_thresholds"`
}

// Thresholds contains memory usage thresholds for alerts and actions
type Thresholds struct {
	HeapWarning  uint64        `json:"heap_warning"`  // bytes
	HeapCritical uint64        `json:"heap_critical"` // bytes
	GoroutineMax int           `json:"goroutine_max"`
	GCPauseMax   time.Duration `json:"gc_pause_max"`
}

// NewStandardMemoryManager creates a new memory manager
func NewStandardMemoryManager(config *ManagerConfig) *StandardMemoryManager {
	if config == nil {
		config = &ManagerConfig{
			MonitorInterval:      time.Second * 30,
			LeakDetectionEnabled: true,
			AutoGCTuning:         true,
			MaxHistorySize:       100,
			MemoryThresholds: Thresholds{
				HeapWarning:  100 * 1024 * 1024, // 100MB
				HeapCritical: 500 * 1024 * 1024, // 500MB
				GoroutineMax: 10000,
				GCPauseMax:   time.Millisecond * 100,
			},
		}
	}

	manager := &StandardMemoryManager{
		config:         config,
		stats:          &MemoryStats{},
		leakDetector:   NewLeakDetector(),
		gcTuner:        NewGCTuner(),
		stopMonitoring: make(chan bool),
		statsHistory:   make([]*MemoryStats, 0, config.MaxHistorySize),
		maxHistorySize: config.MaxHistorySize,
	}

	manager.startMonitoring()
	return manager
}

// startMonitoring begins continuous memory monitoring
func (mm *StandardMemoryManager) startMonitoring() {
	mm.monitoringTicker = time.NewTicker(mm.config.MonitorInterval)

	go func() {
		for {
			select {
			case <-mm.monitoringTicker.C:
				mm.updateStats()
				if mm.config.AutoGCTuning {
					mm.autoTuneGC()
				}
			case <-mm.stopMonitoring:
				mm.monitoringTicker.Stop()
				return
			}
		}
	}()
}

// updateStats updates current memory statistics
func (mm *StandardMemoryManager) updateStats() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Store previous stats for rate calculations
	mm.previousStats = mm.gcStats

	// Get current memory stats
	runtime.ReadMemStats(&mm.gcStats)

	// Calculate derived statistics
	mm.stats = &MemoryStats{
		HeapSize:        mm.gcStats.HeapSys,
		HeapInUse:       mm.gcStats.HeapInuse,
		HeapIdle:        mm.gcStats.HeapIdle,
		HeapReleased:    mm.gcStats.HeapReleased,
		StackInUse:      mm.gcStats.StackInuse,
		StackSys:        mm.gcStats.StackSys,
		MSpanInUse:      mm.gcStats.MSpanInuse,
		MSpanSys:        mm.gcStats.MSpanSys,
		MCacheInUse:     mm.gcStats.MCacheInuse,
		MCacheSys:       mm.gcStats.MCacheSys,
		BuckHashSys:     mm.gcStats.BuckHashSys,
		GCSys:           mm.gcStats.GCSys,
		OtherSys:        mm.gcStats.OtherSys,
		NextGC:          mm.gcStats.NextGC,
		NumGoroutines:   runtime.NumGoroutine(),
		NumGC:           mm.gcStats.NumGC,
		HeapUtilization: float64(mm.gcStats.HeapInuse) / float64(mm.gcStats.HeapSys) * 100,
	}

	// Calculate last GC time
	if mm.gcStats.NumGC > 0 {
		gcTimes := mm.gcStats.PauseNs[(mm.gcStats.NumGC+255)%256]
		mm.stats.LastGC = time.Unix(0, int64(gcTimes))
		mm.stats.GCPauseTime = time.Duration(gcTimes)
	}

	// Calculate rates if we have previous data
	if mm.previousStats.NumGC > 0 {
		timeDiff := time.Since(mm.stats.LastGC)
		if timeDiff > 0 {
			allocDiff := mm.gcStats.TotalAlloc - mm.previousStats.TotalAlloc
			mm.stats.AllocRate = float64(allocDiff) / timeDiff.Seconds()

			gcDiff := mm.gcStats.NumGC - mm.previousStats.NumGC
			mm.stats.GCFrequency = float64(gcDiff) / timeDiff.Seconds()
		}
	}

	// Add to history
	mm.addToHistory(mm.stats)
}

// addToHistory adds current stats to history with size limit
func (mm *StandardMemoryManager) addToHistory(stats *MemoryStats) {
	// Create a copy of stats
	statsCopy := *stats

	mm.statsHistory = append(mm.statsHistory, &statsCopy)

	// Maintain history size limit
	if len(mm.statsHistory) > mm.maxHistorySize {
		mm.statsHistory = mm.statsHistory[1:]
	}
}

// MonitorUsage returns current memory usage statistics
func (mm *StandardMemoryManager) MonitorUsage() *MemoryStats {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Return a copy to avoid race conditions
	if mm.stats == nil {
		mm.updateStats()
	}

	statsCopy := *mm.stats
	return &statsCopy
}

// OptimizeGC performs garbage collection optimization
func (mm *StandardMemoryManager) OptimizeGC() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	currentStats := mm.MonitorUsage()

	// Trigger manual GC if heap usage is high
	if currentStats.HeapInUse > mm.config.MemoryThresholds.HeapWarning {
		runtime.GC()
		runtime.GC() // Double GC to ensure cleanup
	}

	// Optimize GC settings based on current usage
	return mm.gcTuner.OptimizeSettings(currentStats)
}

// autoTuneGC performs automatic GC tuning based on current conditions
func (mm *StandardMemoryManager) autoTuneGC() {
	stats := mm.MonitorUsage()

	// Auto-tune based on memory pressure
	if stats.HeapInUse > mm.config.MemoryThresholds.HeapWarning {
		mm.gcTuner.ReduceGCTarget()
	} else if stats.HeapUtilization < 50 {
		mm.gcTuner.IncreaseGCTarget()
	}

	// Adjust based on GC pause times
	if stats.GCPauseTime > mm.config.MemoryThresholds.GCPauseMax {
		mm.gcTuner.ReduceGCTarget()
	}
}

// DetectLeaks identifies potential memory leaks
func (mm *StandardMemoryManager) DetectLeaks() ([]*MemoryLeak, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if !mm.config.LeakDetectionEnabled {
		return []*MemoryLeak{}, nil
	}

	leaks := []*MemoryLeak{}

	// Detect goroutine leaks
	if mm.stats.NumGoroutines > mm.config.MemoryThresholds.GoroutineMax {
		leak := &MemoryLeak{
			Type:       "goroutine_leak",
			Size:       uint64(mm.stats.NumGoroutines),
			Location:   "runtime",
			DetectedAt: time.Now(),
			Severity:   mm.calculateGoroutineSeverity(mm.stats.NumGoroutines),
			Suggestions: []string{
				"Review goroutine creation patterns",
				"Ensure goroutines are properly terminated",
				"Check for blocked goroutines",
			},
		}
		leaks = append(leaks, leak)
	}

	// Detect heap growth trends
	if len(mm.statsHistory) >= 10 {
		heapGrowthLeak := mm.detectHeapGrowthLeak()
		if heapGrowthLeak != nil {
			leaks = append(leaks, heapGrowthLeak)
		}
	}

	// Use leak detector for more sophisticated detection
	detectedLeaks, err := mm.leakDetector.DetectLeaks(mm.stats, mm.statsHistory)
	if err != nil {
		return nil, fmt.Errorf("leak detection failed: %w", err)
	}

	leaks = append(leaks, detectedLeaks...)

	return leaks, nil
}

// detectHeapGrowthLeak analyzes heap growth patterns for potential leaks
func (mm *StandardMemoryManager) detectHeapGrowthLeak() *MemoryLeak {
	if len(mm.statsHistory) < 10 {
		return nil
	}

	// Analyze last 10 data points
	recent := mm.statsHistory[len(mm.statsHistory)-10:]

	// Calculate growth rate
	startHeap := recent[0].HeapInUse
	endHeap := recent[len(recent)-1].HeapInUse

	growthRate := float64(endHeap-startHeap) / float64(startHeap)

	// If heap has grown significantly and consistently
	if growthRate > 0.5 && mm.isConsistentGrowth(recent) {
		return &MemoryLeak{
			Type:       "heap_growth_leak",
			Size:       endHeap - startHeap,
			Location:   "heap",
			DetectedAt: time.Now(),
			Severity:   mm.calculateHeapGrowthSeverity(growthRate),
			Suggestions: []string{
				"Review object allocation patterns",
				"Check for unreleased references",
				"Consider using object pools",
				"Profile heap usage with pprof",
			},
		}
	}

	return nil
}

// isConsistentGrowth checks if heap growth is consistent across samples
func (mm *StandardMemoryManager) isConsistentGrowth(samples []*MemoryStats) bool {
	growthCount := 0
	for i := 1; i < len(samples); i++ {
		if samples[i].HeapInUse > samples[i-1].HeapInUse {
			growthCount++
		}
	}

	// At least 70% of samples show growth
	return float64(growthCount)/float64(len(samples)-1) >= 0.7
}

// calculateGoroutineSeverity determines severity based on goroutine count
func (mm *StandardMemoryManager) calculateGoroutineSeverity(count int) string {
	threshold := mm.config.MemoryThresholds.GoroutineMax

	if count > threshold*2 {
		return "critical"
	} else if count > threshold*3/2 {
		return "high"
	} else if count > threshold {
		return "medium"
	}
	return "low"
}

// calculateHeapGrowthSeverity determines severity based on heap growth rate
func (mm *StandardMemoryManager) calculateHeapGrowthSeverity(growthRate float64) string {
	if growthRate > 2.0 {
		return "critical"
	} else if growthRate > 1.0 {
		return "high"
	} else if growthRate > 0.5 {
		return "medium"
	}
	return "low"
}

// SetGCTargets sets garbage collection optimization targets
func (mm *StandardMemoryManager) SetGCTargets(targets GCTargets) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	return mm.gcTuner.SetTargets(targets)
}

// GetOptimizationReport generates a comprehensive memory optimization report
func (mm *StandardMemoryManager) GetOptimizationReport() *OptimizationReport {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	currentStats := mm.MonitorUsage()
	leaks, _ := mm.DetectLeaks()

	report := &OptimizationReport{
		GeneratedAt:     time.Now(),
		CurrentStats:    *currentStats,
		DetectedLeaks:   leaks,
		GCEfficiency:    mm.calculateGCEfficiency(),
		Recommendations: mm.generateRecommendations(currentStats, leaks),
		HealthScore:     mm.calculateHealthScore(currentStats, leaks),
	}

	return report
}

// calculateGCEfficiency calculates garbage collection efficiency metrics
func (mm *StandardMemoryManager) calculateGCEfficiency() GCEfficiency {
	if len(mm.statsHistory) < 2 {
		return GCEfficiency{}
	}

	// Calculate average pause time from recent history
	var totalPause time.Duration
	var maxPause time.Duration
	validSamples := 0

	for _, stats := range mm.statsHistory[max(0, len(mm.statsHistory)-10):] {
		if stats.GCPauseTime > 0 {
			totalPause += stats.GCPauseTime
			if stats.GCPauseTime > maxPause {
				maxPause = stats.GCPauseTime
			}
			validSamples++
		}
	}

	avgPause := time.Duration(0)
	if validSamples > 0 {
		avgPause = totalPause / time.Duration(validSamples)
	}

	// Calculate GC overhead (simplified)
	gcOverhead := 0.0
	if mm.stats.GCFrequency > 0 {
		gcOverhead = float64(avgPause) * mm.stats.GCFrequency / float64(time.Second) * 100
	}

	// Calculate efficiency score
	efficiencyScore := mm.calculateEfficiencyScore(avgPause, gcOverhead)

	return GCEfficiency{
		AveragePauseTime: avgPause,
		MaxPauseTime:     maxPause,
		GCOverhead:       gcOverhead,
		CollectionRate:   mm.stats.AllocRate / 1024 / 1024, // MB/s
		EfficiencyScore:  efficiencyScore,
	}
}

// calculateEfficiencyScore calculates GC efficiency score (0-100)
func (mm *StandardMemoryManager) calculateEfficiencyScore(avgPause time.Duration, overhead float64) float64 {
	// Start with perfect score
	score := 100.0

	// Deduct for long pause times
	if avgPause > time.Millisecond*10 {
		score -= float64(avgPause/time.Millisecond) * 0.5
	}

	// Deduct for high overhead
	if overhead > 5 {
		score -= overhead * 2
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

// generateRecommendations generates optimization recommendations
func (mm *StandardMemoryManager) generateRecommendations(stats *MemoryStats, leaks []*MemoryLeak) []string {
	recommendations := []string{}

	// Memory usage recommendations
	if stats.HeapInUse > mm.config.MemoryThresholds.HeapCritical {
		recommendations = append(recommendations, "Critical: Heap usage is very high - consider immediate optimization")
	} else if stats.HeapInUse > mm.config.MemoryThresholds.HeapWarning {
		recommendations = append(recommendations, "Warning: Heap usage is elevated - monitor for growth trends")
	}

	// GC recommendations
	if stats.GCPauseTime > mm.config.MemoryThresholds.GCPauseMax {
		recommendations = append(recommendations, "GC pause times are high - consider tuning GOGC or reducing allocation rate")
	}

	// Goroutine recommendations
	if stats.NumGoroutines > mm.config.MemoryThresholds.GoroutineMax {
		recommendations = append(recommendations, "High goroutine count detected - review goroutine lifecycle management")
	}

	// Heap utilization recommendations
	if stats.HeapUtilization < 30 {
		recommendations = append(recommendations, "Low heap utilization - consider reducing heap size or increasing GOGC")
	} else if stats.HeapUtilization > 90 {
		recommendations = append(recommendations, "Very high heap utilization - heap size may be too small")
	}

	// Leak-specific recommendations
	if len(leaks) > 0 {
		recommendations = append(recommendations, "Memory leaks detected - review allocation patterns and object lifecycle")
	}

	// Rate-based recommendations
	if stats.AllocRate > 100*1024*1024 { // 100MB/s
		recommendations = append(recommendations, "High allocation rate - consider object pooling or reducing allocations")
	}

	return recommendations
}

// calculateHealthScore calculates overall memory health score (0-100)
func (mm *StandardMemoryManager) calculateHealthScore(stats *MemoryStats, leaks []*MemoryLeak) float64 {
	score := 100.0

	// Deduct for high heap usage
	heapUsageRatio := float64(stats.HeapInUse) / float64(mm.config.MemoryThresholds.HeapCritical)
	if heapUsageRatio > 1 {
		score -= 30
	} else if heapUsageRatio > 0.8 {
		score -= 15
	} else if heapUsageRatio > 0.6 {
		score -= 5
	}

	// Deduct for high goroutine count
	goroutineRatio := float64(stats.NumGoroutines) / float64(mm.config.MemoryThresholds.GoroutineMax)
	if goroutineRatio > 1 {
		score -= 20
	} else if goroutineRatio > 0.8 {
		score -= 10
	}

	// Deduct for GC issues
	if stats.GCPauseTime > mm.config.MemoryThresholds.GCPauseMax {
		score -= 15
	}

	// Deduct for detected leaks
	for _, leak := range leaks {
		switch leak.Severity {
		case "critical":
			score -= 25
		case "high":
			score -= 15
		case "medium":
			score -= 10
		case "low":
			score -= 5
		}
	}

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}

	return score
}

// Stop stops the memory manager monitoring
func (mm *StandardMemoryManager) Stop() {
	close(mm.stopMonitoring)
}

// ForceGC forces garbage collection
func (mm *StandardMemoryManager) ForceGC() {
	runtime.GC()
}

// GetMemoryProfile returns current memory profile data
func (mm *StandardMemoryManager) GetMemoryProfile() []byte {
	return debug.Stack()
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
