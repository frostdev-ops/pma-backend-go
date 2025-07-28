package monitor

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MemoryMonitor monitors memory usage and detects potential leaks
type MemoryMonitor struct {
	logger   *logrus.Logger
	mu       sync.RWMutex
	running  bool
	stopChan chan bool

	// Memory thresholds
	maxMemoryUsage uint64 // in bytes
	maxGoroutines  int
	maxHeapAlloc   uint64 // in bytes

	// Monitoring intervals
	checkInterval   time.Duration
	cleanupInterval time.Duration

	// Statistics
	lastCheck      time.Time
	memoryHistory  []MemorySnapshot
	maxHistorySize int

	// Callbacks
	onMemoryPressure func(uint64, uint64) // current, threshold
	onGoroutineLeak  func(int, int)       // current, threshold
}

// MemorySnapshot represents a point-in-time memory measurement
type MemorySnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	HeapAlloc    uint64    `json:"heap_alloc"`
	HeapSys      uint64    `json:"heap_sys"`
	HeapIdle     uint64    `json:"heap_idle"`
	HeapInuse    uint64    `json:"heap_inuse"`
	HeapReleased uint64    `json:"heap_released"`
	HeapObjects  uint64    `json:"heap_objects"`
	Goroutines   int       `json:"goroutines"`
	MemoryUsage  float64   `json:"memory_usage_percent"`
}

// MemoryMonitorConfig holds configuration for the memory monitor
type MemoryMonitorConfig struct {
	MaxMemoryUsage  uint64        `json:"max_memory_usage_bytes"`
	MaxGoroutines   int           `json:"max_goroutines"`
	MaxHeapAlloc    uint64        `json:"max_heap_alloc_bytes"`
	CheckInterval   time.Duration `json:"check_interval"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxHistorySize  int           `json:"max_history_size"`
}

// DefaultMemoryMonitorConfig returns default configuration
func DefaultMemoryMonitorConfig() *MemoryMonitorConfig {
	return &MemoryMonitorConfig{
		MaxMemoryUsage:  1024 * 1024 * 1024, // 1GB
		MaxGoroutines:   10000,
		MaxHeapAlloc:    512 * 1024 * 1024, // 512MB
		CheckInterval:   30 * time.Second,
		CleanupInterval: 5 * time.Minute,
		MaxHistorySize:  100,
	}
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(logger *logrus.Logger, config *MemoryMonitorConfig) *MemoryMonitor {
	if config == nil {
		config = DefaultMemoryMonitorConfig()
	}

	return &MemoryMonitor{
		logger:          logger,
		stopChan:        make(chan bool),
		maxMemoryUsage:  config.MaxMemoryUsage,
		maxGoroutines:   config.MaxGoroutines,
		maxHeapAlloc:    config.MaxHeapAlloc,
		checkInterval:   config.CheckInterval,
		cleanupInterval: config.CleanupInterval,
		maxHistorySize:  config.MaxHistorySize,
		memoryHistory:   make([]MemorySnapshot, 0),
	}
}

// Start starts the memory monitor
func (m *MemoryMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.running = true
	m.logger.Info("Memory monitor started")

	go m.monitorLoop(ctx)
	go m.cleanupLoop(ctx)

	return nil
}

// Stop stops the memory monitor
func (m *MemoryMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	close(m.stopChan)
	m.logger.Info("Memory monitor stopped")
	
	// Create a new stopChan for potential restart
	m.stopChan = make(chan bool)
}

// monitorLoop is the main monitoring loop
func (m *MemoryMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkMemory()
		}
	}
}

// cleanupLoop performs periodic cleanup
func (m *MemoryMonitor) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// checkMemory checks current memory usage and detects potential issues
func (m *MemoryMonitor) checkMemory() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	snapshot := MemorySnapshot{
		Timestamp:    time.Now(),
		HeapAlloc:    memStats.HeapAlloc,
		HeapSys:      memStats.HeapSys,
		HeapIdle:     memStats.HeapIdle,
		HeapInuse:    memStats.HeapInuse,
		HeapReleased: memStats.HeapReleased,
		HeapObjects:  memStats.HeapObjects,
		Goroutines:   runtime.NumGoroutine(),
		MemoryUsage:  float64(memStats.HeapAlloc) / float64(m.maxMemoryUsage) * 100,
	}

	m.mu.Lock()
	m.memoryHistory = append(m.memoryHistory, snapshot)
	if len(m.memoryHistory) > m.maxHistorySize {
		m.memoryHistory = m.memoryHistory[1:]
	}
	m.lastCheck = time.Now()
	m.mu.Unlock()

	// Check for memory pressure
	if memStats.HeapAlloc > m.maxHeapAlloc {
		m.logger.WithFields(logrus.Fields{
			"heap_alloc":     memStats.HeapAlloc,
			"max_heap_alloc": m.maxHeapAlloc,
			"usage_percent":  snapshot.MemoryUsage,
		}).Warn("Memory pressure detected - heap allocation exceeded threshold")

		if m.onMemoryPressure != nil {
			m.onMemoryPressure(memStats.HeapAlloc, m.maxHeapAlloc)
		}
	}

	// Check for goroutine leaks
	if snapshot.Goroutines > m.maxGoroutines {
		m.logger.WithFields(logrus.Fields{
			"goroutines":     snapshot.Goroutines,
			"max_goroutines": m.maxGoroutines,
		}).Warn("Goroutine leak detected - too many goroutines")

		if m.onGoroutineLeak != nil {
			m.onGoroutineLeak(snapshot.Goroutines, m.maxGoroutines)
		}
	}

	// Log memory statistics periodically
	m.logger.WithFields(logrus.Fields{
		"heap_alloc":    memStats.HeapAlloc,
		"heap_sys":      memStats.HeapSys,
		"heap_objects":  memStats.HeapObjects,
		"goroutines":    snapshot.Goroutines,
		"usage_percent": snapshot.MemoryUsage,
	}).Debug("Memory monitor check completed")
}

// cleanup performs memory cleanup operations
func (m *MemoryMonitor) cleanup() {
	// Force garbage collection
	runtime.GC()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.logger.WithFields(logrus.Fields{
		"heap_alloc":   memStats.HeapAlloc,
		"heap_sys":     memStats.HeapSys,
		"heap_objects": memStats.HeapObjects,
		"goroutines":   runtime.NumGoroutine(),
	}).Info("Memory cleanup completed")
}

// GetMemoryStats returns current memory statistics
func (m *MemoryMonitor) GetMemoryStats() *MemorySnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.memoryHistory) == 0 {
		return nil
	}

	return &m.memoryHistory[len(m.memoryHistory)-1]
}

// GetMemoryHistory returns the memory history
func (m *MemoryMonitor) GetMemoryHistory() []MemorySnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history := make([]MemorySnapshot, len(m.memoryHistory))
	copy(history, m.memoryHistory)
	return history
}

// SetMemoryPressureCallback sets the callback for memory pressure events
func (m *MemoryMonitor) SetMemoryPressureCallback(callback func(uint64, uint64)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onMemoryPressure = callback
}

// SetGoroutineLeakCallback sets the callback for goroutine leak events
func (m *MemoryMonitor) SetGoroutineLeakCallback(callback func(int, int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onGoroutineLeak = callback
}

// ForceGC forces garbage collection
func (m *MemoryMonitor) ForceGC() {
	runtime.GC()
	m.logger.Info("Forced garbage collection")
}

// GetMemoryUsage returns current memory usage as a percentage
func (m *MemoryMonitor) GetMemoryUsage() float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return float64(memStats.HeapAlloc) / float64(m.maxMemoryUsage) * 100
}
