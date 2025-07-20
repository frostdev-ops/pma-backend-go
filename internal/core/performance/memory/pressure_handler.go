package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MemoryPressureHandler monitors and responds to memory pressure
type MemoryPressureHandler struct {
	config          *PressureHandlerConfig
	logger          *logrus.Logger
	currentLevel    MemoryPressureLevel
	events          []*MemoryPressureEvent
	callback        func(context.Context, MemoryPressureLevel) error
	mu              sync.RWMutex
	monitorTicker   *time.Ticker
	stopChan        chan bool
	isRunning       bool
	thresholds      *PressureThresholds
	responseTimes   []time.Duration
	recoveryMetrics *RecoveryMetrics
}

// PressureHandlerConfig contains pressure handler configuration
type PressureHandlerConfig struct {
	MonitorInterval       time.Duration       `json:"monitor_interval"`
	Thresholds            *PressureThresholds `json:"thresholds"`
	MaxEventHistory       int                 `json:"max_event_history"`
	ResponseTimeout       time.Duration       `json:"response_timeout"`
	AdaptiveThresholds    bool                `json:"adaptive_thresholds"`
	PredictiveModeEnabled bool                `json:"predictive_mode_enabled"`
}

// PressureThresholds defines memory pressure level thresholds
type PressureThresholds struct {
	LowPressure      *Threshold `json:"low_pressure"`
	MediumPressure   *Threshold `json:"medium_pressure"`
	HighPressure     *Threshold `json:"high_pressure"`
	CriticalPressure *Threshold `json:"critical_pressure"`
}

// Threshold defines conditions for a pressure level
type Threshold struct {
	HeapUtilization   float64       `json:"heap_utilization"`
	GoroutineCount    int           `json:"goroutine_count"`
	AllocationRate    uint64        `json:"allocation_rate"` // bytes per second
	GCFrequency       float64       `json:"gc_frequency"`    // GCs per second
	SustainedDuration time.Duration `json:"sustained_duration"`
}

// RecoveryMetrics tracks pressure recovery performance
type RecoveryMetrics struct {
	TotalEvents          int           `json:"total_events"`
	SuccessfulRecoveries int           `json:"successful_recoveries"`
	FailedRecoveries     int           `json:"failed_recoveries"`
	AverageResponseTime  time.Duration `json:"average_response_time"`
	FastestRecovery      time.Duration `json:"fastest_recovery"`
	SlowestRecovery      time.Duration `json:"slowest_recovery"`
	RecoveryRate         float64       `json:"recovery_rate"`
}

// DefaultPressureConfig returns default pressure handler configuration
func DefaultPressureConfig() *PressureHandlerConfig {
	return &PressureHandlerConfig{
		MonitorInterval: time.Second * 5,
		Thresholds: &PressureThresholds{
			LowPressure: &Threshold{
				HeapUtilization:   50.0,
				GoroutineCount:    1000,
				AllocationRate:    10 * 1024 * 1024, // 10 MB/s
				GCFrequency:       0.1,
				SustainedDuration: time.Second * 30,
			},
			MediumPressure: &Threshold{
				HeapUtilization:   70.0,
				GoroutineCount:    2000,
				AllocationRate:    25 * 1024 * 1024, // 25 MB/s
				GCFrequency:       0.3,
				SustainedDuration: time.Second * 15,
			},
			HighPressure: &Threshold{
				HeapUtilization:   85.0,
				GoroutineCount:    5000,
				AllocationRate:    50 * 1024 * 1024, // 50 MB/s
				GCFrequency:       0.5,
				SustainedDuration: time.Second * 10,
			},
			CriticalPressure: &Threshold{
				HeapUtilization:   95.0,
				GoroutineCount:    10000,
				AllocationRate:    100 * 1024 * 1024, // 100 MB/s
				GCFrequency:       1.0,
				SustainedDuration: time.Second * 5,
			},
		},
		MaxEventHistory:       100,
		ResponseTimeout:       time.Second * 30,
		AdaptiveThresholds:    true,
		PredictiveModeEnabled: false,
	}
}

// NewMemoryPressureHandler creates a new memory pressure handler
func NewMemoryPressureHandler(config *PressureHandlerConfig, logger *logrus.Logger) *MemoryPressureHandler {
	if config == nil {
		config = DefaultPressureConfig()
	}

	if logger == nil {
		logger = logrus.New()
	}

	return &MemoryPressureHandler{
		config:          config,
		logger:          logger,
		currentLevel:    PressureNone,
		events:          make([]*MemoryPressureEvent, 0),
		stopChan:        make(chan bool),
		thresholds:      config.Thresholds,
		responseTimes:   make([]time.Duration, 0),
		recoveryMetrics: &RecoveryMetrics{},
	}
}

// Start begins memory pressure monitoring
func (mph *MemoryPressureHandler) Start(ctx context.Context) error {
	mph.mu.Lock()
	defer mph.mu.Unlock()

	if mph.isRunning {
		return nil
	}

	mph.logger.Info("Starting memory pressure monitoring")

	mph.monitorTicker = time.NewTicker(mph.config.MonitorInterval)
	mph.isRunning = true

	go mph.monitorLoop(ctx)

	return nil
}

// Stop stops memory pressure monitoring
func (mph *MemoryPressureHandler) Stop() {
	mph.mu.Lock()
	defer mph.mu.Unlock()

	if !mph.isRunning {
		return
	}

	mph.logger.Info("Stopping memory pressure monitoring")

	close(mph.stopChan)
	mph.monitorTicker.Stop()
	mph.isRunning = false
}

// SetCallback sets the callback function for pressure events
func (mph *MemoryPressureHandler) SetCallback(callback func(context.Context, MemoryPressureLevel) error) {
	mph.mu.Lock()
	defer mph.mu.Unlock()
	mph.callback = callback
}

// UpdateStats updates current memory statistics for pressure evaluation
func (mph *MemoryPressureHandler) UpdateStats(stats *MemoryStats) {
	mph.mu.Lock()
	defer mph.mu.Unlock()

	// Evaluate pressure level based on current stats
	newLevel := mph.evaluatePressureLevel(stats)

	// If pressure level changed, trigger response
	if newLevel != mph.currentLevel {
		mph.handlePressureLevelChange(newLevel, stats)
	}

	mph.currentLevel = newLevel
}

// GetStatus returns current pressure status
func (mph *MemoryPressureHandler) GetStatus() *PressureStatus {
	mph.mu.RLock()
	defer mph.mu.RUnlock()

	var lastEvent *MemoryPressureEvent
	if len(mph.events) > 0 {
		lastEvent = mph.events[len(mph.events)-1]
	}

	return &PressureStatus{
		CurrentLevel:    mph.currentLevel,
		LastEvent:       lastEvent,
		EventHistory:    mph.getRecentEvents(10),
		EventCount:      len(mph.events),
		TotalRecoveries: mph.recoveryMetrics.SuccessfulRecoveries,
	}
}

// RecordEvent records a pressure event
func (mph *MemoryPressureHandler) RecordEvent(event *MemoryPressureEvent) {
	mph.mu.Lock()
	defer mph.mu.Unlock()

	mph.events = append(mph.events, event)

	// Maintain event history size
	if len(mph.events) > mph.config.MaxEventHistory {
		mph.events = mph.events[1:]
	}

	// Update recovery metrics
	mph.updateRecoveryMetrics(event)
}

// GetRecoveryMetrics returns pressure recovery performance metrics
func (mph *MemoryPressureHandler) GetRecoveryMetrics() *RecoveryMetrics {
	mph.mu.RLock()
	defer mph.mu.RUnlock()

	metricsCopy := *mph.recoveryMetrics
	return &metricsCopy
}

// AdaptThresholds adjusts pressure thresholds based on system behavior
func (mph *MemoryPressureHandler) AdaptThresholds() {
	if !mph.config.AdaptiveThresholds {
		return
	}

	mph.mu.Lock()
	defer mph.mu.Unlock()

	// Analyze recent events to determine if thresholds need adjustment
	recentEvents := mph.getRecentEvents(20)

	if len(recentEvents) < 5 {
		return // Not enough data
	}

	// Check for frequent false positives (events that resolve quickly)
	quickResolutions := 0
	for _, event := range recentEvents {
		if event.RecoveryTime < time.Second*5 {
			quickResolutions++
		}
	}

	falsePositiveRate := float64(quickResolutions) / float64(len(recentEvents))

	// If too many false positives, increase thresholds slightly
	if falsePositiveRate > 0.3 {
		mph.adjustThresholds(1.1) // Increase by 10%
		mph.logger.Info("Adapted pressure thresholds upward due to high false positive rate")
	}

	// If not enough early warnings, decrease thresholds slightly
	if falsePositiveRate < 0.1 && mph.recoveryMetrics.FailedRecoveries > 0 {
		mph.adjustThresholds(0.9) // Decrease by 10%
		mph.logger.Info("Adapted pressure thresholds downward for earlier detection")
	}
}

// GetCurrentPressure returns the current memory pressure status
func (mph *MemoryPressureHandler) GetCurrentPressure() *PressureStatus {
	return mph.GetStatus()
}

// HandleCurrentPressure handles the current memory pressure situation
func (mph *MemoryPressureHandler) HandleCurrentPressure() error {
	mph.mu.RLock()
	currentLevel := mph.currentLevel
	mph.mu.RUnlock()

	if currentLevel == PressureNone || currentLevel == PressureLow {
		return nil // No action needed
	}

	// Trigger callback if set
	if mph.callback != nil {
		ctx := context.Background()
		return mph.callback(ctx, currentLevel)
	}

	return nil
}

// GetConfig returns the current pressure handler configuration
func (mph *MemoryPressureHandler) GetConfig() *PressureHandlerConfig {
	mph.mu.RLock()
	defer mph.mu.RUnlock()

	// Return a copy to prevent external modification
	configCopy := *mph.config
	return &configCopy
}

// UpdateConfig updates the pressure handler configuration
func (mph *MemoryPressureHandler) UpdateConfig(newConfig *PressureHandlerConfig) error {
	if newConfig == nil {
		return fmt.Errorf("config cannot be nil")
	}

	mph.mu.Lock()
	defer mph.mu.Unlock()

	// Validate the new configuration
	if newConfig.MonitorInterval <= 0 {
		return fmt.Errorf("monitor interval must be positive")
	}

	if newConfig.ResponseTimeout <= 0 {
		return fmt.Errorf("response timeout must be positive")
	}

	// Update the configuration
	mph.config = newConfig
	mph.thresholds = newConfig.Thresholds

	return nil
}

// Private methods

func (mph *MemoryPressureHandler) monitorLoop(ctx context.Context) {
	defer mph.monitorTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mph.stopChan:
			return
		case <-mph.monitorTicker.C:
			mph.performAdaptiveActions()
		}
	}
}

func (mph *MemoryPressureHandler) performAdaptiveActions() {
	// Perform adaptive threshold adjustment
	if mph.config.AdaptiveThresholds {
		mph.AdaptThresholds()
	}

	// Clean up old events
	mph.cleanupOldEvents()
}

func (mph *MemoryPressureHandler) evaluatePressureLevel(stats *MemoryStats) MemoryPressureLevel {
	// Check critical pressure first
	if mph.meetsThreshold(stats, mph.thresholds.CriticalPressure) {
		return PressureCritical
	}

	// Check high pressure
	if mph.meetsThreshold(stats, mph.thresholds.HighPressure) {
		return PressureHigh
	}

	// Check medium pressure
	if mph.meetsThreshold(stats, mph.thresholds.MediumPressure) {
		return PressureMedium
	}

	// Check low pressure
	if mph.meetsThreshold(stats, mph.thresholds.LowPressure) {
		return PressureLow
	}

	return PressureNone
}

func (mph *MemoryPressureHandler) meetsThreshold(stats *MemoryStats, threshold *Threshold) bool {
	// Check if any threshold condition is met
	if stats.HeapUtilization >= threshold.HeapUtilization {
		return true
	}

	if stats.NumGoroutines >= threshold.GoroutineCount {
		return true
	}

	if uint64(stats.AllocRate) >= threshold.AllocationRate {
		return true
	}

	if stats.GCFrequency >= threshold.GCFrequency {
		return true
	}

	return false
}

func (mph *MemoryPressureHandler) handlePressureLevelChange(newLevel MemoryPressureLevel, stats *MemoryStats) {
	mph.logger.WithFields(logrus.Fields{
		"old_level":        mph.currentLevel.String(),
		"new_level":        newLevel.String(),
		"heap_utilization": stats.HeapUtilization,
		"goroutines":       stats.NumGoroutines,
	}).Info("Memory pressure level changed")

	// Create pressure event
	event := &MemoryPressureEvent{
		Timestamp:   time.Now(),
		Level:       newLevel,
		HeapUsage:   stats.HeapInUse,
		HeapPercent: stats.HeapUtilization,
		Triggers:    mph.identifyTriggers(stats, newLevel),
		Actions:     make([]string, 0),
	}

	// Only trigger callback for pressure increases or first critical detection
	if newLevel > mph.currentLevel || newLevel >= PressureHigh {
		if mph.callback != nil {
			go func() {
				startTime := time.Now()
				ctx, cancel := context.WithTimeout(context.Background(), mph.config.ResponseTimeout)
				defer cancel()

				err := mph.callback(ctx, newLevel)
				recoveryTime := time.Since(startTime)

				event.RecoveryTime = recoveryTime
				event.Resolved = err == nil

				mph.RecordEvent(event)

				if err != nil {
					mph.logger.WithError(err).WithField("level", newLevel.String()).Error("Pressure response callback failed")
					mph.recoveryMetrics.FailedRecoveries++
				} else {
					mph.recoveryMetrics.SuccessfulRecoveries++
				}

				mph.recoveryMetrics.TotalEvents++
				mph.responseTimes = append(mph.responseTimes, recoveryTime)

				// Update average response time
				mph.updateAverageResponseTime()
			}()
		}
	} else {
		// Record level decrease event without callback
		event.Resolved = true
		mph.RecordEvent(event)
	}
}

func (mph *MemoryPressureHandler) identifyTriggers(stats *MemoryStats, level MemoryPressureLevel) []string {
	triggers := make([]string, 0)

	threshold := mph.getThresholdForLevel(level)
	if threshold == nil {
		return triggers
	}

	if stats.HeapUtilization >= threshold.HeapUtilization {
		triggers = append(triggers, "high_heap_utilization")
	}

	if stats.NumGoroutines >= threshold.GoroutineCount {
		triggers = append(triggers, "goroutine_count_exceeded")
	}

	if uint64(stats.AllocRate) >= threshold.AllocationRate {
		triggers = append(triggers, "high_allocation_rate")
	}

	if stats.GCFrequency >= threshold.GCFrequency {
		triggers = append(triggers, "frequent_garbage_collection")
	}

	return triggers
}

func (mph *MemoryPressureHandler) getThresholdForLevel(level MemoryPressureLevel) *Threshold {
	switch level {
	case PressureLow:
		return mph.thresholds.LowPressure
	case PressureMedium:
		return mph.thresholds.MediumPressure
	case PressureHigh:
		return mph.thresholds.HighPressure
	case PressureCritical:
		return mph.thresholds.CriticalPressure
	default:
		return nil
	}
}

func (mph *MemoryPressureHandler) getRecentEvents(count int) []*MemoryPressureEvent {
	if len(mph.events) == 0 {
		return []*MemoryPressureEvent{}
	}

	start := len(mph.events) - count
	if start < 0 {
		start = 0
	}

	// Create copies to avoid race conditions
	result := make([]*MemoryPressureEvent, 0, count)
	for i := start; i < len(mph.events); i++ {
		eventCopy := *mph.events[i]
		result = append(result, &eventCopy)
	}

	return result
}

func (mph *MemoryPressureHandler) updateRecoveryMetrics(event *MemoryPressureEvent) {
	if event.Resolved {
		mph.recoveryMetrics.SuccessfulRecoveries++
	} else {
		mph.recoveryMetrics.FailedRecoveries++
	}

	mph.recoveryMetrics.TotalEvents++

	// Update recovery rate
	if mph.recoveryMetrics.TotalEvents > 0 {
		mph.recoveryMetrics.RecoveryRate = float64(mph.recoveryMetrics.SuccessfulRecoveries) /
			float64(mph.recoveryMetrics.TotalEvents) * 100
	}

	// Update fastest/slowest recovery times
	if event.RecoveryTime > 0 {
		if mph.recoveryMetrics.FastestRecovery == 0 || event.RecoveryTime < mph.recoveryMetrics.FastestRecovery {
			mph.recoveryMetrics.FastestRecovery = event.RecoveryTime
		}

		if event.RecoveryTime > mph.recoveryMetrics.SlowestRecovery {
			mph.recoveryMetrics.SlowestRecovery = event.RecoveryTime
		}
	}
}

func (mph *MemoryPressureHandler) updateAverageResponseTime() {
	if len(mph.responseTimes) == 0 {
		return
	}

	var total time.Duration
	for _, duration := range mph.responseTimes {
		total += duration
	}

	mph.recoveryMetrics.AverageResponseTime = total / time.Duration(len(mph.responseTimes))

	// Keep only recent response times (last 50)
	if len(mph.responseTimes) > 50 {
		mph.responseTimes = mph.responseTimes[len(mph.responseTimes)-50:]
	}
}

func (mph *MemoryPressureHandler) adjustThresholds(factor float64) {
	// Adjust all thresholds by the given factor
	mph.thresholds.LowPressure.HeapUtilization *= factor
	mph.thresholds.MediumPressure.HeapUtilization *= factor
	mph.thresholds.HighPressure.HeapUtilization *= factor
	mph.thresholds.CriticalPressure.HeapUtilization *= factor

	// Ensure thresholds stay within reasonable bounds
	mph.clampThresholds()
}

func (mph *MemoryPressureHandler) clampThresholds() {
	// Ensure thresholds are within reasonable ranges
	clampHeapThreshold := func(threshold *Threshold, min, max float64) {
		if threshold.HeapUtilization < min {
			threshold.HeapUtilization = min
		}
		if threshold.HeapUtilization > max {
			threshold.HeapUtilization = max
		}
	}

	clampHeapThreshold(mph.thresholds.LowPressure, 30.0, 60.0)
	clampHeapThreshold(mph.thresholds.MediumPressure, 50.0, 80.0)
	clampHeapThreshold(mph.thresholds.HighPressure, 70.0, 90.0)
	clampHeapThreshold(mph.thresholds.CriticalPressure, 90.0, 98.0)
}

func (mph *MemoryPressureHandler) cleanupOldEvents() {
	mph.mu.Lock()
	defer mph.mu.Unlock()

	// Remove events older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)

	var recentEvents []*MemoryPressureEvent
	for _, event := range mph.events {
		if event.Timestamp.After(cutoff) {
			recentEvents = append(recentEvents, event)
		}
	}

	mph.events = recentEvents
}
