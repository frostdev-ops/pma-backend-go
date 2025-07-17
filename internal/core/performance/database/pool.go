package database

import (
	"database/sql"
	"sync"
	"time"
)

// PoolManager defines the interface for connection pool management
type PoolManager interface {
	GetOptimalPoolSize() int
	MonitorConnections() *PoolStats
	OptimizePool() error
	HandleLeaks() error
	GetConnectionHealth() map[string]HealthMetric
}

// PoolStats contains statistics about the connection pool
type PoolStats struct {
	ActiveConnections int           `json:"active_connections"`
	IdleConnections   int           `json:"idle_connections"`
	TotalConnections  int           `json:"total_connections"`
	MaxLifetime       time.Duration `json:"max_lifetime"`
	AverageUsage      time.Duration `json:"average_usage"`
	LeakedConnections int           `json:"leaked_connections"`
	WaitCount         int64         `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
	MaxIdleClosed     int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed int64         `json:"max_lifetime_closed"`
}

// HealthMetric represents the health status of a connection aspect
type HealthMetric struct {
	Status      string    `json:"status"` // healthy, warning, critical
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Description string    `json:"description"`
	LastCheck   time.Time `json:"last_check"`
}

// SQLitePoolManager implements PoolManager for SQLite databases
type SQLitePoolManager struct {
	db               *sql.DB
	config           *PoolConfig
	stats            *PoolStats
	healthMetrics    map[string]HealthMetric
	connectionUsage  map[string]time.Duration
	connectionTimes  map[string]time.Time
	mu               sync.RWMutex
	monitoringTicker *time.Ticker
	stopChan         chan bool
}

// PoolConfig contains configuration for connection pool optimization
type PoolConfig struct {
	MaxOpenConns        int           `json:"max_open_conns"`
	MaxIdleConns        int           `json:"max_idle_conns"`
	ConnMaxLifetime     time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime     time.Duration `json:"conn_max_idle_time"`
	MonitorInterval     time.Duration `json:"monitor_interval"`
	LeakThreshold       time.Duration `json:"leak_threshold"`
	OptimizationEnabled bool          `json:"optimization_enabled"`
}

// NewSQLitePoolManager creates a new SQLite pool manager
func NewSQLitePoolManager(db *sql.DB, config *PoolConfig) *SQLitePoolManager {
	if config == nil {
		config = &PoolConfig{
			MaxOpenConns:        25,
			MaxIdleConns:        10,
			ConnMaxLifetime:     time.Hour,
			ConnMaxIdleTime:     time.Minute * 10,
			MonitorInterval:     time.Second * 30,
			LeakThreshold:       time.Minute * 5,
			OptimizationEnabled: true,
		}
	}

	pm := &SQLitePoolManager{
		db:              db,
		config:          config,
		stats:           &PoolStats{},
		healthMetrics:   make(map[string]HealthMetric),
		connectionUsage: make(map[string]time.Duration),
		connectionTimes: make(map[string]time.Time),
		stopChan:        make(chan bool),
	}

	pm.initializePool()
	pm.startMonitoring()

	return pm
}

// initializePool sets up the connection pool with optimal settings
func (pm *SQLitePoolManager) initializePool() {
	pm.db.SetMaxOpenConns(pm.config.MaxOpenConns)
	pm.db.SetMaxIdleConns(pm.config.MaxIdleConns)
	pm.db.SetConnMaxLifetime(pm.config.ConnMaxLifetime)
	pm.db.SetConnMaxIdleTime(pm.config.ConnMaxIdleTime)
}

// startMonitoring begins continuous monitoring of the connection pool
func (pm *SQLitePoolManager) startMonitoring() {
	pm.monitoringTicker = time.NewTicker(pm.config.MonitorInterval)

	go func() {
		for {
			select {
			case <-pm.monitoringTicker.C:
				pm.updateStats()
				pm.updateHealthMetrics()
				if pm.config.OptimizationEnabled {
					pm.autoOptimize()
				}
			case <-pm.stopChan:
				pm.monitoringTicker.Stop()
				return
			}
		}
	}()
}

// updateStats updates the current pool statistics
func (pm *SQLitePoolManager) updateStats() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	stats := pm.db.Stats()

	pm.stats = &PoolStats{
		ActiveConnections: stats.OpenConnections - stats.Idle,
		IdleConnections:   stats.Idle,
		TotalConnections:  stats.OpenConnections,
		MaxLifetime:       pm.config.ConnMaxLifetime,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxIdleTimeClosed: stats.MaxIdleTimeClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}

	// Calculate average usage
	if len(pm.connectionUsage) > 0 {
		var total time.Duration
		for _, usage := range pm.connectionUsage {
			total += usage
		}
		pm.stats.AverageUsage = total / time.Duration(len(pm.connectionUsage))
	}

	// Count leaked connections
	pm.stats.LeakedConnections = pm.countLeakedConnections()
}

// countLeakedConnections identifies potentially leaked connections
func (pm *SQLitePoolManager) countLeakedConnections() int {
	leaked := 0
	now := time.Now()

	for _, startTime := range pm.connectionTimes {
		if now.Sub(startTime) > pm.config.LeakThreshold {
			leaked++
		}
	}

	return leaked
}

// updateHealthMetrics updates health metrics for the connection pool
func (pm *SQLitePoolManager) updateHealthMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()

	// Connection utilization health
	utilization := float64(pm.stats.ActiveConnections) / float64(pm.config.MaxOpenConns)
	pm.healthMetrics["connection_utilization"] = HealthMetric{
		Status:      pm.getHealthStatus(utilization, 0.8, 0.95),
		Value:       utilization,
		Threshold:   0.8,
		Description: "Percentage of connections in use",
		LastCheck:   now,
	}

	// Wait time health
	avgWaitTime := float64(pm.stats.WaitDuration) / float64(time.Millisecond)
	if pm.stats.WaitCount > 0 {
		avgWaitTime = avgWaitTime / float64(pm.stats.WaitCount)
	}
	pm.healthMetrics["wait_time"] = HealthMetric{
		Status:      pm.getHealthStatus(avgWaitTime, 10, 50), // ms
		Value:       avgWaitTime,
		Threshold:   10,
		Description: "Average wait time for connections (ms)",
		LastCheck:   now,
	}

	// Idle connections health
	idleRatio := float64(pm.stats.IdleConnections) / float64(pm.config.MaxIdleConns)
	pm.healthMetrics["idle_connections"] = HealthMetric{
		Status:      pm.getHealthStatus(1-idleRatio, 0.3, 0.7), // Inverted - too many idle is bad
		Value:       idleRatio,
		Threshold:   0.5,
		Description: "Ratio of idle connections",
		LastCheck:   now,
	}

	// Connection leaks health
	leakRatio := float64(pm.stats.LeakedConnections) / float64(pm.stats.TotalConnections+1)
	pm.healthMetrics["connection_leaks"] = HealthMetric{
		Status:      pm.getHealthStatus(1-leakRatio, 0.9, 0.95), // Inverted - leaks are bad
		Value:       leakRatio,
		Threshold:   0.1,
		Description: "Ratio of potentially leaked connections",
		LastCheck:   now,
	}
}

// getHealthStatus determines health status based on value and thresholds
func (pm *SQLitePoolManager) getHealthStatus(value, warningThreshold, criticalThreshold float64) string {
	if value <= warningThreshold {
		return "healthy"
	} else if value <= criticalThreshold {
		return "warning"
	}
	return "critical"
}

// GetOptimalPoolSize calculates the optimal pool size based on usage patterns
func (pm *SQLitePoolManager) GetOptimalPoolSize() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Base calculation on recent usage patterns
	stats := pm.db.Stats()

	// If we're frequently waiting, increase pool size
	if stats.WaitCount > 0 && pm.stats.WaitDuration > time.Millisecond*10 {
		return min(pm.config.MaxOpenConns+5, 50) // Cap at reasonable max
	}

	// If utilization is consistently low, decrease pool size
	utilization := float64(pm.stats.ActiveConnections) / float64(pm.config.MaxOpenConns)
	if utilization < 0.3 && pm.config.MaxOpenConns > 10 {
		return max(pm.config.MaxOpenConns-5, 5) // Maintain minimum
	}

	// If utilization is high, increase pool size
	if utilization > 0.8 {
		return min(pm.config.MaxOpenConns+5, 50)
	}

	return pm.config.MaxOpenConns
}

// MonitorConnections returns current pool statistics
func (pm *SQLitePoolManager) MonitorConnections() *PoolStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := *pm.stats
	return &statsCopy
}

// OptimizePool applies optimizations to the connection pool
func (pm *SQLitePoolManager) OptimizePool() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	optimalSize := pm.GetOptimalPoolSize()

	// Apply optimizations
	if optimalSize != pm.config.MaxOpenConns {
		pm.config.MaxOpenConns = optimalSize
		pm.db.SetMaxOpenConns(optimalSize)

		// Adjust idle connections proportionally
		pm.config.MaxIdleConns = max(optimalSize/3, 2)
		pm.db.SetMaxIdleConns(pm.config.MaxIdleConns)
	}

	// Optimize connection lifetimes based on usage patterns
	if pm.stats.AverageUsage > 0 {
		// If connections are used for short periods, reduce lifetime
		if pm.stats.AverageUsage < time.Minute*5 {
			pm.config.ConnMaxLifetime = time.Minute * 30
		} else {
			pm.config.ConnMaxLifetime = time.Hour
		}
		pm.db.SetConnMaxLifetime(pm.config.ConnMaxLifetime)
	}

	return nil
}

// HandleLeaks identifies and handles connection leaks
func (pm *SQLitePoolManager) HandleLeaks() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	leakedConnections := []string{}

	// Identify leaked connections
	for connID, startTime := range pm.connectionTimes {
		if now.Sub(startTime) > pm.config.LeakThreshold {
			leakedConnections = append(leakedConnections, connID)
		}
	}

	// Clean up tracking for leaked connections
	for _, connID := range leakedConnections {
		delete(pm.connectionTimes, connID)
		delete(pm.connectionUsage, connID)
	}

	// If too many leaks, force pool reset
	if len(leakedConnections) > pm.config.MaxOpenConns/4 {
		// Temporarily reduce pool size to force connection cycling
		originalMax := pm.config.MaxOpenConns
		pm.db.SetMaxOpenConns(max(originalMax/2, 1))

		// Wait a moment for connections to cycle
		time.Sleep(time.Second)

		// Restore original pool size
		pm.db.SetMaxOpenConns(originalMax)
	}

	return nil
}

// GetConnectionHealth returns health metrics for the connection pool
func (pm *SQLitePoolManager) GetConnectionHealth() map[string]HealthMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Create a copy of health metrics
	healthCopy := make(map[string]HealthMetric)
	for key, value := range pm.healthMetrics {
		healthCopy[key] = value
	}

	return healthCopy
}

// autoOptimize performs automatic optimizations based on current metrics
func (pm *SQLitePoolManager) autoOptimize() {
	// Check if optimization is needed
	health := pm.GetConnectionHealth()

	needsOptimization := false
	for _, metric := range health {
		if metric.Status == "critical" || metric.Status == "warning" {
			needsOptimization = true
			break
		}
	}

	if needsOptimization {
		pm.OptimizePool()
	}

	// Handle leaks if detected
	if pm.stats.LeakedConnections > 0 {
		pm.HandleLeaks()
	}
}

// TrackConnection starts tracking a connection usage
func (pm *SQLitePoolManager) TrackConnection(connID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.connectionTimes[connID] = time.Now()
}

// UntrackConnection stops tracking a connection and records its usage
func (pm *SQLitePoolManager) UntrackConnection(connID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if startTime, exists := pm.connectionTimes[connID]; exists {
		usage := time.Since(startTime)
		pm.connectionUsage[connID] = usage
		delete(pm.connectionTimes, connID)
	}
}

// Stop stops the pool manager and cleanup resources
func (pm *SQLitePoolManager) Stop() {
	close(pm.stopChan)
	if pm.monitoringTicker != nil {
		pm.monitoringTicker.Stop()
	}
}

// GetPoolReport generates a comprehensive pool performance report
func (pm *SQLitePoolManager) GetPoolReport() *PoolReport {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := pm.MonitorConnections()
	health := pm.GetConnectionHealth()

	report := &PoolReport{
		GeneratedAt:     time.Now(),
		Stats:           *stats,
		Health:          health,
		Configuration:   *pm.config,
		Recommendations: pm.generateRecommendations(),
	}

	return report
}

// PoolReport contains comprehensive pool analysis
type PoolReport struct {
	GeneratedAt     time.Time               `json:"generated_at"`
	Stats           PoolStats               `json:"stats"`
	Health          map[string]HealthMetric `json:"health"`
	Configuration   PoolConfig              `json:"configuration"`
	Recommendations []string                `json:"recommendations"`
}

// generateRecommendations generates optimization recommendations
func (pm *SQLitePoolManager) generateRecommendations() []string {
	recommendations := []string{}

	utilization := float64(pm.stats.ActiveConnections) / float64(pm.config.MaxOpenConns)

	if utilization > 0.9 {
		recommendations = append(recommendations, "Consider increasing MaxOpenConns - high utilization detected")
	}

	if pm.stats.WaitCount > 100 {
		recommendations = append(recommendations, "Frequent connection waits detected - consider increasing pool size")
	}

	if pm.stats.LeakedConnections > 0 {
		recommendations = append(recommendations, "Connection leaks detected - review connection handling in application code")
	}

	if pm.stats.IdleConnections > pm.config.MaxIdleConns*3/4 {
		recommendations = append(recommendations, "Many idle connections - consider reducing MaxIdleConns")
	}

	if pm.stats.MaxIdleTimeClosed > pm.stats.MaxLifetimeClosed*2 {
		recommendations = append(recommendations, "Consider reducing ConnMaxIdleTime")
	}

	return recommendations
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
