package memory

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PreallocationManager manages intelligent memory preallocation
type PreallocationManager struct {
	config        *PreallocationConfig
	logger        *logrus.Logger
	strategies    map[string]*PreallocationStrategy
	pools         map[string]*PreallocationPool
	poolManager   *ObjectPoolManager
	mu            sync.RWMutex
	monitorTicker *time.Ticker
	stopChan      chan bool
	isRunning     bool
	usageAnalyzer *UsageAnalyzer
	metrics       *PreallocationMetrics
}

// PreallocationConfig contains preallocation manager configuration
type PreallocationConfig struct {
	MonitorInterval    time.Duration `json:"monitor_interval"`
	AnalysisWindow     time.Duration `json:"analysis_window"`
	MinUsageThreshold  float64       `json:"min_usage_threshold"`
	MaxPreallocationMB int64         `json:"max_preallocation_mb"`
	AdaptiveScaling    bool          `json:"adaptive_scaling"`
	PredictiveEnabled  bool          `json:"predictive_enabled"`
	AggressiveMode     bool          `json:"aggressive_mode"`
	EnabledStrategies  []string      `json:"enabled_strategies"`
}

// PreallocationStrategy defines a memory preallocation strategy
type PreallocationStrategy struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Enabled         bool                   `json:"enabled"`
	Priority        int                    `json:"priority"`
	TriggerPatterns []TriggerPattern       `json:"trigger_patterns"`
	Configuration   map[string]interface{} `json:"configuration"`
	LastExecution   time.Time              `json:"last_execution"`
	ExecutionCount  int64                  `json:"execution_count"`
	SuccessRate     float64                `json:"success_rate"`
}

// TriggerPattern defines when a preallocation strategy should be triggered
type TriggerPattern struct {
	Type         string        `json:"type"`
	Condition    string        `json:"condition"`
	Threshold    float64       `json:"threshold"`
	TimeWindow   time.Duration `json:"time_window"`
	Dependencies []string      `json:"dependencies"`
}

// PreallocationPool represents a preallocated memory pool
type PreallocationPool struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Size          int64       `json:"size"`
	Used          int64       `json:"used"`
	Available     int64       `json:"available"`
	HitCount      int64       `json:"hit_count"`
	MissCount     int64       `json:"miss_count"`
	CreatedAt     time.Time   `json:"created_at"`
	LastUsed      time.Time   `json:"last_used"`
	ExpiresAt     *time.Time  `json:"expires_at,omitempty"`
	Configuration interface{} `json:"configuration"`
}

// UsageAnalyzer analyzes memory usage patterns for preallocation decisions
type UsageAnalyzer struct {
	allocationPatterns map[string]*AllocationPattern
	requestPatterns    map[string]*RequestPattern
	temporalPatterns   *TemporalPattern
	mu                 sync.RWMutex
}

// AllocationPattern represents memory allocation patterns
type AllocationPattern struct {
	Type           string    `json:"type"`
	AverageSize    int64     `json:"average_size"`
	PeakSize       int64     `json:"peak_size"`
	Frequency      float64   `json:"frequency"`
	GrowthRate     float64   `json:"growth_rate"`
	Predictability float64   `json:"predictability"`
	LastAnalyzed   time.Time `json:"last_analyzed"`
}

// RequestPattern represents request-based allocation patterns
type RequestPattern struct {
	Endpoint           string    `json:"endpoint"`
	AverageAllocations int64     `json:"average_allocations"`
	PeakAllocations    int64     `json:"peak_allocations"`
	RequestRate        float64   `json:"request_rate"`
	AllocationSpike    bool      `json:"allocation_spike"`
	LastSeen           time.Time `json:"last_seen"`
}

// TemporalPattern represents time-based allocation patterns
type TemporalPattern struct {
	HourlyPatterns [24]float64 `json:"hourly_patterns"`
	DailyPatterns  [7]float64  `json:"daily_patterns"`
	PeakHours      []int       `json:"peak_hours"`
	PeakDays       []int       `json:"peak_days"`
	Seasonality    float64     `json:"seasonality"`
	LastUpdated    time.Time   `json:"last_updated"`
}

// PreallocationMetrics contains preallocation performance metrics
type PreallocationMetrics struct {
	TotalPools          int       `json:"total_pools"`
	ActivePools         int       `json:"active_pools"`
	TotalPreallocatedMB int64     `json:"total_preallocated_mb"`
	UsedPreallocatedMB  int64     `json:"used_preallocated_mb"`
	HitRate             float64   `json:"hit_rate"`
	EfficiencyScore     float64   `json:"efficiency_score"`
	MemorySavings       int64     `json:"memory_savings"`
	AllocationReduction float64   `json:"allocation_reduction"`
	LastOptimization    time.Time `json:"last_optimization"`
}

// DefaultPreallocationConfig returns default preallocation configuration
func DefaultPreallocationConfig() *PreallocationConfig {
	return &PreallocationConfig{
		MonitorInterval:    time.Minute * 5,
		AnalysisWindow:     time.Hour * 2,
		MinUsageThreshold:  0.6, // 60% usage required for preallocation
		MaxPreallocationMB: 100, // 100MB max preallocation
		AdaptiveScaling:    true,
		PredictiveEnabled:  true,
		AggressiveMode:     false,
		EnabledStrategies: []string{
			"request_based",
			"pattern_based",
			"temporal_based",
			"pool_expansion",
		},
	}
}

// NewPreallocationManager creates a new preallocation manager
func NewPreallocationManager(config *PreallocationConfig, logger *logrus.Logger) *PreallocationManager {
	if config == nil {
		config = DefaultPreallocationConfig()
	}

	if logger == nil {
		logger = logrus.New()
	}

	manager := &PreallocationManager{
		config:        config,
		logger:        logger,
		strategies:    make(map[string]*PreallocationStrategy),
		pools:         make(map[string]*PreallocationPool),
		stopChan:      make(chan bool),
		usageAnalyzer: NewUsageAnalyzer(),
		metrics:       &PreallocationMetrics{},
	}

	// Initialize preallocation strategies
	manager.initializeStrategies()

	return manager
}

// Start begins preallocation management
func (pm *PreallocationManager) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.isRunning {
		return nil
	}

	pm.logger.Info("Starting memory preallocation management")

	pm.monitorTicker = time.NewTicker(pm.config.MonitorInterval)
	pm.isRunning = true

	go pm.managementLoop(ctx)

	return nil
}

// Stop stops preallocation management
func (pm *PreallocationManager) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.isRunning {
		return
	}

	pm.logger.Info("Stopping memory preallocation management")

	close(pm.stopChan)
	pm.monitorTicker.Stop()
	pm.isRunning = false
}

// SetPoolManager sets the object pool manager for integration
func (pm *PreallocationManager) SetPoolManager(poolManager *ObjectPoolManager) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.poolManager = poolManager
}

// AnalyzeAllocationPattern analyzes memory allocation patterns
func (pm *PreallocationManager) AnalyzeAllocationPattern(allocationType string, size int64) {
	pm.usageAnalyzer.RecordAllocation(allocationType, size)
}

// AnalyzeRequestPattern analyzes request-based allocation patterns
func (pm *PreallocationManager) AnalyzeRequestPattern(endpoint string, allocations int64) {
	pm.usageAnalyzer.RecordRequestPattern(endpoint, allocations)
}

// PreallocateForPattern preallocates memory based on detected patterns
func (pm *PreallocationManager) PreallocateForPattern(pattern string, size int64) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if we're within preallocation limits
	if pm.getTotalPreallocatedMB()+size/(1024*1024) > pm.config.MaxPreallocationMB {
		return fmt.Errorf("preallocation would exceed maximum limit")
	}

	poolName := fmt.Sprintf("pattern_%s_%d", pattern, time.Now().Unix())

	pool := &PreallocationPool{
		Name:      poolName,
		Type:      "pattern_based",
		Size:      size,
		Used:      0,
		Available: size,
		CreatedAt: time.Now(),
		Configuration: map[string]interface{}{
			"pattern": pattern,
		},
	}

	pm.pools[poolName] = pool
	pm.logger.WithFields(logrus.Fields{
		"pool_name": poolName,
		"pattern":   pattern,
		"size_mb":   size / (1024 * 1024),
	}).Info("Preallocated memory for pattern")

	return nil
}

// GetPreallocationStats returns current preallocation statistics
func (pm *PreallocationManager) GetStats() *PreallocationStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	activePools := 0
	totalPreallocated := int64(0)
	totalHits := int64(0)
	totalMisses := int64(0)

	for _, pool := range pm.pools {
		if pool.Available > 0 {
			activePools++
		}
		totalPreallocated += pool.Size
		totalHits += pool.HitCount
		totalMisses += pool.MissCount
	}

	hitRate := 0.0
	if totalHits+totalMisses > 0 {
		hitRate = float64(totalHits) / float64(totalHits+totalMisses) * 100
	}

	return &PreallocationStats{
		ActivePools:     activePools,
		PreallocatedMB:  totalPreallocated / (1024 * 1024),
		HitRate:         hitRate,
		EfficiencyScore: pm.calculateEfficiencyScore(),
	}
}

// GetAllocationRecommendations returns memory allocation recommendations
func (pm *PreallocationManager) GetAllocationRecommendations() []*AllocationRecommendation {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	recommendations := make([]*AllocationRecommendation, 0)

	// Analyze patterns and generate recommendations
	patterns := pm.usageAnalyzer.GetAllocationPatterns()

	for patternType, pattern := range patterns {
		if pattern.Predictability > pm.config.MinUsageThreshold {
			recommendation := &AllocationRecommendation{
				Type:        "pattern_based",
				Pattern:     patternType,
				SuggestedMB: pattern.AverageSize / (1024 * 1024),
				Confidence:  pattern.Predictability,
				Reason:      fmt.Sprintf("High predictability pattern detected (%.1f%%)", pattern.Predictability*100),
				Priority:    pm.calculateRecommendationPriority(pattern),
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	// Add temporal-based recommendations
	if pm.config.PredictiveEnabled {
		temporalRecs := pm.generateTemporalRecommendations()
		recommendations = append(recommendations, temporalRecs...)
	}

	return recommendations
}

// OptimizePools optimizes existing preallocation pools
func (pm *PreallocationManager) OptimizePools() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	optimized := 0
	removed := 0

	for poolName, pool := range pm.pools {
		// Remove expired or unused pools
		if pm.shouldRemovePool(pool) {
			delete(pm.pools, poolName)
			removed++
			continue
		}

		// Resize pools based on usage patterns
		if pm.shouldResizePool(pool) {
			pm.resizePool(pool)
			optimized++
		}
	}

	pm.logger.WithFields(logrus.Fields{
		"optimized": optimized,
		"removed":   removed,
	}).Info("Optimized preallocation pools")

	pm.metrics.LastOptimization = time.Now()

	return nil
}

// Private methods

func (pm *PreallocationManager) initializeStrategies() {
	// Request-based preallocation strategy
	pm.strategies["request_based"] = &PreallocationStrategy{
		Name:        "request_based",
		Description: "Preallocate memory based on request patterns",
		Enabled:     pm.isStrategyEnabled("request_based"),
		Priority:    1,
		TriggerPatterns: []TriggerPattern{
			{
				Type:       "request_rate",
				Condition:  "above_threshold",
				Threshold:  100, // requests per minute
				TimeWindow: time.Minute * 5,
			},
		},
	}

	// Pattern-based preallocation strategy
	pm.strategies["pattern_based"] = &PreallocationStrategy{
		Name:        "pattern_based",
		Description: "Preallocate memory based on allocation patterns",
		Enabled:     pm.isStrategyEnabled("pattern_based"),
		Priority:    2,
		TriggerPatterns: []TriggerPattern{
			{
				Type:      "predictability",
				Condition: "above_threshold",
				Threshold: pm.config.MinUsageThreshold,
			},
		},
	}

	// Temporal-based preallocation strategy
	pm.strategies["temporal_based"] = &PreallocationStrategy{
		Name:        "temporal_based",
		Description: "Preallocate memory based on temporal patterns",
		Enabled:     pm.isStrategyEnabled("temporal_based"),
		Priority:    3,
		TriggerPatterns: []TriggerPattern{
			{
				Type:       "time_based",
				Condition:  "peak_approaching",
				TimeWindow: time.Minute * 30,
			},
		},
	}

	// Pool expansion strategy
	pm.strategies["pool_expansion"] = &PreallocationStrategy{
		Name:        "pool_expansion",
		Description: "Expand existing pools based on usage",
		Enabled:     pm.isStrategyEnabled("pool_expansion"),
		Priority:    4,
		TriggerPatterns: []TriggerPattern{
			{
				Type:      "pool_usage",
				Condition: "above_threshold",
				Threshold: 0.8, // 80% usage
			},
		},
	}
}

func (pm *PreallocationManager) managementLoop(ctx context.Context) {
	defer pm.monitorTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case <-pm.monitorTicker.C:
			pm.performManagementCycle(ctx)
		}
	}
}

func (pm *PreallocationManager) performManagementCycle(ctx context.Context) {
	// Update usage analysis
	pm.usageAnalyzer.PerformAnalysis()

	// Execute enabled strategies
	for _, strategy := range pm.strategies {
		if strategy.Enabled {
			pm.executeStrategy(ctx, strategy)
		}
	}

	// Optimize existing pools
	pm.OptimizePools()

	// Update metrics
	pm.updateMetrics()
}

func (pm *PreallocationManager) executeStrategy(ctx context.Context, strategy *PreallocationStrategy) {
	strategy.LastExecution = time.Now()
	strategy.ExecutionCount++

	// Strategy-specific execution logic would go here
	switch strategy.Name {
	case "request_based":
		pm.executeRequestBasedStrategy(ctx, strategy)
	case "pattern_based":
		pm.executePatternBasedStrategy(ctx, strategy)
	case "temporal_based":
		pm.executeTemporalBasedStrategy(ctx, strategy)
	case "pool_expansion":
		pm.executePoolExpansionStrategy(ctx, strategy)
	}
}

func (pm *PreallocationManager) executeRequestBasedStrategy(ctx context.Context, strategy *PreallocationStrategy) {
	// Analyze request patterns and preallocate accordingly
	patterns := pm.usageAnalyzer.GetRequestPatterns()

	for endpoint, pattern := range patterns {
		if pattern.AllocationSpike && pattern.RequestRate > 100 {
			size := pattern.PeakAllocations * 1024 // KB to bytes
			poolName := fmt.Sprintf("request_%s", endpoint)

			if _, exists := pm.pools[poolName]; !exists {
				pm.createPool(poolName, "request_based", size)
			}
		}
	}
}

func (pm *PreallocationManager) executePatternBasedStrategy(ctx context.Context, strategy *PreallocationStrategy) {
	// Analyze allocation patterns and preallocate accordingly
	patterns := pm.usageAnalyzer.GetAllocationPatterns()

	for patternType, pattern := range patterns {
		if pattern.Predictability > pm.config.MinUsageThreshold {
			size := int64(float64(pattern.PeakSize) * 1.2) // 20% buffer
			poolName := fmt.Sprintf("pattern_%s", patternType)

			if _, exists := pm.pools[poolName]; !exists {
				pm.createPool(poolName, "pattern_based", size)
			}
		}
	}
}

func (pm *PreallocationManager) executeTemporalBasedStrategy(ctx context.Context, strategy *PreallocationStrategy) {
	// Analyze temporal patterns and preallocate for upcoming peaks
	if !pm.config.PredictiveEnabled {
		return
	}

	temporal := pm.usageAnalyzer.GetTemporalPattern()
	if temporal == nil {
		return
	}

	currentHour := time.Now().Hour()
	nextHour := (currentHour + 1) % 24

	// If next hour is a peak hour, preallocate
	for _, peakHour := range temporal.PeakHours {
		if nextHour == peakHour {
			size := int64(50 * 1024 * 1024) // 50MB for peak hours
			poolName := fmt.Sprintf("temporal_peak_%d", peakHour)

			if _, exists := pm.pools[poolName]; !exists {
				pm.createPool(poolName, "temporal_based", size)
			}
		}
	}
}

func (pm *PreallocationManager) executePoolExpansionStrategy(ctx context.Context, strategy *PreallocationStrategy) {
	// Expand pools that are heavily used
	for poolName, pool := range pm.pools {
		usage := float64(pool.Used) / float64(pool.Size)

		if usage > 0.8 { // 80% usage threshold
			expansionSize := pool.Size / 2 // Expand by 50%
			pool.Size += expansionSize
			pool.Available += expansionSize

			pm.logger.WithFields(logrus.Fields{
				"pool_name":      poolName,
				"expansion_size": expansionSize / (1024 * 1024),
				"new_size":       pool.Size / (1024 * 1024),
			}).Info("Expanded preallocation pool")
		}
	}
}

func (pm *PreallocationManager) createPool(name, poolType string, size int64) {
	pool := &PreallocationPool{
		Name:      name,
		Type:      poolType,
		Size:      size,
		Used:      0,
		Available: size,
		CreatedAt: time.Now(),
	}

	pm.pools[name] = pool

	pm.logger.WithFields(logrus.Fields{
		"pool_name": name,
		"pool_type": poolType,
		"size_mb":   size / (1024 * 1024),
	}).Info("Created preallocation pool")
}

func (pm *PreallocationManager) shouldRemovePool(pool *PreallocationPool) bool {
	// Remove pools that haven't been used in 24 hours
	if time.Since(pool.LastUsed) > 24*time.Hour {
		return true
	}

	// Remove pools with very low hit rates
	if pool.HitCount+pool.MissCount > 100 {
		hitRate := float64(pool.HitCount) / float64(pool.HitCount+pool.MissCount)
		if hitRate < 0.1 { // Less than 10% hit rate
			return true
		}
	}

	return false
}

func (pm *PreallocationManager) shouldResizePool(pool *PreallocationPool) bool {
	// Resize if usage is consistently high or low
	usage := float64(pool.Used) / float64(pool.Size)

	// Shrink if usage is consistently low
	if usage < 0.2 && time.Since(pool.CreatedAt) > time.Hour {
		return true
	}

	// Expand if usage is consistently high
	if usage > 0.9 {
		return true
	}

	return false
}

func (pm *PreallocationManager) resizePool(pool *PreallocationPool) {
	usage := float64(pool.Used) / float64(pool.Size)

	if usage < 0.2 {
		// Shrink pool by 25%
		reductionSize := pool.Size / 4
		pool.Size -= reductionSize
		pool.Available -= reductionSize
		if pool.Available < 0 {
			pool.Available = 0
		}
	} else if usage > 0.9 {
		// Expand pool by 50%
		expansionSize := pool.Size / 2
		pool.Size += expansionSize
		pool.Available += expansionSize
	}
}

func (pm *PreallocationManager) isStrategyEnabled(strategyName string) bool {
	for _, enabled := range pm.config.EnabledStrategies {
		if enabled == strategyName {
			return true
		}
	}
	return false
}

func (pm *PreallocationManager) getTotalPreallocatedMB() int64 {
	total := int64(0)
	for _, pool := range pm.pools {
		total += pool.Size
	}
	return total / (1024 * 1024)
}

func (pm *PreallocationManager) calculateEfficiencyScore() float64 {
	if len(pm.pools) == 0 {
		return 100.0
	}

	totalHits := int64(0)
	totalMisses := int64(0)
	totalUsage := float64(0)

	for _, pool := range pm.pools {
		totalHits += pool.HitCount
		totalMisses += pool.MissCount
		if pool.Size > 0 {
			totalUsage += float64(pool.Used) / float64(pool.Size)
		}
	}

	hitRate := 0.0
	if totalHits+totalMisses > 0 {
		hitRate = float64(totalHits) / float64(totalHits+totalMisses)
	}

	avgUsage := totalUsage / float64(len(pm.pools))

	// Efficiency score combines hit rate and usage efficiency
	return (hitRate*0.6 + avgUsage*0.4) * 100
}

func (pm *PreallocationManager) calculateRecommendationPriority(pattern *AllocationPattern) int {
	// Higher priority for more predictable and frequent patterns
	priority := int(pattern.Predictability * 10)
	priority += int(pattern.Frequency / 100)

	if priority > 10 {
		priority = 10
	}
	if priority < 1 {
		priority = 1
	}

	return priority
}

func (pm *PreallocationManager) generateTemporalRecommendations() []*AllocationRecommendation {
	recommendations := make([]*AllocationRecommendation, 0)

	temporal := pm.usageAnalyzer.GetTemporalPattern()
	if temporal == nil {
		return recommendations
	}

	currentHour := time.Now().Hour()

	// Check if approaching peak hours
	for _, peakHour := range temporal.PeakHours {
		hoursUntilPeak := (peakHour - currentHour + 24) % 24

		if hoursUntilPeak <= 2 && hoursUntilPeak > 0 {
			recommendation := &AllocationRecommendation{
				Type:        "temporal_peak",
				Pattern:     fmt.Sprintf("peak_hour_%d", peakHour),
				SuggestedMB: 50, // 50MB for peak preparation
				Confidence:  temporal.Seasonality,
				Reason:      fmt.Sprintf("Peak hour %d approaching in %d hours", peakHour, hoursUntilPeak),
				Priority:    5,
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	return recommendations
}

func (pm *PreallocationManager) updateMetrics() {
	pm.metrics.TotalPools = len(pm.pools)
	pm.metrics.ActivePools = 0
	pm.metrics.TotalPreallocatedMB = 0
	pm.metrics.UsedPreallocatedMB = 0

	totalHits := int64(0)
	totalMisses := int64(0)

	for _, pool := range pm.pools {
		if pool.Available > 0 {
			pm.metrics.ActivePools++
		}
		pm.metrics.TotalPreallocatedMB += pool.Size / (1024 * 1024)
		pm.metrics.UsedPreallocatedMB += pool.Used / (1024 * 1024)
		totalHits += pool.HitCount
		totalMisses += pool.MissCount
	}

	if totalHits+totalMisses > 0 {
		pm.metrics.HitRate = float64(totalHits) / float64(totalHits+totalMisses) * 100
	}

	pm.metrics.EfficiencyScore = pm.calculateEfficiencyScore()
}

// Supporting types

type AllocationRecommendation struct {
	Type        string  `json:"type"`
	Pattern     string  `json:"pattern"`
	SuggestedMB int64   `json:"suggested_mb"`
	Confidence  float64 `json:"confidence"`
	Reason      string  `json:"reason"`
	Priority    int     `json:"priority"`
}

// NewUsageAnalyzer creates a new usage analyzer
func NewUsageAnalyzer() *UsageAnalyzer {
	return &UsageAnalyzer{
		allocationPatterns: make(map[string]*AllocationPattern),
		requestPatterns:    make(map[string]*RequestPattern),
		temporalPatterns:   &TemporalPattern{},
	}
}

// RecordAllocation records an allocation for pattern analysis
func (ua *UsageAnalyzer) RecordAllocation(allocationType string, size int64) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	pattern, exists := ua.allocationPatterns[allocationType]
	if !exists {
		pattern = &AllocationPattern{
			Type:         allocationType,
			AverageSize:  size,
			PeakSize:     size,
			Frequency:    1,
			LastAnalyzed: time.Now(),
		}
		ua.allocationPatterns[allocationType] = pattern
	} else {
		// Update running averages
		pattern.AverageSize = (pattern.AverageSize + size) / 2
		if size > pattern.PeakSize {
			pattern.PeakSize = size
		}
		pattern.Frequency++
		pattern.LastAnalyzed = time.Now()
	}
}

// RecordRequestPattern records a request pattern for analysis
func (ua *UsageAnalyzer) RecordRequestPattern(endpoint string, allocations int64) {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	pattern, exists := ua.requestPatterns[endpoint]
	if !exists {
		pattern = &RequestPattern{
			Endpoint:           endpoint,
			AverageAllocations: allocations,
			PeakAllocations:    allocations,
			RequestRate:        1,
			LastSeen:           time.Now(),
		}
		ua.requestPatterns[endpoint] = pattern
	} else {
		pattern.AverageAllocations = (pattern.AverageAllocations + allocations) / 2
		if allocations > pattern.PeakAllocations {
			pattern.PeakAllocations = allocations
			pattern.AllocationSpike = true
		}
		pattern.RequestRate++
		pattern.LastSeen = time.Now()
	}
}

// PerformAnalysis performs comprehensive usage analysis
func (ua *UsageAnalyzer) PerformAnalysis() {
	ua.mu.Lock()
	defer ua.mu.Unlock()

	// Update predictability scores
	for _, pattern := range ua.allocationPatterns {
		ua.updatePredictability(pattern)
	}

	// Update temporal patterns
	ua.updateTemporalPatterns()
}

// GetAllocationPatterns returns current allocation patterns
func (ua *UsageAnalyzer) GetAllocationPatterns() map[string]*AllocationPattern {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	// Return copies to avoid race conditions
	result := make(map[string]*AllocationPattern)
	for k, v := range ua.allocationPatterns {
		patternCopy := *v
		result[k] = &patternCopy
	}
	return result
}

// GetRequestPatterns returns current request patterns
func (ua *UsageAnalyzer) GetRequestPatterns() map[string]*RequestPattern {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	result := make(map[string]*RequestPattern)
	for k, v := range ua.requestPatterns {
		patternCopy := *v
		result[k] = &patternCopy
	}
	return result
}

// GetTemporalPattern returns current temporal patterns
func (ua *UsageAnalyzer) GetTemporalPattern() *TemporalPattern {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	if ua.temporalPatterns == nil {
		return nil
	}

	patternCopy := *ua.temporalPatterns
	return &patternCopy
}

func (ua *UsageAnalyzer) updatePredictability(pattern *AllocationPattern) {
	// Simple predictability calculation based on size variance and frequency
	sizeVariance := float64(pattern.PeakSize-pattern.AverageSize) / float64(pattern.AverageSize)

	// Higher frequency and lower variance = higher predictability
	frequencyScore := math.Min(pattern.Frequency/1000, 1.0) // Cap at 1000 requests
	varianceScore := math.Max(1.0-sizeVariance, 0.0)

	pattern.Predictability = (frequencyScore*0.6 + varianceScore*0.4)
}

func (ua *UsageAnalyzer) updateTemporalPatterns() {
	// Update hourly and daily patterns based on current time
	now := time.Now()
	hour := now.Hour()
	day := int(now.Weekday())

	// Simple pattern update (in real implementation, this would be more sophisticated)
	ua.temporalPatterns.HourlyPatterns[hour] += 1.0
	ua.temporalPatterns.DailyPatterns[day] += 1.0
	ua.temporalPatterns.LastUpdated = now

	// Update peak hours/days
	ua.updatePeakPeriods()
}

func (ua *UsageAnalyzer) updatePeakPeriods() {
	// Find peak hours
	maxHourlyValue := 0.0
	for _, value := range ua.temporalPatterns.HourlyPatterns {
		if value > maxHourlyValue {
			maxHourlyValue = value
		}
	}

	threshold := maxHourlyValue * 0.8 // 80% of peak
	ua.temporalPatterns.PeakHours = ua.temporalPatterns.PeakHours[:0]

	for i, value := range ua.temporalPatterns.HourlyPatterns {
		if value >= threshold {
			ua.temporalPatterns.PeakHours = append(ua.temporalPatterns.PeakHours, i)
		}
	}

	// Similar logic for peak days
	maxDailyValue := 0.0
	for _, value := range ua.temporalPatterns.DailyPatterns {
		if value > maxDailyValue {
			maxDailyValue = value
		}
	}

	threshold = maxDailyValue * 0.8
	ua.temporalPatterns.PeakDays = ua.temporalPatterns.PeakDays[:0]

	for i, value := range ua.temporalPatterns.DailyPatterns {
		if value >= threshold {
			ua.temporalPatterns.PeakDays = append(ua.temporalPatterns.PeakDays, i)
		}
	}
}
