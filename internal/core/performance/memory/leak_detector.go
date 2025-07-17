package memory

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// LeakDetector detects and analyzes memory leaks
type LeakDetector struct {
	mu                 sync.RWMutex
	goroutineBaseline  int
	heapBaseline       uint64
	detectionThreshold DetectionThreshold
	leakPatterns       []LeakPattern
	detectionHistory   []DetectionEvent
}

// DetectionThreshold contains thresholds for leak detection
type DetectionThreshold struct {
	GoroutineIncrease   int           `json:"goroutine_increase"`
	HeapGrowthPercent   float64       `json:"heap_growth_percent"`
	SustainedGrowthTime time.Duration `json:"sustained_growth_time"`
	AllocationRateSpike float64       `json:"allocation_rate_spike"`
}

// LeakPattern represents a pattern that indicates potential leaks
type LeakPattern struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Indicators  []string `json:"indicators"`
	Severity    string   `json:"severity"`
}

// DetectionEvent represents a leak detection event
type DetectionEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Details   string    `json:"details"`
	Resolved  bool      `json:"resolved"`
}

// NewLeakDetector creates a new memory leak detector
func NewLeakDetector() *LeakDetector {
	detector := &LeakDetector{
		goroutineBaseline: runtime.NumGoroutine(),
		detectionThreshold: DetectionThreshold{
			GoroutineIncrease:   100,
			HeapGrowthPercent:   50.0,
			SustainedGrowthTime: time.Minute * 5,
			AllocationRateSpike: 2.0, // 2x normal rate
		},
		leakPatterns:     initializeLeakPatterns(),
		detectionHistory: make([]DetectionEvent, 0),
	}

	// Get initial heap baseline
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	detector.heapBaseline = m.HeapInuse

	return detector
}

// initializeLeakPatterns initializes common memory leak patterns
func initializeLeakPatterns() []LeakPattern {
	return []LeakPattern{
		{
			Name:        "goroutine_explosion",
			Description: "Rapid increase in goroutine count",
			Indicators:  []string{"goroutine_count_spike", "sustained_goroutine_growth"},
			Severity:    "high",
		},
		{
			Name:        "heap_growth_linear",
			Description: "Consistent linear heap growth",
			Indicators:  []string{"linear_heap_increase", "no_gc_cleanup"},
			Severity:    "medium",
		},
		{
			Name:        "allocation_burst",
			Description: "Sudden spike in allocation rate",
			Indicators:  []string{"allocation_rate_spike", "memory_pressure"},
			Severity:    "high",
		},
		{
			Name:        "gc_inefficiency",
			Description: "Garbage collection unable to reclaim memory",
			Indicators:  []string{"gc_frequency_high", "heap_not_decreasing"},
			Severity:    "critical",
		},
		{
			Name:        "resource_accumulation",
			Description: "Gradual accumulation of resources",
			Indicators:  []string{"slow_heap_growth", "sustained_trend"},
			Severity:    "low",
		},
	}
}

// DetectLeaks analyzes current memory statistics and history for leaks
func (ld *LeakDetector) DetectLeaks(currentStats *MemoryStats, history []*MemoryStats) ([]*MemoryLeak, error) {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	leaks := []*MemoryLeak{}

	// Detect different types of leaks
	goroutineLeaks := ld.detectGoroutineLeaks(currentStats)
	heapLeaks := ld.detectHeapLeaks(currentStats, history)
	allocationLeaks := ld.detectAllocationLeaks(currentStats, history)
	gcLeaks := ld.detectGCInefficiency(currentStats, history)

	leaks = append(leaks, goroutineLeaks...)
	leaks = append(leaks, heapLeaks...)
	leaks = append(leaks, allocationLeaks...)
	leaks = append(leaks, gcLeaks...)

	// Record detection events
	for _, leak := range leaks {
		ld.recordDetectionEvent(leak)
	}

	return leaks, nil
}

// detectGoroutineLeaks detects goroutine-related memory leaks
func (ld *LeakDetector) detectGoroutineLeaks(stats *MemoryStats) []*MemoryLeak {
	leaks := []*MemoryLeak{}

	// Check for goroutine explosion
	increase := stats.NumGoroutines - ld.goroutineBaseline
	if increase > ld.detectionThreshold.GoroutineIncrease {
		leak := &MemoryLeak{
			Type:       "goroutine_explosion",
			Size:       uint64(increase),
			Location:   "runtime_goroutines",
			DetectedAt: time.Now(),
			Severity:   ld.calculateGoroutineSeverity(increase),
			Suggestions: []string{
				"Review goroutine creation and termination patterns",
				"Check for goroutines blocked on channels or I/O",
				"Implement goroutine lifecycle management",
				"Use context cancellation for goroutine cleanup",
			},
		}
		leaks = append(leaks, leak)
	}

	return leaks
}

// detectHeapLeaks detects heap-related memory leaks
func (ld *LeakDetector) detectHeapLeaks(stats *MemoryStats, history []*MemoryStats) []*MemoryLeak {
	leaks := []*MemoryLeak{}

	if len(history) < 5 {
		return leaks // Need sufficient history
	}

	// Analyze heap growth patterns
	growthPattern := ld.analyzeHeapGrowthPattern(history)

	switch growthPattern.Type {
	case "linear_growth":
		if growthPattern.Sustained && growthPattern.Rate > ld.detectionThreshold.HeapGrowthPercent {
			leak := &MemoryLeak{
				Type:       "heap_linear_growth",
				Size:       growthPattern.TotalGrowth,
				Location:   "heap",
				DetectedAt: time.Now(),
				Severity:   ld.calculateHeapSeverity(growthPattern.Rate),
				Suggestions: []string{
					"Profile heap usage to identify growing objects",
					"Review object lifecycle and retention",
					"Check for reference cycles",
					"Consider using weak references where appropriate",
				},
			}
			leaks = append(leaks, leak)
		}

	case "exponential_growth":
		leak := &MemoryLeak{
			Type:       "heap_exponential_growth",
			Size:       growthPattern.TotalGrowth,
			Location:   "heap",
			DetectedAt: time.Now(),
			Severity:   "critical",
			Suggestions: []string{
				"Immediate investigation required",
				"Check for recursive allocations",
				"Review caching mechanisms",
				"Implement emergency heap limits",
			},
		}
		leaks = append(leaks, leak)
	}

	return leaks
}

// detectAllocationLeaks detects allocation rate anomalies
func (ld *LeakDetector) detectAllocationLeaks(stats *MemoryStats, history []*MemoryStats) []*MemoryLeak {
	leaks := []*MemoryLeak{}

	if len(history) < 3 {
		return leaks
	}

	// Calculate average allocation rate from history
	var totalRate float64
	validSamples := 0

	for _, h := range history {
		if h.AllocRate > 0 {
			totalRate += h.AllocRate
			validSamples++
		}
	}

	if validSamples == 0 {
		return leaks
	}

	avgRate := totalRate / float64(validSamples)

	// Check for allocation rate spike
	if stats.AllocRate > avgRate*ld.detectionThreshold.AllocationRateSpike {
		leak := &MemoryLeak{
			Type:       "allocation_rate_spike",
			Size:       uint64(stats.AllocRate - avgRate),
			Location:   "allocator",
			DetectedAt: time.Now(),
			Severity:   ld.calculateAllocationSeverity(stats.AllocRate, avgRate),
			Suggestions: []string{
				"Identify source of increased allocations",
				"Review recent code changes",
				"Consider object pooling for frequent allocations",
				"Profile allocation hotspots",
			},
		}
		leaks = append(leaks, leak)
	}

	return leaks
}

// detectGCInefficiency detects garbage collection inefficiency issues
func (ld *LeakDetector) detectGCInefficiency(stats *MemoryStats, history []*MemoryStats) []*MemoryLeak {
	leaks := []*MemoryLeak{}

	if len(history) < 5 {
		return leaks
	}

	// Check if GC frequency is high but heap is not decreasing
	recentHistory := history[len(history)-5:]

	highGCFrequency := false
	heapNotDecreasing := true

	for _, h := range recentHistory {
		if h.GCFrequency > 1.0 { // More than 1 GC per second on average
			highGCFrequency = true
		}
	}

	// Check if heap size is consistently high despite GC
	minHeap := recentHistory[0].HeapInUse
	for _, h := range recentHistory {
		if h.HeapInUse < minHeap {
			minHeap = h.HeapInUse
			heapNotDecreasing = false
		}
	}

	if highGCFrequency && heapNotDecreasing {
		leak := &MemoryLeak{
			Type:       "gc_inefficiency",
			Size:       stats.HeapInUse,
			Location:   "gc_system",
			DetectedAt: time.Now(),
			Severity:   "high",
			Suggestions: []string{
				"Objects may not be eligible for collection",
				"Check for finalizers blocking GC",
				"Review global references and caches",
				"Consider manual GC tuning",
			},
		}
		leaks = append(leaks, leak)
	}

	return leaks
}

// HeapGrowthPattern represents a pattern in heap growth
type HeapGrowthPattern struct {
	Type        string  `json:"type"`
	Rate        float64 `json:"rate"`
	Sustained   bool    `json:"sustained"`
	TotalGrowth uint64  `json:"total_growth"`
}

// analyzeHeapGrowthPattern analyzes heap growth patterns from history
func (ld *LeakDetector) analyzeHeapGrowthPattern(history []*MemoryStats) HeapGrowthPattern {
	if len(history) < 3 {
		return HeapGrowthPattern{Type: "insufficient_data"}
	}

	// Calculate growth rates between consecutive points
	growthRates := make([]float64, 0, len(history)-1)
	totalGrowth := uint64(0)

	for i := 1; i < len(history); i++ {
		prev := history[i-1].HeapInUse
		curr := history[i].HeapInUse

		if prev > 0 {
			rate := float64(curr-prev) / float64(prev) * 100
			growthRates = append(growthRates, rate)

			if curr > prev {
				totalGrowth += curr - prev
			}
		}
	}

	if len(growthRates) == 0 {
		return HeapGrowthPattern{Type: "no_growth"}
	}

	// Analyze growth pattern
	avgGrowthRate := ld.calculateAverage(growthRates)

	// Check for sustained growth (majority of samples show growth)
	positiveGrowthCount := 0
	for _, rate := range growthRates {
		if rate > 0 {
			positiveGrowthCount++
		}
	}

	sustained := float64(positiveGrowthCount)/float64(len(growthRates)) > 0.7

	// Determine pattern type
	patternType := "stable"
	if avgGrowthRate > 50 {
		// Check if growth is accelerating (exponential)
		if ld.isExponentialGrowth(growthRates) {
			patternType = "exponential_growth"
		} else {
			patternType = "linear_growth"
		}
	} else if avgGrowthRate > 10 {
		patternType = "moderate_growth"
	}

	return HeapGrowthPattern{
		Type:        patternType,
		Rate:        avgGrowthRate,
		Sustained:   sustained,
		TotalGrowth: totalGrowth,
	}
}

// isExponentialGrowth checks if growth rates are accelerating
func (ld *LeakDetector) isExponentialGrowth(rates []float64) bool {
	if len(rates) < 4 {
		return false
	}

	// Check if recent rates are significantly higher than earlier ones
	halfPoint := len(rates) / 2
	earlyAvg := ld.calculateAverage(rates[:halfPoint])
	lateAvg := ld.calculateAverage(rates[halfPoint:])

	return lateAvg > earlyAvg*2 // Late period has 2x higher growth rate
}

// calculateAverage calculates the average of a slice of float64 values
func (ld *LeakDetector) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// Severity calculation methods
func (ld *LeakDetector) calculateGoroutineSeverity(increase int) string {
	threshold := ld.detectionThreshold.GoroutineIncrease

	if increase > threshold*3 {
		return "critical"
	} else if increase > threshold*2 {
		return "high"
	} else if increase > threshold {
		return "medium"
	}
	return "low"
}

func (ld *LeakDetector) calculateHeapSeverity(growthRate float64) string {
	if growthRate > 100 {
		return "critical"
	} else if growthRate > 75 {
		return "high"
	} else if growthRate > 50 {
		return "medium"
	}
	return "low"
}

func (ld *LeakDetector) calculateAllocationSeverity(current, average float64) string {
	ratio := current / average

	if ratio > 5 {
		return "critical"
	} else if ratio > 3 {
		return "high"
	} else if ratio > 2 {
		return "medium"
	}
	return "low"
}

// recordDetectionEvent records a detection event in history
func (ld *LeakDetector) recordDetectionEvent(leak *MemoryLeak) {
	event := DetectionEvent{
		Timestamp: leak.DetectedAt,
		Type:      leak.Type,
		Severity:  leak.Severity,
		Details:   fmt.Sprintf("Size: %d, Location: %s", leak.Size, leak.Location),
		Resolved:  false,
	}

	ld.detectionHistory = append(ld.detectionHistory, event)

	// Limit history size
	if len(ld.detectionHistory) > 1000 {
		ld.detectionHistory = ld.detectionHistory[100:] // Keep recent 900 events
	}
}

// UpdateBaseline updates the baseline measurements for leak detection
func (ld *LeakDetector) UpdateBaseline() {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.goroutineBaseline = runtime.NumGoroutine()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ld.heapBaseline = m.HeapInuse
}

// SetThresholds updates detection thresholds
func (ld *LeakDetector) SetThresholds(thresholds DetectionThreshold) {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.detectionThreshold = thresholds
}

// GetDetectionHistory returns the history of leak detection events
func (ld *LeakDetector) GetDetectionHistory() []DetectionEvent {
	ld.mu.RLock()
	defer ld.mu.RUnlock()

	// Return a copy to avoid race conditions
	history := make([]DetectionEvent, len(ld.detectionHistory))
	copy(history, ld.detectionHistory)

	return history
}

// GetLeakPatterns returns the configured leak patterns
func (ld *LeakDetector) GetLeakPatterns() []LeakPattern {
	ld.mu.RLock()
	defer ld.mu.RUnlock()

	// Return a copy to avoid race conditions
	patterns := make([]LeakPattern, len(ld.leakPatterns))
	copy(patterns, ld.leakPatterns)

	return patterns
}
